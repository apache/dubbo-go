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

package global

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

// ProtocolConfig represents the config of protocol.
type ProtocolConfig struct {
	Name string `yaml:"name" json:"name,omitempty" property:"name"`
	Ip   string `yaml:"ip"  json:"ip,omitempty" property:"ip"`
	Port string `yaml:"port" json:"port,omitempty" property:"port"`

	// TODO: maybe Params is useless, find a ideal way to config dubbo protocol, ref: TripleConfig.
	Params any `yaml:"params" json:"params,omitempty" property:"params"`

	// TripleConfig holds the Triple protocol configuration.
	TripleConfig *TripleConfig `yaml:"triple" json:"triple,omitempty" property:"triple"`
}

// DefaultProtocolConfig returns a default ProtocolConfig instance.
func DefaultProtocolConfig() *ProtocolConfig {
	return &ProtocolConfig{
		Name:         constant.TriProtocol,
		Port:         constant.DefaultTripleProtocolPort,
		TripleConfig: DefaultTripleConfig(),
	}
}

// Clone a new ProtocolConfig
func (c *ProtocolConfig) Clone() *ProtocolConfig {
	if c == nil {
		return nil
	}

	return &ProtocolConfig{
		Name:         c.Name,
		Ip:           c.Ip,
		Port:         c.Port,
		Params:       c.Params,
		TripleConfig: c.TripleConfig.Clone(),
	}
}
