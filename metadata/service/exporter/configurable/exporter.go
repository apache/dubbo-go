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
	"sync"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/config"
	_ "dubbo.apache.org/dubbo-go/v3/metadata/mapping/metadata"
	"dubbo.apache.org/dubbo-go/v3/metadata/service"
	"dubbo.apache.org/dubbo-go/v3/metadata/service/exporter"
	_ "dubbo.apache.org/dubbo-go/v3/metadata/service/remote"
	_ "dubbo.apache.org/dubbo-go/v3/protocol/dubbo"
)

// MetadataServiceExporter is the ConfigurableMetadataServiceExporter which implement MetadataServiceExporter interface
type MetadataServiceExporter struct {
	ServiceConfig   *config.ServiceConfig
	lock            sync.RWMutex
	metadataService service.MetadataService
}

func init() {
	extension.SetMetadataServiceExporter(constant.DEFAULT_Key, NewMetadataServiceExporter)
}

// NewMetadataServiceExporter will return a service_exporter.MetadataServiceExporter with the specified  metadata service
func NewMetadataServiceExporter(metadataService service.MetadataService) exporter.MetadataServiceExporter {
	return &MetadataServiceExporter{
		metadataService: metadataService,
	}
}

// Export will export the metadataService
func (exporter *MetadataServiceExporter) Export(url *common.URL) error {
	if !exporter.IsExported() {
		version, _ := exporter.metadataService.Version()
		exporter.lock.Lock()
		defer exporter.lock.Unlock()
		exporter.ServiceConfig = config.NewServiceConfigBuilder().
			SetServiceID(constant.SIMPLE_METADATA_SERVICE_NAME).
			SetProtocolIDs(constant.DEFAULT_PROTOCOL).
			AddRCProtocol(constant.DEFAULT_PROTOCOL, config.NewProtocolConfigBuilder().
				SetName(constant.DEFAULT_PROTOCOL).
				Build()).
			SetRegistryIDs("N/A").
			SetInterface(constant.METADATA_SERVICE_NAME).
			SetGroup(config.GetApplicationConfig().Name).
			SetVersion(version).
			SetProxyFactoryKey(constant.DEFAULT_Key).
			SetMetadataType(constant.REMOTE_METADATA_STORAGE_TYPE).
			Build()
		exporter.ServiceConfig.Implement(exporter.metadataService)
		err := exporter.ServiceConfig.Export()

		logger.Infof("The MetadataService exports urls : %v ", exporter.ServiceConfig.GetExportedUrls())
		return err
	}
	logger.Warnf("The MetadataService has been exported : %v ", exporter.ServiceConfig.GetExportedUrls())
	return nil
}

// Unexport will unexport the metadataService
func (exporter *MetadataServiceExporter) Unexport() {
	if exporter.IsExported() {
		exporter.ServiceConfig.Unexport()
	}
}

// GetExportedURLs will return the urls that export use.
// Notice！The exported url is not same as url in registry , for example it lack the ip.
func (exporter *MetadataServiceExporter) GetExportedURLs() []*common.URL {
	return exporter.ServiceConfig.GetExportedUrls()
}

// isExported will return is metadataServiceExporter exported or not
func (exporter *MetadataServiceExporter) IsExported() bool {
	exporter.lock.RLock()
	defer exporter.lock.RUnlock()
	return exporter.ServiceConfig != nil && exporter.ServiceConfig.IsExport()
}
