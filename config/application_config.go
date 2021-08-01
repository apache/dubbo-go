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

// ApplicationConfig is a configuration for current applicationConfig, whether the applicationConfig is a provider or a consumer
type ApplicationConfig struct {
	Organization string `default:"dubbo.io" yaml:"organization" json:"organization,omitempty" property:"organization"`
	Name         string `default:"dubbo.io" yaml:"name" json:"name,omitempty" property:"name"`
	Module       string `default:"sample" yaml:"module" json:"module,omitempty" property:"module"`
	Version      string `default:"0.0.1" yaml:"version" json:"version,omitempty" property:"version"`
	Owner        string `default:"dubbo-go" yaml:"owner" json:"owner,omitempty" property:"owner"`
	Environment  string `default:"dev" yaml:"environment" json:"environment,omitempty" property:"environment"`
	// the metadata type. remote or local
	MetadataType string `default:"local" yaml:"metadataType" json:"metadataType,omitempty" property:"metadataType"`
}

func NewApplicationConfig() *ApplicationConfig {
	return &ApplicationConfig{}
}

// Prefix dubbo.applicationConfig
func (ApplicationConfig) Prefix() string {
	return constant.DUBBO + ".applicationConfig"
}

func (a *ApplicationConfig) CheckConfig() error {
	// todo check
	defaults.MustSet(a)
	return verify(a)
}

func (a *ApplicationConfig) Validate() {
	defaults.MustSet(a)
	// todo set default application
}
