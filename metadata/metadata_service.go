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
	"fmt"
	"strings"
)

import (
	"github.com/dubbogo/gost/log/logger"
	gxnet "github.com/dubbogo/gost/net"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

// version will be used by Version func
const (
	version              = "1.0.0"
	allServiceInterfaces = "*"
)

var metadataUrl *common.URL

// MetadataService is used to define meta data related behaviors
// usually the implementation should be singleton
type MetadataService interface {
	// GetExportedURLs will get the target exported url in metadata, the url should be unique
	GetExportedURLs(serviceInterface string, group string, version string, protocol string) ([]*common.URL, error)
	// GetExportedServiceURLs will return exported service urls
	GetExportedServiceURLs() ([]*common.URL, error)
	// GetSubscribedURLs will get the exported urls in metadata
	GetSubscribedURLs() ([]*common.URL, error)
	Version() (string, error)
	// GetMetadataInfo will return metadata info
	GetMetadataInfo(revision string) (*info.MetadataInfo, error)
	// GetMetadataServiceURL will return the url of metadata service
	GetMetadataServiceURL() (*common.URL, error)
}

// DefaultMetadataService is store and query the metadata info in memory when each service registry
type DefaultMetadataService struct {
}

// GetExportedURLs get all exported urls
func (mts *DefaultMetadataService) GetExportedURLs(serviceInterface string, group string, version string, protocol string) ([]*common.URL, error) {
	if allServiceInterfaces == serviceInterface {
		return mts.GetExportedServiceURLs()
	}
	all, err := mts.GetExportedServiceURLs()
	if err != nil {
		return nil, err
	}
	urls := make([]*common.URL, 0)
	for _, url := range all {
		if url.GetParam(constant.InterfaceKey, "") == serviceInterface &&
			url.GetParam(constant.GroupKey, "") == group &&
			url.GetParam(constant.ProtocolKey, "") == protocol &&
			url.GetParam(constant.VersionKey, "") == version {
			urls = append(urls, url)
		}
	}
	return urls, nil
}

// GetMetadataInfo can get metadata in memory
func (mts *DefaultMetadataService) GetMetadataInfo(revision string) (*info.MetadataInfo, error) {
	if revision == "" {
		return nil, nil
	}
	for _, metadataInfo := range mts.getAllMetadata() {
		if metadataInfo.Revision == revision {
			return metadataInfo, nil
		}
	}
	logger.Warnf("metadata not found for revision: %s", revision)
	return nil, nil
}

func (mts *DefaultMetadataService) getAllMetadata() []*info.MetadataInfo {
	sds := make([]*info.MetadataInfo, 0)
	p := extension.GetProtocol(constant.RegistryProtocol)
	if factory, ok := p.(registry.RegistryFactory); ok {
		for _, v := range factory.GetRegistries() {
			if sd, ok := v.(registry.ServiceDiscoveryRegistry); ok {
				sds = append(sds, sd.GetLocalMetadata())
			}
		}
	}
	return sds
}

// GetExportedServiceURLs get exported service urls
func (mts *DefaultMetadataService) GetExportedServiceURLs() ([]*common.URL, error) {
	urls := make([]*common.URL, 0)
	for _, metadataInfo := range mts.getAllMetadata() {
		urls = append(urls, metadataInfo.GetExportedServiceURLs()...)
	}
	return urls, nil
}

// Version will return the version of metadata service
func (mts *DefaultMetadataService) Version() (string, error) {
	return version, nil
}

// GetMetadataServiceURL get url of MetadataService
func (mts *DefaultMetadataService) GetMetadataServiceURL() (*common.URL, error) {
	return metadataUrl, nil
}

func (mts *DefaultMetadataService) GetSubscribedURLs() ([]*common.URL, error) {
	urls := make([]*common.URL, 0)
	for _, metadataInfo := range mts.getAllMetadata() {
		urls = append(urls, metadataInfo.GetSubscribedURLs()...)
	}
	return urls, nil
}

// MethodMapper only for rename exported function, for example: rename the function GetMetadataInfo to getMetadataInfo
func (mts *DefaultMetadataService) MethodMapper() map[string]string {
	return map[string]string{
		"GetExportedURLs": "getExportedURLs",
		"GetMetadataInfo": "getMetadataInfo",
	}
}

// ServiceExporter is the ConfigurableMetadataServiceExporter which implement MetadataServiceExporter interface
type ServiceExporter struct {
	app, metadataType string
	service           MetadataService
	protocolExporter  protocol.Exporter
}

// Export will export the metadataService
func (exporter *ServiceExporter) Export() error {
	version, _ := exporter.service.Version()
	tcp, err := gxnet.ListenOnTCPRandomPort("")
	if err != nil {
		panic(perrors.New(fmt.Sprintf("Get tcp port error, err is {%v}", err)))
	}
	ivkURL := common.NewURLWithOptions(
		common.WithPath(constant.MetadataServiceName),
		common.WithProtocol(constant.DefaultProtocol),
		common.WithPort(strings.Split(tcp.Addr().String(), ":")[1]),
		common.WithParamsValue(constant.GroupKey, exporter.app),
		common.WithParamsValue(constant.SerializationKey, constant.Hessian2Serialization),
		common.WithParamsValue(constant.ReleaseKey, constant.Version),
		common.WithParamsValue(constant.VersionKey, version),
		common.WithParamsValue(constant.InterfaceKey, constant.MetadataServiceName),
		common.WithParamsValue(constant.BeanNameKey, constant.SimpleMetadataServiceName),
		common.WithParamsValue(constant.MetadataTypeKey, exporter.metadataType),
	)
	methods, err := common.ServiceMap.Register(ivkURL.Interface(), ivkURL.Protocol, ivkURL.Group(), ivkURL.Version(), exporter.service)
	if err != nil {
		formatErr := perrors.Errorf("The service %v needExport the protocol %v error! Error message is %v.",
			ivkURL.Interface(), ivkURL.Protocol, err.Error())
		logger.Errorf(formatErr.Error())
		return formatErr
	}
	ivkURL.Methods = strings.Split(methods, ",")
	proxyFactory := extension.GetProxyFactory("")
	invoker := proxyFactory.GetInvoker(ivkURL)
	exporter.protocolExporter = extension.GetProtocol(ivkURL.Protocol).Export(invoker)
	metadataUrl = exporter.protocolExporter.GetInvoker().GetURL()
	logger.Infof("[Metadata Service] The MetadataService exports urls : %v ", exporter.protocolExporter.GetInvoker().GetURL())
	return nil
}

// UnExport will unExport the metadataService
func (exporter *ServiceExporter) UnExport() {
	exporter.protocolExporter.UnExport()
}
