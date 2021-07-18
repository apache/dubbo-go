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

package generic

import (
	"context"
	"net/url"
	"testing"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/mock"
)

// test isCallingToGenericService branch
func TestFilter_Invoke(t *testing.T) {
	invokeUrl := common.NewURLWithOptions(
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.GENERIC_KEY, constant.GenericSerializationDefault))
	filter := &Filter{}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	normalInvocation := invocation.NewRPCInvocation("Hello", []interface{}{"arg1"}, make(map[string]interface{}))

	mockInvoker := mock.NewMockInvoker(ctrl)
	mockInvoker.EXPECT().GetUrl().Return(invokeUrl).Times(2)
	mockInvoker.EXPECT().Invoke(gomock.Not(normalInvocation)).DoAndReturn(
		func(invocation protocol.Invocation) protocol.Result {
			assert.Equal(t, constant.GENERIC, invocation.MethodName())
			args := invocation.Arguments()
			assert.Equal(t, "Hello", args[0])
			assert.Equal(t, "java.lang.String", args[1].([]string)[0])
			assert.Equal(t, "arg1", args[2].([]hessian.Object)[0].(string))
			assert.Equal(t, constant.GenericSerializationDefault, invocation.AttachmentsByKey(constant.GENERIC_KEY, ""))
			return &protocol.RPCResult{}
		})

	result := filter.Invoke(context.Background(), mockInvoker, normalInvocation)
	assert.NotNil(t, result)
}

// test isMakingAGenericCall branch
func TestFilter_InvokeWithGenericCall(t *testing.T) {
	invokeUrl := common.NewURLWithOptions(
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.GENERIC_KEY, constant.GenericSerializationDefault))
	filter := &Filter{}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	genericInvocation := invocation.NewRPCInvocation(constant.GENERIC, []interface{}{
		"hello",
		[]string{"java.lang.String"},
		[]string{"arg1"},
	}, make(map[string]interface{}))

	mockInvoker := mock.NewMockInvoker(ctrl)
	mockInvoker.EXPECT().GetUrl().Return(invokeUrl).Times(3)
	mockInvoker.EXPECT().Invoke(gomock.Any()).DoAndReturn(
		func(invocation protocol.Invocation) protocol.Result {
			assert.Equal(t, constant.GENERIC, invocation.MethodName())
			args := invocation.Arguments()
			assert.Equal(t, "hello", args[0])
			assert.Equal(t, "java.lang.String", args[1].([]string)[0])
			assert.Equal(t, "arg1", args[2].([]string)[0])
			assert.Equal(t, constant.GenericSerializationDefault, invocation.AttachmentsByKey(constant.GENERIC_KEY, ""))
			return &protocol.RPCResult{}
		})

	result := filter.Invoke(context.Background(), mockInvoker, genericInvocation)
	assert.NotNil(t, result)
}
