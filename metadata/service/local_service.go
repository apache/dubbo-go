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
	"context"
	"sync"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	triple_api "dubbo.apache.org/dubbo-go/v3/metadata/triple_api/proto"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

// MetadataService is used to define meta data related behaviors
// usually the implementation should be singleton
type MetadataService interface {
	common.ReferencedRPCService
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
	GetExportedURLs(serviceInterface string, group string, version string, protocol string) ([]*common.URL, error)
	// MethodMapper for rename dubbo method name
	MethodMapper() map[string]string
	// GetSubscribedURLs will get the target subscribed url in metadata the url should be unique
	GetSubscribedURLs() ([]*common.URL, error)
	// GetServiceDefinition will get the target service info store in metadata
	GetServiceDefinition(interfaceName string, group string, version string) (string, error)
	// GetServiceDefinitionByServiceKey will get the target service info store in metadata by service key
	GetServiceDefinitionByServiceKey(serviceKey string) (string, error)
	// RefreshMetadata will refresh the metadata
	RefreshMetadata(exportedRevision string, subscribedRevision string) (bool, error)
	// Version will return the metadata service version
	Version() (string, error)
	// GetMetadataInfo will return metadata info
	GetMetadataInfo(revision string) (*common.MetadataInfo, error)
	// GetExportedServiceURLs will return exported service urls
	GetExportedServiceURLs() ([]*common.URL, error)
	// GetMetadataServiceURL will return the url of metadata service
	GetMetadataServiceURL() (*common.URL, error)
	// SetMetadataServiceURL will save the url of metadata service
	SetMetadataServiceURL(*common.URL) error
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
		"GetMetadataInfo": "getMetadataInfo",
	}
}

// ServiceName can get the service's name in meta service , which is application name
func (mts *BaseMetadataService) ServiceName() (string, error) {
	return mts.serviceName, nil
}

// Reference will return the reference id of metadata service
func (mts *BaseMetadataService) Reference() string {
	return constant.SimpleMetadataServiceName
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
	return b.creator(ins)
}

func getExportedServicesRevision(serviceInstance registry.ServiceInstance) string {
	metaData := serviceInstance.GetMetadata()
	return metaData[constant.ExportedServicesRevisionPropertyName]
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

// MetadataServiceV2 is a client for the org.apache.dubbo.metadata.MetadataServiceV2 service.
type MetadataServiceV2 interface {
	GetMetadataInfo(ctx context.Context, req *triple_api.MetadataRequest) (*triple_api.MetadataInfoV2, error)
}

type MetadataServiceProxyFactoryV2 interface {
	GetProxy(ins registry.ServiceInstance) MetadataServiceV2
}

type MetadataServiceProxyCreatorV2 func(ins registry.ServiceInstance) MetadataServiceV2

type BaseMetadataServiceProxyFactoryV2 struct {
	proxies sync.Map
	creator MetadataServiceProxyCreatorV2
}

func (b *BaseMetadataServiceProxyFactoryV2) GetProxy(ins registry.ServiceInstance) MetadataServiceV2 {
	return b.creator(ins)
}

func NewBaseMetadataServiceProxyFactoryV2(creator MetadataServiceProxyCreatorV2) *BaseMetadataServiceProxyFactoryV2 {
	return &BaseMetadataServiceProxyFactoryV2{
		creator: creator,
	}
}

type MetadataServiceV1 interface {
	GetMetadataInfo(ctx context.Context, req string) (*triple_api.MetadataInfo, error)
}
