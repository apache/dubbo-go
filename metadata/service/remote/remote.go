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

package remote

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

// RemoteMetadataService for save and get metadata
type RemoteMetadataService interface {
	// PublishMetadata publish the medata info of service from report
	PublishMetadata(service string)
	// GetMetadata get the medata info of service from report
	GetMetadata(instance registry.ServiceInstance) (*common.MetadataInfo, error)
	// PublishServiceDefinition will call remote metadata's StoreProviderMetadata to store url info and service definition
	PublishServiceDefinition(url *common.URL) error
}
