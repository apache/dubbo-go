package rest

import (
	"context"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/protocol/invocation"
	"github.com/apache/dubbo-go/protocol/rest/rest_client"
	_ "github.com/apache/dubbo-go/protocol/rest/rest_config_reader"
	"github.com/apache/dubbo-go/protocol/rest/rest_interface"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestRestInvoker_Invoke(t *testing.T) {
	// Refer
	proto := GetRestProtocol()
	defer proto.Destroy()
	url, err := common.NewURL(context.Background(), "rest://127.0.0.1:8877/com.ikurento.user.UserProvider?anyhost=true&"+
		"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&"+
		"environment=dev&interface=com.ikurento.user.UserProvider&ip=192.168.56.1&methods=GetUser%2C&"+
		"module=dubbogo+user-info+server&org=ikurento.com&owner=ZX&pid=1447&revision=0.0.1&"+
		"side=provider&timeout=3000&timestamp=1556509797245")
	assert.NoError(t, err)
	_, err = common.ServiceMap.Register(url.Protocol, &UserProvider{})
	assert.NoError(t, err)
	con := config.ProviderConfig{}
	config.SetProviderConfig(con)
	configMap := make(map[string]*rest_interface.RestConfig)
	methodConfigMap := make(map[string]*rest_interface.RestMethodConfig)
	queryParamsMap := make(map[int]string)
	queryParamsMap[1] = "age"
	queryParamsMap[2] = "name"
	pathParamsMap := make(map[int]string)
	pathParamsMap[0] = "userid"
	methodConfigMap["GetUserOne"] = &rest_interface.RestMethodConfig{
		InterfaceName:  "",
		MethodName:     "GetUserOne",
		Path:           "/GetUserOne",
		Produces:       "application/json",
		Consumes:       "application/json",
		MethodType:     "POST",
		PathParams:     "",
		PathParamsMap:  nil,
		QueryParams:    "",
		QueryParamsMap: nil,
		Body:           0,
	}
	methodConfigMap["GetUser"] = &rest_interface.RestMethodConfig{
		InterfaceName:  "",
		MethodName:     "GetUser",
		Path:           "/GetUser/{userid}",
		Produces:       "application/json",
		Consumes:       "application/json",
		MethodType:     "GET",
		PathParams:     "",
		PathParamsMap:  pathParamsMap,
		QueryParams:    "",
		QueryParamsMap: queryParamsMap,
		Body:           -1,
	}

	configMap["com.ikurento.user.UserProvider"] = &rest_interface.RestConfig{
		Server:               "go-restful",
		RestMethodConfigsMap: methodConfigMap,
	}
	SetRestProviderServiceConfigMap(configMap)
	proxyFactory := extension.GetProxyFactory("default")
	proto.Export(proxyFactory.GetInvoker(url))
	time.Sleep(5 * time.Second)
	configMap = make(map[string]*rest_interface.RestConfig)
	configMap["com.ikurento.user.UserProvider"] = &rest_interface.RestConfig{
		RestMethodConfigsMap: methodConfigMap,
	}
	restClient := rest_client.GetRestyClient(&rest_interface.RestOptions{ConnectTimeout: 3 * time.Second, RequestTimeout: 3 * time.Second})
	invoker := NewRestInvoker(url, &restClient, methodConfigMap)
	user := &User{}
	inv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("GetUser"),
		invocation.WithArguments([]interface{}{1, int32(23), "username"}), invocation.WithReply(user))
	res := invoker.Invoke(context.Background(), inv)
	assert.NoError(t, res.Error())
	assert.Equal(t, User{Id: 1, Age: int32(23), Name: "username"}, *res.Result().(*User))
	time.Sleep(3 * time.Second)
	now := time.Now()
	inv = invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("GetUserOne"),
		invocation.WithArguments([]interface{}{&User{1, &now, int32(23), "username"}}), invocation.WithReply(user))
	res = invoker.Invoke(context.Background(), inv)
	assert.NoError(t, res.Error())
	assert.Equal(t, 1, res.Result().(*User).Id)
	assert.Equal(t, now.Unix(), res.Result().(*User).Time.Unix())
	assert.Equal(t, int32(23), res.Result().(*User).Age)
	assert.Equal(t, "username", res.Result().(*User).Name)
	err = common.ServiceMap.UnRegister(url.Protocol, "com.ikurento.user.UserProvider")
	assert.NoError(t, err)
}
