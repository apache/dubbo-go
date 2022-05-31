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
	"strconv"
	"strings"
)

import (
	"github.com/creasty/defaults"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	common "dubbo.apache.org/dubbo-go/v3/protocol/rest/config"
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
	rootConfig            *RootConfig
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

func (c *RestConsumerConfig) Init(rc *RootConfig) error {
	restConsumerConfig := rc.RestConsumer

	if restConsumerConfig == nil {
		logger.Debugf("[rest-protocol] no consumer config")
		return nil
	}

	restConsumerServices := make(map[string]*RestServiceConfig, len(restConsumerConfig.RestServiceConfigsMap))
	for key, rc := range restConsumerConfig.RestServiceConfigsMap {
		rc.Client = common.GetNotEmptyStr(rc.Client, restConsumerConfig.Client, constant.DefaultRestClient)
		rc.RestMethodConfigs = initMethodConfigMap(rc, restConsumerConfig.Consumes, restConsumerConfig.Produces)
		restConsumerServices[key] = rc
	}

	SetRestConsumerServiceConfigMap(restConsumerServices)

	return nil
}

// initProviderRestConfig ...
func initMethodConfigMap(rc *RestServiceConfig, consumes string, produces string) map[string]*RestMethodConfig {
	mcm := make(map[string]*RestMethodConfig, len(rc.RestMethodConfigs))
	for _, mc := range rc.RestMethodConfigs {
		mc.InterfaceName = rc.InterfaceName
		mc.Path = rc.Path + mc.Path
		mc.Consumes = common.GetNotEmptyStr(mc.Consumes, rc.Consumes, consumes)
		mc.Produces = common.GetNotEmptyStr(mc.Produces, rc.Produces, produces)
		mc.MethodType = common.GetNotEmptyStr(mc.MethodType, rc.MethodType)
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
			logger.Warnf("[RestConfig] Path Param parse error:%v", err)
		} else {
			methodConfig.PathParamsMap = paramsMap
		}
	}
	if len(methodConfig.QueryParamsMap) == 0 && len(methodConfig.QueryParams) > 0 {
		paramsMap, err := parseParamsString2Map(methodConfig.QueryParams)
		if err != nil {
			logger.Warnf("[RestConfig] Query Param parse error:%v", err)
		} else {
			methodConfig.QueryParamsMap = paramsMap
		}
	}
	if len(methodConfig.HeadersMap) == 0 && len(methodConfig.Headers) > 0 {
		headersMap, err := parseParamsString2Map(methodConfig.Headers)
		if err != nil {
			logger.Warnf("[RestConfig] Header parse error:%v", err)
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

// nolint
type RestProviderConfig struct {
	Server                string                        `default:"go-restful" yaml:"rest_server" json:"rest_server,omitempty" property:"rest_server"`
	Produces              string                        `default:"*/*" yaml:"rest_produces"  json:"rest_produces,omitempty" property:"rest_produces"`
	Consumes              string                        `default:"*/*" yaml:"rest_consumes"  json:"rest_consumes,omitempty" property:"rest_consumes"`
	RestServiceConfigsMap map[string]*RestServiceConfig `yaml:"services" json:"services,omitempty" property:"services"`
	rootConfig            *RootConfig
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
	InterfaceName     string                       `required:"true"  yaml:"interface"  json:"interface,omitempty" property:"interface"`
	URL               string                       `yaml:"url"  json:"url,omitempty" property:"url"`
	Path              string                       `yaml:"rest_path"  json:"rest_path,omitempty" property:"rest_path"`
	Produces          string                       `yaml:"rest_produces"  json:"rest_produces,omitempty" property:"rest_produces"`
	Consumes          string                       `yaml:"rest_consumes"  json:"rest_consumes,omitempty" property:"rest_consumes"`
	MethodType        string                       `yaml:"rest_method"  json:"rest_method,omitempty" property:"rest_method"`
	Client            string                       `yaml:"rest_client" json:"rest_client,omitempty" property:"rest_client"`
	Server            string                       `yaml:"rest_server" json:"rest_server,omitempty" property:"rest_server"`
	RestMethodConfigs map[string]*RestMethodConfig `yaml:"methods" json:"methods,omitempty" property:"methods"`
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

type RestCommonConfig struct {
	Path        string `yaml:"path"  json:"rest_path,omitempty" property:"rest_path"`
	MethodType  string `yaml:"method"  json:"rest_method,omitempty" property:"rest_method"`
	QueryParams string `yaml:"query_params"  json:"rest_query_params,omitempty" property:"rest_query_params"`
	PathParams  string `yaml:"path_params" json:"rest_path_params,omitempty" property:"rest_query_params"`
}

// nolint
type RestMethodConfig struct {
	InterfaceName    string
	MethodName       string `required:"true" yaml:"name"  json:"name,omitempty" property:"name"`
	URL              string `yaml:"url"  json:"url,omitempty" property:"url"`
	Path             string `yaml:"rest_path"  json:"rest_path,omitempty" property:"rest_path"`
	Produces         string `yaml:"rest_produces"  json:"rest_produces,omitempty" property:"rest_produces"`
	Consumes         string `yaml:"rest_consumes"  json:"rest_consumes,omitempty" property:"rest_consumes"`
	MethodType       string `yaml:"rest_method"  json:"rest_method,omitempty" property:"rest_method"`
	PathParams       string `yaml:"rest_path_params"  json:"rest_path_params,omitempty" property:"rest_path_params"`
	PathParamsMap    map[int]string
	QueryParams      string            `yaml:"rest_query_params"  json:"rest_query_params,omitempty" property:"rest_query_params"`
	RestCommonConfig *RestCommonConfig `yaml:"rest" json:"rest_common,omitempty" property:"rest_common"`
	QueryParamsMap   map[int]string
	Body             int    `default:"-1" yaml:"rest_body"  json:"rest_body,omitempty" property:"rest_body"`
	Headers          string `yaml:"rest_headers"  json:"rest_headers,omitempty" property:"rest_headers"`
	HeadersMap       map[int]string
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

	if restProviderConfig == nil {
		logger.Debugf("[rest-protocol] no provider config")
		return nil
	}

	restProviderServiceConfigMap := make(map[string]*RestServiceConfig, len(restProviderConfig.RestServiceConfigsMap))
	for key, rc := range restProviderConfig.RestServiceConfigsMap {
		rc.Server = common.GetNotEmptyStr(rc.Server, restProviderConfig.Server, constant.DefaultRestServer)
		rc.RestMethodConfigs = initMethodConfigMap(rc, restProviderConfig.Consumes, restProviderConfig.Produces)
		restProviderServiceConfigMap[key] = rc
	}
	SetRestProviderServiceConfigMap(restProviderServiceConfigMap)

	return nil
}

type RestProviderConfigBuilder struct {
	restProviderConfig *RestProviderConfig
}

func NewRestProviderConfigBuilder() *RestProviderConfigBuilder {
	return &RestProviderConfigBuilder{restProviderConfig: newEmptyRestProviderConfig()}
}

func (pcb *RestProviderConfigBuilder) Build() *RestProviderConfig {
	return pcb.restProviderConfig
}

func newEmptyRestProviderConfig() *RestProviderConfig {
	newProviderConfig := &RestProviderConfig{
		Server:   "go-restful",
		Produces: "*/*",
		Consumes: "*/*",
	}
	return newProviderConfig
}

type RestConsumerConfigBuilder struct {
	restConsumerConfig *RestConsumerConfig
}

func NewRestConsumerConfigBuilder() *RestConsumerConfigBuilder {
	return &RestConsumerConfigBuilder{restConsumerConfig: newEmptyRestConsumerConfig()}
}

func newEmptyRestConsumerConfig() *RestConsumerConfig {
	return &RestConsumerConfig{
		Client:   "resty",
		Produces: "application/json",
		Consumes: "application/json",
	}
}

func (ccb *RestConsumerConfigBuilder) Build() *RestConsumerConfig {
	return ccb.restConsumerConfig
}
