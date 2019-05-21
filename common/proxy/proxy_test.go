package proxy

import (
	"context"
	"errors"
	"reflect"
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
	MethodOne   func(context.Context, int, bool, *interface{}) error
	MethodTwo   func([]interface{}, *interface{}) error
	MethodThree func(int, bool) (interface{}, error)
	MethodFour  func(int, bool) (*interface{}, error) `dubbo:"methodFour"`
	Echo        func(interface{}, *interface{}) error
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
	err := p.Get().(*TestService).MethodOne(nil, 0, false, nil)
	assert.NoError(t, err)
	err = p.Get().(*TestService).MethodTwo(nil, nil)
	assert.NoError(t, err)
	ret, err := p.Get().(*TestService).MethodThree(0, false)
	assert.NoError(t, err)
	assert.Nil(t, ret) // ret is nil, because it doesn't be injection yet
	ret2, err := p.Get().(*TestService).MethodFour(0, false)
	assert.NoError(t, err)
	assert.Equal(t, "*interface {}", reflect.TypeOf(ret2).String())
	err = p.Get().(*TestService).Echo(nil, nil)
	assert.NoError(t, err)

	// inherit & lowercase
	p.rpc = nil
	type S1 struct {
		TestService
		methodOne func(context.Context, interface{}, *struct{}) error
	}
	s1 := &S1{TestService: *s, methodOne: func(i context.Context, i2 interface{}, i3 *struct{}) error {
		return errors.New("errors")
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

	// reply type
	p.rpc = nil
	type S3 struct {
		TestService
		MethodOne func(context.Context, []interface{}, struct{}) error
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

}
