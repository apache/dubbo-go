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

package nacos

import (
	"net"
	"strconv"
	"strings"
	"time"
)

import (
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/clients/naming_client"
	nacosConstant "github.com/nacos-group/nacos-sdk-go/common/constant"
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/config"
)

// NewNacosConfigClient read the config from url and build an instance
func NewNacosConfigClient(url *common.URL) (config_client.IConfigClient, error) {
	nacosConfig, err := getNacosConfig(url)
	if err != nil {
		return nil, err
	}
	return clients.CreateConfigClient(nacosConfig)
}

// getNacosConfig will return the nacos config
func getNacosConfig(url *common.URL) (map[string]interface{}, error) {
	if url == nil {
		return nil, perrors.New("url is empty!")
	}
	if len(url.Location) == 0 {
		return nil, perrors.New("url.location is empty!")
	}
	configMap := make(map[string]interface{}, 2)

	addresses := strings.Split(url.Location, ",")
	serverConfigs := make([]nacosConstant.ServerConfig, 0, len(addresses))
	for _, addr := range addresses {
		ip, portStr, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, perrors.WithMessagef(err, "split [%s] ", addr)
		}
		port, _ := strconv.Atoi(portStr)
		serverConfigs = append(serverConfigs, nacosConstant.ServerConfig{
			IpAddr: ip,
			Port:   uint64(port),
		})
	}
	configMap["serverConfigs"] = serverConfigs

	var clientConfig nacosConstant.ClientConfig
	timeout, err := time.ParseDuration(url.GetParam(constant.REGISTRY_TIMEOUT_KEY, constant.DEFAULT_REG_TIMEOUT))
	if err != nil {
		return nil, err
	}
	clientConfig.TimeoutMs = uint64(timeout.Seconds() * 1000)
	clientConfig.ListenInterval = 2 * clientConfig.TimeoutMs
	clientConfig.CacheDir = url.GetParam(constant.NACOS_CACHE_DIR_KEY, "")
	clientConfig.LogDir = url.GetParam(constant.NACOS_LOG_DIR_KEY, "")
	clientConfig.Endpoint = url.GetParam(constant.NACOS_ENDPOINT, "")
	clientConfig.NamespaceId = url.GetParam(constant.NACOS_NAMESPACE_ID, "")
	clientConfig.Username = url.GetParam(constant.NACOS_USERNAME, "")
	clientConfig.Password = url.GetParam(constant.NACOS_PASSWORD, "")
	clientConfig.NamespaceId = url.GetParam(constant.NACOS_NAMESPACE_ID, "")
	clientConfig.NotLoadCacheAtStart = true
	configMap["clientConfig"] = clientConfig

	return configMap, nil
}

// NewNacosClient creates an instance with the config
func NewNacosClient(rc *config.RemoteConfig) (naming_client.INamingClient, error) {
	if len(rc.Address) == 0 {
		return nil, perrors.New("nacos address is empty!")
	}
	configMap := make(map[string]interface{}, 2)

	addresses := strings.Split(rc.Address, ",")
	serverConfigs := make([]nacosConstant.ServerConfig, 0, len(addresses))
	for _, addr := range addresses {
		ip, portStr, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, perrors.WithMessagef(err, "split [%s] ", addr)
		}
		port, _ := strconv.Atoi(portStr)
		serverConfigs = append(serverConfigs, nacosConstant.ServerConfig{
			IpAddr: ip,
			Port:   uint64(port),
		})
	}
	configMap["serverConfigs"] = serverConfigs

	var clientConfig nacosConstant.ClientConfig
	timeout := rc.Timeout()
	clientConfig.TimeoutMs = uint64(timeout.Nanoseconds() / constant.MsToNanoRate)
	clientConfig.ListenInterval = 2 * clientConfig.TimeoutMs
	clientConfig.CacheDir = rc.GetParam(constant.NACOS_CACHE_DIR_KEY, "")
	clientConfig.LogDir = rc.GetParam(constant.NACOS_LOG_DIR_KEY, "")
	clientConfig.Endpoint = rc.Address
	clientConfig.Username = rc.Username
	clientConfig.Password = rc.Password
	clientConfig.NotLoadCacheAtStart = true
	clientConfig.NamespaceId = rc.GetParam(constant.NACOS_NAMESPACE_ID, "")
	configMap["clientConfig"] = clientConfig

	return clients.CreateNamingClient(configMap)
}
