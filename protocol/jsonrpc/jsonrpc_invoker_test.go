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

package jsonrpc

//
//import (
//	"context"
//	"testing"
//	"time"
//)
//
//import (
//	"github.com/stretchr/testify/assert"
//)
//
//import (
//	"dubbo.apache.org/dubbo-go/v3/common"
//	"dubbo.apache.org/dubbo-go/v3/proxy/proxy_factory"
//	"dubbo.apache.org/dubbo-go/v3/protocol"
//	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
//)
//
//func TestJsonrpcInvokerInvoke(t *testing.T) {
//	methods, err := common.ServiceMap.Register("com.ikurento.user.UserProvider", "jsonrpc", "", "", &UserProvider{})
//	assert.NoError(t, err)
//	assert.Equal(t, "GetUser,GetUser0,GetUser1,GetUser2,GetUser3,GetUser4", methods)
//
//	// Export
//	proto := GetProtocol()
//	url, err := common.NewURL("jsonrpc://127.0.0.1:20001/com.ikurento.user.UserProvider?anyhost=true&" +
//		"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&" +
//		"environment=dev&interface=com.ikurento.user.UserProvider&ip=192.168.56.1&methods=GetUser%2C&" +
//		"module=dubbogo+user-info+server&org=ikurento.com&owner=ZX&pid=1447&revision=0.0.1&" +
//		"side=provider&timeout=3000&timestamp=1556509797245&bean.name=UserProvider")
//	assert.NoError(t, err)
//	proto.Export(&proxy_factory.ProxyInvoker{
//		BaseInvoker: *protocol.NewBaseInvoker(url),
//	})
//	time.Sleep(time.Second * 2)
//
//	client := NewHTTPClient(&HTTPOptions{
//		HandshakeTimeout: time.Second,
//		HTTPTimeout:      time.Second,
//	})
//
//	jsonInvoker := NewJsonrpcInvoker(url, client)
//	user := &User{}
//	res := jsonInvoker.Invoke(context.Background(), invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("GetUser"), invocation.WithArguments([]interface{}{"1", "username"}),
//		invocation.WithReply(user)))
//
//	assert.NoError(t, res.Error())
//	assert.Equal(t, User{ID: "1", Name: "username"}, *res.Result().(*User))
//
//	// destroy
//	proto.Destroy()
//}
