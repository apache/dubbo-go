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
	"strings"
	"time"
)

import (
	gxetcd "github.com/dubbogo/gost/database/kv/etcd/v3"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/metadata/identifier"
	"dubbo.apache.org/dubbo-go/v3/metadata/report"
	"dubbo.apache.org/dubbo-go/v3/metadata/report/factory"
)

const DEFAULT_ROOT = "dubbo"

func init() {
	extension.SetMetadataReportFactory(constant.ETCDV3_KEY, func() factory.MetadataReportFactory {
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
	// TODO will implement
	panic("implement me")
}

// PublishAppMetadata publish metadata info to etcd
func (e *etcdMetadataReport) PublishAppMetadata(metadataIdentifier *identifier.SubscriberMetadataIdentifier, info *common.MetadataInfo) error {
	// TODO will implement
	panic("implement me")
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

type etcdMetadataReportFactory struct{}

// CreateMetadataReport get the MetadataReport instance of etcd
func (e *etcdMetadataReportFactory) CreateMetadataReport(url *common.URL) report.MetadataReport {
	timeout, _ := time.ParseDuration(url.GetParam(constant.REGISTRY_TIMEOUT_KEY, constant.DEFAULT_REG_TIMEOUT))
	addresses := strings.Split(url.Location, ",")
	client, err := gxetcd.NewClient(gxetcd.MetadataETCDV3Client, addresses, timeout, 1)
	if err != nil {
		logger.Errorf("Could not create etcd metadata report. URL: %s,error:{%v}", url.String(), err)
		return nil
	}
	group := url.GetParam(constant.GROUP_KEY, DEFAULT_ROOT)
	group = constant.PATH_SEPARATOR + strings.TrimPrefix(group, constant.PATH_SEPARATOR)
	return &etcdMetadataReport{client: client, root: group}
}

func (e *etcdMetadataReport) getNodeKey(MetadataIdentifier identifier.IMetadataIdentifier) string {
	var rootDir string
	if e.root == constant.PATH_SEPARATOR {
		rootDir = e.root
	} else {
		rootDir = e.root + constant.PATH_SEPARATOR
	}
	return rootDir + MetadataIdentifier.GetFilePathKey()
}
