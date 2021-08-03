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
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"github.com/creasty/defaults"
)

// ProtocolConfig is protocol configuration
type ProtocolConfig struct {
	Name string `default:"dubbo" validate:"required" yaml:"name" json:"name,omitempty" property:"name"`
	Ip   string `default:"127.0.0.1" yaml:"ip"  json:"ip,omitempty" property:"ip"`
	Port string `default:"2000" yaml:"port" json:"port,omitempty" property:"port"`
}

func initProtocolsConfig(rc *RootConfig) error {
	protocols := rc.Protocols
	if len(protocols) <= 0 {
		protocol := new(ProtocolConfig)
		protocols = make(map[string]*ProtocolConfig, 1)
		protocols[constant.DUBBO] = protocol
		rc.Protocols = protocols
		return protocol.check()
	}
	for _, protocol := range protocols {
		if err := protocol.check(); err != nil {
			return err
		}
	}
	rc.Protocols = protocols
	return nil
}

func (p *ProtocolConfig) check() error {
	defaults.MustSet(p)
	return verify(p)
}
