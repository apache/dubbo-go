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

package exec_limit

import (
	"context"
	"net/url"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

func TestFilterInvokeIgnored(t *testing.T) {
	methodName := "hello"
	invoc := invocation.NewRPCInvocation(methodName, []interface{}{"OK"}, make(map[string]interface{}))

	invokeUrl := common.NewURLWithOptions(
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.InterfaceKey, methodName))

	limitFilter := newFilter()

	result := limitFilter.Invoke(context.Background(), protocol.NewBaseInvoker(invokeUrl), invoc)
	assert.NotNil(t, result)
	assert.Nil(t, result.Error())
}

func TestFilterInvokeConfigureError(t *testing.T) {
	methodName := "hello1"
	invoc := invocation.NewRPCInvocation(methodName, []interface{}{"OK"}, make(map[string]interface{}))

	invokeUrl := common.NewURLWithOptions(
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.InterfaceKey, methodName),
		common.WithParamsValue(constant.ExecuteLimitKey, "13a"),
	)

	limitFilter := newFilter()

	result := limitFilter.Invoke(context.Background(), protocol.NewBaseInvoker(invokeUrl), invoc)
	assert.NotNil(t, result)
	assert.Nil(t, result.Error())
}

func TestFilterInvoke(t *testing.T) {
	methodName := "hello1"
	invoc := invocation.NewRPCInvocation(methodName, []interface{}{"OK"}, make(map[string]interface{}))

	invokeUrl := common.NewURLWithOptions(
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.InterfaceKey, methodName),
		common.WithParamsValue(constant.ExecuteLimitKey, "20"),
	)

	limitFilter := newFilter()

	result := limitFilter.Invoke(context.Background(), protocol.NewBaseInvoker(invokeUrl), invoc)
	assert.NotNil(t, result)
	assert.Nil(t, result.Error())
}
