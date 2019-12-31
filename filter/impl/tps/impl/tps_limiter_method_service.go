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
package impl

import (
	"fmt"
	"strconv"
	"sync"
)

import (
	"github.com/modern-go/concurrent"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/filter/impl/tps"
	"github.com/apache/dubbo-go/protocol"
)

const name = "method-service"

func init() {
	extension.SetTpsLimiter(constant.DEFAULT_KEY, GetMethodServiceTpsLimiter)
	extension.SetTpsLimiter(name, GetMethodServiceTpsLimiter)
}

/**
 * This implementation allows developer to config both method-level and service-level tps limiter.
 * for example:
 * "UserProvider":
 *   registry: "hangzhouzk"
 *   protocol : "dubbo"
 *   interface : "com.ikurento.user.UserProvider"
 *   ... # other configuration
 *   tps.limiter: "method-service" or "default" # the name of MethodServiceTpsLimiterImpl. It's the default limiter too.
 *   tps.limit.interval: 5000 # interval, the time unit is ms
 *   tps.limit.rate: 300 # the max value in the interval. <0 means that the service will not be limited.
 *   methods:
 *    - name: "GetUser"
 *      tps.interval: 3000
 *      tps.limit.rate: 20, # in this case, this configuration in service-level will be ignored.
 *    - name: "UpdateUser"
 *      tps.limit.rate: -1, # If the rate<0, the method will be ignored
 *
 *
 * More examples:
 * case1:
 * "UserProvider":
 *   registry: "hangzhouzk"
 *   protocol : "dubbo"
 *   interface : "com.ikurento.user.UserProvider"
 *   ... # other configuration
 *   tps.limiter: "method-service" or "default" # the name of MethodServiceTpsLimiterImpl. It's the default limiter too.
 *   tps.limit.interval: 5000 # interval, the time unit is ms
 *   tps.limit.rate: 300 # the max value in the interval. <0 means that the service will not be limited.
 *   methods:
 *    - name: "GetUser"
 *    - name: "UpdateUser"
 *      tps.limit.rate: -1,
 * in this case, the method UpdateUser will be ignored,
 * which means that only GetUser will be limited by service-level configuration.
 *
 * case2:
 * "UserProvider":
 *   registry: "hangzhouzk"
 *   protocol : "dubbo"
 *   interface : "com.ikurento.user.UserProvider"
 *   ... # other configuration
 *   tps.limiter: "method-service" or "default" # the name of MethodServiceTpsLimiterImpl. It's the default limiter too.
 *   tps.limit.interval: 5000 # interval, the time unit is ms
 *   tps.limit.rate: 300 # the max value in the interval. <0 means that the service will not be limited.
 *   methods:
 *    - name: "GetUser"
 *    - name: "UpdateUser"
 *      tps.limit.rate: 30,
 * In this case, the GetUser will be limited by service-level configuration(300 times in 5000ms),
 * but UpdateUser will be limited by its configuration (30 times in 60000ms)
 *
 * case3:
 * "UserProvider":
 *   registry: "hangzhouzk"
 *   protocol : "dubbo"
 *   interface : "com.ikurento.user.UserProvider"
 *   ... # other configuration
 *   tps.limiter: "method-service" or "default" # the name of MethodServiceTpsLimiterImpl. It's the default limiter too.
 *   methods:
 *    - name: "GetUser"
 *    - name: "UpdateUser"
 *      tps.limit.rate: 70,
 *      tps.limit.interval: 40000
 * In this case, only UpdateUser will be limited by its configuration (70 times in 40000ms)
 */
type MethodServiceTpsLimiterImpl struct {
	tpsState *concurrent.Map
}

func (limiter MethodServiceTpsLimiterImpl) IsAllowable(url common.URL, invocation protocol.Invocation) bool {

	methodConfigPrefix := "methods." + invocation.MethodName() + "."

	methodLimitRateConfig := url.GetParam(methodConfigPrefix+constant.TPS_LIMIT_RATE_KEY, "")
	methodIntervalConfig := url.GetParam(methodConfigPrefix+constant.TPS_LIMIT_INTERVAL_KEY, "")

	limitTarget := url.ServiceKey()

	// method-level tps limit
	if len(methodIntervalConfig) > 0 || len(methodLimitRateConfig) > 0 {
		limitTarget = limitTarget + "#" + invocation.MethodName()
	}

	limitState, found := limiter.tpsState.Load(limitTarget)
	if found {
		return limitState.(tps.TpsLimitStrategy).IsAllowable()
	}

	limitRate := getLimitConfig(methodLimitRateConfig, url, invocation,
		constant.TPS_LIMIT_RATE_KEY,
		constant.DEFAULT_TPS_LIMIT_RATE)

	if limitRate < 0 {
		return true
	}

	limitInterval := getLimitConfig(methodIntervalConfig, url, invocation,
		constant.TPS_LIMIT_INTERVAL_KEY,
		constant.DEFAULT_TPS_LIMIT_INTERVAL)
	if limitInterval <= 0 {
		panic(fmt.Sprintf("The interval must be positive, please check your configuration! url: %s", url.String()))
	}

	limitStrategyConfig := url.GetParam(methodConfigPrefix+constant.TPS_LIMIT_STRATEGY_KEY,
		url.GetParam(constant.TPS_LIMIT_STRATEGY_KEY, constant.DEFAULT_KEY))
	limitStateCreator := extension.GetTpsLimitStrategyCreator(limitStrategyConfig)
	limitState, _ = limiter.tpsState.LoadOrStore(limitTarget, limitStateCreator.Create(int(limitRate), int(limitInterval)))
	return limitState.(tps.TpsLimitStrategy).IsAllowable()
}

func getLimitConfig(methodLevelConfig string,
	url common.URL,
	invocation protocol.Invocation,
	configKey string,
	defaultVal string) int64 {

	if len(methodLevelConfig) > 0 {
		result, err := strconv.ParseInt(methodLevelConfig, 0, 0)
		if err != nil {
			panic(fmt.Sprintf("The %s for invocation %s # %s must be positive, please check your configuration!",
				configKey, url.ServiceKey(), invocation.MethodName()))
		}
		return result
	}

	result, err := strconv.ParseInt(url.GetParam(configKey, defaultVal), 0, 0)

	if err != nil {
		panic(fmt.Sprintf("Cannot parse the configuration %s, please check your configuration!", configKey))
	}
	return result
}

var methodServiceTpsLimiterInstance *MethodServiceTpsLimiterImpl
var methodServiceTpsLimiterOnce sync.Once

func GetMethodServiceTpsLimiter() tps.TpsLimiter {
	methodServiceTpsLimiterOnce.Do(func() {
		methodServiceTpsLimiterInstance = &MethodServiceTpsLimiterImpl{
			tpsState: concurrent.NewMap(),
		}
	})
	return methodServiceTpsLimiterInstance
}
