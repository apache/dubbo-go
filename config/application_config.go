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

// ApplicationConfig is a configuration for current application, whether the application is a provider or a consumer
type ApplicationConfig struct {
	Organization string `yaml:"organization" json:"organization,omitempty" property:"organization"`
	Name         string `yaml:"name" json:"name,omitempty" property:"name"`
	Module       string `yaml:"module" json:"module,omitempty" property:"module"`
	Version      string `yaml:"version" json:"version,omitempty" property:"version"`
	Owner        string `yaml:"owner" json:"owner,omitempty" property:"owner"`
	Environment  string `yaml:"environment" json:"environment,omitempty" property:"environment"`
	// the metadata type. remote or local
	MetadataType string `default:"local" yaml:"metadataType" json:"metadataType,omitempty" property:"metadataType"`
}

// nolint
func (*ApplicationConfig) Prefix() string {
	return constant.DUBBO + ".application."
}

// UnmarshalYAML unmarshals the ApplicationConfig by @unmarshal function
func (c *ApplicationConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	if err := defaults.Set(c); err != nil {
		return err
	}
	type plain ApplicationConfig
	return unmarshal((*plain)(c))
}
