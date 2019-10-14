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

	"github.com/modern-go/concurrent"

	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/filter"
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
 *   tps.limiter: "method-service" # the name of MethodServiceTpsLimiterImpl. It's the default limiter too.
 *   tps.interval: 5000 # interval, the time unit is ms
 *   tps.rate: 300 # the max value in the interval. <0 means that the service will not be limited.
 *   methods:
 *    - name: "GetUser"
 *      tps.interval: 3000
 *      tps.rate: 20, # in this case, this configuration in service-level will be ignored.
 *    - name: "UpdateUser"
 *      tps.rate: -1, # If the rate<0, the method will be ignored
 */
type MethodServiceTpsLimiterImpl struct {
	tpsState *concurrent.Map
}

func (limiter MethodServiceTpsLimiterImpl) IsAllowable(url common.URL, invocation protocol.Invocation) bool {

	serviceLimitRate, err:= strconv.ParseInt(url.GetParam(constant.TPS_LIMIT_RATE_KEY,
		constant.DEFAULT_TPS_LIMIT_RATE), 0, 0)

	if err != nil {
		panic(fmt.Sprintf("Can not parse the %s for url %s, please check your configuration!",
			constant.TPS_LIMIT_RATE_KEY, url.String()))
	}
	methodLimitRateConfig := invocation.AttachmentsByKey(constant.TPS_LIMIT_RATE_KEY, "")

	// both method-level and service-level don't have the configuration of tps limit
	if serviceLimitRate < 0 && len(methodLimitRateConfig) <= 0 {
		return true
	}

	limitRate := serviceLimitRate
	// the method has tps limit configuration
	if len(methodLimitRateConfig) >0 {
		limitRate, err = strconv.ParseInt(methodLimitRateConfig, 0, 0)
		if err != nil {
			panic(fmt.Sprintf("Can not parse the %s for invocation %s # %s, please check your configuration!",
				constant.TPS_LIMIT_RATE_KEY, url.ServiceKey(), invocation.MethodName()))
		}
	}

	// 1. the serviceLimitRate < 0 and methodRateConfig is empty string
	// 2. the methodLimitRate < 0
	if limitRate < 0{
		return true
	}

	serviceInterval, err := strconv.ParseInt(url.GetParam(constant.TPS_LIMIT_INTERVAL_KEY,
		constant.DEFAULT_TPS_LIMIT_INTERVAL), 0, 0)

	if err != nil || serviceInterval <= 0{
		panic(fmt.Sprintf("The %s must be positive, please check your configuration!",
			constant.TPS_LIMIT_INTERVAL_KEY))
	}
	limitInterval := serviceInterval
	methodIntervalConfig := invocation.AttachmentsByKey(constant.TPS_LIMIT_INTERVAL_KEY, "")
	// there is the interval configuration of method-level
	if len(methodIntervalConfig) > 0 {
		limitInterval, err = strconv.ParseInt(methodIntervalConfig, 0, 0)
		if err != nil || limitInterval <= 0{
			panic(fmt.Sprintf("The %s for invocation %s # %s must be positive, please check your configuration!",
				constant.TPS_LIMIT_INTERVAL_KEY, url.ServiceKey(), invocation.MethodName()))
		}
	}

	limitTarget := url.ServiceKey()

	// method-level tps limit
	if len(methodIntervalConfig) > 0 || len(methodLimitRateConfig) >0  {
		limitTarget = limitTarget + "#" + invocation.MethodName()
	}

	limitStrategyConfig := invocation.AttachmentsByKey(constant.TPS_LIMIT_STRATEGY_KEY,
		url.GetParam(constant.TPS_LIMIT_STRATEGY_KEY, constant.DEFAULT_KEY))
	limitStateCreator := extension.GetTpsLimitStrategyCreator(limitStrategyConfig)
	limitState, _ := limiter.tpsState.LoadOrStore(limitTarget, limitStateCreator(int(limitRate), int(limitInterval)))
	return limitState.(filter.TpsLimitStrategy).IsAllowable()
}

var methodServiceTpsLimiterInstance *MethodServiceTpsLimiterImpl
var methodServiceTpsLimiterOnce sync.Once

func GetMethodServiceTpsLimiter() filter.TpsLimiter {
	methodServiceTpsLimiterOnce.Do(func() {
		methodServiceTpsLimiterInstance = &MethodServiceTpsLimiterImpl{
			tpsState: concurrent.NewMap(),
		}
	})
	return methodServiceTpsLimiterInstance
}
