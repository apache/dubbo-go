package config

import (
	"context"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

type TestService struct {
}

func (s *TestService) MethodOne(ctx context.Context, args []interface{}, rsp *struct{}) error {
	return nil
}

func (s *TestService) MethodTwo(ctx context.Context, args []interface{}, rsp *struct{}) error {
	return nil

}

func (s *TestService) Service() string {
	return "com.test.Service"
}
func (s *TestService) Version() string {
	return ""
}

func TestServiceMap_Register(t *testing.T) {
	s := &TestService{}
	methods, err := ServiceMap.Register("testporotocol", s)
	assert.NoError(t, err)
	assert.Equal(t, "MethodOne,MethodTwo", methods)
}
