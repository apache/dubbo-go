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
	"dubbo.apache.org/dubbo-go/v3/metadata/identifier"
	"dubbo.apache.org/dubbo-go/v3/metadata/report"
	"dubbo.apache.org/dubbo-go/v3/metadata/report/factory"
)

const DEFAULT_ROOT = "dubbo"

func init() {
	extension.SetMetadataReportFactory(constant.EtcdV3Key, func() factory.MetadataReportFactory {
		return &etcdMetadataReportFactory{}
	})
}

// etcdMetadataReport is the implementation of MetadataReport based etcd
type etcdMetadataReport struct {
	client *gxetcd.Client
	root   string
}

// GetAppMetadata get metadata info from etcd
func (e *etcdMetadataReport) GetAppMetadata(metadataIdentifier *identifier.SubscriberMetadataIdentifier) (*common.MetadataInfo, error) {
	key := e.getNodeKey(metadataIdentifier)
	data, err := e.client.Get(key)
	if err != nil {
		return nil, err
	}

	info := &common.MetadataInfo{}
	return info, json.Unmarshal([]byte(data), info)
}

// PublishAppMetadata publish metadata info to etcd
func (e *etcdMetadataReport) PublishAppMetadata(metadataIdentifier *identifier.SubscriberMetadataIdentifier, info *common.MetadataInfo) error {
	key := e.getNodeKey(metadataIdentifier)
	value, err := json.Marshal(info)
	if err == nil {
		err = e.client.Put(key, string(value))
	}

	return err
}

// StoreProviderMetadata will store the metadata
// metadata including the basic info of the server, provider info, and other user custom info
func (e *etcdMetadataReport) StoreProviderMetadata(providerIdentifier *identifier.MetadataIdentifier, serviceDefinitions string) error {
	key := e.getNodeKey(providerIdentifier)
	return e.client.Put(key, serviceDefinitions)
}

// StoreConsumerMetadata will store the metadata
// metadata including the basic info of the server, consumer info, and other user custom info
func (e *etcdMetadataReport) StoreConsumerMetadata(consumerMetadataIdentifier *identifier.MetadataIdentifier, serviceParameterString string) error {
	key := e.getNodeKey(consumerMetadataIdentifier)
	return e.client.Put(key, serviceParameterString)
}

// SaveServiceMetadata will store the metadata
// metadata including the basic info of the server, service info, and other user custom info
func (e *etcdMetadataReport) SaveServiceMetadata(metadataIdentifier *identifier.ServiceMetadataIdentifier, url *common.URL) error {
	key := e.getNodeKey(metadataIdentifier)
	return e.client.Put(key, url.String())
}

// RemoveServiceMetadata will remove the service metadata
func (e *etcdMetadataReport) RemoveServiceMetadata(metadataIdentifier *identifier.ServiceMetadataIdentifier) error {
	return e.client.Delete(e.getNodeKey(metadataIdentifier))
}

// GetExportedURLs will look up the exported urls.
// if not found, an empty list will be returned.
func (e *etcdMetadataReport) GetExportedURLs(metadataIdentifier *identifier.ServiceMetadataIdentifier) ([]string, error) {
	content, err := e.client.Get(e.getNodeKey(metadataIdentifier))
	if err != nil {
		logger.Errorf("etcdMetadataReport GetExportedURLs err:{%v}", err.Error())
		return []string{}, err
	}
	if content == "" {
		return []string{}, nil
	}
	return []string{content}, nil
}

// SaveSubscribedData will convert the urlList to json array and then store it
func (e *etcdMetadataReport) SaveSubscribedData(subscriberMetadataIdentifier *identifier.SubscriberMetadataIdentifier, urls string) error {
	key := e.getNodeKey(subscriberMetadataIdentifier)
	return e.client.Put(key, urls)
}

// GetSubscribedURLs will lookup the url
// if not found, an empty list will be returned
func (e *etcdMetadataReport) GetSubscribedURLs(subscriberMetadataIdentifier *identifier.SubscriberMetadataIdentifier) ([]string, error) {
	content, err := e.client.Get(e.getNodeKey(subscriberMetadataIdentifier))
	if err != nil {
		logger.Errorf("etcdMetadataReport GetSubscribedURLs err:{%v}", err.Error())
		return nil, err
	}
	return []string{content}, nil
}

// GetServiceDefinition will lookup the service definition
func (e *etcdMetadataReport) GetServiceDefinition(metadataIdentifier *identifier.MetadataIdentifier) (string, error) {
	key := e.getNodeKey(metadataIdentifier)
	content, err := e.client.Get(key)
	if err != nil {
		logger.Errorf("etcdMetadataReport GetServiceDefinition err:{%v}", err.Error())
		return "", err
	}
	return content, nil
}

// RegisterServiceAppMapping map the specified Dubbo service interface to current Dubbo app name
func (e *etcdMetadataReport) RegisterServiceAppMapping(key string, group string, value string) error {
	path := e.root + constant.PathSeparator + group + constant.PathSeparator + key
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
func (e *etcdMetadataReport) GetServiceAppMapping(key string, group string) (*gxset.HashSet, error) {
	path := e.root + constant.PathSeparator + group + constant.PathSeparator + key
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
	group := url.GetParam(constant.GroupKey, DEFAULT_ROOT)
	group = constant.PathSeparator + strings.TrimPrefix(group, constant.PathSeparator)
	return &etcdMetadataReport{client: client, root: group}
}

func (e *etcdMetadataReport) getNodeKey(MetadataIdentifier identifier.IMetadataIdentifier) string {
	var rootDir string
	if e.root == constant.PathSeparator {
		rootDir = e.root
	} else {
		rootDir = e.root + constant.PathSeparator
	}
	return rootDir + MetadataIdentifier.GetFilePathKey()
}
