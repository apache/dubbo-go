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

package customizer

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/metadata"
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

const defaultRevision = "N/A"

func init() {
	extension.AddCustomizers(&exportedServicesRevisionMetadataCustomizer{})
	extension.AddCustomizers(&subscribedServicesRevisionMetadataCustomizer{})
}

type exportedServicesRevisionMetadataCustomizer struct{}

// GetPriority will return 1 so that it will be invoked in front of user defining Customizer
func (e *exportedServicesRevisionMetadataCustomizer) GetPriority() int {
	return 1
}

// Customize calculates the revision of exported services scoped to the registry,
// preventing different instances from getting the same revision due to a merged cross-registry service list in multi-registry setups.
func (e *exportedServicesRevisionMetadataCustomizer) Customize(instance registry.ServiceInstance) {
	registryId := instance.GetMetadata()[constant.RegistryIdKey]
	if len(registryId) == 0 {
		logger.Errorf("[Registry][ServiceDiscovery] instance has no registryId in metadata; " +
			"exported revision will be \"0\" and this instance will be invisible to consumers")
	}
	metaInfo := metadata.GetMetadataInfo(registryId)
	var urls []*common.URL
	if metaInfo != nil {
		urls = metaInfo.GetExportedServiceURLs()
	}
	revision := resolveRevision(urls)
	if len(revision) == 0 {
		revision = defaultRevision
	}
	instance.GetMetadata()[constant.ExportedServicesRevisionPropertyName] = revision
}

type subscribedServicesRevisionMetadataCustomizer struct{}

// GetPriority will return 2 so that it will be invoked in front of user defining Customizer
func (e *subscribedServicesRevisionMetadataCustomizer) GetPriority() int {
	return 2
}

// Customize calculates the revision of subscribed services scoped to the registry.
func (e *subscribedServicesRevisionMetadataCustomizer) Customize(instance registry.ServiceInstance) {
	registryId := instance.GetMetadata()[constant.RegistryIdKey]
	if len(registryId) == 0 {
		logger.Errorf("[Registry][ServiceDiscovery] instance has no registryId in metadata; " +
			"subscribed revision will be \"0\" and this instance will be invisible to consumers")
	}
	metaInfo := metadata.GetMetadataInfo(registryId)
	var urls []*common.URL
	if metaInfo != nil {
		urls = metaInfo.GetSubscribedURLs()
	}
	revision := resolveRevision(urls)
	if len(revision) == 0 {
		revision = defaultRevision
	}
	instance.GetMetadata()[constant.SubscribedServicesRevisionPropertyName] = revision
}

// resolveRevision calculates a deterministic revision from the given URLs.
// It converts URLs to canonical ServiceInfo objects and delegates to info.CalRevision,
// aligning with Java dubbo's MetadataInfo.calAndGetRevision().
func resolveRevision(urls []*common.URL) string {
	if len(urls) == 0 {
		return "0"
	}

	// build canonical ServiceInfo map from URLs, keyed by MatchKey
	services := make(map[string]*info.ServiceInfo, len(urls))
	app := ""
	for _, u := range urls {
		si := info.NewServiceInfoWithURL(u)
		services[si.GetMatchKey()] = si
		if app == "" {
			app = u.GetParam(constant.ApplicationKey, "")
		}
	}

	return info.CalRevision(app, services)
}
