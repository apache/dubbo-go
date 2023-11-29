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

package servicediscovery

import (
	"context"
	"strconv"
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
	metadataInstance "dubbo.apache.org/dubbo-go/v3/metadata/report/instance"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

type ServiceMeta struct {
	metadataInfo *info.MetadataInfo
	instance     registry.ServiceInstance
}

func NewServiceMeta() *ServiceMeta {
	return &ServiceMeta{metadataInfo: info.NewMetadataInfWithApp()}
}

func (sd *ServiceMeta) GetLocalMetadata() *info.MetadataInfo {
	return sd.metadataInfo
}

func (sd *ServiceMeta) GetRemoteMetadata(revision string, instance registry.ServiceInstance) (*info.MetadataInfo, error) {
	meta, err := getMetadataFromCache(revision)
	if err != nil || meta == nil {
		meta, err = getMetadataFromMetadataReport(revision, instance)
		if err != nil || meta == nil {
			meta, err = getMetadataFromRpc(revision, instance)
		}
		// TODO : need to update cache
	}
	return meta, err
}

func (sd *ServiceMeta) createInstance() registry.ServiceInstance {
	metadata := make(map[string]string, 8)
	metadata[constant.MetadataStorageTypePropertyName] = metadataInstance.GetMetadataType()
	instance := &registry.DefaultServiceInstance{
		ServiceName:     sd.metadataInfo.App,
		Enable:          true,
		Healthy:         true,
		Metadata:        metadata,
		ServiceMetadata: sd.metadataInfo,
	}

	for _, cus := range extension.GetCustomizers() {
		cus.Customize(instance)
	}
	sd.instance = instance
	return instance
}

//func (sd *ServiceMeta) calOrUpdateInstanceRevision(instance registry.ServiceInstance) bool {
//	oldRevision := getRevision(instance)
//	newRevision := instance.GetServiceMetadata().CalAndGetRevision()
//	if oldRevision != newRevision {
//		instance.GetMetadata()[constant.ExportedServicesRevisionPropertyName] = newRevision
//		return true
//	}
//	return false
//}

//func getRevision(instance registry.ServiceInstance) string {
//	if instance.GetServiceMetadata() != nil && instance.GetServiceMetadata().Revision != "" {
//		return instance.GetServiceMetadata().Revision
//	}
//	return instance.GetMetadata()[constant.ExportedServicesRevisionPropertyName]
//}

func getMetadataFromCache(revision string) (*info.MetadataInfo, error) {
	// TODO metadata cache
	return nil, nil
}

func getMetadataFromMetadataReport(revision string, instance registry.ServiceInstance) (*info.MetadataInfo, error) {
	report := metadataInstance.GetMetadataReport()
	return report.GetAppMetadata(instance.GetServiceName(), revision)
}

func getMetadataFromRpc(revision string, instance registry.ServiceInstance) (*info.MetadataInfo, error) {
	service, destroy := createRpcClient(instance)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(5000))
	defer cancel()
	defer destroy()
	return service.GetMetadataInfo(ctx, revision)
}

type metadataService struct {
	GetExportedURLs       func(context context.Context, serviceInterface string, group string, version string, protocol string) ([]*common.URL, error) `dubbo:"getExportedURLs"`
	GetMetadataInfo       func(context context.Context, revision string) (*info.MetadataInfo, error)                                                   `dubbo:"getMetadataInfo"`
	GetMetadataServiceURL func(context context.Context) (*common.URL, error)
	GetSubscribedURLs     func(context context.Context) ([]*common.URL, error)
	Version               func(context context.Context) (string, error)
}

func createRpcClient(instance registry.ServiceInstance) (*metadataService, func()) {
	url := common.NewURLWithOptions(
		common.WithProtocol(constant.Dubbo),
		common.WithIp(instance.GetHost()),
		common.WithPort(strconv.Itoa(instance.GetPort())),
	)
	url.SetParam(constant.SideKey, constant.Consumer)
	url.SetParam(constant.VersionKey, "1.0.0")
	url.SetParam(constant.InterfaceKey, constant.MetadataServiceName)
	url.SetParam(constant.GroupKey, instance.GetServiceName())
	rpcService := &metadataService{}
	invoker := extension.GetProtocol(constant.Dubbo).Refer(url)
	proxy := extension.GetProxyFactory("").GetProxy(invoker, url)
	proxy.Implement(rpcService)
	destroy := func() {
		invoker.Destroy()
	}
	return rpcService, destroy
}
