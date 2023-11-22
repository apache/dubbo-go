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
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
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
	// MethodMapper only for rename exported function, for example: rename the function GetMetadataInfo to getMetadataInfo
	MethodMapper() map[string]string
}

type BaseMetadataService struct {
}

func (mts *BaseMetadataService) MethodMapper() map[string]string {
	return map[string]string{
		"GetExportedURLs": "getExportedURLs",
		"GetMetadataInfo": "getMetadataInfo",
	}
}
