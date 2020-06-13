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

	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/registry"
)

// ProtocolPortsMetadataCustomizer will update
type ProtocolPortsMetadataCustomizer struct {
}

// GetPriority will return 0, which means it will be invoked at the beginning
func (p *ProtocolPortsMetadataCustomizer) GetPriority() int {
	return 0
}

// Customize will
func (p *ProtocolPortsMetadataCustomizer) Customize(instance registry.ServiceInstance) {
	metadataService, err := getMetadataService()
	if err != nil {
		logger.Errorf("Could not init the MetadataService", err)
		return
	}

	// 4 is enough...
	protocolMap := make(map[string]int, 4)

	list, err := metadataService.GetExportedURLs(constant.ANY_VALUE, constant.ANY_VALUE, constant.ANY_VALUE, constant.ANY_VALUE)
	if err != nil || list == nil {
		logger.Errorf("Could not find exported urls", err)
		return
	}

	for _, ui := range list {
		u := ui.(common.URL)
		if len(u.Protocol) == 0 {
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

func endpointsStr(protocolMap map[string]int) string {
	if len(protocolMap) == 0 {
		return ""
	}

	endpoints := make([]endpoint, 0, len(protocolMap))
	for k, v := range protocolMap {
		endpoints = append(endpoints, endpoint{
			port:     v,
			protocol: k,
		})
	}

	str, err := json.Marshal(endpoints)
	if err != nil {
		logger.Errorf("could not convert the endpoints to json", err)
		return ""
	}
	return string(str)
}

type endpoint struct {
	port     int
	protocol string
}
