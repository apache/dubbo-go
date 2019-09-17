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
package config

import (
	"strings"
)

import (
	"github.com/apache/dubbo-go/common/constant"
)

type ProtocolConfig struct {
	Name string `required:"true" yaml:"name"  json:"name,omitempty" property:"name"`
	Ip   string `required:"true" yaml:"ip"  json:"ip,omitempty" property:"ip"`
	Port string `required:"true" yaml:"port"  json:"port,omitempty" property:"port"`
}

func (c *ProtocolConfig) Prefix() string {
	return constant.ProtocolConfigPrefix
}

func loadProtocol(protocolsIds string, protocols map[string]*ProtocolConfig) []*ProtocolConfig {
	returnProtocols := []*ProtocolConfig{}
	for _, v := range strings.Split(protocolsIds, ",") {
		for _, prot := range protocols {
			if v == prot.Name {
				returnProtocols = append(returnProtocols, prot)
			}
		}

	}
	return returnProtocols
}
