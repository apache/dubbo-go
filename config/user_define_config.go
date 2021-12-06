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

type CustomConfig struct {
	Name         string                 `default:"user-config" yaml:"name" json:"name,omitempty" property:"name"`
	Version      string                 `default:"v1.0" yaml:"version" json:"version,omitempty" property:"version"`
	DefineConfig map[string]interface{} `yaml:"define-config" json:"define-config,omitempty" property:"define-config"`
}

func (*CustomConfig) Prefix() string {
	return constant.CustomConfigPrefix
}

func (c *CustomConfig) Init() error {
	return c.check()
}

func (c *CustomConfig) check() error {
	if err := defaults.Set(c); err != nil {
		return err
	}
	return verify(c)
}

func (c *CustomConfig) GetDefineValue(key string, default_value interface{}) interface{} {
	if define_value, ok := c.DefineConfig[key]; ok {
		return define_value
	}
	return default_value
}

func GetDefineValue(key string, default_value interface{}) interface{} {
	rt := GetRootConfig()
	if rt.Custom == nil {
		return default_value
	}
	return rt.Custom.GetDefineValue(key, default_value)
}

type CustomConfigBuilder struct {
	customConfig *CustomConfig
}

func NewCustomConfigBuilder() *CustomConfigBuilder {
	return &CustomConfigBuilder{customConfig: &CustomConfig{}}
}

func (ccb *CustomConfigBuilder) SetName(name string) *CustomConfigBuilder {
	ccb.customConfig.Name = name
	return ccb
}

func (ccb *CustomConfigBuilder) SetVersion(version string) *CustomConfigBuilder {
	ccb.customConfig.Version = version
	return ccb
}

func (ccb *CustomConfigBuilder) SetDefineConfig(key string, val interface{}) *CustomConfigBuilder {
	if ccb.customConfig.DefineConfig == nil {
		ccb.customConfig.DefineConfig = make(map[string]interface{})
	}
	ccb.customConfig.DefineConfig[key] = val
	return ccb
}

func (ccb *CustomConfigBuilder) Build() *CustomConfig {
	return ccb.customConfig
}
