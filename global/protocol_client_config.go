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

// TODO: find a better name replace ProtocolClientConfig
type ProtocolClientConfig struct {
	// TODO: maybe we could use this field
	Name string `yaml:"name" json:"name,omitempty" property:"name"`

	TripleConfig *TripleConfig `yaml:"triple" json:"triple,omitempty" property:"triple"`
}

// DefaultProtocolConfig returns a default ProtocolConfig instance.
func DefaultProtocolClientConfig() *ProtocolClientConfig {
	return &ProtocolClientConfig{
		Name:         constant.TriProtocol,
		TripleConfig: DefaultTripleConfig(),
	}
}

// Clone a new ProtocolConfig
func (c *ProtocolClientConfig) Clone() *ProtocolClientConfig {
	if c == nil {
		return nil
	}

	return &ProtocolClientConfig{
		Name:         c.Name,
		TripleConfig: c.TripleConfig.Clone(),
	}
}
