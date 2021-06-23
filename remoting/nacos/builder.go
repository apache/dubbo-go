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
	nacosClient "github.com/dubbogo/gost/database/kv/nacos"
	nacosConstant "github.com/nacos-group/nacos-sdk-go/common/constant"
	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config"
)

// NewNacosConfigClientByUrl read the config from url and build an instance
func NewNacosConfigClientByUrl(url *common.URL) (*nacosClient.NacosConfigClient, error) {
	sc, cc, err := GetNacosConfig(url)
	if err != nil {
		return nil, err
	}
	return nacosClient.NewNacosConfigClient(getNacosClientName(), true, sc, cc)
}

// GetNacosConfig will return the nacos config
func GetNacosConfig(url *common.URL) ([]nacosConstant.ServerConfig, nacosConstant.ClientConfig, error) {
	if url == nil {
		return []nacosConstant.ServerConfig{}, nacosConstant.ClientConfig{}, perrors.New("url is empty!")
	}

	if len(url.Location) == 0 {
		return []nacosConstant.ServerConfig{}, nacosConstant.ClientConfig{},
			perrors.New("url.location is empty!")
	}

	addresses := strings.Split(url.Location, ",")
	serverConfigs := make([]nacosConstant.ServerConfig, 0, len(addresses))
	for _, addr := range addresses {
		ip, portStr, err := net.SplitHostPort(addr)
		if err != nil {
			return []nacosConstant.ServerConfig{}, nacosConstant.ClientConfig{},
				perrors.WithMessagef(err, "split [%s] ", addr)
		}
		port, _ := strconv.Atoi(portStr)
		serverConfigs = append(serverConfigs, nacosConstant.ServerConfig{IpAddr: ip, Port: uint64(port)})
	}

	var clientConfig nacosConstant.ClientConfig
	timeout, err := time.ParseDuration(url.GetParam(constant.REGISTRY_TIMEOUT_KEY, constant.DEFAULT_REG_TIMEOUT))
	if err != nil {
		return []nacosConstant.ServerConfig{}, nacosConstant.ClientConfig{}, err
	}
	//enable local cache when nacos can not connect.
	notLoadCache, err := strconv.ParseBool(url.GetParam(constant.NACOS_NOT_LOAD_LOCAL_CACHE, "true"))
	if err != nil {
		notLoadCache = false
	}
	clientConfig.TimeoutMs = uint64(timeout.Seconds() * 1000)
	// clientConfig.ListenInterval = 2 * clientConfig.TimeoutMs
	clientConfig.CacheDir = url.GetParam(constant.NACOS_CACHE_DIR_KEY, "")
	clientConfig.LogDir = url.GetParam(constant.NACOS_LOG_DIR_KEY, "")
	clientConfig.Endpoint = url.GetParam(constant.NACOS_ENDPOINT, "")
	clientConfig.NamespaceId = url.GetParam(constant.NACOS_NAMESPACE_ID, "")
	clientConfig.Username = url.GetParam(constant.NACOS_USERNAME, "")
	clientConfig.Password = url.GetParam(constant.NACOS_PASSWORD, "")
	clientConfig.NotLoadCacheAtStart = notLoadCache

	return serverConfigs, clientConfig, nil
}

// NewNacosClient create an instance with the config
func NewNacosClient(rc *config.RemoteConfig) (*nacosClient.NacosNamingClient, error) {
	if len(rc.Address) == 0 {
		return nil, perrors.New("nacos address is empty!")
	}
	addresses := strings.Split(rc.Address, ",")
	scs := make([]nacosConstant.ServerConfig, 0, len(addresses))
	for _, addr := range addresses {
		ip, portStr, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, perrors.WithMessagef(err, "split [%s] ", addr)
		}
		port, _ := strconv.Atoi(portStr)
		scs = append(scs, nacosConstant.ServerConfig{
			IpAddr: ip,
			Port:   uint64(port),
		})
	}

	var cc nacosConstant.ClientConfig
	timeout := rc.Timeout()
	//enable local cache when nacos can not connect.
	notLoadCache, err := strconv.ParseBool(rc.GetParam(constant.NACOS_NOT_LOAD_LOCAL_CACHE, "true"))
	if err != nil {
		notLoadCache = false
	}
	cc.TimeoutMs = uint64(timeout.Nanoseconds() / constant.MsToNanoRate)
	// cc.ListenInterval = 2 * cc.TimeoutMs
	cc.CacheDir = rc.GetParam(constant.NACOS_CACHE_DIR_KEY, "")
	cc.LogDir = rc.GetParam(constant.NACOS_LOG_DIR_KEY, "")
	cc.Endpoint = rc.GetParam(constant.NACOS_ENDPOINT, "")
	cc.NamespaceId = rc.GetParam(constant.NACOS_NAMESPACE_ID, "")
	cc.Username = rc.Username
	cc.Password = rc.Password
	cc.NotLoadCacheAtStart = notLoadCache

	return nacosClient.NewNacosNamingClient(getNacosClientName(), true, scs, cc)
}

// NewNacosClientByUrl created
func NewNacosClientByUrl(url *common.URL) (*nacosClient.NacosNamingClient, error) {
	scs, cc, err := GetNacosConfig(url)
	if err != nil {
		return nil, err
	}
	return nacosClient.NewNacosNamingClient(getNacosClientName(), true, scs, cc)
}

// getNacosClientName get nacos client name
func getNacosClientName() string {
	name := config.GetApplicationConfig().Name
	if len(name) > 0 {
		return name
	}
	return "nacos-client"
}
