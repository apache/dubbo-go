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
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"github.com/creasty/defaults"
	"strconv"
	"strings"
)

var (
	// pi todo Encapsulation it , do not export
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

// pi todo refactor
func (c *RestConsumerConfig) Init(rc *RootConfig) error {
	restConsumerConfig := rc.RestConsumer

	restConsumerServiceConfigMap := make(map[string]*RestServiceConfig, len(restConsumerConfig.RestServiceConfigsMap))
	for key, rc := range restConsumerConfig.RestServiceConfigsMap {
		rc.Client = getNotEmptyStr(rc.Client, restConsumerConfig.Client, constant.DefaultRestClient)
		rc.RestMethodConfigsMap = initMethodConfigMap(rc, restConsumerConfig.Consumes, restConsumerConfig.Produces)
		restConsumerServiceConfigMap[key] = rc
	}

	SetRestConsumerServiceConfigMap(restConsumerServiceConfigMap)

	return nil
}

// initProviderRestConfig ...
func initMethodConfigMap(rc *RestServiceConfig, consumes string, produces string) map[string]*RestMethodConfig {
	mcm := make(map[string]*RestMethodConfig, len(rc.RestMethodConfigs))
	for _, mc := range rc.RestMethodConfigs {
		mc.InterfaceName = rc.InterfaceName
		mc.Path = rc.Path + mc.Path
		mc.Consumes = getNotEmptyStr(mc.Consumes, rc.Consumes, consumes)
		mc.Produces = getNotEmptyStr(mc.Produces, rc.Produces, produces)
		mc.MethodType = getNotEmptyStr(mc.MethodType, rc.MethodType)
		mc = transformMethodConfig(mc)
		mcm[mc.MethodName] = mc
	}
	return mcm
}

// transformMethodConfig
func transformMethodConfig(methodConfig *RestMethodConfig) *RestMethodConfig {
	if len(methodConfig.PathParamsMap) == 0 && len(methodConfig.PathParams) > 0 {
		paramsMap, err := parseParamsString2Map(methodConfig.PathParams)
		if err != nil {
			logger.Warnf("[Rest ShutdownConfig] Path Param parse error:%v", err)
		} else {
			methodConfig.PathParamsMap = paramsMap
		}
	}
	if len(methodConfig.QueryParamsMap) == 0 && len(methodConfig.QueryParams) > 0 {
		paramsMap, err := parseParamsString2Map(methodConfig.QueryParams)
		if err != nil {
			logger.Warnf("[Rest ShutdownConfig] Argument Param parse error:%v", err)
		} else {
			methodConfig.QueryParamsMap = paramsMap
		}
	}
	if len(methodConfig.HeadersMap) == 0 && len(methodConfig.Headers) > 0 {
		headersMap, err := parseParamsString2Map(methodConfig.Headers)
		if err != nil {
			logger.Warnf("[Rest ShutdownConfig] Argument Param parse error:%v", err)
		} else {
			methodConfig.HeadersMap = headersMap
		}
	}
	return methodConfig
}

// transform a string to a map
// for example:
// string "0:id,1:name" => map [0:id,1:name]
func parseParamsString2Map(params string) (map[int]string, error) {
	m := make(map[int]string, 8)
	for _, p := range strings.Split(params, ",") {
		pa := strings.Split(p, ":")
		key, err := strconv.Atoi(pa[0])
		if err != nil {
			return nil, err
		}
		m[key] = pa[1]
	}
	return m, nil
}

// function will return first not empty string ..
func getNotEmptyStr(args ...string) string {
	var r string
	for _, t := range args {
		if len(t) > 0 {
			r = t
			break
		}
	}
	return r
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
	URL                  string              `yaml:"url"  json:"url,omitempty" property:"url"`
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
	URL            string `yaml:"url"  json:"url,omitempty" property:"url"`
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

func (c *RestProviderConfig) Init(rc *RootConfig) error {

	restProviderConfig := rc.RestProvider
	restProviderServiceConfigMap := make(map[string]*RestServiceConfig, len(restProviderConfig.RestServiceConfigsMap))
	for key, rc := range restProviderConfig.RestServiceConfigsMap {
		rc.Server = getNotEmptyStr(rc.Server, restProviderConfig.Server, constant.DefaultRestServer)
		rc.RestMethodConfigsMap = initMethodConfigMap(rc, restProviderConfig.Consumes, restProviderConfig.Produces)
		restProviderServiceConfigMap[key] = rc
	}
	SetRestProviderServiceConfigMap(restProviderServiceConfigMap)

	return nil
}
