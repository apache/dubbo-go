package rest

import (
	"context"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/protocol/invocation"
	"github.com/apache/dubbo-go/protocol/rest/rest_client"
	_ "github.com/apache/dubbo-go/protocol/rest/rest_config_reader"
	"github.com/apache/dubbo-go/protocol/rest/rest_interface"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type User struct {
}

func TestRestInvoker_Invoke(t *testing.T) {
	// Refer
	proto := GetRestProtocol()
	url, err := common.NewURL(context.Background(), "rest://127.0.0.1:8888/com.ikurento.user.UserProvider?anyhost=true&"+
		"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&"+
		"environment=dev&interface=com.ikurento.user.UserProvider&ip=192.168.56.1&methods=GetUser%2C&"+
		"module=dubbogo+user-info+server&org=ikurento.com&owner=ZX&pid=1447&revision=0.0.1&"+
		"side=provider&timeout=3000&timestamp=1556509797245")
	assert.NoError(t, err)
	con := config.ConsumerConfig{
		ConnectTimeout: 1 * time.Second,
		RequestTimeout: 1 * time.Second,
		RestConfigType: "default",
	}
	config.SetConsumerConfig(con)
	configMap := make(map[string]*rest_interface.RestConfig)
	methodConfigMap := make(map[string]*rest_interface.RestMethodConfig)
	methodConfigMap["GetUser"] = &rest_interface.RestMethodConfig{
		InterfaceName:  "",
		MethodName:     "GetUser",
		Path:           "/GetUser",
		Produces:       "application/json",
		Consumes:       "application/json",
		MethodType:     "GET",
		PathParams:     "",
		PathParamsMap:  nil,
		QueryParams:    "",
		QueryParamsMap: nil,
		Body:           "",
		BodyMap:        nil,
	}
	configMap["com.ikurento.user.UserProvider"] = &rest_interface.RestConfig{
		RestMethodConfigsMap: methodConfigMap,
	}
	restClient := rest_client.GetRestyClient(&rest_interface.RestOptions{ConnectTimeout: 5 * time.Second, RequestTimeout: 5 * time.Second})
	invoker := NewRestInvoker(url, &restClient, methodConfigMap)
	user := &User{}
	inv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("GetUser"),
		invocation.WithArguments([]interface{}{"1", "username"}), invocation.WithReply(user))
	invoker.Invoke(context.Background(), inv)

	// make sure url
	eq := invoker.GetUrl().URLEqual(url)
	assert.True(t, eq)

	// make sure invokers after 'Destroy'
	invokersLen := len(proto.(*RestProtocol).Invokers())
	assert.Equal(t, 1, invokersLen)
	proto.Destroy()
	invokersLen = len(proto.(*RestProtocol).Invokers())
	assert.Equal(t, 0, invokersLen)
}
