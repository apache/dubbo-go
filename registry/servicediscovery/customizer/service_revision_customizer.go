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
	"fmt"
	"hash/crc32"
	"sort"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/metadata"
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

// Customize 按注册中心范围计算 exported services 的 revision，
// 避免多注册中心时不同 instance 因使用跨注册中心合并的服务列表而得到相同 revision。
func (e *exportedServicesRevisionMetadataCustomizer) Customize(instance registry.ServiceInstance) {
	registryId := instance.GetMetadata()[constant.RegistryIdKey]
	if len(registryId) == 0 {
		logger.Warnf("[Registry][ServiceDiscovery] instance has no registryId in metadata, exported revision will be empty")
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

// Customize 按注册中心范围计算 subscribed services 的 revision。
func (e *subscribedServicesRevisionMetadataCustomizer) Customize(instance registry.ServiceInstance) {
	registryId := instance.GetMetadata()[constant.RegistryIdKey]
	if len(registryId) == 0 {
		logger.Warnf("[Registry][ServiceDiscovery] instance has no registryId in metadata, subscribed revision will be empty")
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

// resolveRevision provides the actual pattern to calculate the revision.
// please refer to dubbo-java's method, org.apache.dubbo.metadata.Metadata#calAndGetRevision
func resolveRevision(urls []*common.URL) string {
	if len(urls) == 0 {
		return "0"
	}
	candidates := make([]string, 0, len(urls))

	for _, u := range urls {
		desc := u.GetParam(constant.ApplicationKey, "") + u.Path + u.GetParam(constant.VersionKey, "") + u.Port

		if len(u.Methods) == 0 {
			candidates = append(candidates, desc)
		} else {
			for _, m := range u.Methods {
				// methods are part of candidates
				candidates = append(candidates, desc+constant.KeySeparator+m)
			}
		}

	}
	sort.Strings(candidates)

	// it's nearly impossible to be overflow
	res := uint64(0)
	for _, c := range candidates {
		res += uint64(crc32.ChecksumIEEE([]byte(c)))
	}
	return fmt.Sprint(res)
}
