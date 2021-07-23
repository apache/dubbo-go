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

package protocol

import (
	"dubbo.apache.org/dubbo-go/v3/config"
	"github.com/knadh/koanf"
	"strings"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

// Config is protocol configuration
type Config struct {
	Name string `required:"true" yaml:"name"  json:"name,omitempty" property:"name"`
	Ip   string `required:"true" yaml:"ip"  json:"ip,omitempty" property:"ip"`
	Port string `required:"true" yaml:"port"  json:"port,omitempty" property:"port"`
}

// Prefix dubbo.protocols
func (Config) Prefix() string {
	return constant.ProtocolConfigPrefix
}

func GetProtocolsConfig(protocols map[string]*Config, k *koanf.Koanf) map[string]*Config {
	if protocols != nil {
		return protocols
	}

	conf := new(Config)
	if value := k.Get(conf.Prefix()); value != nil {
		conf = value.(*Config)
	} else {

	}
	config.Koanf.Get(conf.Prefix())
	return nil
}
func loadProtocol(protocolsIds string, protocols map[string]*Config) []*Config {
	returnProtocols := make([]*Config, 0, len(protocols))
	for _, v := range strings.Split(protocolsIds, ",") {
		for k, protocol := range protocols {
			if v == k {
				returnProtocols = append(returnProtocols, protocol)
			}
		}
	}
	return returnProtocols
}
