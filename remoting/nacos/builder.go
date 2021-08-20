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

	timeout := url.GetParamDuration(constant.CONFIG_TIMEOUT_KEY, constant.DEFAULT_REG_TIMEOUT)

	clientConfig := nacosConstant.ClientConfig{
		TimeoutMs:           uint64(int32(timeout / time.Millisecond)),
		BeatInterval:        url.GetParamInt(constant.NACOS_BEAT_INTERVAL_KEY, 5000),
		NamespaceId:         url.GetParam(constant.NACOS_NAMESPACE_ID, ""),
		AppName:             url.GetParam(constant.NACOS_APP_NAME_KEY, ""),
		Endpoint:            url.GetParam(constant.NACOS_ENDPOINT, ""),
		RegionId:            url.GetParam(constant.NACOS_REGION_ID_KEY, ""),
		AccessKey:           url.GetParam(constant.NACOS_ACCESS_KEY, ""),
		SecretKey:           url.GetParam(constant.NACOS_SECRET_KEY, ""),
		OpenKMS:             url.GetParamBool(constant.NACOS_OPEN_KMS_KEY, false),
		CacheDir:            url.GetParam(constant.NACOS_CACHE_DIR_KEY, ""),
		UpdateThreadNum:     url.GetParamByIntValue(constant.NACOS_UPDATE_THREAD_NUM_KEY, 20),
		NotLoadCacheAtStart: url.GetParamBool(constant.NACOS_NOT_LOAD_LOCAL_CACHE, true),
		Username:            url.GetParam(constant.NACOS_USERNAME, ""),
		Password:            url.GetParam(constant.NACOS_PASSWORD, ""),
		LogDir:              url.GetParam(constant.NACOS_LOG_DIR_KEY, ""),
		LogLevel:            url.GetParam(constant.NACOS_LOG_LEVEL_KEY, "info"),
	}
	return serverConfigs, clientConfig, nil
}

// NewNacosClient create an instance with the config
func NewNacosClient(rc *config.RemoteConfig) (*nacosClient.NacosNamingClient, error) {
	url, err := rc.ToURL()
	if err != nil {
		return nil, err
	}
	scs, cc, err := GetNacosConfig(url)

	if err != nil {
		return nil, err
	}
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
