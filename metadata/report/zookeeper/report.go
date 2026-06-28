/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package zookeeper

import (
	"encoding/json"
	"fmt"
	"strings"
)

import (
	"github.com/dubbogo/go-zookeeper/zk"

	gxset "github.com/dubbogo/gost/container/set"
	gxzookeeper "github.com/dubbogo/gost/database/kv/zk"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
	"dubbo.apache.org/dubbo-go/v3/metadata/mapping"
	"dubbo.apache.org/dubbo-go/v3/metadata/mapping/metadata"
	"dubbo.apache.org/dubbo-go/v3/metadata/report"
	"dubbo.apache.org/dubbo-go/v3/remoting/zookeeper"
)

// zkClient abstracts the ZookeeperClient operations used by zookeeperMetadataReport.
type zkClient interface {
	GetContent(path string) ([]byte, *zk.Stat, error)
	SetContent(path string, data []byte, version int32) (*zk.Stat, error)
	CreateWithValue(path string, data []byte) error
	Delete(path string) error
	Exists(path string) (bool, *zk.Stat, error)
	Children(path string) ([]string, *zk.Stat, error)
	Get(path string) ([]byte, *zk.Stat, error)
}

// zkClientWrapper wraps *gxzookeeper.ZookeeperClient to implement zkClient.
// Methods that exist on Conn (Exists, Children, Get) are delegated.
type zkClientWrapper struct {
	*gxzookeeper.ZookeeperClient
}

func (w zkClientWrapper) Exists(path string) (bool, *zk.Stat, error) {
	return w.Conn.Exists(path)
}

func (w zkClientWrapper) Children(path string) ([]string, *zk.Stat, error) {
	return w.Conn.Children(path)
}

func (w zkClientWrapper) Get(path string) ([]byte, *zk.Stat, error) {
	return w.Conn.Get(path)
}

func init() {
	mf := &zookeeperMetadataReportFactory{}
	extension.SetMetadataReportFactory("zookeeper", func() report.MetadataReportFactory {
		return mf
	})
}

// zookeeperMetadataReport is the implementation of
// MetadataReport based on zookeeper.
type zookeeperMetadataReport struct {
	client        zkClient
	rootDir       string
	listener      *zookeeper.ZkEventListener
	cacheListener *CacheListener
	url           *common.URL
}

// URL returns the URL used to create this metadata report.
func (m *zookeeperMetadataReport) URL() *common.URL {
	return m.url
}

// GetAppMetadata get metadata info from zookeeper
func (m *zookeeperMetadataReport) GetAppMetadata(application, revision string) (*info.MetadataInfo, error) {
	k := m.rootDir + application + constant.PathSeparator + revision
	data, _, err := m.client.GetContent(k)
	if err != nil {
		return nil, err
	}
	var metadataInfo info.MetadataInfo
	err = json.Unmarshal(data, &metadataInfo)
	if err != nil {
		return nil, err
	}
	return &metadataInfo, nil
}

// PublishAppMetadata publish metadata info to zookeeper
func (m *zookeeperMetadataReport) PublishAppMetadata(application, revision string, meta *info.MetadataInfo) error {
	k := m.rootDir + application + constant.PathSeparator + revision
	data, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	err = m.client.CreateWithValue(k, data)
	if perrors.Is(err, zk.ErrNodeExists) {
		_, err = m.client.SetContent(k, data, -1)
	}
	return err
}

// UnPublishAppMetadata removes metadata for a specific revision from zookeeper.
// This operation is idempotent.
func (m *zookeeperMetadataReport) UnPublishAppMetadata(application, revision string) error {
	k := m.rootDir + application + constant.PathSeparator + revision
	err := m.client.Delete(k)
	if perrors.Is(err, zk.ErrNoNode) {
		return nil
	}
	return err
}

// ListAppRevisions lists all stored revisions for an application from zookeeper.
func (m *zookeeperMetadataReport) ListAppRevisions(application string) ([]report.AppRevision, error) {
	parent := m.rootDir + application
	children, _, err := m.client.Children(parent)
	if err != nil {
		if perrors.Is(err, zk.ErrNoNode) {
			return nil, nil
		}
		return nil, err
	}
	result := make([]report.AppRevision, 0, len(children))
	for _, rev := range children {
		path := parent + constant.PathSeparator + rev
		data, _, err := m.client.Get(path)
		if err != nil {
			continue // skip if node disappeared between listing and reading
		}
		result = append(result, report.AppRevision{
			Revision:   rev,
			ModifyTime: report.ParseMetadataLastUpdatedTime(data),
		})
	}
	return result, nil
}

// RegisterServiceAppMapping map the specified Dubbo service interface to current Dubbo app name
func (m *zookeeperMetadataReport) RegisterServiceAppMapping(key string, group string, value string) error {
	path := m.rootDir + group + constant.PathSeparator + key
	v, state, err := m.client.GetContent(path)
	if perrors.Is(err, zk.ErrNoNode) {
		if cErr := m.client.CreateWithValue(path, []byte(value)); cErr != nil {
			if perrors.Is(cErr, zk.ErrNodeExists) {
				return fmt.Errorf("create mapping %s: %w", path, report.ErrMappingCASConflict)
			}
			return cErr
		}
		return nil
	} else if err != nil {
		return err
	}
	merged, changed := report.MergeServiceAppMapping(string(v), value)
	if !changed {
		return nil
	}
	if _, sErr := m.client.SetContent(path, []byte(merged), state.Version); sErr != nil {
		if perrors.Is(sErr, zk.ErrBadVersion) {
			return fmt.Errorf("update mapping %s: %w", path, report.ErrMappingCASConflict)
		}
		return sErr
	}
	return nil
}

// GetServiceAppMapping get the app names from the specified Dubbo service interface
func (m *zookeeperMetadataReport) GetServiceAppMapping(key string, group string, listener mapping.MappingListener) (*gxset.HashSet, error) {
	path := m.rootDir + group + constant.PathSeparator + key

	// listen to mapping changes first
	if listener != nil {
		m.cacheListener.AddListener(path, listener)
	}

	v, _, err := m.client.GetContent(path)
	if err != nil {
		return nil, err
	}
	return report.DecodeServiceAppNames(string(v)), nil
}

func (m *zookeeperMetadataReport) RemoveServiceAppMappingListener(key string, group string) error {
	path := m.rootDir + group + constant.PathSeparator + key
	m.cacheListener.RemoveKeyListeners(path)
	return nil
}

type zookeeperMetadataReportFactory struct{}

// CreateMetadataReport creates the zookeeper-based metadata report implementation.
func (mf *zookeeperMetadataReportFactory) CreateMetadataReport(url *common.URL) report.MetadataReport {
	client, err := gxzookeeper.NewZookeeperClient(
		"zookeeperMetadataReport",
		strings.Split(url.Location, ","),
		false,
		gxzookeeper.WithZkTimeOut(url.GetParamDuration(constant.TimeoutKey, "15s")),
	)
	if err != nil {
		panic(err)
	}

	rootDir := url.GetParam(constant.MetadataReportGroupKey, "dubbo")
	if !strings.HasPrefix(rootDir, constant.PathSeparator) {
		rootDir = constant.PathSeparator + rootDir
	}
	if rootDir != constant.PathSeparator {
		rootDir = rootDir + constant.PathSeparator
	}

	reporter := &zookeeperMetadataReport{
		client:   zkClientWrapper{client},
		rootDir:  rootDir,
		listener: zookeeper.NewZkEventListener(client),
		url:      url,
	}

	reporter.cacheListener = NewCacheListener(rootDir, reporter.listener)
	reporter.listener.ListenConfigurationEvent(rootDir+metadata.DefaultGroup, reporter.cacheListener)
	return reporter
}
