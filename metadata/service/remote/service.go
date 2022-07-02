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

package remote

import (
	"sync"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"go.uber.org/atomic"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/metadata/definition"
	"dubbo.apache.org/dubbo-go/v3/metadata/identifier"
	"dubbo.apache.org/dubbo-go/v3/metadata/report/delegate"
	"dubbo.apache.org/dubbo-go/v3/metadata/service"
	"dubbo.apache.org/dubbo-go/v3/metadata/service/local"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

type MetadataService struct {
	*local.MetadataService
	exportedRevision   atomic.String
	subscribedRevision atomic.String
	delegateReport     *delegate.MetadataReport
}

var (
	metadataServiceOnce           sync.Once
	remoteMetadataServiceInstance service.RemoteMetadataService
)

func init() {
	extension.SetRemoteMetadataService(GetRemoteMetadataService)
}

// GetRemoteMetadataService will create a new remote MetadataService instance
func GetRemoteMetadataService() (service.RemoteMetadataService, error) {
	var err error
	metadataServiceOnce.Do(func() {
		var mr *delegate.MetadataReport
		mr, err = delegate.NewMetadataReport()
		if err != nil {
			return
		}
		// it will never return error
		inms, _ := local.GetLocalMetadataService()
		remoteMetadataServiceInstance = &MetadataService{
			// todo serviceName
			//BaseMetadataService:     service.NewBaseMetadataService(""),
			MetadataService: inms.(*local.MetadataService),
			delegateReport:  mr,
		}
	})
	return remoteMetadataServiceInstance, err
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
	revision := instance.GetMetadata()[constant.ExportedServicesRevisionPropertyName]
	id := identifier.NewSubscriberMetadataIdentifier(instance.GetServiceName(), revision)
	return s.delegateReport.GetAppMetadata(id)
}

// PublishServiceDefinition will call remote metadata's StoreProviderMetadata to store url info and service definition
func (s *MetadataService) PublishServiceDefinition(url *common.URL) error {
	interfaceName := url.GetParam(constant.InterfaceKey, "")
	isGeneric := url.GetParamBool(constant.GenericKey, false)
	if common.RoleType(common.PROVIDER).Role() == url.GetParam(constant.SideKey, "") {
		if len(interfaceName) > 0 && !isGeneric {
			sv := common.ServiceMap.GetServiceByServiceKey(url.Protocol, url.ServiceKey())
			sd := definition.BuildServiceDefinition(*sv, url)
			id := &identifier.MetadataIdentifier{
				BaseMetadataIdentifier: identifier.BaseMetadataIdentifier{
					ServiceInterface: interfaceName,
					Version:          url.GetParam(constant.VersionKey, ""),
					Group:            url.GetParam(constant.GroupKey, constant.Dubbo),
					Side:             url.GetParam(constant.SideKey, constant.ProviderProtocol),
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
				Version:          url.GetParam(constant.VersionKey, ""),
				Group:            url.GetParam(constant.GroupKey, constant.Dubbo),
				Side:             url.GetParam(constant.SideKey, "consumer"),
			},
		}
		s.delegateReport.StoreConsumerMetadata(id, params)
		return nil
	}

	return nil
}
