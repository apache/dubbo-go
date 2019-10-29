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
	filterCommon "github.com/apache/dubbo-go/filter/common"
	filterCommonImpl "github.com/apache/dubbo-go/filter/common/impl"
	"github.com/apache/dubbo-go/filter/impl/tps"
	"github.com/apache/dubbo-go/filter/impl/tps/impl"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/protocol/invocation"
)

func TestTpsLimitFilter_Invoke_With_No_TpsLimiter(t *testing.T) {
	tpsFilter := GetTpsLimitFilter()
	invokeUrl := common.NewURLWithOptions(
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.TPS_LIMITER_KEY, ""))
	attch := make(map[string]string, 0)

	result := tpsFilter.Invoke(protocol.NewBaseInvoker(*invokeUrl),
		invocation.NewRPCInvocation("MethodName", []interface{}{"OK"}, attch))
	assert.Nil(t, result.Error())
	assert.Nil(t, result.Result())

}

func TestGenericFilter_Invoke_With_Default_TpsLimiter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockLimiter := impl.NewMockTpsLimiter(ctrl)
	mockLimiter.EXPECT().IsAllowable(gomock.Any(), gomock.Any()).Return(true).Times(1)
	extension.SetTpsLimiter(constant.DEFAULT_KEY, func() tps.TpsLimiter {
		return mockLimiter
	})

	tpsFilter := GetTpsLimitFilter()
	invokeUrl := common.NewURLWithOptions(
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.TPS_LIMITER_KEY, constant.DEFAULT_KEY))
	attch := make(map[string]string, 0)

	result := tpsFilter.Invoke(protocol.NewBaseInvoker(*invokeUrl),
		invocation.NewRPCInvocation("MethodName", []interface{}{"OK"}, attch))
	assert.Nil(t, result.Error())
	assert.Nil(t, result.Result())
}

func TestGenericFilter_Invoke_With_Default_TpsLimiter_Not_Allow(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockLimiter := impl.NewMockTpsLimiter(ctrl)
	mockLimiter.EXPECT().IsAllowable(gomock.Any(), gomock.Any()).Return(false).Times(1)
	extension.SetTpsLimiter(constant.DEFAULT_KEY, func() tps.TpsLimiter {
		return mockLimiter
	})

	mockResult := &protocol.RPCResult{}
	mockRejectedHandler := filterCommonImpl.NewMockRejectedExecutionHandler(ctrl)
	mockRejectedHandler.EXPECT().RejectedExecution(gomock.Any(), gomock.Any()).Return(mockResult).Times(1)

	extension.SetRejectedExecutionHandler(constant.DEFAULT_KEY, func() filterCommon.RejectedExecutionHandler {
		return mockRejectedHandler
	})

	tpsFilter := GetTpsLimitFilter()
	invokeUrl := common.NewURLWithOptions(
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.TPS_LIMITER_KEY, constant.DEFAULT_KEY))
	attch := make(map[string]string, 0)

	result := tpsFilter.Invoke(protocol.NewBaseInvoker(*invokeUrl),
		invocation.NewRPCInvocation("MethodName", []interface{}{"OK"}, attch))
	assert.Nil(t, result.Error())
	assert.Nil(t, result.Result())
}
