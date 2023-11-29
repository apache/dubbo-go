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

// Package metadata collects and exposes information of all services for service discovery purpose.
package metadata

import (
	"sync"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

var (
	GlobalMetadataService MetadataService = &DefaultMetadataService{}
	exportOnce            sync.Once
	Factory               ExporterFactory
)

// ExporterFactory to create ServiceExporter, avoid cycle import
type ExporterFactory func(app, metadataType string, service MetadataService) ServiceExporter

func SetExporterFactory(factory ExporterFactory) {
	Factory = factory
}

func ExportMetadataService(app, metadataType string) {
	if Factory != nil {
		exportOnce.Do(func() {
			if metadataType != constant.RemoteMetadataStorageType {
				err := Factory(app, metadataType, GlobalMetadataService).Export()
				if err != nil {
					logger.Errorf("export metadata service failed, got error %#v", err)
				}
			}
		})
	} else {
		logger.Warn("no metadata service exporter found, MetadataService will not be Exported")
	}
}
