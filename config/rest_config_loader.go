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
	"log"
	"strconv"
	"strings"
)

import (
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/config/rest"
	_ "github.com/apache/dubbo-go/config/rest/config_reader/reader_impl"
)

var (
	restConsumerServiceConfigMap map[string]*rest.RestServiceConfig
	restProviderServiceConfigMap map[string]*rest.RestServiceConfig
)

func init() {
	if err := initConsumerRestConfig(); err != nil {
		log.Printf("[initConsumerRestConfig] %#v", err)
	}
	if err := initProviderRestConfig(); err != nil {
		log.Printf("[initProviderRestConfig] %#v", err)
	}
}

// initConsumerRestConfig ...
func initConsumerRestConfig() error {
	consumerConfigType := GetConsumerConfig().RestConfigType
	consumerConfigReader := extension.GetSingletonRestConfigReader(consumerConfigType)
	var restConsumerConfig *rest.RestConsumerConfig
	var err error
	if restConsumerConfig, err = consumerConfigReader.ReadConsumerConfig(); err != nil {
		return err
	}
	if restConsumerConfig == nil || len(restConsumerConfig.RestServiceConfigsMap) == 0 {
		return perrors.New("Consumer don't has RestServiceConfigsMap ")
	}
	restConsumerServiceConfigMap = make(map[string]*rest.RestServiceConfig, len(restConsumerConfig.RestServiceConfigsMap))
	for key, rc := range restConsumerConfig.RestServiceConfigsMap {
		rc.Client = getNotEmptyStr(rc.Client, restConsumerConfig.Client, constant.DEFAULT_REST_CLIENT)
		rc.RestMethodConfigsMap = initMethodConfigMap(rc, restConsumerConfig.Consumes, restConsumerConfig.Produces)
		restConsumerServiceConfigMap[strings.TrimPrefix(key, "/")] = rc
	}
	return nil
}

// initProviderRestConfig ...
func initProviderRestConfig() error {
	providerConfigType := GetProviderConfig().RestConfigType
	providerConfigReader := extension.GetSingletonRestConfigReader(providerConfigType)
	var restProviderConfig *rest.RestProviderConfig
	var err error
	if restProviderConfig, err = providerConfigReader.ReadProviderConfig(); err != nil {
		return err
	}
	if restProviderConfig == nil || len(restProviderConfig.RestServiceConfigsMap) == 0 {
		return perrors.New("Provider don't has RestServiceConfigsMap is nil")
	}
	restProviderServiceConfigMap = make(map[string]*rest.RestServiceConfig, len(restProviderConfig.RestServiceConfigsMap))
	for key, rc := range restProviderConfig.RestServiceConfigsMap {
		rc.Server = getNotEmptyStr(rc.Server, restProviderConfig.Server, constant.DEFAULT_REST_SERVER)
		rc.RestMethodConfigsMap = initMethodConfigMap(rc, restProviderConfig.Consumes, restProviderConfig.Produces)
		restProviderServiceConfigMap[strings.TrimPrefix(key, "/")] = rc
	}
	return nil
}

// initProviderRestConfig ...
func initMethodConfigMap(rc *rest.RestServiceConfig, consumes string, produces string) map[string]*rest.RestMethodConfig {
	mcm := make(map[string]*rest.RestMethodConfig, len(rc.RestMethodConfigs))
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

// transformMethodConfig
func transformMethodConfig(methodConfig *rest.RestMethodConfig) *rest.RestMethodConfig {
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

// GetRestConsumerServiceConfig ...
func GetRestConsumerServiceConfig(path string) *rest.RestServiceConfig {
	return restConsumerServiceConfigMap[path]
}

// GetRestProviderServiceConfig ...
func GetRestProviderServiceConfig(path string) *rest.RestServiceConfig {
	return restProviderServiceConfigMap[path]
}

// SetRestConsumerServiceConfigMap ...
func SetRestConsumerServiceConfigMap(configMap map[string]*rest.RestServiceConfig) {
	restConsumerServiceConfigMap = configMap
}

// SetRestProviderServiceConfigMap ...
func SetRestProviderServiceConfigMap(configMap map[string]*rest.RestServiceConfig) {
	restProviderServiceConfigMap = configMap
}
