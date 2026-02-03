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
	"errors"
	"testing"
)

import (
	"github.com/afex/hystrix-go/hystrix"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

func TestNewHystrixFilterError(t *testing.T) {
	get := NewHystrixFilterError(errors.New("test"), true)
	assert.True(t, get.(*FilterError).FailByHystrix())
	assert.Equal(t, "test", get.Error())
}

func TestGetHystrixFilter(t *testing.T) {
	filterGot := newFilterConsumer()
	assert.NotNil(t, filterGot)

	filterGot = newFilterProvider()
	assert.NotNil(t, filterGot)
}

func TestHystrixFilter_Invoke(t *testing.T) {
	// Configure hystrix command for testing
	// Resource name format: dubbo:consumer:InterfaceName:group:version:Method(paramTypes)
	cmdName := "dubbo:consumer:com.ikurento.user.UserProvider:::TestMethod()"
	hystrix.ConfigureCommand(cmdName, hystrix.CommandConfig{
		Timeout:                1000,
		MaxConcurrentRequests:  10,
		RequestVolumeThreshold: 5,
		SleepWindow:            1000,
		ErrorPercentThreshold:  50,
	})

	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider")
	require.NoError(t, err)
	mockInvoker := base.NewBaseInvoker(url)

	filter := &Filter{COrP: true}
	mockInvocation := invocation.NewRPCInvocation("TestMethod", []any{"OK"}, make(map[string]any))

	ctx := context.Background()
	result := filter.Invoke(ctx, mockInvoker, mockInvocation)
	assert.NotNil(t, result)
	assert.NoError(t, result.Error())
}

func TestHystrixFilter_OnResponse(t *testing.T) {
	filter := &Filter{COrP: true}
	ctx := context.Background()
	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider")
	require.NoError(t, err)
	mockInvoker := base.NewBaseInvoker(url)
	mockInvocation := invocation.NewRPCInvocation("TestMethod", []any{}, make(map[string]any))

	result := filter.OnResponse(ctx, nil, mockInvoker, mockInvocation)
	assert.Nil(t, result)
}
