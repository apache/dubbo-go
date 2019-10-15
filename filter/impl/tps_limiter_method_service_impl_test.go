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
	"github.com/coreos/etcd/pkg/testutil"
	"github.com/golang/mock/gomock"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/filter"
	"github.com/apache/dubbo-go/protocol/invocation"
)

const methodName = "hello"

var methodConfigPrefix = "methods." + methodName + "."
var invoc = invocation.NewRPCInvocation(methodName, []interface{}{"OK"}, make(map[string]string, 0))

func TestMethodServiceTpsLimiterImpl_IsAllowable_Only_Service_Level(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invokeUrl := common.NewURLWithOptions(
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.TPS_LIMIT_RATE_KEY, "20"))

	mockStrategyImpl := filter.NewMockTpsLimitStrategy(ctrl)
	mockStrategyImpl.EXPECT().IsAllowable().Return(true).Times(1)
	extension.SetTpsLimitStrategy(constant.DEFAULT_KEY, func(rate int, interval int) filter.TpsLimitStrategy {
		testutil.AssertEqual(t, 20, rate)
		testutil.AssertEqual(t, 60000, interval)
		return mockStrategyImpl
	})

	limiter := GetMethodServiceTpsLimiter()
	result := limiter.IsAllowable(*invokeUrl, invoc)
	testutil.AssertTrue(t, result)
}

func TestMethodServiceTpsLimiterImpl_IsAllowable_No_Config(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invokeUrl := common.NewURLWithOptions(
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.TPS_LIMIT_RATE_KEY, ""))

	limiter := GetMethodServiceTpsLimiter()
	result := limiter.IsAllowable(*invokeUrl, invoc)
	testutil.AssertTrue(t, result)
}

func TestMethodServiceTpsLimiterImpl_IsAllowable_Method_Level_Override(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invokeUrl := common.NewURLWithOptions(
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.TPS_LIMIT_RATE_KEY, "20"),
		common.WithParamsValue(constant.TPS_LIMIT_INTERVAL_KEY, "3000"),
		common.WithParamsValue(constant.TPS_LIMIT_STRATEGY_KEY, "invalid"),
		common.WithParamsValue(methodConfigPrefix+constant.TPS_LIMIT_RATE_KEY, "40"),
		common.WithParamsValue(methodConfigPrefix+constant.TPS_LIMIT_INTERVAL_KEY, "7000"),
		common.WithParamsValue(methodConfigPrefix+constant.TPS_LIMIT_STRATEGY_KEY, "default"),
	)

	mockStrategyImpl := filter.NewMockTpsLimitStrategy(ctrl)
	mockStrategyImpl.EXPECT().IsAllowable().Return(true).Times(1)
	extension.SetTpsLimitStrategy(constant.DEFAULT_KEY, func(rate int, interval int) filter.TpsLimitStrategy {
		testutil.AssertEqual(t, 40, rate)
		testutil.AssertEqual(t, 7000, interval)
		return mockStrategyImpl
	})

	limiter := GetMethodServiceTpsLimiter()
	result := limiter.IsAllowable(*invokeUrl, invoc)
	testutil.AssertTrue(t, result)
}

func TestMethodServiceTpsLimiterImpl_IsAllowable_Both_Method_And_Service(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invokeUrl := common.NewURLWithOptions(
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.TPS_LIMIT_RATE_KEY, "20"),
		common.WithParamsValue(constant.TPS_LIMIT_INTERVAL_KEY, "3000"),
		common.WithParamsValue(methodConfigPrefix+constant.TPS_LIMIT_RATE_KEY, "40"),
	)

	mockStrategyImpl := filter.NewMockTpsLimitStrategy(ctrl)
	mockStrategyImpl.EXPECT().IsAllowable().Return(true).Times(1)
	extension.SetTpsLimitStrategy(constant.DEFAULT_KEY, func(rate int, interval int) filter.TpsLimitStrategy {
		testutil.AssertEqual(t, 40, rate)
		testutil.AssertEqual(t, 3000, interval)
		return mockStrategyImpl
	})

	limiter := GetMethodServiceTpsLimiter()
	result := limiter.IsAllowable(*invokeUrl, invoc)
	testutil.AssertTrue(t, result)
}
