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

package metadata

import (
	"github.com/apache/dubbo-go/common"
	gxset "github.com/dubbogo/gost/container/set"
)

// Metadata service is a built-in service around the metadata of Dubbo services,
// whose interface is provided by Dubbo Framework and exported automatically before subscription after other services exporting,
// which may be used for Dubbo subscribers and admin.
type MetadataService interface {
	ServiceName() string
	ExportURL(url *common.URL) bool
	UnexportURL(url *common.URL) bool
	RefreshMetadata(exportedRevision string, subscribedRevision string) bool
	SubscribeURL(url *common.URL) bool
	UnsubscribeURL(url *common.URL) bool
	PublishServiceDefinition(url *common.URL)

	GetExportedURLs(serviceInterface string, group string, version string, protocol string) gxset.HashSet
	GetServiceDefinition(interfaceName string, version string, group string) string
	GetServiceDefinitionByServiceKey(serviceKey string) string
}
