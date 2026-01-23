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
	"testing"
)

import (
	"github.com/golang/mock/gomock"

	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/filter/generic/generalizer"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/mock"
)

func TestIsCallingToGenericService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	u1, _ := common.NewURL("dubbo://127.0.0.1:20000/ComTest?generic=true")
	invoker1 := mock.NewMockInvoker(ctrl)
	invoker1.EXPECT().GetURL().Return(u1).AnyTimes()
	invoc1 := invocation.NewRPCInvocation("GetUser", []any{"ID"}, nil)
	assert.True(t, isCallingToGenericService(invoker1, invoc1))

	u2, _ := common.NewURL("dubbo://127.0.0.1:20000/ComTest")
	invoker2 := mock.NewMockInvoker(ctrl)
	invoker2.EXPECT().GetURL().Return(u2).AnyTimes()
	assert.False(t, isCallingToGenericService(invoker2, invoc1))

	invoc3 := invocation.NewRPCInvocation(constant.Generic, []any{"m", "t", "a"}, nil)
	assert.False(t, isCallingToGenericService(invoker1, invoc3))
}

func TestIsMakingAGenericCall(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	u, _ := common.NewURL("dubbo://127.0.0.1:20000/ComTest?generic=true")
	invoker := mock.NewMockInvoker(ctrl)
	invoker.EXPECT().GetURL().Return(u).AnyTimes()

	invoc1 := invocation.NewRPCInvocation(constant.Generic, []any{"method", "types", "args"}, nil)
	assert.True(t, isMakingAGenericCall(invoker, invoc1))

	invoc2 := invocation.NewRPCInvocation(constant.Generic, []any{"method"}, nil)
	assert.False(t, isMakingAGenericCall(invoker, invoc2))

	invoc3 := invocation.NewRPCInvocation("GetUser", []any{"a", "b", "c"}, nil)
	assert.False(t, isMakingAGenericCall(invoker, invoc3))
}

func TestIsGeneric(t *testing.T) {
	assert.True(t, isGeneric("true"))
	assert.True(t, isGeneric("True"))
	assert.True(t, isGeneric("gson"))
	assert.True(t, isGeneric("Gson"))
	assert.True(t, isGeneric("protobuf-json"))
	assert.True(t, isGeneric("Protobuf-Json"))
	assert.False(t, isGeneric("false"))
	assert.False(t, isGeneric(""))
	assert.False(t, isGeneric("bean"))
}

func TestGetGeneralizer(t *testing.T) {
	g1 := getGeneralizer(constant.GenericSerializationDefault)
	assert.IsType(t, generalizer.GetMapGeneralizer(), g1)

	g2 := getGeneralizer(constant.GenericSerializationGson)
	assert.IsType(t, generalizer.GetGsonGeneralizer(), g2)

	g3 := getGeneralizer(constant.GenericSerializationProtobufJson)
	assert.IsType(t, generalizer.GetProtobufJsonGeneralizer(), g3)

	// test case insensitive
	g4 := getGeneralizer("Protobuf-Json")
	assert.IsType(t, generalizer.GetProtobufJsonGeneralizer(), g4)

	g5 := getGeneralizer("unsupported_type")
	assert.IsType(t, generalizer.GetMapGeneralizer(), g5)
}
