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
	"fmt"
	"net/url"
	"testing"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"

	"github.com/golang/mock/gomock"

	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/protocol/invocation"
	"github.com/apache/dubbo-go/protocol/mock"
)

type MockHelloService struct{}

func (s *MockHelloService) Hello(who string) (string, error) {
	return fmt.Sprintf("hello, %s", who), nil
}

func (s *MockHelloService) JavaClassName() string {
	return "org.apache.dubbo.hello"
}

func (s *MockHelloService) Reference() string {
	return "org.apache.dubbo.test"
}

func TestServiceFilter_Invoke(t *testing.T) {
	filter := &GenericServiceFilter{}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInvoker := mock.NewMockInvoker(ctrl)

	// methodName is not "$invoke"
	invocation1 := invocation.NewRPCInvocation("test", nil, nil)
	mockInvoker.EXPECT().Invoke(gomock.Eq(invocation1))
	_ = filter.Invoke(context.Background(), mockInvoker, invocation1)
	// arguments are nil
	invocation2 := invocation.NewRPCInvocation(constant.GENERIC, nil, nil)
	mockInvoker.EXPECT().Invoke(gomock.Eq(invocation2))
	_ = filter.Invoke(context.Background(), mockInvoker, invocation2)
	// the number of arguments is not 3
	invocation3 := invocation.NewRPCInvocation(constant.GENERIC, []interface{}{"hello"}, nil)
	mockInvoker.EXPECT().Invoke(gomock.Eq(invocation3))
	_ = filter.Invoke(context.Background(), mockInvoker, invocation3)

	// hello service
	service := &MockHelloService{}
	// invoke URL
	ivkUrl := common.NewURLWithOptions(
		common.WithProtocol("test"),
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.INTERFACE_KEY, service.Reference()),
		common.WithParamsValue(constant.GENERIC_KEY, GENERIC_SERIALIZATION_DEFAULT))
	// registry RPC service
	_, err := common.ServiceMap.Register(ivkUrl.GetParam(constant.INTERFACE_KEY, ""),
		ivkUrl.Protocol,
		"",
		"",
		service)
	assert.Nil(t, err)

	// mock
	mockInvoker.EXPECT().GetURL().Return(ivkUrl).Times(3)

	// invoke a method without errors using default generalization
	invocation4 := invocation.NewRPCInvocation(constant.GENERIC,
		[]interface{}{
			"Hello",
			[]string{"java.lang.String"},
			[]hessian.Object{"world"},
		}, map[string]interface{}{
			constant.GENERIC_KEY: "true",
		})
	// invoke a non-existed method
	invocation5 := invocation.NewRPCInvocation(constant.GENERIC,
		[]interface{}{
			"hello11",
			[]string{"java.lang.String"},
			[]hessian.Object{"world"},
		}, map[string]interface{}{
			constant.GENERIC_KEY: "true",
		})
	// invoke a method with incorrect arguments
	invocation6 := invocation.NewRPCInvocation(constant.GENERIC,
		[]interface{}{
			"Hello",
			[]string{"java.lang.String", "java.lang.String"},
			[]hessian.Object{"world", "haha"},
		}, map[string]interface{}{
			constant.GENERIC_KEY: "true",
		})

	mockInvoker.EXPECT().Invoke(gomock.All(
		gomock.Not(invocation1),
		gomock.Not(invocation2),
		gomock.Not(invocation3),
	)).DoAndReturn(
		func(invocation protocol.Invocation) protocol.Result {
			switch invocation.MethodName() {
			case "Hello":
				who := invocation.Arguments()[0].(string)
				result, _ := service.Hello(who)
				return &protocol.RPCResult{
					Rest: result,
				}
			default:
				panic("this branch shouldn't be reached")
			}
		}).AnyTimes()

	result := filter.Invoke(context.Background(), mockInvoker, invocation4)
	assert.Nil(t, result.Error())
	assert.Equal(t, "hello, world", result.Result())

	result = filter.Invoke(context.Background(), mockInvoker, invocation5)
	assert.Equal(t,
		fmt.Sprintf("\"hello11\" method is not found, service key: %s", ivkUrl.ServiceKey()),
		fmt.Sprintf("%v", result.Error().(error)))

	result = filter.Invoke(context.Background(), mockInvoker, invocation6)
	assert.Equal(t,
		"the number of args(=2) is not matched with \"Hello\" method",
		fmt.Sprintf("%v", result.Error().(error)))
}

func TestServiceFilter_OnResponse(t *testing.T) {
	filter := &GenericServiceFilter{}

	// invoke a method without errors
	invocation1 := invocation.NewRPCInvocation(constant.GENERIC,
		[]interface{}{
			"hello",
			[]interface{}{"java.lang.String"},
			[]interface{}{"world"},
		}, map[string]interface{}{
			constant.GENERIC_KEY: "true",
		})

	rpcResult := &protocol.RPCResult{
		Rest: "result",
	}

	result := filter.OnResponse(context.Background(), rpcResult, nil, invocation1)
	assert.Equal(t, "result", result.Result())
}
