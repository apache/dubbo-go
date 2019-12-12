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
	"net/url"
	"testing"
)
import (
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/filter/impl/tps"
	"github.com/apache/dubbo-go/protocol/invocation"
)

func TestMethodServiceTpsLimiterImpl_IsAllowable_Only_Service_Level(t *testing.T) {
	methodName := "hello"
	invoc := invocation.NewRPCInvocation(methodName, []interface{}{"OK"}, make(map[string]string, 0))

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invokeUrl := common.NewURLWithOptions(
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.INTERFACE_KEY, methodName),
		common.WithParamsValue(constant.TPS_LIMIT_RATE_KEY, "20"))

	mockStrategyImpl := NewMockTpsLimitStrategy(ctrl)
	mockStrategyImpl.EXPECT().IsAllowable().Return(true).Times(1)

	extension.SetTpsLimitStrategy(constant.DEFAULT_KEY, &mockStrategyCreator{
		rate:     40,
		interval: 60000,
		t:        t,
		strategy: mockStrategyImpl,
	})

	limiter := GetMethodServiceTpsLimiter()
	result := limiter.IsAllowable(*invokeUrl, invoc)
	assert.True(t, result)
}

func TestMethodServiceTpsLimiterImpl_IsAllowable_No_Config(t *testing.T) {
	methodName := "hello1"
	invoc := invocation.NewRPCInvocation(methodName, []interface{}{"OK"}, make(map[string]string, 0))
	// ctrl := gomock.NewController(t)
	// defer ctrl.Finish()

	invokeUrl := common.NewURLWithOptions(
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.INTERFACE_KEY, methodName),
		common.WithParamsValue(constant.TPS_LIMIT_RATE_KEY, ""))

	limiter := GetMethodServiceTpsLimiter()
	result := limiter.IsAllowable(*invokeUrl, invoc)
	assert.True(t, result)
}

func TestMethodServiceTpsLimiterImpl_IsAllowable_Method_Level_Override(t *testing.T) {
	methodName := "hello2"
	methodConfigPrefix := "methods." + methodName + "."
	invoc := invocation.NewRPCInvocation(methodName, []interface{}{"OK"}, make(map[string]string, 0))
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invokeUrl := common.NewURLWithOptions(
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.INTERFACE_KEY, methodName),
		common.WithParamsValue(constant.TPS_LIMIT_RATE_KEY, "20"),
		common.WithParamsValue(constant.TPS_LIMIT_INTERVAL_KEY, "3000"),
		common.WithParamsValue(constant.TPS_LIMIT_STRATEGY_KEY, "invalid"),
		common.WithParamsValue(methodConfigPrefix+constant.TPS_LIMIT_RATE_KEY, "40"),
		common.WithParamsValue(methodConfigPrefix+constant.TPS_LIMIT_INTERVAL_KEY, "7000"),
		common.WithParamsValue(methodConfigPrefix+constant.TPS_LIMIT_STRATEGY_KEY, "default"),
	)

	mockStrategyImpl := NewMockTpsLimitStrategy(ctrl)
	mockStrategyImpl.EXPECT().IsAllowable().Return(true).Times(1)

	extension.SetTpsLimitStrategy(constant.DEFAULT_KEY, &mockStrategyCreator{
		rate:     40,
		interval: 7000,
		t:        t,
		strategy: mockStrategyImpl,
	})

	limiter := GetMethodServiceTpsLimiter()
	result := limiter.IsAllowable(*invokeUrl, invoc)
	assert.True(t, result)
}

func TestMethodServiceTpsLimiterImpl_IsAllowable_Both_Method_And_Service(t *testing.T) {
	methodName := "hello3"
	methodConfigPrefix := "methods." + methodName + "."
	invoc := invocation.NewRPCInvocation(methodName, []interface{}{"OK"}, make(map[string]string, 0))
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invokeUrl := common.NewURLWithOptions(
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.INTERFACE_KEY, methodName),
		common.WithParamsValue(constant.TPS_LIMIT_RATE_KEY, "20"),
		common.WithParamsValue(constant.TPS_LIMIT_INTERVAL_KEY, "3000"),
		common.WithParamsValue(methodConfigPrefix+constant.TPS_LIMIT_RATE_KEY, "40"),
	)

	mockStrategyImpl := NewMockTpsLimitStrategy(ctrl)
	mockStrategyImpl.EXPECT().IsAllowable().Return(true).Times(1)

	extension.SetTpsLimitStrategy(constant.DEFAULT_KEY, &mockStrategyCreator{
		rate:     40,
		interval: 3000,
		t:        t,
		strategy: mockStrategyImpl,
	})

	limiter := GetMethodServiceTpsLimiter()
	result := limiter.IsAllowable(*invokeUrl, invoc)
	assert.True(t, result)
}

type mockStrategyCreator struct {
	rate     int
	interval int
	t        *testing.T
	strategy tps.TpsLimitStrategy
}

func (creator *mockStrategyCreator) Create(rate int, interval int) tps.TpsLimitStrategy {
	assert.Equal(creator.t, creator.rate, rate)
	assert.Equal(creator.t, creator.interval, interval)
	return creator.strategy
}
