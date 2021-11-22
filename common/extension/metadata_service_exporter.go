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

package extension

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/metadata/service"
	"dubbo.apache.org/dubbo-go/v3/metadata/service/exporter"
)

type MetadataServiceExporterCreator func(service.MetadataService) exporter.MetadataServiceExporter

var (
	metadataServiceExporterInsMap = make(map[string]MetadataServiceExporterCreator, 2)
)

// SetMetadataServiceExporter will store the type => creator pair
func SetMetadataServiceExporter(key string, creator MetadataServiceExporterCreator) {
	metadataServiceExporterInsMap[key] = creator
}

// GetMetadataServiceExporter will create a MetadataServiceExporter instance
func GetMetadataServiceExporter(key string, s service.MetadataService) exporter.MetadataServiceExporter {
	if key == "" {
		key = constant.DefaultKey
	}
	if creator, ok := metadataServiceExporterInsMap[key]; ok {
		return creator(s)
	}
	return nil
}
