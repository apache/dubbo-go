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

package rest

import (
	"strconv"
	"strings"
)

import (
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/config"
	_ "github.com/apache/dubbo-go/protocol/rest/rest_config_reader"
	"github.com/apache/dubbo-go/protocol/rest/rest_interface"
)

var (
	restConsumerConfig           *rest_interface.RestConsumerConfig
	restProviderConfig           *rest_interface.RestProviderConfig
	restConsumerServiceConfigMap map[string]*rest_interface.RestConfig
	restProviderServiceConfigMap map[string]*rest_interface.RestConfig
)

func init() {
	initConsumerRestConfig()
	initProviderRestConfig()
}

func initConsumerRestConfig() {
	consumerConfigType := config.GetConsumerConfig().RestConfigType
	consumerConfigReader := extension.GetSingletonRestConfigReader(consumerConfigType)
	restConsumerConfig = consumerConfigReader.ReadConsumerConfig()
	if restConsumerConfig == nil {
		return
	}
	restConsumerServiceConfigMap = make(map[string]*rest_interface.RestConfig, len(restConsumerConfig.RestConfigMap))
	for _, rc := range restConsumerConfig.RestConfigMap {
		rc.Client = getNotEmptyStr(rc.Client, restConsumerConfig.Client, constant.DEFAULT_REST_CLIENT)
		rc.RestMethodConfigsMap = initMethodConfigMap(rc, restConsumerConfig.Consumes, restConsumerConfig.Produces)
		restConsumerServiceConfigMap[rc.InterfaceName] = rc
	}
}

func initProviderRestConfig() {
	providerConfigType := config.GetProviderConfig().RestConfigType
	providerConfigReader := extension.GetSingletonRestConfigReader(providerConfigType)
	restProviderConfig = providerConfigReader.ReadProviderConfig()
	if restProviderConfig == nil {
		return
	}
	restProviderServiceConfigMap = make(map[string]*rest_interface.RestConfig, len(restProviderConfig.RestConfigMap))
	for _, rc := range restProviderConfig.RestConfigMap {
		rc.Server = getNotEmptyStr(rc.Server, restProviderConfig.Server, constant.DEFAULT_REST_SERVER)
		rc.RestMethodConfigsMap = initMethodConfigMap(rc, restProviderConfig.Consumes, restProviderConfig.Produces)
		restProviderServiceConfigMap[rc.InterfaceName] = rc
	}
}

func initMethodConfigMap(rc *rest_interface.RestConfig, consumes string, produces string) map[string]*rest_interface.RestMethodConfig {
	mcm := make(map[string]*rest_interface.RestMethodConfig, len(rc.RestMethodConfigs))
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

func transformMethodConfig(methodConfig *rest_interface.RestMethodConfig) *rest_interface.RestMethodConfig {
	if len(methodConfig.PathParamsMap) == 0 && len(methodConfig.PathParams) > 0 {
		paramsMap, err := parseParamsString2Map(methodConfig.PathParams)
		if err != nil {
			logger.Warnf("[Rest Config] Path Param parse error:%v", err)
		} else {
			methodConfig.PathParamsMap = paramsMap
		}
	}
	if len(methodConfig.QueryParamsMap) == 0 && len(methodConfig.QueryParams) > 0 {
		paramsMap, err := parseParamsString2Map(methodConfig.QueryParams)
		if err != nil {
			logger.Warnf("[Rest Config] Argument Param parse error:%v", err)
		} else {
			methodConfig.QueryParamsMap = paramsMap
		}
	}
	if len(methodConfig.HeadersMap) == 0 && len(methodConfig.Headers) > 0 {
		headersMap, err := parseParamsString2Map(methodConfig.Headers)
		if err != nil {
			logger.Warnf("[Rest Config] Argument Param parse error:%v", err)
		} else {
			methodConfig.HeadersMap = headersMap
		}
	}
	return methodConfig
}

func parseParamsString2Map(params string) (map[int]string, error) {
	m := make(map[int]string)
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

func GetRestConsumerServiceConfig(service string) *rest_interface.RestConfig {
	return restConsumerServiceConfigMap[service]
}

func GetRestProviderServiceConfig(service string) *rest_interface.RestConfig {
	return restProviderServiceConfigMap[service]
}

func SetRestConsumerServiceConfigMap(configMap map[string]*rest_interface.RestConfig) {
	restConsumerServiceConfigMap = configMap
}

func SetRestProviderServiceConfigMap(configMap map[string]*rest_interface.RestConfig) {
	restProviderServiceConfigMap = configMap
}
