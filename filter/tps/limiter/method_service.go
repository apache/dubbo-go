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

package limiter

import (
	"fmt"
	"strconv"
	"sync"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"github.com/modern-go/concurrent"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

const (
	name = "method-service"
)

func init() {
	extension.SetTpsLimiter(constant.DefaultKey, GetMethodServiceTpsLimiter)
	extension.SetTpsLimiter(name, GetMethodServiceTpsLimiter)
}

// MethodServiceTpsLimiter allows developer to config both method-level and service-level tps limiter.
/**
 * for example:
 * "UserProvider":
 *   registry: "hangzhouzk"
 *   protocol : "dubbo"
 *   interface : "com.ikurento.user.UserProvider"
 *   ... # other configuration
 *   tps.limiter: "method-service" or "default" # the name of MethodServiceTpsLimiter. It's the default limiter too.
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
 *   tps.limiter: "method-service" or "default" # the name of MethodServiceTpsLimiter. It's the default limiter too.
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
 *   tps.limiter: "method-service" or "default" # the name of MethodServiceTpsLimiter. It's the default limiter too.
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
 *   tps.limiter: "method-service" or "default" # the name of MethodServiceTpsLimiter. It's the default limiter too.
 *   methods:
 *    - name: "GetUser"
 *    - name: "UpdateUser"
 *      tps.limit.rate: 70,
 *      tps.limit.interval: 40000
 * In this case, only UpdateUser will be limited by its configuration (70 times in 40000ms)
 */
type MethodServiceTpsLimiter struct {
	tpsState *concurrent.Map
}

// IsAllowable based on method-level and service-level.
// The method-level has high priority which means that if there is any rate limit configuration for the method,
// the service-level rate limit strategy will be ignored.
// The key point is how to keep thread-safe
// This implementation use concurrent map + loadOrStore to make implementation thread-safe
// You can image that even multiple threads create limiter, but only one could store the limiter into tpsState
func (limiter MethodServiceTpsLimiter) IsAllowable(url *common.URL, invocation protocol.Invocation) bool {
	methodConfigPrefix := "methods." + invocation.MethodName() + "."

	methodLimitRateConfig := url.GetParam(methodConfigPrefix+constant.TPSLimitRateKey, "")
	methodIntervalConfig := url.GetParam(methodConfigPrefix+constant.TPSLimitIntervalKey, "")

	// service-level tps limit
	limitTarget := url.ServiceKey()

	// method-level tps limit
	if len(methodIntervalConfig) > 0 || len(methodLimitRateConfig) > 0 {
		// it means that if the method-level rate limit exist, we will use method-level rate limit strategy
		limitTarget = limitTarget + "#" + invocation.MethodName()
	}

	// looking up the limiter from 'cache'
	limitState, found := limiter.tpsState.Load(limitTarget)
	if found {
		// the limiter has been cached, we return its result
		return limitState.(filter.TpsLimitStrategy).IsAllowable()
	}

	// we could not find the limiter, and try to create one.

	limitRate := getLimitConfig(methodLimitRateConfig, url, invocation,
		constant.TPSLimitRateKey,
		constant.DefaultTPSLimitRate)

	if limitRate < 0 {
		// the limitTarget is not necessary to be limited.
		logger.Errorf("Found error configuration value of tps.limit.rate for the invocation %s, ignores TPS Limiter", url.ServiceKey()+"#"+invocation.MethodName())
		return true
	}

	limitInterval := getLimitConfig(methodIntervalConfig, url, invocation,
		constant.TPSLimitIntervalKey,
		constant.DefaultTPSLimitInterval)
	if limitInterval <= 0 {
		logger.Errorf(fmt.Sprintf("Found error configuration value of tps.limit.interval for the invocation %s, ignores TPS Limiter", url.ServiceKey()+"#"+invocation.MethodName()))
		return true
	}

	// find the strategy config and then create one
	limitStrategyConfig := url.GetParam(methodConfigPrefix+constant.TPSLimitStrategyKey,
		url.GetParam(constant.TPSLimitStrategyKey, constant.DefaultKey))
	limitStateCreator, err := extension.GetTpsLimitStrategyCreator(limitStrategyConfig)
	if err != nil {
		logger.Warn(err)
		return true
	}

	// we using loadOrStore to ensure thread-safe
	limitState, _ = limiter.tpsState.LoadOrStore(limitTarget, limitStateCreator.Create(int(limitRate), int(limitInterval)))

	return limitState.(filter.TpsLimitStrategy).IsAllowable()
}

// getLimitConfig will try to fetch the configuration from url.
// If we can convert the methodLevelConfig to int64, return;
// Or, we will try to look up server-level configuration and then convert it to int64
func getLimitConfig(methodLevelConfig string,
	url *common.URL,
	invocation protocol.Invocation,
	configKey string,
	defaultVal int64) int64 {

	if len(methodLevelConfig) > 0 {
		result, err := strconv.ParseInt(methodLevelConfig, 0, 0)
		if err != nil {
			logger.Error(fmt.Sprintf("The %s for invocation %s # %s must be positive, please check your configuration!",
				configKey, url.ServiceKey(), invocation.MethodName()))
			return defaultVal
		}
		return result
	}

	// actually there is no method-level configuration, so we use the service-level configuration

	result, err := strconv.ParseInt(url.GetParam(configKey, ""), 0, 0)
	if err != nil {
		logger.Errorf(fmt.Sprintf("Cannot parse the configuration %s, please check your configuration!", configKey))
		return defaultVal
	}
	return result
}

var (
	methodServiceTpsLimiterInstance *MethodServiceTpsLimiter
	methodServiceTpsLimiterOnce     sync.Once
)

// GetMethodServiceTpsLimiter will return an MethodServiceTpsLimiter instance.
func GetMethodServiceTpsLimiter() filter.TpsLimiter {
	methodServiceTpsLimiterOnce.Do(func() {
		methodServiceTpsLimiterInstance = &MethodServiceTpsLimiter{
			tpsState: concurrent.NewMap(),
		}
	})
	return methodServiceTpsLimiterInstance
}
