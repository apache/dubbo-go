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

package hystrix

import (
	"context"
	"fmt"
	"regexp"
	"testing"
)

import (
	"github.com/afex/hystrix-go/hystrix"

	"github.com/pkg/errors"

	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

func init() {
	mockInitHystrixConfig()
}

func TestNewHystrixFilterError(t *testing.T) {
	get := NewHystrixFilterError(errors.New("test"), true)
	assert.True(t, get.(*FilterError).FailByHystrix())
	assert.Equal(t, "test", get.Error())
}

func mockInitHystrixConfig() {
	// Mock config
	confConsumer = &FilterConfig{
		make(map[string]*CommandConfigWithError),
		"Default",
		make(map[string]ServiceHystrixConfig),
	}
	confConsumer.Configs["Default"] = &CommandConfigWithError{
		Timeout:                1000,
		MaxConcurrentRequests:  600,
		RequestVolumeThreshold: 5,
		SleepWindow:            5000,
		ErrorPercentThreshold:  5,
		Error:                  nil,
	}
	confConsumer.Configs["userp"] = &CommandConfigWithError{
		Timeout:                2000,
		MaxConcurrentRequests:  64,
		RequestVolumeThreshold: 15,
		SleepWindow:            4000,
		ErrorPercentThreshold:  45,
		Error:                  nil,
	}
	confConsumer.Configs["userp_m"] = &CommandConfigWithError{
		Timeout:                1200,
		MaxConcurrentRequests:  64,
		RequestVolumeThreshold: 5,
		SleepWindow:            6000,
		ErrorPercentThreshold:  60,
		Error: []string{
			"exception",
		},
	}
	confConsumer.Services["com.ikurento.user.UserProvider"] = ServiceHystrixConfig{
		"userp",
		map[string]string{
			"GetUser": "userp_m",
		},
	}
}

func TestGetHystrixFilter(t *testing.T) {
	filterGot := newFilterConsumer()
	assert.NotNil(t, filterGot)
}

func TestGetConfig1(t *testing.T) {
	mockInitHystrixConfig()
	configGot := getConfig("com.ikurento.user.UserProvider", "GetUser", true)
	assert.NotNil(t, configGot)
	assert.Equal(t, 1200, configGot.Timeout)
	assert.Equal(t, 64, configGot.MaxConcurrentRequests)
	assert.Equal(t, 6000, configGot.SleepWindow)
	assert.Equal(t, 60, configGot.ErrorPercentThreshold)
	assert.Equal(t, 5, configGot.RequestVolumeThreshold)
}

func TestGetConfig2(t *testing.T) {
	mockInitHystrixConfig()
	configGot := getConfig("com.ikurento.user.UserProvider", "GetUser0", true)
	assert.NotNil(t, configGot)
	assert.Equal(t, 2000, configGot.Timeout)
	assert.Equal(t, 64, configGot.MaxConcurrentRequests)
	assert.Equal(t, 4000, configGot.SleepWindow)
	assert.Equal(t, 45, configGot.ErrorPercentThreshold)
	assert.Equal(t, 15, configGot.RequestVolumeThreshold)
}

func TestGetConfig3(t *testing.T) {
	mockInitHystrixConfig()
	// This should use default
	configGot := getConfig("Mock.Service", "GetMock", true)
	assert.NotNil(t, configGot)
	assert.Equal(t, 1000, configGot.Timeout)
	assert.Equal(t, 600, configGot.MaxConcurrentRequests)
	assert.Equal(t, 5000, configGot.SleepWindow)
	assert.Equal(t, 5, configGot.ErrorPercentThreshold)
	assert.Equal(t, 5, configGot.RequestVolumeThreshold)
}

type testMockSuccessInvoker struct {
	base.BaseInvoker
}

func (iv *testMockSuccessInvoker) Invoke(_ context.Context, _ base.Invocation) result.Result {
	return &result.RPCResult{
		Rest: "Success",
		Err:  nil,
	}
}

type testMockFailInvoker struct {
	base.BaseInvoker
}

func (iv *testMockFailInvoker) Invoke(_ context.Context, _ base.Invocation) result.Result {
	return &result.RPCResult{
		Err: errors.Errorf("exception"),
	}
}

func TestHystrixFilterInvokeSuccess(t *testing.T) {
	hf := &Filter{}
	testUrl, err := common.NewURL(
		fmt.Sprintf("dubbo://%s:%d/com.ikurento.user.UserProvider", constant.LocalHostValue, constant.DefaultPort))
	assert.NoError(t, err)
	testInvoker := testMockSuccessInvoker{*base.NewBaseInvoker(testUrl)}
	result := hf.Invoke(context.Background(), &testInvoker, &invocation.RPCInvocation{})
	assert.NotNil(t, result)
	assert.NoError(t, result.Error())
	assert.NotNil(t, result.Result())
}

func TestHystrixFilterInvokeFail(t *testing.T) {
	hf := &Filter{}
	testUrl, err := common.NewURL(
		fmt.Sprintf("dubbo://%s:%d/com.ikurento.user.UserProvider", constant.LocalHostValue, constant.DefaultPort))
	assert.NoError(t, err)
	testInvoker := testMockFailInvoker{*base.NewBaseInvoker(testUrl)}
	result := hf.Invoke(context.Background(), &testInvoker, &invocation.RPCInvocation{})
	assert.NotNil(t, result)
	assert.Error(t, result.Error())
}

func TestHystricFilterInvokeCircuitBreak(t *testing.T) {
	mockInitHystrixConfig()
	hystrix.Flush()
	hf := &Filter{COrP: true}
	resChan := make(chan result.Result, 50)
	for i := 0; i < 50; i++ {
		go func() {
			testUrl, err := common.NewURL(
				fmt.Sprintf("dubbo://%s:%d/com.ikurento.user.UserProvider", constant.LocalHostValue, constant.DefaultPort))
			assert.NoError(t, err)
			testInvoker := testMockSuccessInvoker{*base.NewBaseInvoker(testUrl)}
			result := hf.Invoke(context.Background(), &testInvoker, &invocation.RPCInvocation{})
			resChan <- result
		}()
	}
	// This can not always pass the test when on travis due to concurrency, you can uncomment the code below and test it locally

	//var lastRest bool
	//for i := 0; i < 50; i++ {
	//	lastRest = (<-resChan).Error().(*FilterError).FailByHystrix()
	//}
	//Normally the last result should be true, which means the circuit has been opened
	//
	//assert.True(t, lastRest)
}

func TestHystricFilterInvokeCircuitBreakOmitException(t *testing.T) {
	mockInitHystrixConfig()
	hystrix.Flush()
	reg, _ := regexp.Compile(".*exception.*")
	regs := []*regexp.Regexp{reg}
	hf := &Filter{res: map[string][]*regexp.Regexp{"": regs}, COrP: true}
	resChan := make(chan result.Result, 50)
	for i := 0; i < 50; i++ {
		go func() {
			testUrl, err := common.NewURL(
				fmt.Sprintf("dubbo://%s:%d/com.ikurento.user.UserProvider", constant.LocalHostValue, constant.DefaultPort))
			assert.NoError(t, err)
			testInvoker := testMockSuccessInvoker{*base.NewBaseInvoker(testUrl)}
			result := hf.Invoke(context.Background(), &testInvoker, &invocation.RPCInvocation{})
			resChan <- result
		}()
	}
	// This can not always pass the test when on travis due to concurrency, you can uncomment the code below and test it locally

	//time.Sleep(time.Second * 6)
	//var lastRest bool
	//for i := 0; i < 50; i++ {
	//	lastRest = (<-resChan).Error().(*FilterError).FailByHystrix()
	//}
	//
	//assert.False(t, lastRest)
}

func TestGetHystrixFilterConsumer(t *testing.T) {
	get := newFilterConsumer()
	assert.NotNil(t, get)
	assert.True(t, get.(*Filter).COrP)
}

func TestGetHystrixFilterProvider(t *testing.T) {
	get := newFilterProvider()
	assert.NotNil(t, get)
	assert.False(t, get.(*Filter).COrP)
}
