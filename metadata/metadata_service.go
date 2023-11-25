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

package metadata

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

// version will be used by Version func
const (
	version              = "1.0.0"
	allServiceInterfaces = "*"
)

// MetadataService is used to define meta data related behaviors
// usually the implementation should be singleton
type MetadataService interface {
	// GetExportedURLs will get the target exported url in metadata, the url should be unique
	GetExportedURLs(serviceInterface string, group string, version string, protocol string) []*common.URL
	// GetExportedServiceURLs will return exported service urls
	GetExportedServiceURLs() []*common.URL
	// GetSubscribedURLs will get the exported urls in metadata
	GetSubscribedURLs() []*common.URL
	Version() string
	// GetMetadataInfo will return metadata info
	GetMetadataInfo(revision string) *info.MetadataInfo
	// GetMetadataServiceURL will return the url of metadata service
	GetMetadataServiceURL() *common.URL
	// SetMetadataServiceURL exporter to set url of metadata service
	SetMetadataServiceURL(*common.URL)
}

type ServiceExporter interface {
	Export() error
	UnExport()
}

// MetadataService is store and query the metadata info in memory when each service registry
type metadataService struct {
	url *common.URL
}

func (mts *metadataService) SetMetadataServiceURL(url *common.URL) {
	mts.url = url
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
	for _, sd := range mts.getServiceDiscoveryMetadata() {
		meta := sd.GetLocalMetadata()
		if meta.Revision == revision {
			return meta
		}
	}
	logger.Warnf("metadata not found for revision: %s", revision)
	return nil
}

func (mts *metadataService) getServiceDiscoveryMetadata() []registry.ServiceDiscoveryRegistry {
	sds := make([]registry.ServiceDiscoveryRegistry, 0)
	p := extension.GetProtocol(constant.RegistryProtocol)
	if factory, ok := p.(registry.RegistryFactory); ok {
		for _, v := range factory.GetRegistries() {
			if sd, ok := v.(registry.ServiceDiscoveryRegistry); ok {
				sds = append(sds, sd)
			}
		}
	}
	return sds
}

// GetExportedServiceURLs get exported service urls
func (mts *metadataService) GetExportedServiceURLs() []*common.URL {
	urls := make([]*common.URL, 0)
	for _, sd := range mts.getServiceDiscoveryMetadata() {
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
	return mts.url
}

func (mts *metadataService) GetSubscribedURLs() []*common.URL {
	urls := make([]*common.URL, 0)
	for _, sd := range mts.getServiceDiscoveryMetadata() {
		urls = append(urls, sd.GetLocalMetadata().GetSubscribedURLs()...)
	}
	return urls
}

// MethodMapper only for rename exported function, for example: rename the function GetMetadataInfo to getMetadataInfo
func (mts *metadataService) MethodMapper() map[string]string {
	return map[string]string{
		"GetExportedURLs": "getExportedURLs",
		"GetMetadataInfo": "getMetadataInfo",
	}
}
