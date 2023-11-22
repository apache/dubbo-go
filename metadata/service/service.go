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
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
	_ "dubbo.apache.org/dubbo-go/v3/protocol/dubbo"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

// version will be used by Version func
const (
	version              = "1.0.0"
	allServiceInterfaces = "*"
)

var (
	metadataServiceUrl    *common.URL
	GlobalMetadataService MetadataService = &metadataService{}
	Exporter                              = &MetadataServiceExporter{}
)

// MetadataService is store and query the metadata info in memory when each service registry
type metadataService struct {
	BaseMetadataService
}

// GetExportedURLs get all exported urls
func (mts *metadataService) GetExportedURLs(serviceInterface string, group string, version string, protocol string) []*common.URL {
	if allServiceInterfaces == serviceInterface {
		return mts.GetExportedServiceURLs()
	}
	all := mts.GetExportedServiceURLs()
	urls := make([]*common.URL, 0)
	for _, url := range all {
		if url.GetParam(constant.InterfaceKey, "") == serviceInterface &&
			url.GetParam(constant.GroupKey, "") == group &&
			url.GetParam(constant.ProtocolKey, "") == protocol &&
			url.GetParam(constant.VersionKey, "") == version {
			urls = append(urls, url)
		}
	}
	return urls
}

// GetMetadataInfo can get metadata in memory
func (mts *metadataService) GetMetadataInfo(revision string) *info.MetadataInfo {
	if revision == "" {
		return nil
	}
	for _, sd := range mts.getServiceDiscoveries() {
		meta := sd.GetLocalMetadata()
		if meta.Revision == revision {
			return meta
		}
	}
	logger.Warnf("metadata not found for revision: %s", revision)
	return nil
}

func (mts *metadataService) getServiceDiscoveries() []registry.ServiceDiscovery {
	sds := make([]registry.ServiceDiscovery, 0)
	p := extension.GetProtocol(constant.RegistryProtocol)
	if factory, ok := p.(registry.RegistryFactory); ok {
		for _, v := range factory.GetRegistries() {
			if sd, ok := v.(registry.ServiceDiscoveryHolder); ok {
				sds = append(sds, sd.GetServiceDiscovery())
			}
		}
	}
	return sds
}

// GetExportedServiceURLs get exported service urls
func (mts *metadataService) GetExportedServiceURLs() []*common.URL {
	urls := make([]*common.URL, 0)
	for _, sd := range mts.getServiceDiscoveries() {
		urls = append(urls, sd.GetLocalMetadata().GetExportedServiceURLs()...)
	}
	return urls
}

// Version will return the version of metadata service
func (mts *metadataService) Version() string {
	return version
}

// GetMetadataServiceURL get url of MetadataService
func (mts *metadataService) GetMetadataServiceURL() *common.URL {
	return metadataServiceUrl
}

func (mts *metadataService) GetSubscribedURLs() []*common.URL {
	urls := make([]*common.URL, 0)
	for _, sd := range mts.getServiceDiscoveries() {
		urls = append(urls, sd.GetLocalMetadata().GetSubscribedURLs()...)
	}
	return urls
}

// MetadataServiceExporter is the ConfigurableMetadataServiceExporter which implement MetadataServiceExporter interface
type MetadataServiceExporter struct {
	ServiceConfig *config.ServiceConfig
	lock          sync.RWMutex
}

// Export will export the metadataService
func (exporter *MetadataServiceExporter) Export(app, metadataType string) error {
	if !exporter.IsExported() {
		exporter.lock.Lock()
		defer exporter.lock.Unlock()
		exporter.ServiceConfig = config.NewServiceConfigBuilder().
			SetServiceID(constant.SimpleMetadataServiceName).
			SetProtocolIDs(constant.DefaultProtocol).
			AddRCProtocol(constant.DefaultProtocol, config.NewProtocolConfigBuilder().
				SetName(constant.DefaultProtocol).
				Build()).
			SetRegistryIDs("N/A").
			SetInterface(constant.MetadataServiceName).
			SetGroup(app).
			SetVersion(version).
			SetProxyFactoryKey(constant.DefaultKey).
			SetMetadataType(metadataType).
			Build()
		exporter.ServiceConfig.Implement(GlobalMetadataService)
		err := exporter.ServiceConfig.Export()
		metadataServiceUrl = exporter.ServiceConfig.GetExportedUrls()[0]
		logger.Infof("[Metadata Service] The MetadataService exports urls : %v ", exporter.ServiceConfig.GetExportedUrls())
		return err
	}
	logger.Warnf("[Metadata Service] The MetadataService has been exported : %v ", exporter.ServiceConfig.GetExportedUrls())
	return nil
}

// UnExport will unExport the metadataService
func (exporter *MetadataServiceExporter) UnExport() {
	if exporter.IsExported() {
		exporter.ServiceConfig.Unexport()
	}
}

// IsExported will return is metadataServiceExporter exported or not
func (exporter *MetadataServiceExporter) IsExported() bool {
	exporter.lock.RLock()
	defer exporter.lock.RUnlock()
	return exporter.ServiceConfig != nil && exporter.ServiceConfig.IsExport()
}
