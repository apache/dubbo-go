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

package configurable

import (
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/protocolwrapper"
	"dubbo.apache.org/dubbo-go/v3/server"
	"strconv"
	"strings"
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
	_ "dubbo.apache.org/dubbo-go/v3/metadata/mapping/metadata"
	"dubbo.apache.org/dubbo-go/v3/metadata/service"
	"dubbo.apache.org/dubbo-go/v3/metadata/service/exporter"
	_ "dubbo.apache.org/dubbo-go/v3/metadata/service/remote"
	_ "dubbo.apache.org/dubbo-go/v3/protocol/dubbo"
)

// MetadataServiceExporter is the ConfigurableMetadataServiceExporter which implement MetadataServiceExporter interface
type MetadataServiceExporter struct {
	ServiceConfig     *config.ServiceConfig
	v2Exporter        protocol.Exporter
	lock              sync.RWMutex
	metadataService   service.MetadataService
	metadataServiceV2 service.MetadataServiceV2
}

func init() {
	extension.SetMetadataServiceExporter(constant.DefaultKey, NewMetadataServiceExporter)
}

// NewMetadataServiceExporter will return a service_exporter.MetadataServiceExporter with the specified  metadata service
func NewMetadataServiceExporter(metadataService service.MetadataService, metadataServiceV2 service.MetadataServiceV2) exporter.MetadataServiceExporter {
	return &MetadataServiceExporter{
		metadataService:   metadataService,
		metadataServiceV2: metadataServiceV2,
	}
}

// Export will export the metadataService
func (exporter *MetadataServiceExporter) Export(url *common.URL) error {
	if !exporter.IsExported() {
		exporter.lock.Lock()
		defer exporter.lock.Unlock()
		var err error
		err = exporter.exportV1()
		err = exporter.exportV2()
		return err
	}
	logger.Warnf("[Metadata Service] The MetadataService has been exported : %v ", exporter.ServiceConfig.GetExportedUrls())
	return nil
}

func (exporter *MetadataServiceExporter) exportV1() error {
	version, _ := exporter.metadataService.Version()
	pro, port := getMetadataProtocolAndPort()
	if pro == constant.DefaultProtocol {
		exporter.ServiceConfig = config.NewServiceConfigBuilder().
			SetServiceID(constant.SimpleMetadataServiceName).
			SetProtocolIDs(pro).
			AddRCProtocol(pro, config.NewProtocolConfigBuilder().SetName(pro).SetPort(port).Build()).
			SetRegistryIDs("N/A").
			SetInterface(constant.MetadataServiceName).
			SetGroup(config.GetApplicationConfig().Name).
			SetVersion(version).
			SetProxyFactoryKey(constant.DefaultKey).
			SetMetadataType(config.GetApplicationConfig().MetadataType).
			Build()
		exporter.ServiceConfig.Implement(exporter.metadataService)
		err := exporter.ServiceConfig.Export()
		logger.Infof("[Metadata Service] The MetadataService exports urls : %v ", exporter.ServiceConfig.GetExportedUrls())
		return err
	} else {
		ivkURL := common.NewURLWithOptions(
			common.WithPath(constant.MetadataServiceName),
			common.WithProtocol("tri"),
			common.WithPort(port),
			common.WithParamsValue(constant.GroupKey, config.GetApplicationConfig().Name),
			common.WithParamsValue(constant.VersionKey, version),
			common.WithInterface(constant.MetadataServiceName),
			common.WithMethods(strings.Split("getMetadataInfo,GetMetadataInfo", ",")),
			common.WithAttribute(constant.ServiceInfoKey, &info),
		)

		invoker := server.NewInternalInvoker(ivkURL, &info, exporter.metadataServiceV2)
		exporter.v2Exporter = extension.GetProtocol(protocolwrapper.FILTER).Export(invoker)

		logger.Infof("[Metadata Service] MetadataServiceV2 has been exported at url : %v ", invoker.GetURL())
		return nil
	}
}

func (exporter *MetadataServiceExporter) exportV2() error {
	info := server.MetadataServiceV2_ServiceInfo
	pro, port := getMetadataProtocolAndPort()
	if pro == constant.DefaultProtocol {
		port = "-1"
	}
	ivkURL := common.NewURLWithOptions(
		common.WithPath(constant.MetadataServiceV2Name),
		common.WithProtocol("tri"),
		common.WithPort(port),
		common.WithParamsValue(constant.GroupKey, config.GetApplicationConfig().Name),
		common.WithParamsValue(constant.VersionKey, "2.0.0"),
		common.WithInterface(constant.MetadataServiceV2Name),
		common.WithMethods(strings.Split("getMetadataInfo,GetMetadataInfo", ",")),
		common.WithAttribute(constant.ServiceInfoKey, &info),
	)

	invoker := server.NewInternalInvoker(ivkURL, &info, exporter.metadataServiceV2)
	exporter.v2Exporter = extension.GetProtocol(protocolwrapper.FILTER).Export(invoker)

	logger.Infof("[Metadata Service] MetadataServiceV2 has been exported at url : %v ", invoker.GetURL())
	return nil
}

func getMetadataProtocolAndPort() (string, string) {
	rootConfig := config.GetRootConfig()
	port := rootConfig.Application.MetadataServicePort
	protocol := "tri"
	if port == "" {
		protocolConfig, ok := rootConfig.Protocols[protocol]
		if ok {
			port = protocolConfig.Port
		} else {
			defaultProtocol := rootConfig.Protocols[constant.DefaultProtocol]
			if defaultProtocol != nil {
				port = defaultProtocol.Port
				protocol = constant.DefaultProtocol
			} else {
				port = strconv.Itoa(constant.DefaultPort)
			}
		}
	}
	return protocol, port
}

// Unexport will unexport the metadataService
func (exporter *MetadataServiceExporter) Unexport() {
	if exporter.IsExported() {
		exporter.ServiceConfig.Unexport()
	}
	if exporter.v2Exporter != nil {
		exporter.v2Exporter.UnExport()
	}
}

// GetExportedURLs will return the urls that export use.
// Notice！The exported url is not same as url in registry , for example it lack the ip.
func (exporter *MetadataServiceExporter) GetExportedURLs() []*common.URL {
	return exporter.ServiceConfig.GetExportedUrls()
}

// IsExported will return is metadataServiceExporter exported or not
func (exporter *MetadataServiceExporter) IsExported() bool {
	exporter.lock.RLock()
	defer exporter.lock.RUnlock()
	return exporter.ServiceConfig != nil && exporter.ServiceConfig.IsExport()
}
