package proxy

import (
	"context"
	"errors"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/dubbo/go-for-apache-dubbo/common"
	"github.com/dubbo/go-for-apache-dubbo/common/constant"
	"github.com/dubbo/go-for-apache-dubbo/protocol"
)

type TestService struct {
	MethodOne   func(context.Context, []interface{}, *struct{}) error
	MethodTwo   func([]interface{}, *struct{}) error
	MethodThree func([]interface{}, *struct{}) error `dubbo:"methodThree"`
	Echo        func([]interface{}, *struct{}) error
}

func (s *TestService) Service() string {
	return "com.test.Path"
}
func (s *TestService) Version() string {
	return ""
}

type TestServiceInt int

func (s *TestServiceInt) Service() string {
	return "com.test.TestServiceInt"
}
func (s *TestServiceInt) Version() string {
	return ""
}

func TestProxy_Implement(t *testing.T) {

	invoker := protocol.NewBaseInvoker(common.URL{})
	p := NewProxy(invoker, nil, map[string]string{constant.ASYNC_KEY: "false"})
	s := &TestService{}
	p.Implement(s)
	err := p.Get().(*TestService).MethodOne(nil, nil, nil)
	assert.NoError(t, err)
	err = p.Get().(*TestService).MethodTwo(nil, nil)
	assert.NoError(t, err)
	err = p.Get().(*TestService).MethodThree(nil, nil)
	assert.NoError(t, err)
	err = p.Get().(*TestService).Echo(nil, nil)
	assert.NoError(t, err)

	// inherit & lowercase
	p.rpc = nil
	type S1 struct {
		TestService
		methodOne func(context.Context, []interface{}, *struct{}) error
	}
	s1 := &S1{TestService: *s, methodOne: func(i context.Context, i2 []interface{}, i3 *struct{}) error {
		return errors.New("errors")
	}}
	p.Implement(s1)
	err = s1.MethodOne(nil, nil, nil)
	assert.NoError(t, err)
	err = s1.methodOne(nil, nil, nil)
	assert.EqualError(t, err, "errors")

	// no struct
	p.rpc = nil
	it := TestServiceInt(1)
	p.Implement(&it)
	assert.Nil(t, p.rpc)

	// args number
	p.rpc = nil
	type S2 struct {
		TestService
		MethodOne func([]interface{}) error
	}
	s2 := &S2{TestService: *s}
	p.Implement(s2)
	assert.Nil(t, s2.MethodOne)

	// returns number
	p.rpc = nil
	type S3 struct {
		TestService
		MethodOne func(context.Context, []interface{}, *struct{}) (interface{}, error)
	}
	s3 := &S3{TestService: *s}
	p.Implement(s3)
	assert.Nil(t, s3.MethodOne)

	// returns type
	p.rpc = nil
	type S4 struct {
		TestService
		MethodOne func(context.Context, []interface{}, *struct{}) interface{}
	}
	s4 := &S4{TestService: *s}
	p.Implement(s4)
	assert.Nil(t, s4.MethodOne)

	// reply type for number 3
	p.rpc = nil
	type S5 struct {
		TestService
		MethodOne func(context.Context, []interface{}, interface{}) error
	}
	s5 := &S5{TestService: *s}
	p.Implement(s5)
	assert.Nil(t, s5.MethodOne)

	// reply type for number 2
	p.rpc = nil
	type S6 struct {
		TestService
		MethodOne func([]interface{}, interface{}) error
	}
	s6 := &S6{TestService: *s}
	p.Implement(s6)
	assert.Nil(t, s5.MethodOne)
}
