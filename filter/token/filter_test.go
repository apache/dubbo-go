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

package token

import (
	"context"
	"net/url"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

func TestTokenFilterInvoke(t *testing.T) {
	filter := &tokenFilter{}

	baseUrl := common.NewURLWithOptions(
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.TokenKey, "ori_key"))
	attch := make(map[string]any)
	attch[constant.TokenKey] = "ori_key"
	result := filter.Invoke(context.Background(),
		base.NewBaseInvoker(baseUrl),
		invocation.NewRPCInvocation("MethodName",
			[]any{"OK"}, attch))
	require.NoError(t, result.Error())
	assert.Nil(t, result.Result())
}

func TestTokenFilterInvokeEmptyToken(t *testing.T) {
	filter := &tokenFilter{}

	testUrl := common.URL{}
	attch := make(map[string]any)
	attch[constant.TokenKey] = "ori_key"
	result := filter.Invoke(context.Background(), base.NewBaseInvoker(&testUrl), invocation.NewRPCInvocation("MethodName", []any{"OK"}, attch))
	require.NoError(t, result.Error())
	assert.Nil(t, result.Result())
}

func TestTokenFilterInvokeEmptyAttach(t *testing.T) {
	filter := &tokenFilter{}

	testUrl := common.NewURLWithOptions(
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.TokenKey, "ori_key"))
	attch := make(map[string]any)
	result := filter.Invoke(context.Background(), base.NewBaseInvoker(testUrl), invocation.NewRPCInvocation("MethodName", []any{"OK"}, attch))
	assert.Error(t, result.Error())
}

func TestTokenFilterInvokeNotEqual(t *testing.T) {
	filter := &tokenFilter{}

	testUrl := common.NewURLWithOptions(
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.TokenKey, "ori_key"))
	attch := make(map[string]any)
	attch[constant.TokenKey] = "err_key"
	result := filter.Invoke(context.Background(),
		base.NewBaseInvoker(testUrl), invocation.NewRPCInvocation("MethodName", []any{"OK"}, attch))
	assert.Error(t, result.Error())
}
