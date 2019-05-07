package imp

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/dubbo/go-for-apache-dubbo/config"
	"github.com/dubbo/go-for-apache-dubbo/protocol"
	"github.com/dubbo/go-for-apache-dubbo/protocol/support"
)

func TestEchoFilter_Invoke(t *testing.T) {
	filter := GetFilter()
	result := filter.Invoke(protocol.NewBaseInvoker(config.URL{}),
		support.NewRPCInvocationForProvider("Echo", []interface{}{"OK"}, nil))
	assert.Equal(t, "OK", result.Result())

	result = filter.Invoke(protocol.NewBaseInvoker(config.URL{}),
		support.NewRPCInvocationForProvider("MethodName", []interface{}{"OK"}, nil))
	assert.Nil(t, result.Error())
	assert.Nil(t, result.Result())
}
