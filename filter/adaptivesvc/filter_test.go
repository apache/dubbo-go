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

package adaptivesvc

import (
	"context"
	"testing"

	"dubbo.apache.org/dubbo-go/v3/filter/adaptivesvc/limiter"
)

import (
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/mock"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

type mockUpdater struct {
	called bool
}

func (m *mockUpdater) DoUpdate() error {
	m.called = true
	return nil
}

func (m *mockUpdater) Report(uint64) {}

func TestAdaptiveServiceProviderFilter_Invoke(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	u, _ := common.NewURL("dubbo://127.0.0.1:20000/com.test.Service")
	methodName := "GetInfo"
	filter := newAdaptiveServiceProviderFilter()

	t.Run("AdaptiveDisabled", func(t *testing.T) {
		invoc := invocation.NewRPCInvocation(methodName, nil, nil)
		invoker := mock.NewMockInvoker(ctrl)
		invoker.EXPECT().Invoke(gomock.Any(), gomock.Any()).Return(&result.RPCResult{Rest: "ok"})

		res := filter.Invoke(context.Background(), invoker, invoc)
		assert.Nil(t, res.Error())
	})

	t.Run("AdaptiveEnabled_AcquireSuccess", func(t *testing.T) {
		invoc := invocation.NewRPCInvocation(methodName, nil, map[string]interface{}{
			constant.AdaptiveServiceEnabledKey: constant.AdaptiveServiceIsEnabled,
		})
		invoker := mock.NewMockInvoker(ctrl)
		invoker.EXPECT().GetURL().Return(u).AnyTimes()
		invoker.EXPECT().Invoke(gomock.Any(), gomock.Any()).Return(&result.RPCResult{Rest: "ok"})

		res := filter.Invoke(context.Background(), invoker, invoc)
		assert.Nil(t, res.Error())

		updater, _ := invoc.GetAttribute(constant.AdaptiveServiceUpdaterKey)
		assert.NotNil(t, updater)
	})
}

func TestAdaptiveServiceProviderFilter_OnResponse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	u, _ := common.NewURL("dubbo://127.0.0.1:20000/com.test.Service")
	methodName := "GetInfo"
	filter := newAdaptiveServiceProviderFilter()

	t.Run("DisabledInResponse", func(t *testing.T) {
		invoc := invocation.NewRPCInvocation(methodName, nil, nil)
		res := &result.RPCResult{Rest: "ok"}
		invoker := mock.NewMockInvoker(ctrl)

		ret := filter.OnResponse(context.Background(), res, invoker, invoc)
		assert.Equal(t, res, ret)
	})

	t.Run("InterruptedErrorShouldSkip", func(t *testing.T) {
		invoc := invocation.NewRPCInvocation(methodName, nil, nil)
		res := &result.RPCResult{Err: wrapErrAdaptiveSvcInterrupted("limit exceeded")}
		invoker := mock.NewMockInvoker(ctrl)

		ret := filter.OnResponse(context.Background(), res, invoker, invoc)
		assert.True(t, isErrAdaptiveSvcInterrupted(ret.Error()))
	})

	t.Run("SuccessWithAttachments", func(t *testing.T) {
		invoc := invocation.NewRPCInvocation(methodName, nil, nil)
		updater := &mockUpdater{}
		invoc.SetAttribute(constant.AdaptiveServiceUpdaterKey, updater)

		res := &result.RPCResult{Rest: "ok"}
		res.AddAttachment(constant.AdaptiveServiceEnabledKey, constant.AdaptiveServiceIsEnabled)

		invoker := mock.NewMockInvoker(ctrl)
		invoker.EXPECT().GetURL().Return(u).AnyTimes()

		_, _ = limiterMapperSingleton.newAndSetMethodLimiter(u, methodName, limiter.HillClimbingLimiter)

		ret := filter.OnResponse(context.Background(), res, invoker, invoc)

		assert.Nil(t, ret.Error())
		assert.NotEmpty(t, ret.Attachment(constant.AdaptiveServiceRemainingKey, ""))
		assert.NotEmpty(t, ret.Attachment(constant.AdaptiveServiceInflightKey, ""))
		assert.True(t, updater.called)
	})
}
