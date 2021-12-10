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

package tps

import (
	"context"
	"net/url"
	"testing"
)

import (
	"github.com/golang/mock/gomock"

	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/filter/handler"
	"dubbo.apache.org/dubbo-go/v3/filter/tps/limiter"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

func TestTpsLimitFilterInvokeWithNoTpsLimiter(t *testing.T) {
	tpsFilter := &tpsLimitFilter{}
	invokeUrl := common.NewURLWithOptions(
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.TPSLimiterKey, ""))
	attch := make(map[string]interface{})

	result := tpsFilter.Invoke(context.Background(),
		protocol.NewBaseInvoker(invokeUrl),
		invocation.NewRPCInvocation("MethodName",
			[]interface{}{"OK"}, attch))
	assert.Nil(t, result.Error())
	assert.Nil(t, result.Result())
}

func TestGenericFilterInvokeWithDefaultTpsLimiter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockLimiter := limiter.NewMockTpsLimiter(ctrl)
	mockLimiter.EXPECT().IsAllowable(gomock.Any(), gomock.Any()).Return(true).Times(1)
	extension.SetTpsLimiter(constant.DefaultKey, func() filter.TpsLimiter {
		return mockLimiter
	})

	tpsFilter := &tpsLimitFilter{}
	invokeUrl := common.NewURLWithOptions(
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.TPSLimiterKey, constant.DefaultKey))
	attch := make(map[string]interface{})

	result := tpsFilter.Invoke(context.Background(),
		protocol.NewBaseInvoker(invokeUrl),
		invocation.NewRPCInvocation("MethodName",
			[]interface{}{"OK"}, attch))
	assert.Nil(t, result.Error())
	assert.Nil(t, result.Result())
}

func TestGenericFilterInvokeWithDefaultTpsLimiterNotAllow(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockLimiter := limiter.NewMockTpsLimiter(ctrl)
	mockLimiter.EXPECT().IsAllowable(gomock.Any(), gomock.Any()).Return(false).Times(1)
	extension.SetTpsLimiter(constant.DefaultKey, func() filter.TpsLimiter {
		return mockLimiter
	})

	mockResult := &protocol.RPCResult{}
	mockRejectedHandler := handler.NewMockRejectedExecutionHandler(ctrl)
	mockRejectedHandler.EXPECT().RejectedExecution(gomock.Any(), gomock.Any()).Return(mockResult).Times(1)

	extension.SetRejectedExecutionHandler(constant.DefaultKey, func() filter.RejectedExecutionHandler {
		return mockRejectedHandler
	})

	tpsFilter := &tpsLimitFilter{}
	invokeUrl := common.NewURLWithOptions(
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.TPSLimiterKey, constant.DefaultKey))
	attch := make(map[string]interface{})

	result := tpsFilter.Invoke(context.Background(),
		protocol.NewBaseInvoker(invokeUrl),
		invocation.NewRPCInvocation("MethodName", []interface{}{"OK"}, attch))
	assert.Nil(t, result.Error())
	assert.Nil(t, result.Result())
}
