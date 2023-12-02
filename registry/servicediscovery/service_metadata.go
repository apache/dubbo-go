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
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
	metadataInstance "dubbo.apache.org/dubbo-go/v3/metadata/report/instance"
	_ "dubbo.apache.org/dubbo-go/v3/protocol/dubbo"
	_ "dubbo.apache.org/dubbo-go/v3/proxy/proxy_factory"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

type ServiceMeta struct {
	metadataInfo *info.MetadataInfo
	instance     registry.ServiceInstance
}

func NewServiceMeta() *ServiceMeta {
	return &ServiceMeta{metadataInfo: info.NewMetadataInfo()}
}

func (sd *ServiceMeta) GetLocalMetadata() *info.MetadataInfo {
	return sd.metadataInfo
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
