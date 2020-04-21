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
	"github.com/Workiva/go-datastructures/slice/skip"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/config"
)

// Metadataservice is used to define meta data related behaviors
type MetadataService interface {
	// ServiceName will get the service's name in meta service , which is application name
	ServiceName() (string, error)
	// ExportURL will store the exported url in metadata
	ExportURL(url common.URL) (bool, error)
	// UnexportURL will delete the exported url in metadata
	UnexportURL(url common.URL) error
	// SubscribeURL will store the subscribed url in metadata
	SubscribeURL(url common.URL) (bool, error)
	// UnsubscribeURL will delete the subscribed url in metadata
	UnsubscribeURL(url common.URL) error
	// PublishServiceDefinition will generate the target url's code info
	PublishServiceDefinition(url common.URL) error
	// GetExportedURLs will get the target exported url in metadata
	GetExportedURLs(serviceInterface string, group string, version string, protocol string) (*skip.SkipList, error)
	// GetExportedURLs will get the target subscribed url in metadata
	GetSubscribedURLs() (*skip.SkipList, error)
	// GetServiceDefinition will get the target service info store in metadata
	GetServiceDefinition(interfaceName string, group string, version string) (string, error)
	// GetServiceDefinition will get the target service info store in metadata by service key
	GetServiceDefinitionByServiceKey(serviceKey string) (string, error)
	// Version will return the metadata service version
	Version() string
	common.RPCService
}

// BaseMetadataService is used for the common logic for struct who will implement interface MetadataService
type BaseMetadataService struct {
}

// ServiceName can get the service's name in meta service , which is application name
func (mts *BaseMetadataService) ServiceName() (string, error) {
	return config.GetApplicationConfig().Name, nil
}
