package jsonrpc

import (
	"context"
	"github.com/dubbo/go-for-apache-dubbo/protocol/invocation"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/dubbo/go-for-apache-dubbo/common"
	"github.com/dubbo/go-for-apache-dubbo/protocol"
)

func TestJsonrpcInvoker_Invoke(t *testing.T) {

	methods, err := common.ServiceMap.Register("jsonrpc", &UserProvider{})
	assert.NoError(t, err)
	assert.Equal(t, "GetUser,GetUser1", methods)

	// Export
	proto := GetProtocol()
	url, err := common.NewURL(context.Background(), "jsonrpc://127.0.0.1:20000/com.ikurento.user.UserProvider?anyhost=true&"+
		"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&"+
		"environment=dev&interface=com.ikurento.user.UserProvider&ip=192.168.56.1&methods=GetUser%2C&"+
		"module=dubbogo+user-info+server&org=ikurento.com&owner=ZX&pid=1447&revision=0.0.1&"+
		"side=provider&timeout=3000&timestamp=1556509797245")
	assert.NoError(t, err)
	proto.Export(protocol.NewBaseInvoker(url))

	client := NewHTTPClient(&HTTPOptions{
		HandshakeTimeout: time.Second,
		HTTPTimeout:      time.Second,
	})

	jsonInvoker := NewJsonrpcInvoker(url, client)
	user := &User{}
	res := jsonInvoker.Invoke(invocation.NewRPCInvocationForConsumer("GetUser", nil, []interface{}{"1", "username"}, user, nil, url, nil))
	assert.NoError(t, res.Error())
	assert.Equal(t, User{Id: "1", Name: "username"}, *res.Result().(*User))

	// destroy
	proto.Destroy()
}
