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

package filter_impl

import (
	"context"
	"net/url"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/protocol/invocation"
)

func TestExecuteLimitFilterInvokeIgnored(t *testing.T) {
	methodName := "hello"
	invoc := invocation.NewRPCInvocation(methodName, []interface{}{"OK"}, make(map[string]interface{}, 0))

	invokeUrl := common.NewURLWithOptions(
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.INTERFACE_KEY, methodName))

	limitFilter := GetExecuteLimitFilter()

	result := limitFilter.Invoke(context.Background(), protocol.NewBaseInvoker(invokeUrl), invoc)
	assert.NotNil(t, result)
	assert.Nil(t, result.Error())
}

func TestExecuteLimitFilterInvokeConfigureError(t *testing.T) {
	methodName := "hello1"
	invoc := invocation.NewRPCInvocation(methodName, []interface{}{"OK"}, make(map[string]interface{}, 0))

	invokeUrl := common.NewURLWithOptions(
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.INTERFACE_KEY, methodName),
		common.WithParamsValue(constant.EXECUTE_LIMIT_KEY, "13a"),
	)

	limitFilter := GetExecuteLimitFilter()

	result := limitFilter.Invoke(context.Background(), protocol.NewBaseInvoker(invokeUrl), invoc)
	assert.NotNil(t, result)
	assert.Nil(t, result.Error())
}

func TestExecuteLimitFilterInvoke(t *testing.T) {
	methodName := "hello1"
	invoc := invocation.NewRPCInvocation(methodName, []interface{}{"OK"}, make(map[string]interface{}, 0))

	invokeUrl := common.NewURLWithOptions(
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.INTERFACE_KEY, methodName),
		common.WithParamsValue(constant.EXECUTE_LIMIT_KEY, "20"),
	)

	limitFilter := GetExecuteLimitFilter()

	result := limitFilter.Invoke(context.Background(), protocol.NewBaseInvoker(invokeUrl), invoc)
	assert.NotNil(t, result)
	assert.Nil(t, result.Error())
}
