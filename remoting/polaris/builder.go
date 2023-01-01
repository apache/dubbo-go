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

package polaris

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
)

import (
	perrors "github.com/pkg/errors"

	"github.com/polarismesh/polaris-go"
	"github.com/polarismesh/polaris-go/api"
	"github.com/polarismesh/polaris-go/pkg/config"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

var (
	once               sync.Once
	namesapce          string
	sdkCtx             api.SDKContext
	openPolarisAbility bool
)

var (
	ErrorNoOpenPolarisAbility = errors.New("polaris ability not open")
	ErrorSDKContextNotInit    = errors.New("polaris SDKContext not init")
)

// GetConsumerAPI creates one polaris ConsumerAPI instance
func GetConsumerAPI() (polaris.ConsumerAPI, error) {
	if err := Check(); err != nil {
		return nil, err
	}

	return polaris.NewConsumerAPIByContext(sdkCtx), nil
}

// GetProviderAPI creates one polaris ProviderAPI instance
func GetProviderAPI() (polaris.ProviderAPI, error) {
	if err := Check(); err != nil {
		return nil, err
	}

	return polaris.NewProviderAPIByContext(sdkCtx), nil
}

// GetRouterAPI create one polaris RouterAPI instance
func GetRouterAPI() (polaris.RouterAPI, error) {
	if err := Check(); err != nil {
		return nil, err
	}

	return polaris.NewRouterAPIByContext(sdkCtx), nil
}

// GetLimiterAPI creates one polaris LimiterAPI instance
func GetLimiterAPI() (polaris.LimitAPI, error) {
	if err := Check(); err != nil {
		return nil, err
	}

	return polaris.NewLimitAPIByContext(sdkCtx), nil
}

func Check() error {
	if !openPolarisAbility {
		return ErrorNoOpenPolarisAbility
	}
	if sdkCtx == nil {
		return ErrorSDKContextNotInit
	}
	return nil
}

// GetNamespace gets user defined namespace info
func GetNamespace() string {
	return namesapce
}

// InitSDKContext inits polaris SDKContext by URL
func InitSDKContext(url *common.URL) error {
	if url == nil {
		return errors.New("url is empty!")
	}

	openPolarisAbility = true

	var rerr error
	once.Do(func() {
		addresses := strings.Split(url.Location, ",")
		serverConfigs := make([]string, 0, len(addresses))
		for _, addr := range addresses {
			ip, portStr, err := net.SplitHostPort(addr)
			if err != nil {
				rerr = perrors.WithMessagef(err, "split [%s] ", addr)
			}
			port, _ := strconv.Atoi(portStr)
			serverConfigs = append(serverConfigs, fmt.Sprintf("%s:%d", ip, uint64(port)))
		}

		polarisConf := config.NewDefaultConfiguration(serverConfigs)
		_sdkCtx, err := api.InitContextByConfig(polarisConf)
		rerr = err
		sdkCtx = _sdkCtx
		namesapce = url.GetParam(constant.RegistryNamespaceKey, constant.PolarisDefaultNamespace)
	})

	return rerr
}

func mergePolarisConfiguration(easy, complexConf config.Configuration) {

	easySvrList := easy.GetGlobal().GetServerConnector().GetAddresses()

	complexSvrList := complexConf.GetGlobal().GetServerConnector().GetAddresses()

	result := make(map[string]bool)

	for i := range complexSvrList {
		result[complexSvrList[i]] = true
	}

	for i := range easySvrList {
		if _, exist := result[easySvrList[i]]; !exist {
			result[easySvrList[i]] = true
		}
	}

	finalSvrList := make([]string, 0)
	for k := range result {
		finalSvrList = append(finalSvrList, k)
	}

	complexConf.GetGlobal().GetServerConnector().SetAddresses(finalSvrList)
}
