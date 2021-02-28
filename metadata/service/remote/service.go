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
	"go.uber.org/atomic"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/metadata/definition"
	"github.com/apache/dubbo-go/metadata/identifier"
	"github.com/apache/dubbo-go/metadata/report/delegate"
	"github.com/apache/dubbo-go/metadata/service"
	"github.com/apache/dubbo-go/metadata/service/inmemory"
)

// version will be used by Version func
const (
	version = "1.0.0"
	remote  = "remote"
)

func init() {
	extension.SetMetadataService(remote, newMetadataService)
}

// MetadataService is a implement of metadata service which will delegate the remote metadata report
// This is singleton
type MetadataService struct {
	service.BaseMetadataService
	inMemoryMetadataService *inmemory.MetadataService
	exportedRevision        atomic.String
	subscribedRevision      atomic.String
	delegateReport          *delegate.MetadataReport
}

var (
	metadataServiceOnce     sync.Once
	metadataServiceInstance *MetadataService
)

// newMetadataService will create a new remote MetadataService instance
func newMetadataService() (service.MetadataService, error) {
	var err error
	metadataServiceOnce.Do(func() {
		var mr *delegate.MetadataReport
		mr, err = delegate.NewMetadataReport()
		if err != nil {
			return
		}
		// it will never return error
		inms, _ := inmemory.NewMetadataService()
		metadataServiceInstance = &MetadataService{
			BaseMetadataService:     service.NewBaseMetadataService(config.GetApplicationConfig().Name),
			inMemoryMetadataService: inms.(*inmemory.MetadataService),
			delegateReport:          mr,
		}
	})
	return metadataServiceInstance, err
}

// setInMemoryMetadataService will replace the in memory metadata service by the specific param
func (mts *MetadataService) setInMemoryMetadataService(metadata *inmemory.MetadataService) {
	mts.inMemoryMetadataService = metadata
}

// ExportURL will be implemented by in memory service
func (mts *MetadataService) ExportURL(url *common.URL) (bool, error) {
	return mts.inMemoryMetadataService.ExportURL(url)
}

// UnexportURL remove @url's metadata
func (mts *MetadataService) UnexportURL(url *common.URL) error {
	smi := identifier.NewServiceMetadataIdentifier(url)
	smi.Revision = mts.exportedRevision.Load()
	return mts.delegateReport.RemoveServiceMetadata(smi)
}

// SubscribeURL will be implemented by in memory service
func (mts *MetadataService) SubscribeURL(url *common.URL) (bool, error) {
	return mts.inMemoryMetadataService.SubscribeURL(url)
}

// UnsubscribeURL will be implemented by in memory service
func (mts *MetadataService) UnsubscribeURL(url *common.URL) error {
	// TODO remove call local.
	return nil
	//return mts.UnsubscribeURL(url)
}

// PublishServiceDefinition will call remote metadata's StoreProviderMetadata to store url info and service definition
func (mts *MetadataService) PublishServiceDefinition(url *common.URL) error {
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
			mts.delegateReport.StoreProviderMetadata(id, sd)
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
		mts.delegateReport.StoreConsumerMetadata(id, params)
		return nil
	}

	return nil
}

// GetExportedURLs will be implemented by in memory service
func (mts *MetadataService) GetExportedURLs(serviceInterface string, group string, version string, protocol string) ([]interface{}, error) {
	return mts.inMemoryMetadataService.GetExportedURLs(serviceInterface, group, version, protocol)
}

// GetSubscribedURLs will be implemented by in memory service
func (mts *MetadataService) GetSubscribedURLs() ([]*common.URL, error) {
	return mts.inMemoryMetadataService.GetSubscribedURLs()
}

// GetServiceDefinition will be implemented by in memory service
func (mts *MetadataService) GetServiceDefinition(interfaceName string, group string, version string) (string, error) {
	return mts.inMemoryMetadataService.GetServiceDefinition(interfaceName, group, version)
}

// GetServiceDefinitionByServiceKey will be implemented by in memory service
func (mts *MetadataService) GetServiceDefinitionByServiceKey(serviceKey string) (string, error) {
	return mts.inMemoryMetadataService.GetServiceDefinitionByServiceKey(serviceKey)
}

// RefreshMetadata will refresh the exported & subscribed metadata to remote metadata report from the inmemory metadata service
func (mts *MetadataService) RefreshMetadata(exportedRevision string, subscribedRevision string) (bool, error) {
	if len(exportedRevision) != 0 && exportedRevision != mts.exportedRevision.Load() {
		mts.exportedRevision.Store(exportedRevision)
		urls, err := mts.inMemoryMetadataService.GetExportedURLs(constant.ANY_VALUE, "", "", "")
		if err != nil {
			logger.Errorf("Error occur when execute remote.MetadataService.RefreshMetadata, error message is %+v", err)
			return false, err
		}
		logger.Infof("urls length = %v", len(urls))
		for _, ui := range urls {

			u, err := common.NewURL(ui.(string))
			if err != nil {
				logger.Errorf("this is not valid url string: %s ", ui.(string))
				continue
			}
			id := identifier.NewServiceMetadataIdentifier(u)
			id.Revision = mts.exportedRevision.Load()
			if err := mts.delegateReport.SaveServiceMetadata(id, u); err != nil {
				logger.Errorf("Error occur when execute remote.MetadataService.RefreshMetadata, error message is %+v", err)
				return false, err
			}
		}
	}

	if len(subscribedRevision) != 0 && subscribedRevision != mts.subscribedRevision.Load() {
		mts.subscribedRevision.Store(subscribedRevision)
		urls, err := mts.inMemoryMetadataService.GetSubscribedURLs()
		if err != nil {
			logger.Errorf("Error occur when execute remote.MetadataService.RefreshMetadata, error message is %v+", err)
			return false, err
		}
		if len(urls) > 0 {
			id := &identifier.SubscriberMetadataIdentifier{
				MetadataIdentifier: identifier.MetadataIdentifier{
					Application: config.GetApplicationConfig().Name,
				},
				Revision: subscribedRevision,
			}
			if err := mts.delegateReport.SaveSubscribedData(id, urls); err != nil {
				logger.Errorf("Error occur when execute remote.MetadataService.RefreshMetadata, error message is %+v", err)
				return false, err
			}
		}
	}
	return true, nil
}

// Version will return the remote service version
func (MetadataService) Version() (string, error) {
	return version, nil
}
