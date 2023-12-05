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

package exporter

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/metadata"
	_ "dubbo.apache.org/dubbo-go/v3/protocol/dubbo"
	_ "dubbo.apache.org/dubbo-go/v3/proxy/proxy_factory"
)

func init() {
	metadata.SetExporterFactory(NewMetadataServiceExporter)
}

// MetadataServiceExporter is the ConfigurableMetadataServiceExporter which implement MetadataServiceExporter interface
type MetadataServiceExporter struct {
	app, metadataType string
	ServiceConfig     *config.ServiceConfig
	service           metadata.MetadataService
}

func NewMetadataServiceExporter(app, metadataType string, service metadata.MetadataService) metadata.ServiceExporter {
	return &MetadataServiceExporter{app: app, metadataType: metadataType, service: service}
}

// Export will export the metadataService
func (exporter *MetadataServiceExporter) Export() error {
	version, _ := exporter.service.Version()
	exporter.ServiceConfig = config.NewServiceConfigBuilder().
		SetServiceID(constant.SimpleMetadataServiceName).
		SetProtocolIDs(constant.DefaultProtocol).
		AddRCProtocol(constant.DefaultProtocol, config.NewProtocolConfigBuilder().
			SetName(constant.DefaultProtocol).
			Build()).
		SetRegistryIDs("N/A").
		SetInterface(constant.MetadataServiceName).
		SetGroup(exporter.app).
		SetVersion(version).
		SetProxyFactoryKey(constant.DefaultKey).
		SetMetadataType(exporter.metadataType).
		Build()
	exporter.ServiceConfig.Implement(exporter.service)
	err := exporter.ServiceConfig.Export()
	if err != nil {
		return err
	}
	url := exporter.ServiceConfig.GetExportedUrls()[0]
	exporter.service.SetMetadataServiceURL(url)
	logger.Infof("[Metadata Service] The MetadataService exports urls : %v ", exporter.ServiceConfig.GetExportedUrls())
	return nil
}

// UnExport will unExport the metadataService
func (exporter *MetadataServiceExporter) UnExport() {
	exporter.ServiceConfig.Unexport()
}
