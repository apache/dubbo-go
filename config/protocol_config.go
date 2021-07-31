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
	"github.com/creasty/defaults"
)

// ProtocolConfig is protocol configuration
type ProtocolConfig struct {
	Name string `default:"dubbo" validate:"required" yaml:"name"  json:"name,omitempty" property:"name"`
	Ip   string `default:"127.0.0.1" yaml:"ip"  json:"ip,omitempty" property:"ip"`
	Port string `default:"0" yaml:"port" json:"port,omitempty" property:"port"`
}

func (p *ProtocolConfig) CheckConfig() error {
	// todo check
	defaults.MustSet(p)
	return verify(p)
}

func (p *ProtocolConfig) Validate() {

	// todo set default application
}
