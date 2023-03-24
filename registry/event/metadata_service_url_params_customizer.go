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
	"encoding/json"
)

import (
	gxset "github.com/dubbogo/gost/container/set"
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/metadata/service/local"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

func init() {
	exceptKeys := gxset.NewSet(
		// remove ApplicationKey because service name must be present
		constant.ApplicationKey,
		// remove GroupKey, always uses service name.
		constant.GroupKey,
		// remove TimestampKey because it's nonsense
		constant.TimestampKey)
	extension.AddCustomizers(&metadataServiceURLParamsMetadataCustomizer{exceptKeys: exceptKeys})
}

type metadataServiceURLParamsMetadataCustomizer struct {
	exceptKeys *gxset.HashSet
}

// GetPriority will return 0 so that it will be invoked in front of user defining Customizer
func (m *metadataServiceURLParamsMetadataCustomizer) GetPriority() int {
	return 0
}

func (m *metadataServiceURLParamsMetadataCustomizer) Customize(instance registry.ServiceInstance) {
	ms, err := local.GetLocalMetadataService()
	if err != nil {
		logger.Errorf("could not find the metadata service", err)
		return
	}
	url, err := ms.GetMetadataServiceURL()
	if url == nil || err != nil {
		logger.Errorf("the metadata service url is nil")
		return
	}
	ps := m.convertToParams(url)
	str, err := json.Marshal(ps)
	if err != nil {
		logger.Errorf("could not transfer the map to json", err)
		return
	}
	instance.GetMetadata()[constant.MetadataServiceURLParamsPropertyName] = string(str)
}

func (m *metadataServiceURLParamsMetadataCustomizer) convertToParams(url *common.URL) map[string]string {
	// those keys are useless
	p := make(map[string]string, len(url.GetParams()))
	for k, v := range url.GetParams() {
		// we will ignore that
		if !common.IncludeKeys.Contains(k) || len(v) == 0 || len(v[0]) == 0 {
			continue
		}
		p[k] = v[0]
	}
	p[constant.PortKey] = url.Port
	p[constant.ProtocolKey] = url.Protocol
	return p
}
