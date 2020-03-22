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
	"github.com/emirpasic/gods/sets"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/config"
)

// Metadataservice is used to define meta data related behaviors
type MetadataService interface {
	ServiceName() string
	ExportURL(url common.URL) bool
	UnexportURL(url common.URL) bool
	RefreshMetadata(exportedRevision string, subscribedRevision string) bool
	SubscribeURL(url common.URL) bool
	UnsubscribeURL(url common.URL) bool
	PublishServiceDefinition(url common.URL)

	GetExportedURLs(serviceInterface string, group string, version string, protocol string) sets.Set
	GetServiceDefinition(interfaceName string, version string, group string) string
	GetServiceDefinitionByServiceKey(serviceKey string) string
}

// BaseMetadataService: is used for the common logic for struct who will implement interface MetadataService
type BaseMetadataService struct {
}

// ServiceName: get the service's name in meta service , which is application name
func (mts *BaseMetadataService) ServiceName() string {
	return config.GetApplicationConfig().Name
}

// RefreshMetadata: used for event listener's calling, to refresh metadata
func (mts *BaseMetadataService) RefreshMetadata(exportedRevision string, subscribedRevision string) bool {
	return true
}
