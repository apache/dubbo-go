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

package etcd

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	gxset "github.com/dubbogo/gost/container/set"

	gxetcd "github.com/dubbogo/gost/database/kv/etcd/v3"
	"github.com/dubbogo/gost/log/logger"

	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
	"dubbo.apache.org/dubbo-go/v3/metadata/mapping"
	"dubbo.apache.org/dubbo-go/v3/metadata/report"
	perrors "github.com/pkg/errors"
)

// etcdClient abstracts the etcd Client operations used by etcdMetadataReport.
type etcdClient interface {
	Get(key string) (string, error)
	Put(key, value string) error
	Delete(key string) error
	GetChildren(key string) ([]string, []string, error)
}

// etcdClientWrapper wraps *gxetcd.Client to implement etcdClient.
type etcdClientWrapper struct {
	*gxetcd.Client
}

func (w etcdClientWrapper) Put(key, value string) error {
	return w.Client.Put(key, value)
}

const DEFAULT_ROOT = "dubbo"

func init() {
	extension.SetMetadataReportFactory(constant.EtcdV3Key, func() report.MetadataReportFactory {
		return &etcdMetadataReportFactory{}
	})
}

// etcdMetadataReport is the implementation of MetadataReport based etcd
type etcdMetadataReport struct {
	client  etcdClient
	rootDir string
	url     *common.URL
}

// URL returns the URL used to create this metadata report.
func (e *etcdMetadataReport) URL() *common.URL {
	return e.url
}

// GetAppMetadata get metadata info from etcd
func (e *etcdMetadataReport) GetAppMetadata(application, revision string) (*info.MetadataInfo, error) {
	key := e.rootDir + constant.PathSeparator + application + constant.PathSeparator + revision
	data, err := e.client.Get(key)
	if err != nil {
		return nil, err
	}

	meta := &info.MetadataInfo{}
	return meta, json.Unmarshal([]byte(data), meta)
}

// PublishAppMetadata publish metadata info to etcd
func (e *etcdMetadataReport) PublishAppMetadata(application, revision string, info *info.MetadataInfo) error {
	key := e.rootDir + constant.PathSeparator + application + constant.PathSeparator + revision
	value, err := json.Marshal(info)
	if err == nil {
		err = e.client.Put(key, string(value))
	}

	return err
}

// RegisterServiceAppMapping map the specified Dubbo service interface to current Dubbo app name
func (e *etcdMetadataReport) RegisterServiceAppMapping(key string, group string, value string) error {
	path := e.rootDir + constant.PathSeparator + group + constant.PathSeparator + key
	oldVal, rev, err := e.client.GetValAndRev(path)
	if perrors.Cause(err) == gxetcd.ErrKVPairNotFound {
		if cErr := e.client.Create(path, value); cErr != nil {
			if perrors.Cause(cErr) == gxetcd.ErrCompareFail {
				return fmt.Errorf("create mapping %s: %w", path, report.ErrMappingCASConflict)
			}
			return cErr
		}
		return nil
	} else if err != nil {
		return err
	}
	merged, changed := report.MergeServiceAppMapping(oldVal, value)
	if !changed {
		return nil
	}
	if uErr := e.client.UpdateWithRev(path, merged, rev); uErr != nil {
		if perrors.Cause(uErr) == gxetcd.ErrCompareFail {
			return fmt.Errorf("update mapping %s: %w", path, report.ErrMappingCASConflict)
		}
		return uErr
	}
	return nil
}

// GetServiceAppMapping get the app names from the specified Dubbo service interface
func (e *etcdMetadataReport) GetServiceAppMapping(key string, group string, listener mapping.MappingListener) (*gxset.HashSet, error) {
	path := e.rootDir + constant.PathSeparator + group + constant.PathSeparator + key
	if listener != nil {
		logger.Warnf("etcd metadata report does not support service mapping listener, "+
			"mapping changes of %s will not be notified", path)
	}
	v, err := e.client.Get(path)
	if err != nil {
		return nil, err
	}
	return report.DecodeServiceAppNames(v), nil
}

// RemoveServiceAppMappingListener is a no-op: etcd metadata report does not register a mapping
// listener (see GetServiceAppMapping), so there is nothing to remove.
func (e *etcdMetadataReport) RemoveServiceAppMappingListener(key string, group string) error {
	return nil
}

// UnPublishAppMetadata removes metadata for a specific revision from etcd.
// This operation is idempotent.
func (e *etcdMetadataReport) UnPublishAppMetadata(application, revision string) error {
	key := e.rootDir + constant.PathSeparator + application + constant.PathSeparator + revision
	return e.client.Delete(key)
}

func (e *etcdMetadataReport) ListAppRevisions(application string) ([]report.AppRevision, error) {
	prefix := e.rootDir + constant.PathSeparator + application + constant.PathSeparator
	keys, values, err := e.client.GetChildren(prefix)
	if err != nil {
		if errors.Is(perrors.Cause(err), gxetcd.ErrKVPairNotFound) {
			return nil, nil
		}
		return nil, err
	}

	result := make([]report.AppRevision, 0, len(keys))
	for i, key := range keys {
		// Extract revision from key suffix (key is full path, revision is last segment)
		revision := key[strings.LastIndex(key, constant.PathSeparator)+1:]
		result = append(result, report.AppRevision{
			Revision:   revision,
			ModifyTime: report.ParseMetadataLastUpdatedTime([]byte(values[i])),
		})
	}
	return result, nil
}

type etcdMetadataReportFactory struct{}

// CreateMetadataReport get the MetadataReport instance of etcd
func (e *etcdMetadataReportFactory) CreateMetadataReport(url *common.URL) report.MetadataReport {
	timeout := url.GetParamDuration(constant.TimeoutKey, constant.DefaultRegTimeout)
	addresses := strings.Split(url.Location, ",")
	client, err := gxetcd.NewClient(gxetcd.MetadataETCDV3Client, addresses, timeout, 1)
	if err != nil {
		logger.Errorf("[Metadata][Etcd] could not create etcd metadata report, url=%s, err=%v", url.String(), err)
		return nil
	}
	group := url.GetParam(constant.MetadataReportGroupKey, DEFAULT_ROOT)
	group = constant.PathSeparator + strings.TrimPrefix(group, constant.PathSeparator)
	return &etcdMetadataReport{client: etcdClientWrapper{client}, rootDir: group, url: url}
}
