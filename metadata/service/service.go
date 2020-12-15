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

package service

import (
	"sync"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/registry"
)

// MetadataService is used to define meta data related behaviors
// usually the implementation should be singleton
type MetadataService interface {
	common.RPCService
	// ServiceName will get the service's name in meta service , which is application name
	ServiceName() (string, error)
	// ExportURL will store the exported url in metadata
	ExportURL(url *common.URL) (bool, error)
	// UnexportURL will delete the exported url in metadata
	UnexportURL(url *common.URL) error
	// SubscribeURL will store the subscribed url in metadata
	SubscribeURL(url *common.URL) (bool, error)
	// UnsubscribeURL will delete the subscribed url in metadata
	UnsubscribeURL(url *common.URL) error
	// PublishServiceDefinition will generate the target url's code info
	PublishServiceDefinition(url *common.URL) error
	// GetExportedURLs will get the target exported url in metadata
	// the url should be unique
	// due to dubbo-go only support return array []interface{} in RPCService, so we should declare the return type as []interface{}
	// actually, it's []String
	GetExportedURLs(serviceInterface string, group string, version string, protocol string) ([]interface{}, error)

	MethodMapper() map[string]string

	// GetExportedURLs will get the target subscribed url in metadata
	// the url should be unique
	GetSubscribedURLs() ([]*common.URL, error)
	// GetServiceDefinition will get the target service info store in metadata
	GetServiceDefinition(interfaceName string, group string, version string) (string, error)
	// GetServiceDefinition will get the target service info store in metadata by service key
	GetServiceDefinitionByServiceKey(serviceKey string) (string, error)
	// RefreshMetadata will refresh the metadata
	RefreshMetadata(exportedRevision string, subscribedRevision string) (bool, error)
	// Version will return the metadata service version
	Version() (string, error)
}

// BaseMetadataService is used for the event logic for struct who will implement interface MetadataService
type BaseMetadataService struct {
	serviceName string
}

func NewBaseMetadataService(serviceName string) BaseMetadataService {
	return BaseMetadataService{
		serviceName: serviceName,
	}
}

func (mts *BaseMetadataService) MethodMapper() map[string]string {
	return map[string]string{
		"GetExportedURLs": "getExportedURLs",
	}
}

// ServiceName can get the service's name in meta service , which is application name
func (mts *BaseMetadataService) ServiceName() (string, error) {
	return mts.serviceName, nil
}

// Version will return the version of metadata service
func (mts *BaseMetadataService) Reference() string {
	return constant.SIMPLE_METADATA_SERVICE_NAME
}

type MetadataServiceProxyFactory interface {
	GetProxy(ins registry.ServiceInstance) MetadataService
}

type MetadataServiceProxyCreator func(ins registry.ServiceInstance) MetadataService

type BaseMetadataServiceProxyFactory struct {
	proxies sync.Map
	creator MetadataServiceProxyCreator
}

func NewBaseMetadataServiceProxyFactory(creator MetadataServiceProxyCreator) *BaseMetadataServiceProxyFactory {
	return &BaseMetadataServiceProxyFactory{
		creator: creator,
	}
}

func (b *BaseMetadataServiceProxyFactory) GetProxy(ins registry.ServiceInstance) MetadataService {
	key := ins.GetServiceName() + "##" + getExportedServicesRevision(ins)
	if proxy, ok := b.proxies.Load(key); ok {
		return proxy.(MetadataService)
	}
	v, _ := b.proxies.LoadOrStore(key, b.creator(ins))
	return v.(MetadataService)
}

func getExportedServicesRevision(serviceInstance registry.ServiceInstance) string {
	metaData := serviceInstance.GetMetadata()
	return metaData[constant.EXPORTED_SERVICES_REVISION_PROPERTY_NAME]
}

func ConvertURLArrToIntfArr(urls []*common.URL) []interface{} {
	if len(urls) == 0 {
		return []interface{}{}
	}

	res := make([]interface{}, 0, len(urls))
	for _, u := range urls {
		res = append(res, u.String())
	}
	return res
}
