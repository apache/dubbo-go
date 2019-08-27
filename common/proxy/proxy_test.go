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

package proxy

import (
	"context"
	"reflect"
	"testing"
)

import (
	perrors "github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/protocol"
)

type TestService struct {
	MethodOne   func(context.Context, int, bool, *interface{}) error
	MethodTwo   func([]interface{}) error
	MethodThree func(int, bool) (interface{}, error)
	MethodFour  func(int, bool) (*interface{}, error) `dubbo:"methodFour"`
	MethodFive  func() error
	Echo        func(interface{}, *interface{}) error
}

func (s *TestService) Reference() string {
	return "com.test.Path"
}

type TestServiceInt int

func (s *TestServiceInt) Reference() string {
	return "com.test.TestServiceInt"
}

func TestProxy_Implement(t *testing.T) {

	invoker := protocol.NewBaseInvoker(common.URL{})
	p := NewProxy(invoker, nil, map[string]string{constant.ASYNC_KEY: "false"})
	s := &TestService{}
	p.Implement(s)

	err := p.Get().(*TestService).MethodOne(nil, 0, false, nil)
	assert.NoError(t, err)

	err = p.Get().(*TestService).MethodTwo(nil)
	assert.NoError(t, err)
	ret, err := p.Get().(*TestService).MethodThree(0, false)
	assert.NoError(t, err)
	assert.Nil(t, ret) // ret is nil, because it doesn't be injection yet

	ret2, err := p.Get().(*TestService).MethodFour(0, false)
	assert.NoError(t, err)
	assert.Equal(t, "*interface {}", reflect.TypeOf(ret2).String())
	err = p.Get().(*TestService).Echo(nil, nil)
	assert.NoError(t, err)

	err = p.Get().(*TestService).MethodFive()
	assert.NoError(t, err)

	// inherit & lowercase
	p.rpc = nil
	type S1 struct {
		TestService
		methodOne func(context.Context, interface{}, *struct{}) error
	}
	s1 := &S1{TestService: *s, methodOne: func(i context.Context, i2 interface{}, i3 *struct{}) error {
		return perrors.New("errors")
	}}
	p.Implement(s1)
	err = s1.MethodOne(nil, 0, false, nil)
	assert.NoError(t, err)
	err = s1.methodOne(nil, nil, nil)
	assert.EqualError(t, err, "errors")

	// no struct
	p.rpc = nil
	it := TestServiceInt(1)
	p.Implement(&it)
	assert.Nil(t, p.rpc)

	// return number
	p.rpc = nil
	type S2 struct {
		TestService
		MethodOne func([]interface{}) (*struct{}, int, error)
	}
	s2 := &S2{TestService: *s}
	p.Implement(s2)
	assert.Nil(t, s2.MethodOne)

	// returns type
	p.rpc = nil
	type S3 struct {
		TestService
		MethodOne func(context.Context, []interface{}, *struct{}) interface{}
	}
	s3 := &S3{TestService: *s}
	p.Implement(s3)
	assert.Nil(t, s3.MethodOne)

}
