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
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/registry"
)

func init() {
	exceptKeys := gxset.NewSet(
		// remove APPLICATION_KEY because service name must be present
		constant.APPLICATION_KEY,
		// remove GROUP_KEY, always uses service name.
		constant.GROUP_KEY,
		// remove TIMESTAMP_KEY because it's nonsense
		constant.TIMESTAMP_KEY)
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
	ms, err := getMetadataService()
	if err != nil {
		logger.Errorf("could not find the metadata service", err)
		return
	}
	serviceName := constant.METADATA_SERVICE_NAME
	// error always is nil
	version, _ := ms.Version()
	group := instance.GetServiceName()
	urls, err := ms.GetExportedURLs(serviceName, group, version, constant.ANY_VALUE)
	if err != nil || len(urls) == 0 {
		logger.Info("could not find the exported urls", err)
		return
	}
	ps := m.convertToParams(urls)
	str, err := json.Marshal(ps)
	if err != nil {
		logger.Errorf("could not transfer the map to json", err)
		return
	}
	instance.GetMetadata()[constant.METADATA_SERVICE_URL_PARAMS_PROPERTY_NAME] = string(str)
}

func (m *metadataServiceURLParamsMetadataCustomizer) convertToParams(urls []interface{}) map[string]map[string]string {

	// usually there will be only one protocol
	res := make(map[string]map[string]string, 1)
	// those keys are useless

	for _, ui := range urls {
		u, err := common.NewURL(ui.(string))
		if err != nil {
			logger.Errorf("could not parse the string to url: %s", ui.(string), err)
			continue
		}
		p := make(map[string]string, len(u.GetParams()))
		for k, v := range u.GetParams() {
			// we will ignore that
			if m.exceptKeys.Contains(k) || len(v) == 0 || len(v[0]) == 0 {
				continue
			}
			p[k] = v[0]
		}
		p[constant.PORT_KEY] = u.Port
		res[u.Protocol] = p
	}
	return res
}
