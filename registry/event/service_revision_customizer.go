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

package event

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
	"dubbo.apache.org/dubbo-go/v3/metadata/service/local"
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
	ms, err := local.GetLocalMetadataService()
	if err != nil {
		logger.Errorf("could not get metadata service", err)
		return
	}

	urls, err := ms.GetExportedURLs(constant.AnyValue, constant.AnyValue, constant.AnyValue, constant.AnyValue)
	if err != nil {
		logger.Errorf("could not find the exported url", err)
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

// Customize calculate the revision for subscribed urls and then put it into instance metadata
func (e *subscribedServicesRevisionMetadataCustomizer) Customize(instance registry.ServiceInstance) {
	ms, err := local.GetLocalMetadataService()
	if err != nil {
		logger.Errorf("could not get metadata service", err)
		return
	}

	urls, err := ms.GetSubscribedURLs()
	if err != nil {
		logger.Errorf("could not find the subscribed url", err)
	}

	revision := resolveRevision(urls)
	if len(revision) == 0 {
		revision = defaultRevision
	}
	instance.GetMetadata()[constant.SubscribedServicesRevisionPropertyName] = revision
}

// resolveRevision is different from Dubbo because golang doesn't support overload
// so that we could use interface + method name as identifier and ignore the method params
// per my understanding, it's enough because Dubbo actually ignore the url params.
// please refer org.apache.dubbo.common.URL#toParameterString(java.lang.String...)
func resolveRevision(urls []*common.URL) string {
	if len(urls) == 0 {
		return "0"
	}
	candidates := make([]string, 0, len(urls))

	for _, u := range urls {
		sk := u.GetParam(constant.InterfaceKey, "")

		if len(u.Methods) == 0 {
			candidates = append(candidates, sk)
		} else {
			for _, m := range u.Methods {
				// methods are part of candidates
				candidates = append(candidates, sk+constant.KeySeparator+m)
			}
		}

		// append url params if we need it
	}
	sort.Strings(candidates)

	// it's nearly impossible to be overflow
	res := uint64(0)
	for _, c := range candidates {
		res += uint64(crc32.ChecksumIEEE([]byte(c)))
	}
	return fmt.Sprint(res)
}
