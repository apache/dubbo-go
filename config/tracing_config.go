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

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

// TracingConfig is the configuration of the tracing.
type TracingConfig struct {
	Name        string `default:"jaeger" yaml:"name" json:"name,omitempty" property:"name"` // jaeger or zipkin(todo)
	ServiceName string `yaml:"serviceName" json:"serviceName,omitempty" property:"serviceName"`
	Address     string `yaml:"address" json:"address,omitempty" property:"address"`
	UseAgent    *bool  `default:"false" yaml:"use-agent" json:"use-agent,omitempty" property:"use-agent"`
}

// Prefix dubbo.router
func (TracingConfig) Prefix() string {
	return constant.TracingConfigPrefix
}

func (c *TracingConfig) Init() error {
	if err := defaults.Set(c); err != nil {
		return err
	}
	return verify(c)
}
