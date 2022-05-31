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

package rest

import (
	"context"
	"testing"
	"time"
)

import (
	"github.com/emicklei/go-restful/v3"

	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config"
	rest_config "dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/rest/client"
	"dubbo.apache.org/dubbo-go/v3/protocol/rest/client/client_impl"
	server_impl "dubbo.apache.org/dubbo-go/v3/protocol/rest/server"
)

const (
	mockRestCommonUrl = "rest://127.0.0.1:8877/com.ikurento.user.UserProvider?anyhost=true&" +
		"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&" +
		"environment=dev&interface=com.ikurento.user.UserProvider&ip=192.168.56.1&methods=GetUser%2C&" +
		"module=dubbogo+user-info+server&org=ikurento.com&owner=ZX&pid=1447&revision=0.0.1&" +
		"side=provider&timeout=3000&timestamp=1556509797245&bean.name=com.ikurento.user.UserProvider"
)

func TestRestInvokerInvoke(t *testing.T) {
	// Refer
	proto := GetRestProtocol()
	defer proto.Destroy()
	var filterNum int
	server_impl.AddGoRestfulServerFilter(func(request *restful.Request, response *restful.Response, chain *restful.FilterChain) {
		println(request.SelectedRoutePath())
		filterNum = filterNum + 1
		chain.ProcessFilter(request, response)
	})
	server_impl.AddGoRestfulServerFilter(func(request *restful.Request, response *restful.Response, chain *restful.FilterChain) {
		println("filter2")
		filterNum = filterNum + 1
		chain.ProcessFilter(request, response)
	})

	url, err := common.NewURL(mockRestCommonUrl)
	assert.NoError(t, err)
	_, err = common.ServiceMap.Register(url.Service(), url.Protocol, "", "", &UserProvider{})
	assert.NoError(t, err)
	con := config.ProviderConfig{}
	rest_config.SetProviderService(con)
	configMap := make(map[string]*rest_config.RestServiceConfig)
	methodConfigMap := make(map[string]*rest_config.RestMethodConfig)
	queryParamsMap := make(map[int]string)
	queryParamsMap[1] = "age"
	queryParamsMap[2] = "name"
	pathParamsMap := make(map[int]string)
	pathParamsMap[0] = "userid"
	headersMap := make(map[int]string)
	headersMap[3] = "Content-Type"
	methodConfigMap["GetUserOne"] = &rest_config.RestMethodConfig{
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
	methodConfigMap["GetUserTwo"] = &rest_config.RestMethodConfig{
		InterfaceName:  "",
		MethodName:     "GetUserTwo",
		Path:           "/GetUserTwo",
		Produces:       "application/json",
		Consumes:       "application/json",
		MethodType:     "POST",
		PathParams:     "",
		PathParamsMap:  nil,
		QueryParams:    "",
		QueryParamsMap: nil,
		Body:           0,
	}
	methodConfigMap["GetUserThree"] = &rest_config.RestMethodConfig{
		InterfaceName:  "",
		MethodName:     "GetUserThree",
		Path:           "/GetUserThree",
		Produces:       "application/json",
		Consumes:       "application/json",
		MethodType:     "POST",
		PathParams:     "",
		PathParamsMap:  nil,
		QueryParams:    "",
		QueryParamsMap: nil,
		Body:           0,
	}
	methodConfigMap["GetUserFour"] = &rest_config.RestMethodConfig{
		InterfaceName:  "",
		MethodName:     "GetUserFour",
		Path:           "/GetUserFour",
		Produces:       "application/json",
		Consumes:       "application/json",
		MethodType:     "POST",
		PathParams:     "",
		PathParamsMap:  nil,
		QueryParams:    "",
		QueryParamsMap: nil,
		Body:           0,
	}
	methodConfigMap["GetUserFive"] = &rest_config.RestMethodConfig{
		InterfaceName: "",
		MethodName:    "GetUserFive",
		Path:          "/GetUserFive",
		Produces:      "*/*",
		Consumes:      "*/*",
		MethodType:    "GET",
	}
	methodConfigMap["GetUser"] = &rest_config.RestMethodConfig{
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
		HeadersMap:     headersMap,
	}

	configMap["com.ikurento.user.UserProvider"] = &rest_config.RestServiceConfig{
		Server:            "go-restful",
		RestMethodConfigs: methodConfigMap,
	}
	rest_config.SetRestProviderServiceConfigMap(configMap)
	proxyFactory := extension.GetProxyFactory("default")
	proto.Export(proxyFactory.GetInvoker(url))
	time.Sleep(5 * time.Second)
	configMap = make(map[string]*rest_config.RestServiceConfig)
	configMap["com.ikurento.user.UserProvider"] = &rest_config.RestServiceConfig{
		RestMethodConfigs: methodConfigMap,
		InterfaceName:     "com.ikurento.user.UserProvider",
	}
	restClient := client_impl.NewRestyClient(&client.RestOptions{ConnectTimeout: 3 * time.Second, RequestTimeout: 3 * time.Second})
	invoker := NewRestInvoker(url, &restClient, methodConfigMap)
	user := &User{}
	inv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("GetUser"),
		invocation.WithArguments([]interface{}{1, int32(23), "username", "application/json"}), invocation.WithReply(user))
	res := invoker.Invoke(context.Background(), inv)
	assert.NoError(t, res.Error())
	assert.Equal(t, User{ID: 1, Age: int32(23), Name: "username"}, *res.Result().(*User))
	now := time.Now()
	inv = invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("GetUserOne"),
		invocation.WithArguments([]interface{}{&User{1, &now, int32(23), "username"}}), invocation.WithReply(user))
	res = invoker.Invoke(context.Background(), inv)
	assert.NoError(t, res.Error())
	assert.NotNil(t, res.Result())
	assert.Equal(t, 1, res.Result().(*User).ID)
	assert.Equal(t, now.Unix(), res.Result().(*User).Time.Unix())
	assert.Equal(t, int32(23), res.Result().(*User).Age)
	assert.Equal(t, "username", res.Result().(*User).Name)
	// test 1
	inv = invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("GetUserTwo"),
		invocation.WithArguments([]interface{}{&User{1, &now, int32(23), "username"}}), invocation.WithReply(user))
	res = invoker.Invoke(context.Background(), inv)
	assert.NoError(t, res.Error())
	assert.NotNil(t, res.Result())
	assert.Equal(t, "username", res.Result().(*User).Name)
	// test 2
	inv = invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("GetUserThree"),
		invocation.WithArguments([]interface{}{&User{1, &now, int32(23), "username"}}), invocation.WithReply(user))
	res = invoker.Invoke(context.Background(), inv)
	assert.NoError(t, res.Error())
	assert.NotNil(t, res.Result())
	assert.Equal(t, "username", res.Result().(*User).Name)
	// test 3
	inv = invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("GetUserFour"),
		invocation.WithArguments([]interface{}{[]User{{1, nil, int32(23), "username"}}}), invocation.WithReply(user))
	res = invoker.Invoke(context.Background(), inv)
	assert.NoError(t, res.Error())
	assert.NotNil(t, res.Result())
	assert.Equal(t, "username", res.Result().(*User).Name)
	// test 4
	inv = invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("GetUserFive"), invocation.WithReply(user))
	res = invoker.Invoke(context.Background(), inv)
	assert.Error(t, res.Error(), "test error")

	assert.Equal(t, filterNum, 12)
	err = common.ServiceMap.UnRegister(url.Service(), url.Protocol, url.ServiceKey())
	assert.NoError(t, err)
}
