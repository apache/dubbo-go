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
	"dubbo.apache.org/dubbo-go/v3/protocol"
	_ "dubbo.apache.org/dubbo-go/v3/protocol/dubbo"
	"dubbo.apache.org/dubbo-go/v3/protocol/protocolwrapper"
	"dubbo.apache.org/dubbo-go/v3/server"
)

// MetadataServiceExporter is the ConfigurableMetadataServiceExporter which implement MetadataServiceExporter interface
type MetadataServiceExporter struct {
	ServiceConfig     *config.ServiceConfig
	v2Exporter        protocol.Exporter
	Exporter          protocol.Exporter
	lock              sync.RWMutex
	metadataService   service.MetadataService
	metadataServiceV1 service.MetadataServiceV1
	metadataServiceV2 service.MetadataServiceV2
}

func init() {
	extension.SetMetadataServiceExporter(constant.DefaultKey, NewMetadataServiceExporter)
}

// NewMetadataServiceExporter will return a service_exporter.MetadataServiceExporter with the specified  metadata service
func NewMetadataServiceExporter(metadataService service.MetadataService, metadataServiceV1 service.MetadataServiceV1, metadataServiceV2 service.MetadataServiceV2) exporter.MetadataServiceExporter {
	return &MetadataServiceExporter{
		metadataService:   metadataService,
		metadataServiceV1: metadataServiceV1,
		metadataServiceV2: metadataServiceV2,
	}
}

// Export will export the metadataService
func (exporter *MetadataServiceExporter) Export() error {
	if !exporter.IsExported() {
		exporter.lock.Lock()
		defer exporter.lock.Unlock()
		var err error
		pro, port := getMetadataProtocolAndPort()
		err = exporter.exportV1(pro, port)
		if err != nil {
			return err
		}
		err = exporter.exportV2(pro, port)
		return err
	}
	logger.Warnf("[Metadata Service] The MetadataService has been exported : %v ", exporter.ServiceConfig.GetExportedUrls())
	return nil
}

func (exporter *MetadataServiceExporter) exportV1(pro, port string) error {
	version, _ := exporter.metadataService.Version()
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
		logger.Infof("[Metadata Service] MetadataService has been exported at url : %v ", exporter.ServiceConfig.GetExportedUrls())
		exporter.metadataService.SetMetadataServiceURL(exporter.ServiceConfig.GetExportedUrls()[0])
		return err
	} else {
		info := server.MetadataService_ServiceInfo

		ivkURL := common.NewURLWithOptions(
			common.WithPath(constant.MetadataServiceName),
			common.WithProtocol(constant.TriProtocol),
			common.WithPort(port),
			common.WithParamsValue(constant.GroupKey, config.GetApplicationConfig().Name),
			common.WithParamsValue(constant.VersionKey, version),
			common.WithInterface(constant.MetadataServiceName),
			common.WithMethods(strings.Split("getMetadataInfo,GetMetadataInfo", ",")),
			common.WithAttribute(constant.ServiceInfoKey, &info),
			common.WithParamsValue(constant.SerializationKey, constant.Hessian2Serialization),
		)

		invoker := server.NewInternalInvoker(ivkURL, &info, exporter.metadataServiceV1)
		exporter.Exporter = extension.GetProtocol(protocolwrapper.FILTER).Export(invoker)
		logger.Infof("[Metadata Service] MetadataService has been exported at url : %v ", invoker.GetURL())
		exporter.metadataService.SetMetadataServiceURL(invoker.GetURL())
		return nil
	}
}

func (exporter *MetadataServiceExporter) exportV2(pro, port string) error {
	info := server.MetadataServiceV2_ServiceInfo
	// v2 only supports triple protocol
	if pro == constant.DefaultProtocol {
		return nil
	}
	ivkURL := common.NewURLWithOptions(
		common.WithPath(constant.MetadataServiceV2Name),
		common.WithProtocol(constant.TriProtocol),
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
	// do not set, because it will override MetadataService
	//exporter.metadataService.SetMetadataServiceURL(ivkURL)
	return nil
}

func getMetadataProtocolAndPort() (string, string) {
	rootConfig := config.GetRootConfig()
	port := rootConfig.Application.MetadataServicePort
	protocol := rootConfig.Application.MetadataServiceProtocol
	if protocol != "" && port != "" {
		return protocol, port
	}

	var protocolConfig *config.ProtocolConfig
	for _, pc := range rootConfig.Protocols {
		if pc.Name != constant.DefaultProtocol && pc.Name != constant.TriProtocol {
			continue
		}
		if pc.Name == protocol {
			protocolConfig = pc
			break
		}
	}

	if protocol == "" {
		for _, pc := range rootConfig.Protocols {
			if pc.Name == constant.TriProtocol {
				protocolConfig = pc
				break
			}
		}
	}

	if protocolConfig == nil {
		port = common.GetRandomPort("0")
		if protocol == "" || protocol == constant.TriProtocol {
			protocolConfig = &config.ProtocolConfig{
				Name: constant.TriProtocol,
				Port: port, // use a random port
			}
		} else {
			protocolConfig = &config.ProtocolConfig{
				Name: constant.DefaultProtocol,
				Port: port, // use a random port
			}
		}
	}

	if port == "" {
		return protocolConfig.Name, protocolConfig.Port
	} else {
		return protocolConfig.Name, port
	}
}

// Unexport will unexport the metadataService
func (exporter *MetadataServiceExporter) Unexport() {
	if exporter.IsExported() && exporter.ServiceConfig != nil {
		exporter.ServiceConfig.Unexport()
		exporter.ServiceConfig = nil
	}
	if exporter.v2Exporter != nil {
		exporter.v2Exporter.UnExport()
		exporter.v2Exporter = nil
	}
	if exporter.Exporter != nil {
		exporter.Exporter.UnExport()
		exporter.Exporter = nil
	}
}

// GetExportedURLs will return the urls that export use.
// Notice！The exported url is not same as url in registry , for example it lack the ip.
func (exporter *MetadataServiceExporter) GetExportedURLs() []*common.URL {
	url, _ := exporter.metadataService.GetMetadataServiceURL()
	return []*common.URL{url}
}

// IsExported will return is metadataServiceExporter exported or not
func (exporter *MetadataServiceExporter) IsExported() bool {
	exporter.lock.RLock()
	defer exporter.lock.RUnlock()
	return exporter.ServiceConfig != nil && exporter.ServiceConfig.IsExport() || exporter.Exporter != nil
}
