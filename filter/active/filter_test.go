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

package active

import (
	"context"
	"errors"
	"strconv"
	"testing"
)

import (
	"github.com/golang/mock/gomock"

	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/mock"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

const ProviderUrl = "dubbo://192.168.10.10:20000/com.ikurento.user.UserProvider"

func TestFilterInvoke(t *testing.T) {
	invoc := invocation.NewRPCInvocation("test", []any{"OK"}, make(map[string]any))
	url, _ := common.NewURL(ProviderUrl)
	filter := activeFilter{}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	invoker := mock.NewMockInvoker(ctrl)
	invoker.EXPECT().Invoke(gomock.Any(), gomock.Any()).Return(nil)
	invoker.EXPECT().GetURL().Return(url).Times(1)
	filter.Invoke(context.Background(), invoker, invoc)
	assert.True(t, invoc.GetAttachmentWithDefaultValue(dubboInvokeStartTime, "") != "")
}

func TestFilterOnResponse(t *testing.T) {
	c := base.CurrentTimeMillis()
	elapsed := 100
	invoc := invocation.NewRPCInvocation("test", []any{"OK"}, map[string]any{
		dubboInvokeStartTime: strconv.FormatInt(c-int64(elapsed), 10),
	})
	url, _ := common.NewURL(ProviderUrl)
	filter := activeFilter{}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	invoker := mock.NewMockInvoker(ctrl)
	invoker.EXPECT().GetURL().Return(url).Times(1)
	rpcResult := &result.RPCResult{
		Err: errors.New("test"),
	}
	filter.OnResponse(context.TODO(), rpcResult, invoker, invoc)
	methodStatus := base.GetMethodStatus(url, "test")
	urlStatus := base.GetURLStatus(url)

	assert.Equal(t, int32(1), methodStatus.GetTotal())
	assert.Equal(t, int32(1), urlStatus.GetTotal())
	assert.Equal(t, int32(1), methodStatus.GetFailed())
	assert.Equal(t, int32(1), urlStatus.GetFailed())
	assert.Equal(t, int32(1), methodStatus.GetSuccessiveRequestFailureCount())
	assert.Equal(t, int32(1), urlStatus.GetSuccessiveRequestFailureCount())
	assert.True(t, methodStatus.GetFailedElapsed() >= int64(elapsed))
	assert.True(t, urlStatus.GetFailedElapsed() >= int64(elapsed))
	assert.True(t, urlStatus.GetLastRequestFailedTimestamp() != int64(0))
	assert.True(t, methodStatus.GetLastRequestFailedTimestamp() != int64(0))
}

func TestFilterOnResponseWithDefer(t *testing.T) {
	base.CleanAllStatus()

	// Test scenario 1: dubboInvokeStartTime is parsed successfully and the result is correct.
	t.Run("ParseSuccessAndResultSuccess", func(t *testing.T) {
		defer base.CleanAllStatus()

		c := base.CurrentTimeMillis()
		invoc := invocation.NewRPCInvocation("test1", []any{"OK"}, map[string]any{
			dubboInvokeStartTime: strconv.FormatInt(c, 10),
		})
		url, _ := common.NewURL(ProviderUrl)
		filter := activeFilter{}
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		invoker := mock.NewMockInvoker(ctrl)
		invoker.EXPECT().GetURL().Return(url).Times(1)
		rpcResult := &result.RPCResult{}

		filter.OnResponse(context.TODO(), rpcResult, invoker, invoc)

		methodStatus := base.GetMethodStatus(url, "test1")
		urlStatus := base.GetURLStatus(url)

		assert.Equal(t, int32(1), methodStatus.GetTotal())
		assert.Equal(t, int32(1), urlStatus.GetTotal())
		assert.Equal(t, int32(0), methodStatus.GetFailed())
		assert.Equal(t, int32(0), urlStatus.GetFailed())
		assert.Equal(t, int32(0), methodStatus.GetSuccessiveRequestFailureCount())
		assert.Equal(t, int32(0), urlStatus.GetSuccessiveRequestFailureCount())
		assert.True(t, methodStatus.GetTotalElapsed() >= int64(0))
		assert.True(t, urlStatus.GetTotalElapsed() >= int64(0))
	})

	// Test scenario 2: dubboInvokeStartTime is parsed successfully, but the result is incorrect
	t.Run("ParseSuccessAndResultFailed", func(t *testing.T) {
		defer base.CleanAllStatus()

		c := base.CurrentTimeMillis()
		invoc := invocation.NewRPCInvocation("test2", []any{"OK"}, map[string]any{
			dubboInvokeStartTime: strconv.FormatInt(c, 10),
		})
		url, _ := common.NewURL(ProviderUrl)
		filter := activeFilter{}
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		invoker := mock.NewMockInvoker(ctrl)
		invoker.EXPECT().GetURL().Return(url).Times(1)
		rpcResult := &result.RPCResult{
			Err: errors.New("test error"),
		}

		filter.OnResponse(context.TODO(), rpcResult, invoker, invoc)

		methodStatus := base.GetMethodStatus(url, "test2")
		urlStatus := base.GetURLStatus(url)

		assert.Equal(t, int32(1), methodStatus.GetTotal())
		assert.Equal(t, int32(1), urlStatus.GetTotal())
		assert.Equal(t, int32(1), methodStatus.GetFailed())
		assert.Equal(t, int32(1), urlStatus.GetFailed())
		assert.Equal(t, int32(1), methodStatus.GetSuccessiveRequestFailureCount())
		assert.Equal(t, int32(1), urlStatus.GetSuccessiveRequestFailureCount())
		assert.True(t, methodStatus.GetFailedElapsed() >= int64(0))
		assert.True(t, urlStatus.GetFailedElapsed() >= int64(0))
		assert.True(t, urlStatus.GetLastRequestFailedTimestamp() != int64(0))
		assert.True(t, methodStatus.GetLastRequestFailedTimestamp() != int64(0))
	})

	// Test Scenario 3: dubboInvokeStartTime parsing failed (non-numeric string)
	t.Run("ParseFailedWithInvalidString", func(t *testing.T) {
		defer base.CleanAllStatus()

		invoc := invocation.NewRPCInvocation("test3", []any{"OK"}, map[string]any{
			dubboInvokeStartTime: "invalid-time",
		})
		url, _ := common.NewURL(ProviderUrl)
		filter := activeFilter{}
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		invoker := mock.NewMockInvoker(ctrl)
		invoker.EXPECT().GetURL().Return(url).Times(1)
		rpcResult := &result.RPCResult{}

		result := filter.OnResponse(context.TODO(), rpcResult, invoker, invoc)
		assert.NotNil(t, result.Error())

		methodStatus := base.GetMethodStatus(url, "test3")
		urlStatus := base.GetURLStatus(url)

		// Verification count and status - should use the default duration of 1 and be marked as failed
		assert.Equal(t, int32(1), methodStatus.GetTotal())
		assert.Equal(t, int32(1), urlStatus.GetTotal())
		assert.Equal(t, int32(1), methodStatus.GetFailed())
		assert.Equal(t, int32(1), urlStatus.GetFailed())
		assert.Equal(t, int32(1), methodStatus.GetSuccessiveRequestFailureCount())
		assert.Equal(t, int32(1), urlStatus.GetSuccessiveRequestFailureCount())
		assert.True(t, methodStatus.GetFailedElapsed() >= int64(1))
		assert.True(t, urlStatus.GetFailedElapsed() >= int64(1))
	})

	// Test scenario 4: dubboInvokeStartTime does not exist (use the default value 0)
	t.Run("ParseFailedWithDefaultValue", func(t *testing.T) {
		defer base.CleanAllStatus()

		invoc := invocation.NewRPCInvocation("test4", []any{"OK"}, make(map[string]any))
		url, _ := common.NewURL(ProviderUrl)
		filter := activeFilter{}
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		invoker := mock.NewMockInvoker(ctrl)
		invoker.EXPECT().GetURL().Return(url).Times(1)
		rpcResult := &result.RPCResult{}

		filter.OnResponse(context.TODO(), rpcResult, invoker, invoc)

		methodStatus := base.GetMethodStatus(url, "test4")
		urlStatus := base.GetURLStatus(url)

		assert.Equal(t, int32(1), methodStatus.GetTotal())
		assert.Equal(t, int32(1), urlStatus.GetTotal())
		assert.Equal(t, int32(0), methodStatus.GetFailed())
		assert.Equal(t, int32(0), urlStatus.GetFailed())
		assert.Equal(t, int32(0), methodStatus.GetSuccessiveRequestFailureCount())
		assert.Equal(t, int32(0), urlStatus.GetSuccessiveRequestFailureCount())
		assert.True(t, methodStatus.GetTotalElapsed() > int64(0))
		assert.True(t, urlStatus.GetTotalElapsed() > int64(0))
	})
}
