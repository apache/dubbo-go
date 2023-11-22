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
	"strings"
)

import (
	"github.com/dubbogo/go-zookeeper/zk"

	gxset "github.com/dubbogo/gost/container/set"
	gxzookeeper "github.com/dubbogo/gost/database/kv/zk"
	"github.com/dubbogo/gost/log/logger"

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

var emptyStrSlice = make([]string, 0)

func init() {
	mf := &zookeeperMetadataReportFactory{}
	extension.SetMetadataReportFactory("zookeeper", func() report.MetadataReportFactory {
		return mf
	})
}

// zookeeperMetadataReport is the implementation of
// MetadataReport based on zookeeper.
type zookeeperMetadataReport struct {
	client        *gxzookeeper.ZookeeperClient
	rootDir       string
	listener      *zookeeper.ZkEventListener
	cacheListener *CacheListener
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
		logger.Debugf("Try to create the node data failed. In most cases, it's not a problem. ")
		return nil
	}
	return err
}

// RegisterServiceAppMapping map the specified Dubbo service interface to current Dubbo app name
func (m *zookeeperMetadataReport) RegisterServiceAppMapping(key string, group string, value string) error {
	path := m.rootDir + group + constant.PathSeparator + key
	v, state, err := m.client.GetContent(path)
	if err == zk.ErrNoNode {
		return m.client.CreateWithValue(path, []byte(value))
	} else if err != nil {
		return err
	}
	oldValue := string(v)
	if strings.Contains(oldValue, value) {
		return nil
	}
	value = oldValue + constant.CommaSeparator + value
	_, err = m.client.SetContent(path, []byte(value), state.Version)
	return err
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
	appNames := strings.Split(string(v), constant.CommaSeparator)
	set := gxset.NewSet()
	for _, e := range appNames {
		set.Add(e)
	}
	return set, nil
}

// GetConfigKeysByGroup will return all keys with the group
func (m *zookeeperMetadataReport) GetConfigKeysByGroup(group string) (*gxset.HashSet, error) {
	path := m.rootDir + group
	result, err := m.client.GetChildren(path)
	if err != nil {
		return nil, perrors.WithStack(err)
	}

	if len(result) == 0 {
		return nil, perrors.New("could not find keys with group: " + group)
	}
	set := gxset.NewSet()
	for _, e := range result {
		set.Add(e)
	}
	return set, nil
}

func (m *zookeeperMetadataReport) RemoveServiceAppMappingListener(key string, group string) error {
	return nil
}

type zookeeperMetadataReportFactory struct{}

// nolint
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
		client:   client,
		rootDir:  rootDir,
		listener: zookeeper.NewZkEventListener(client),
	}

	reporter.cacheListener = NewCacheListener(rootDir, reporter.listener)
	reporter.listener.ListenConfigurationEvent(rootDir+constant.PathSeparator+metadata.DefaultGroup, reporter.cacheListener)
	return reporter
}
