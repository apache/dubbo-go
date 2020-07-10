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
	"strconv"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/registry"
)

func init() {
	extension.AddCustomizers(&ProtocolPortsMetadataCustomizer{})
}

// ProtocolPortsMetadataCustomizer will update the endpoints
type ProtocolPortsMetadataCustomizer struct {
}

// GetPriority will return 0, which means it will be invoked at the beginning
func (p *ProtocolPortsMetadataCustomizer) GetPriority() int {
	return 0
}

// Customize put the the string like [{"protocol": "dubbo", "port": 123}] into instance's metadata
func (p *ProtocolPortsMetadataCustomizer) Customize(instance registry.ServiceInstance) {
	metadataService, err := getMetadataService()
	if err != nil {
		logger.Errorf("Could not init the MetadataService", err)
		return
	}

	// 4 is enough... we don't have many protocol
	protocolMap := make(map[string]int, 4)

	list, err := metadataService.GetExportedURLs(constant.ANY_VALUE, constant.ANY_VALUE, constant.ANY_VALUE, constant.ANY_VALUE)
	if err != nil || len(list) == 0 {
		logger.Debugf("Could not find exported urls", err)
		return
	}

	for _, ui := range list {
		u, err := common.NewURL(ui.(string))
		if err != nil || len(u.Protocol) == 0 {
			logger.Errorf("the url string is invalid: %s", ui.(string), err)
			continue
		}

		port, err := strconv.Atoi(u.Port)
		if err != nil {
			logger.Errorf("Could not customize the metadata of port. ", err)
		}
		protocolMap[u.Protocol] = port
	}

	instance.GetMetadata()[constant.SERVICE_INSTANCE_ENDPOINTS] = endpointsStr(protocolMap)
}

// endpointsStr convert the map to json like [{"protocol": "dubbo", "port": 123}]
func endpointsStr(protocolMap map[string]int) string {
	if len(protocolMap) == 0 {
		return ""
	}

	endpoints := make([]endpoint, 0, len(protocolMap))
	for k, v := range protocolMap {
		endpoints = append(endpoints, endpoint{
			Port:     v,
			Protocol: k,
		})
	}

	str, err := json.Marshal(endpoints)
	if err != nil {
		logger.Errorf("could not convert the endpoints to json", err)
		return ""
	}
	return string(str)
}

// nolint
type endpoint struct {
	Port     int    `json:"port, omitempty"`
	Protocol string `json:"protocol, omitempty"`
}
