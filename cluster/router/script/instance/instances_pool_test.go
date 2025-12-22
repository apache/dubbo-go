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

package instance

import (
	"context"
	"testing"
)

import (
	"github.com/golang/mock/gomock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/mock"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

func TestGetInstancesJavascript(t *testing.T) {
	instance, err := GetInstances("javascript")
	assert.NoError(t, err)
	assert.NotNil(t, instance)
}

func TestGetInstancesJavascriptCaseInsensitive(t *testing.T) {
	tests := []string{
		"javascript",
		"JavaScript",
		"JAVASCRIPT",
		"JaVaScRiPt",
	}

	for _, scriptType := range tests {
		instance, err := GetInstances(scriptType)
		assert.NoError(t, err)
		assert.NotNil(t, instance)
	}
}

func TestGetInstancesNonExistent(t *testing.T) {
	instance, err := GetInstances("python")
	assert.Error(t, err)
	assert.Nil(t, instance)
	assert.Contains(t, err.Error(), "script type not be loaded")
}

func TestGetInstancesEmptyString(t *testing.T) {
	instance, err := GetInstances("")
	assert.Error(t, err)
	assert.Nil(t, instance)
}

func TestRangeInstances(t *testing.T) {
	count := 0
	RangeInstances(func(instance ScriptInstances) bool {
		count++
		assert.NotNil(t, instance)
		return true
	})

	assert.Greater(t, count, 0, "should have at least one instance")
}

func TestRangeInstancesStopEarly(t *testing.T) {
	count := 0
	RangeInstances(func(instance ScriptInstances) bool {
		count++
		return false // Stop after first iteration
	})

	assert.Equal(t, 1, count, "should stop after first instance")
}

func TestNewScriptInvokerImpl(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInvoker := mock.NewMockInvoker(ctrl)
	testURL, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService")
	require.NoError(t, err)
	mockInvoker.EXPECT().GetURL().Return(testURL).Times(1)

	wrapper := newScriptInvokerImpl(mockInvoker)

	assert.NotNil(t, wrapper)
	assert.False(t, wrapper.isRan)
	assert.NotNil(t, wrapper.copiedURL)
	assert.Equal(t, mockInvoker, wrapper.invoker)
}

func TestScriptInvokerWrapperGetURL(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInvoker := mock.NewMockInvoker(ctrl)
	testURL, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService")
	require.NoError(t, err)
	mockInvoker.EXPECT().GetURL().Return(testURL).Times(1)

	wrapper := newScriptInvokerImpl(mockInvoker)
	url := wrapper.GetURL()

	assert.NotNil(t, url)
	assert.Equal(t, testURL.Location, url.Location)
}

func TestScriptInvokerWrapperIsAvailableBeforeRan(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInvoker := mock.NewMockInvoker(ctrl)
	testURL, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService")
	require.NoError(t, err)
	mockInvoker.EXPECT().GetURL().Return(testURL).Times(1)

	wrapper := newScriptInvokerImpl(mockInvoker)

	// Before ran mode is set, should always return true
	assert.True(t, wrapper.IsAvailable())
}

func TestScriptInvokerWrapperIsAvailableAfterRan(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInvoker := mock.NewMockInvoker(ctrl)
	testURL, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService")
	require.NoError(t, err)
	mockInvoker.EXPECT().GetURL().Return(testURL).Times(1)
	mockInvoker.EXPECT().IsAvailable().Return(true).Times(1)

	wrapper := newScriptInvokerImpl(mockInvoker)
	wrapper.setRanMode()

	// After ran mode is set, should delegate to wrapped invoker
	assert.True(t, wrapper.IsAvailable())
}

func TestScriptInvokerWrapperIsAvailableAfterRanFalse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInvoker := mock.NewMockInvoker(ctrl)
	testURL, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService")
	require.NoError(t, err)
	mockInvoker.EXPECT().GetURL().Return(testURL).Times(1)
	mockInvoker.EXPECT().IsAvailable().Return(false).Times(1)

	wrapper := newScriptInvokerImpl(mockInvoker)
	wrapper.setRanMode()

	assert.False(t, wrapper.IsAvailable())
}

func TestScriptInvokerWrapperInvokePanicBeforeRan(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInvoker := mock.NewMockInvoker(ctrl)
	testURL, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService")
	require.NoError(t, err)
	mockInvoker.EXPECT().GetURL().Return(testURL).Times(1)

	wrapper := newScriptInvokerImpl(mockInvoker)
	inv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("test"))

	// Should panic when Invoke is called before ran mode is set
	assert.Panics(t, func() {
		wrapper.Invoke(context.Background(), inv)
	})
}

func TestScriptInvokerWrapperInvokeAfterRan(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInvoker := mock.NewMockInvoker(ctrl)
	testURL, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService")
	require.NoError(t, err)
	expectedResult := &result.RPCResult{}

	mockInvoker.EXPECT().GetURL().Return(testURL).Times(1)
	mockInvoker.EXPECT().Invoke(gomock.Any(), gomock.Any()).Return(expectedResult).Times(1)

	wrapper := newScriptInvokerImpl(mockInvoker)
	wrapper.setRanMode()

	inv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("test"))
	res := wrapper.Invoke(context.Background(), inv)

	assert.Equal(t, expectedResult, res)
}

func TestScriptInvokerWrapperDestroyPanicBeforeRan(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInvoker := mock.NewMockInvoker(ctrl)
	testURL, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService")
	require.NoError(t, err)
	mockInvoker.EXPECT().GetURL().Return(testURL).Times(1)

	wrapper := newScriptInvokerImpl(mockInvoker)

	// Should panic when Destroy is called before ran mode is set
	assert.Panics(t, func() {
		wrapper.Destroy()
	})
}

func TestScriptInvokerWrapperDestroyAfterRan(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInvoker := mock.NewMockInvoker(ctrl)
	testURL, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService")
	require.NoError(t, err)

	mockInvoker.EXPECT().GetURL().Return(testURL).Times(1)
	mockInvoker.EXPECT().Destroy().Times(1)

	wrapper := newScriptInvokerImpl(mockInvoker)
	wrapper.setRanMode()

	wrapper.Destroy()
	// Should not panic
}

func TestScriptInvokerWrapperSetRanMode(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInvoker := mock.NewMockInvoker(ctrl)
	testURL, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService")
	require.NoError(t, err)
	mockInvoker.EXPECT().GetURL().Return(testURL).Times(1)

	wrapper := newScriptInvokerImpl(mockInvoker)
	assert.False(t, wrapper.isRan)

	wrapper.setRanMode()
	assert.True(t, wrapper.isRan)
}

func TestScriptInvokerWrapperURLCloning(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInvoker := mock.NewMockInvoker(ctrl)
	testURL, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService")
	require.NoError(t, err)
	mockInvoker.EXPECT().GetURL().Return(testURL).Times(1)

	wrapper := newScriptInvokerImpl(mockInvoker)

	// The copied URL should be a clone, not the same reference
	copiedURL := wrapper.GetURL()
	assert.Equal(t, testURL.Location, copiedURL.Location)

	// Modifying the copied URL should not affect the original
	copiedURL.SetParam("test", "value")
	assert.NotEqual(t, testURL.GetParam("test", ""), copiedURL.GetParam("test", ""))
}
