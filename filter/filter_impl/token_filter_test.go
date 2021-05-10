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
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

func TestTokenFilterInvoke(t *testing.T) {
	filter := GetTokenFilter()

	url := common.NewURLWithOptions(
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.TOKEN_KEY, "ori_key"))
	attch := make(map[string]interface{})
	attch[constant.TOKEN_KEY] = "ori_key"
	result := filter.Invoke(context.Background(),
		protocol.NewBaseInvoker(url),
		invocation.NewRPCInvocation("MethodName",
			[]interface{}{"OK"}, attch))
	assert.Nil(t, result.Error())
	assert.Nil(t, result.Result())
}

func TestTokenFilterInvokeEmptyToken(t *testing.T) {
	filter := GetTokenFilter()

	testUrl := common.URL{}
	attch := make(map[string]interface{})
	attch[constant.TOKEN_KEY] = "ori_key"
	result := filter.Invoke(context.Background(), protocol.NewBaseInvoker(&testUrl), invocation.NewRPCInvocation("MethodName", []interface{}{"OK"}, attch))
	assert.Nil(t, result.Error())
	assert.Nil(t, result.Result())
}

func TestTokenFilterInvokeEmptyAttach(t *testing.T) {
	filter := GetTokenFilter()

	testUrl := common.NewURLWithOptions(
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.TOKEN_KEY, "ori_key"))
	attch := make(map[string]interface{})
	result := filter.Invoke(context.Background(), protocol.NewBaseInvoker(testUrl), invocation.NewRPCInvocation("MethodName", []interface{}{"OK"}, attch))
	assert.NotNil(t, result.Error())
}

func TestTokenFilterInvokeNotEqual(t *testing.T) {
	filter := GetTokenFilter()

	testUrl := common.NewURLWithOptions(
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.TOKEN_KEY, "ori_key"))
	attch := make(map[string]interface{})
	attch[constant.TOKEN_KEY] = "err_key"
	result := filter.Invoke(context.Background(),
		protocol.NewBaseInvoker(testUrl), invocation.NewRPCInvocation("MethodName", []interface{}{"OK"}, attch))
	assert.NotNil(t, result.Error())
}
