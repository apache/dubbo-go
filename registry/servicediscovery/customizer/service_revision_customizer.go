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

// Customize calculate the revision for exported urls and then put it into instance metadata
func (e *exportedServicesRevisionMetadataCustomizer) Customize(instance registry.ServiceInstance) {
	metaInfo := instance.GetServiceMetadata()
	if metaInfo == nil {
		logger.Warn("[Registry][ServiceDiscovery] exportedServicesRevision customizer: instance service metadata is nil")
		return
	}
	revision := resolveRevision(metaInfo.GetExportedServiceURLs())
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

// Customize calculate the revision for subscribed urls and then put it into instance metadata
func (e *subscribedServicesRevisionMetadataCustomizer) Customize(instance registry.ServiceInstance) {
	metaInfo := instance.GetServiceMetadata()
	if metaInfo == nil {
		logger.Warn("[Registry][ServiceDiscovery] subscribedServicesRevision customizer: instance service metadata is nil")
		return
	}
	revision := resolveRevision(metaInfo.GetSubscribedURLs())
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
