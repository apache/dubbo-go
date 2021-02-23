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

var (
	restConsumerServiceConfigMap map[string]*RestServiceConfig
	restProviderServiceConfigMap map[string]*RestServiceConfig
)

// nolint
type RestConsumerConfig struct {
	Client                string                        `default:"resty" yaml:"rest_client" json:"rest_client,omitempty" property:"rest_client"`
	Produces              string                        `default:"application/json" yaml:"rest_produces"  json:"rest_produces,omitempty" property:"rest_produces"`
	Consumes              string                        `default:"application/json" yaml:"rest_consumes"  json:"rest_consumes,omitempty" property:"rest_consumes"`
	RestServiceConfigsMap map[string]*RestServiceConfig `yaml:"references" json:"references,omitempty" property:"references"`
}

// UnmarshalYAML unmarshals the RestConsumerConfig by @unmarshal function
func (c *RestConsumerConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	if err := defaults.Set(c); err != nil {
		return err
	}
	type plain RestConsumerConfig
	if err := unmarshal((*plain)(c)); err != nil {
		return err
	}
	return nil
}

// nolint
type RestProviderConfig struct {
	Server                string                        `default:"go-restful" yaml:"rest_server" json:"rest_server,omitempty" property:"rest_server"`
	Produces              string                        `default:"*/*" yaml:"rest_produces"  json:"rest_produces,omitempty" property:"rest_produces"`
	Consumes              string                        `default:"*/*" yaml:"rest_consumes"  json:"rest_consumes,omitempty" property:"rest_consumes"`
	RestServiceConfigsMap map[string]*RestServiceConfig `yaml:"services" json:"services,omitempty" property:"services"`
}

// UnmarshalYAML unmarshals the RestProviderConfig by @unmarshal function
func (c *RestProviderConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	if err := defaults.Set(c); err != nil {
		return err
	}
	type plain RestProviderConfig
	if err := unmarshal((*plain)(c)); err != nil {
		return err
	}
	return nil
}

// nolint
type RestServiceConfig struct {
	InterfaceName        string              `required:"true"  yaml:"interface"  json:"interface,omitempty" property:"interface"`
	Url                  string              `yaml:"url"  json:"url,omitempty" property:"url"`
	Path                 string              `yaml:"rest_path"  json:"rest_path,omitempty" property:"rest_path"`
	Produces             string              `yaml:"rest_produces"  json:"rest_produces,omitempty" property:"rest_produces"`
	Consumes             string              `yaml:"rest_consumes"  json:"rest_consumes,omitempty" property:"rest_consumes"`
	MethodType           string              `yaml:"rest_method"  json:"rest_method,omitempty" property:"rest_method"`
	Client               string              `yaml:"rest_client" json:"rest_client,omitempty" property:"rest_client"`
	Server               string              `yaml:"rest_server" json:"rest_server,omitempty" property:"rest_server"`
	RestMethodConfigs    []*RestMethodConfig `yaml:"methods" json:"methods,omitempty" property:"methods"`
	RestMethodConfigsMap map[string]*RestMethodConfig
}

// UnmarshalYAML unmarshals the RestServiceConfig by @unmarshal function
func (c *RestServiceConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	if err := defaults.Set(c); err != nil {
		return err
	}
	type plain RestServiceConfig
	if err := unmarshal((*plain)(c)); err != nil {
		return err
	}
	return nil
}

// nolint
type RestMethodConfig struct {
	InterfaceName  string
	MethodName     string `required:"true" yaml:"name"  json:"name,omitempty" property:"name"`
	Url            string `yaml:"url"  json:"url,omitempty" property:"url"`
	Path           string `yaml:"rest_path"  json:"rest_path,omitempty" property:"rest_path"`
	Produces       string `yaml:"rest_produces"  json:"rest_produces,omitempty" property:"rest_produces"`
	Consumes       string `yaml:"rest_consumes"  json:"rest_consumes,omitempty" property:"rest_consumes"`
	MethodType     string `yaml:"rest_method"  json:"rest_method,omitempty" property:"rest_method"`
	PathParams     string `yaml:"rest_path_params"  json:"rest_path_params,omitempty" property:"rest_path_params"`
	PathParamsMap  map[int]string
	QueryParams    string `yaml:"rest_query_params"  json:"rest_query_params,omitempty" property:"rest_query_params"`
	QueryParamsMap map[int]string
	Body           int    `default:"-1" yaml:"rest_body"  json:"rest_body,omitempty" property:"rest_body"`
	Headers        string `yaml:"rest_headers"  json:"rest_headers,omitempty" property:"rest_headers"`
	HeadersMap     map[int]string
}

// UnmarshalYAML unmarshals the RestMethodConfig by @unmarshal function
func (c *RestMethodConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	if err := defaults.Set(c); err != nil {
		return err
	}
	type plain RestMethodConfig
	if err := unmarshal((*plain)(c)); err != nil {
		return err
	}
	return nil
}

// nolint
func GetRestConsumerServiceConfig(id string) *RestServiceConfig {
	return restConsumerServiceConfigMap[id]
}

// nolint
func GetRestProviderServiceConfig(id string) *RestServiceConfig {
	return restProviderServiceConfigMap[id]
}

// nolint
func SetRestConsumerServiceConfigMap(configMap map[string]*RestServiceConfig) {
	restConsumerServiceConfigMap = configMap
}

// nolint
func SetRestProviderServiceConfigMap(configMap map[string]*RestServiceConfig) {
	restProviderServiceConfigMap = configMap
}

// nolint
func GetRestConsumerServiceConfigMap() map[string]*RestServiceConfig {
	return restConsumerServiceConfigMap
}

// nolint
func GetRestProviderServiceConfigMap() map[string]*RestServiceConfig {
	return restProviderServiceConfigMap
}
