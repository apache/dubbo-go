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
	"strings"
)

import (
	gxset "github.com/dubbogo/gost/container/set"
	gxetcd "github.com/dubbogo/gost/database/kv/etcd/v3"
	"github.com/dubbogo/gost/log/logger"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
	"dubbo.apache.org/dubbo-go/v3/metadata/mapping"
	"dubbo.apache.org/dubbo-go/v3/metadata/report"
)

const DEFAULT_ROOT = "dubbo"

func init() {
	extension.SetMetadataReportFactory(constant.EtcdV3Key, func() report.MetadataReportFactory {
		return &etcdMetadataReportFactory{}
	})
}

// etcdMetadataReport is the implementation of MetadataReport based etcd
type etcdMetadataReport struct {
	client  *gxetcd.Client
	rootDir string
}

// GetAppMetadata get metadata info from etcd
func (e *etcdMetadataReport) GetAppMetadata(application, revision string) (*info.MetadataInfo, error) {
	key := e.rootDir + application + constant.PathSeparator + revision
	data, err := e.client.Get(key)
	if err != nil {
		return nil, err
	}

	meta := &info.MetadataInfo{}
	return meta, json.Unmarshal([]byte(data), meta)
}

// PublishAppMetadata publish metadata info to etcd
func (e *etcdMetadataReport) PublishAppMetadata(application, revision string, info *info.MetadataInfo) error {
	key := e.rootDir + application + constant.PathSeparator + revision
	value, err := json.Marshal(info)
	if err == nil {
		err = e.client.Put(key, string(value))
	}

	return err
}

// RegisterServiceAppMapping map the specified Dubbo service interface to current Dubbo app name
func (e *etcdMetadataReport) RegisterServiceAppMapping(key string, group string, value string) error {
	path := e.rootDir + constant.PathSeparator + group + constant.PathSeparator + key
	oldVal, err := e.client.Get(path)
	if perrors.Cause(err) == gxetcd.ErrKVPairNotFound {
		return e.client.Put(path, value)
	} else if err != nil {
		return err
	}
	if strings.Contains(oldVal, value) {
		return nil
	}
	value = oldVal + constant.CommaSeparator + value
	return e.client.Put(path, value)
}

// GetServiceAppMapping get the app names from the specified Dubbo service interface
func (e *etcdMetadataReport) GetServiceAppMapping(key string, group string, listener mapping.MappingListener) (*gxset.HashSet, error) {
	path := e.rootDir + constant.PathSeparator + group + constant.PathSeparator + key
	v, err := e.client.Get(path)
	if err != nil {
		return nil, err
	}
	appNames := strings.Split(v, constant.CommaSeparator)
	set := gxset.NewSet()
	for _, app := range appNames {
		set.Add(app)
	}
	return set, nil
}

func (e *etcdMetadataReport) RemoveServiceAppMappingListener(key string, group string) error {
	return nil
}

type etcdMetadataReportFactory struct{}

// CreateMetadataReport get the MetadataReport instance of etcd
func (e *etcdMetadataReportFactory) CreateMetadataReport(url *common.URL) report.MetadataReport {
	timeout := url.GetParamDuration(constant.TimeoutKey, constant.DefaultRegTimeout)
	addresses := strings.Split(url.Location, ",")
	client, err := gxetcd.NewClient(gxetcd.MetadataETCDV3Client, addresses, timeout, 1)
	if err != nil {
		logger.Errorf("Could not create etcd metadata report. URL: %s,error:{%v}", url.String(), err)
		return nil
	}
	group := url.GetParam(constant.MetadataReportGroupKey, DEFAULT_ROOT)
	group = constant.PathSeparator + strings.TrimPrefix(group, constant.PathSeparator)
	return &etcdMetadataReport{client: client, rootDir: group}
}
