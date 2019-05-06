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
	"github.com/dubbo/go-for-apache-dubbo/config"
	"github.com/dubbo/go-for-apache-dubbo/protocol"
)

type TestService struct {
	MethodOne func(context.Context, []interface{}, *struct{}) error
}

func (s *TestService) Service() string {
	return "com.test.Path"
}
func (s *TestService) Version() string {
	return ""
}

func TestProxy_Implement(t *testing.T) {

	invoker := protocol.NewBaseInvoker(config.URL{})
	p := NewProxy(invoker, nil, nil)
	s := &TestService{MethodOne: func(i context.Context, i2 []interface{}, i3 *struct{}) error {
		return errors.New("errors")
	}}
	p.Implement(s)
	err := s.MethodOne(nil, nil, nil)
	assert.NoError(t, err)

	// inherit & lowercase
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

	// args number
	type S2 struct {
		TestService
		MethodOne func(context.Context, []interface{}) error
	}
	s2 := &S2{TestService: *s}
	p.Implement(s2)
	assert.Nil(t, s2.MethodOne)
	//assert.EqualError(t, err, "method MethodOne of mtype func(context.Context, []interface {}) error has wrong number of in parameters 2; needs exactly 3/4")

	// returns number
	type S3 struct {
		TestService
		MethodOne func(context.Context, []interface{}, *struct{}) (interface{}, error)
	}
	s3 := &S3{TestService: *s}
	p.Implement(s3)
	assert.Nil(t, s3.MethodOne)
	//assert.EqualError(t, err, "method \"MethodOne\" has 2 out parameters; needs exactly 1")

	// returns type
	type S4 struct {
		TestService
		MethodOne func(context.Context, []interface{}, *struct{}) interface{}
	}
	s4 := &S4{TestService: *s}
	p.Implement(s4)
	assert.Nil(t, s4.MethodOne)
	//assert.EqualError(t, err, "return type interface {} of method \"MethodOne\" is not error")

	// reply type
	type S5 struct {
		TestService
		MethodOne func(context.Context, []interface{}, interface{}) error
	}
	s5 := &S5{TestService: *s}
	p.Implement(s5)
	assert.Nil(t, s5.MethodOne)
	//assert.EqualError(t, err, "reply type of method \"MethodOne\" is not a pointer interface {}")

}
