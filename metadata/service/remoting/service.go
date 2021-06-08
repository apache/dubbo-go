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

package remoting

import (
	"sync"
)

import (
	"go.uber.org/atomic"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/metadata/definition"
	"dubbo.apache.org/dubbo-go/v3/metadata/identifier"
	"dubbo.apache.org/dubbo-go/v3/metadata/report/delegate"
	"dubbo.apache.org/dubbo-go/v3/metadata/service"
	"dubbo.apache.org/dubbo-go/v3/metadata/service/inmemory"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

type MetadataService struct {
	*inmemory.MetadataService
	exportedRevision   atomic.String
	subscribedRevision atomic.String
	delegateReport     *delegate.MetadataReport
}

var (
	metadataServiceOnce             sync.Once
	remotingMetadataServiceInstance service.RemotingMetadataService
)

func init() {
	extension.SetRemotingMetadataService(GetRemotingMetadataService)
}

// GetRemotingMetadataService will create a new remote MetadataService instance
func GetRemotingMetadataService() (service.RemotingMetadataService, error) {
	var err error
	metadataServiceOnce.Do(func() {
		var mr *delegate.MetadataReport
		mr, err = delegate.NewMetadataReport()
		if err != nil {
			return
		}
		// it will never return error
		inms, _ := inmemory.GetInMemoryMetadataService()
		remotingMetadataServiceInstance = &MetadataService{
			// todo serviceName
			//BaseMetadataService:     service.NewBaseMetadataService(""),
			MetadataService: inms.(*inmemory.MetadataService),
			delegateReport:  mr,
		}
	})
	return remotingMetadataServiceInstance, err
}

// PublishMetadata publishes the metadata info of @service to remote metadata center
func (s *MetadataService) PublishMetadata(service string) {
	info, err := s.MetadataService.GetMetadataInfo("")
	if err != nil {
		logger.Errorf("GetMetadataInfo error[%v]", err)
		return
	}
	if info.HasReported() {
		return
	}
	id := identifier.NewSubscriberMetadataIdentifier(service, info.CalAndGetRevision())
	err = s.delegateReport.PublishAppMetadata(id, info)
	if err != nil {
		logger.Errorf("Publishing metadata to error[%v]", err)
		return
	}
	info.MarkReported()
}

// GetMetadata get the medata info of service from report
func (s *MetadataService) GetMetadata(instance registry.ServiceInstance) (*common.MetadataInfo, error) {
	revision := instance.GetMetadata()[constant.EXPORTED_SERVICES_REVISION_PROPERTY_NAME]
	id := identifier.NewSubscriberMetadataIdentifier(instance.GetServiceName(), revision)
	return s.delegateReport.GetAppMetadata(id)
}

// PublishServiceDefinition will call remote metadata's StoreProviderMetadata to store url info and service definition
func (s *MetadataService) PublishServiceDefinition(url *common.URL) error {
	interfaceName := url.GetParam(constant.INTERFACE_KEY, "")
	isGeneric := url.GetParamBool(constant.GENERIC_KEY, false)
	if common.RoleType(common.PROVIDER).Role() == url.GetParam(constant.SIDE_KEY, "") {
		if len(interfaceName) > 0 && !isGeneric {
			sv := common.ServiceMap.GetServiceByServiceKey(url.Protocol, url.ServiceKey())
			sd := definition.BuildServiceDefinition(*sv, url)
			id := &identifier.MetadataIdentifier{
				BaseMetadataIdentifier: identifier.BaseMetadataIdentifier{
					ServiceInterface: interfaceName,
					Version:          url.GetParam(constant.VERSION_KEY, ""),
					Group:            url.GetParam(constant.GROUP_KEY, constant.DUBBO),
					Side:             url.GetParam(constant.SIDE_KEY, constant.PROVIDER_PROTOCOL),
				},
			}
			s.delegateReport.StoreProviderMetadata(id, sd)
			return nil
		}
		logger.Errorf("publishProvider interfaceName is empty . providerUrl:%v ", url)
	} else {
		params := make(map[string]string, len(url.GetParams()))
		url.RangeParams(func(key, value string) bool {
			params[key] = value
			return true
		})
		id := &identifier.MetadataIdentifier{
			BaseMetadataIdentifier: identifier.BaseMetadataIdentifier{
				ServiceInterface: interfaceName,
				Version:          url.GetParam(constant.VERSION_KEY, ""),
				Group:            url.GetParam(constant.GROUP_KEY, constant.DUBBO),
				Side:             url.GetParam(constant.SIDE_KEY, "consumer"),
			},
		}
		s.delegateReport.StoreConsumerMetadata(id, params)
		return nil
	}

	return nil
}
