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
	"github.com/Workiva/go-datastructures/slice/skip"
	"go.uber.org/atomic"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/metadata/definition"
	"github.com/apache/dubbo-go/metadata/identifier"
	"github.com/apache/dubbo-go/metadata/report/delegate"
	"github.com/apache/dubbo-go/metadata/service"
	"github.com/apache/dubbo-go/metadata/service/inmemory"
)

// version will be used by Version func
const version = "1.0.0"

// MetadataService is a implement of metadata service which will delegate the remote metadata report
type MetadataService struct {
	service.BaseMetadataService
	inMemoryMetadataService *inmemory.MetadataService
	exportedRevision        atomic.String
	subscribedRevision      atomic.String
	delegateReport          *delegate.MetadataReport
}

// NewMetadataService will create a new remote MetadataService instance
func NewMetadataService() (*MetadataService, error) {
	mr, err := delegate.NewMetadataReport()
	if err != nil {
		return nil, err
	}
	return &MetadataService{
		inMemoryMetadataService: inmemory.NewMetadataService(),
		delegateReport:          mr,
	}, nil
}

// setInMemoryMetadataService will replace the in memory metadata service by the specific param
func (mts *MetadataService) setInMemoryMetadataService(metadata *inmemory.MetadataService) {
	mts.inMemoryMetadataService = metadata
}

// ExportURL will be implemented by in memory service
func (mts *MetadataService) ExportURL(url common.URL) (bool, error) {
	return mts.inMemoryMetadataService.ExportURL(url)
}

// UnexportURL
func (mts *MetadataService) UnexportURL(url common.URL) error {
	smi := identifier.NewServiceMetadataIdentifier(url)
	smi.Revision = mts.exportedRevision.Load()
	return mts.delegateReport.RemoveServiceMetadata(smi)
}

// SubscribeURL will be implemented by in memory service
func (mts *MetadataService) SubscribeURL(url common.URL) (bool, error) {
	return mts.inMemoryMetadataService.SubscribeURL(url)
}

// UnsubscribeURL will be implemented by in memory service
func (mts *MetadataService) UnsubscribeURL(url common.URL) error {
	return mts.UnsubscribeURL(url)
}

// PublishServiceDefinition will call remote metadata's StoreProviderMetadata to store url info and service definition
func (mts *MetadataService) PublishServiceDefinition(url common.URL) error {
	interfaceName := url.GetParam(constant.INTERFACE_KEY, "")
	isGeneric := url.GetParamBool(constant.GENERIC_KEY, false)
	if len(interfaceName) > 0 && !isGeneric {
		service := common.ServiceMap.GetService(url.Protocol, url.GetParam(constant.BEAN_NAME_KEY, url.Service()))
		sd := definition.BuildServiceDefinition(*service, url)
		id := &identifier.MetadataIdentifier{
			BaseMetadataIdentifier: identifier.BaseMetadataIdentifier{
				ServiceInterface: interfaceName,
				Version:          url.GetParam(constant.VERSION_KEY, ""),
				Group:            url.GetParam(constant.GROUP_KEY, ""),
			},
		}
		mts.delegateReport.StoreProviderMetadata(id, sd)
	}
	logger.Errorf("publishProvider interfaceName is empty . providerUrl:%v ", url)
	return nil
}

// GetExportedURLs will be implemented by in memory service
func (MetadataService) GetExportedURLs(serviceInterface string, group string, version string, protocol string) (*skip.SkipList, error) {
	return nil, nil
}

// GetSubscribedURLs will be implemented by in memory service
func (MetadataService) GetSubscribedURLs() (*skip.SkipList, error) {
	return nil, nil
}

// GetServiceDefinition will be implemented by in memory service
func (MetadataService) GetServiceDefinition(interfaceName string, group string, version string) (string, error) {
	return "", nil
}

// GetServiceDefinitionByServiceKey will be implemented by in memory service
func (MetadataService) GetServiceDefinitionByServiceKey(serviceKey string) (string, error) {
	return "", nil
}

// RefreshMetadata will refresh the exported & subscribed metadata to remote metadata report from the inmemory metadata service
func (mts *MetadataService) RefreshMetadata(exportedRevision string, subscribedRevision string) bool {
	result := true
	if len(exportedRevision) != 0 && exportedRevision != mts.exportedRevision.Load() {
		mts.exportedRevision.Store(exportedRevision)
		urls, err := mts.inMemoryMetadataService.GetExportedURLs(constant.ANY_VALUE, "", "", "")
		if err != nil {
			logger.Errorf("Error occur when execute remote.MetadataService.RefreshMetadata, error message is %+v", err)
			result = false
		}
		iterator := urls.Iter(inmemory.Comparator{})
		logger.Infof("urls length = %v", urls.Len())
		for {
			if !iterator.Next() {
				break
			}
			url := iterator.Value().(inmemory.Comparator)
			id := identifier.NewServiceMetadataIdentifier(common.URL(url))
			id.Revision = mts.exportedRevision.Load()
			if err := mts.delegateReport.SaveServiceMetadata(id, common.URL(url)); err != nil {
				logger.Errorf("Error occur when execute remote.MetadataService.RefreshMetadata, error message is %+v", err)
				result = false
			}
		}
	}

	if len(subscribedRevision) != 0 && subscribedRevision != mts.subscribedRevision.Load() {
		mts.subscribedRevision.Store(subscribedRevision)
		urls, err := mts.inMemoryMetadataService.GetSubscribedURLs()
		if err != nil {
			logger.Errorf("Error occur when execute remote.MetadataService.RefreshMetadata, error message is %v+", err)
			result = false
		}
		if urls != nil && urls.Len() > 0 {
			id := &identifier.SubscriberMetadataIdentifier{
				MetadataIdentifier: identifier.MetadataIdentifier{
					Application: config.GetApplicationConfig().Name,
				},
				Revision: subscribedRevision,
			}
			if err := mts.delegateReport.SaveSubscribedData(id, convertUrls(urls)); err != nil {
				logger.Errorf("Error occur when execute remote.MetadataService.RefreshMetadata, error message is %+v", err)
				result = false
			}
		}
	}
	return result
}

// Version will return the remote service version
func (MetadataService) Version() string {
	return version
}

// convertUrls will convert the skip list to slice
func convertUrls(list *skip.SkipList) []common.URL {
	urls := make([]common.URL, list.Len())
	iterator := list.Iter(inmemory.Comparator{})
	for {
		if iterator.Value() == nil {
			break
		}
		url := iterator.Value().(inmemory.Comparator)
		urls = append(urls, common.URL(url))
		if !iterator.Next() {
			break
		}
	}
	return urls
}
