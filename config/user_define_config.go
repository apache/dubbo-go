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

import(
	"github.com/creasty/defaults"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

type UserDefineConfig struct {
	Name         string                 `default:"user-config" yaml:"name" json:"name,omitempty" property:"name"`
	Version      string                 `default:"v1.0" yaml:"version" json:"version,omitempty" property:"version"`
	DefineConfig map[string]interface{} `yaml:"define-config" json:"define-config,omitempty" property:"define-config"`
}

func (*UserDefineConfig) Prefix() string {
	return constant.UserDefineConfigPrefix
}

func (udc *UserDefineConfig) Init() error {
	return udc.check()
}

func (udc *UserDefineConfig) check() error {
	if err := defaults.Set(udc); err != nil {
		return err
	}
	return verify(udc)
}

func (udc *UserDefineConfig) GetDefineValue(key string, default_value interface{}) interface{} {
	if define_value, ok := udc.DefineConfig[key]; ok {
		return define_value
	}
	return default_value
}

type UserDefineConfigBuilder struct {
	userDefineConfig *UserDefineConfig
}

func NewUserDefineConfigBuilder() *UserDefineConfigBuilder {
	return &UserDefineConfigBuilder{userDefineConfig: &UserDefineConfig{}}
}

func (udcb *UserDefineConfigBuilder) SetName(name string) *UserDefineConfigBuilder {
	udcb.userDefineConfig.Name = name
	return udcb
}

func (udcb *UserDefineConfigBuilder) SetVersion(version string) *UserDefineConfigBuilder {
	udcb.userDefineConfig.Version = version
	return udcb
}

func (udcb *UserDefineConfigBuilder) SetDefineConfig(key string, val interface{}) *UserDefineConfigBuilder {
	udcb.userDefineConfig.DefineConfig[key] = val
	return udcb
}

func (udcb *UserDefineConfigBuilder) Build() *UserDefineConfig {
	return udcb.userDefineConfig
}
