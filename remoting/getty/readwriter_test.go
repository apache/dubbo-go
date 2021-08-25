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
package getty

import (
	"context"
	"reflect"
	"testing"
	"time"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"

	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/proxy/proxy_factory"
	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/protocol/dubbo/impl"
	"github.com/apache/dubbo-go/protocol/invocation"
	"github.com/apache/dubbo-go/remoting"
)

func TestTCPPackageHandle(t *testing.T) {
	initTestEnvironment(t)
	adminUrl := initAdminUrl(t)
	server := getServer(adminUrl)
	client := getClient(adminUrl)

	testDecodeTCPPackage(t, server, client)
	server.Stop()
}

//////////////////////////////////
// before execute getty_test
// 1. init config
// 2. init url
// 3. init server
// 4. init client
//////////////////////////////////

func initTestEnvironment(t *testing.T) {
	SetClientConf(ClientConfig{
		ConnectionNum:   2,
		HeartbeatPeriod: "5s",
		SessionTimeout:  "20s",
		GettySessionParam: GettySessionParam{
			CompressEncoding: false,
			TcpNoDelay:       true,
			TcpKeepAlive:     true,
			KeepAlivePeriod:  "120s",
			TcpRBufSize:      262144,
			TcpWBufSize:      65536,
			TcpReadTimeout:   "4s",
			TcpWriteTimeout:  "5s",
			WaitTimeout:      "1s",
			MaxMsgLen:        10240000000,
			SessionName:      "client",
		},
	})
	assert.NoError(t, clientConf.CheckValidity())
	SetServerConfig(ServerConfig{
		SessionNumber:  700,
		SessionTimeout: "20s",
		GettySessionParam: GettySessionParam{
			CompressEncoding: false,
			TcpNoDelay:       true,
			TcpKeepAlive:     true,
			KeepAlivePeriod:  "120s",
			TcpRBufSize:      262144,
			TcpWBufSize:      65536,
			TcpReadTimeout:   "1s",
			TcpWriteTimeout:  "5s",
			WaitTimeout:      "1s",
			MaxMsgLen:        10240000000,
			SessionName:      "server",
		}})
	assert.NoError(t, srvConf.CheckValidity())
}

func initAdminUrl(t *testing.T) *common.URL {
	hessian.RegisterPOJO(&User{})
	remoting.RegistryCodec("dubbo", &DubboTestCodec{})

	url, err := common.NewURL("dubbo://127.0.0.1:20061/?anyhost=true&" +
		"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&" +
		"environment=dev&interface=com.ikurento.user.AdminProvider&ip=127.0.0.1&methods=GetAdmin%2C&" +
		"module=dubbogo+user-info+server&org=ikurento.com&owner=ZX&pid=1447&revision=0.0.1&" +
		"side=provider&timeout=3000&timestamp=1556509797245&bean.name=AdminProvider")
	assert.NoError(t, err)

	methods, err := common.ServiceMap.Register("com.ikurento.user.AdminProvider", url.Protocol, "", "", &AdminProvider{})
	assert.NoError(t, err)
	assert.Equal(t, "GetAdmin", methods)

	return url
}

func getServer(url *common.URL) *Server {
	invoker := &proxy_factory.ProxyInvoker{
		BaseInvoker: *protocol.NewBaseInvoker(url),
	}
	handler := func(invocation *invocation.RPCInvocation) protocol.RPCResult {
		//result := protocol.RPCResult{}
		r := invoker.Invoke(context.Background(), invocation)
		result := protocol.RPCResult{
			Err:   r.Error(),
			Rest:  r.Result(),
			Attrs: r.Attachments(),
		}
		return result
	}
	server := NewServer(url, handler)
	server.Start()

	time.Sleep(time.Second * 2)

	return server
}

func getClient(url *common.URL) *Client {
	client := NewClient(Options{
		ConnectTimeout: config.GetConsumerConfig().ConnectTimeout,
	})

	if err := client.Connect(url); err != nil {
		return nil
	}
	return client
}

//////////////////////////////////
// test util
//////////////////////////////////

func createInvocation(methodName string, callback interface{}, reply interface{}, arguments []interface{},
	parameterValues []reflect.Value) *invocation.RPCInvocation {
	return invocation.NewRPCInvocationWithOptions(invocation.WithMethodName(methodName),
		invocation.WithArguments(arguments), invocation.WithReply(reply),
		invocation.WithCallBack(callback), invocation.WithParameterValues(parameterValues))
}

func setAttachment(invocation *invocation.RPCInvocation, attachments map[string]string) {
	for key, value := range attachments {
		invocation.SetAttachments(key, value)
	}
}

//////////////////////////////////
// test cases
//////////////////////////////////

func testDecodeTCPPackage(t *testing.T, svr *Server, client *Client) {
	request := remoting.NewRequest("2.0.2")
	ap := &AdminProvider{}
	rpcInvocation := createInvocation("GetAdmin", nil, nil, []interface{}{[]interface{}{"1", "username"}},
		[]reflect.Value{reflect.ValueOf([]interface{}{"1", "username"}), reflect.ValueOf(ap)})
	attachment := map[string]string{
		constant.INTERFACE_KEY: "com.ikurento.user.AdminProvider",
		constant.PATH_KEY:      "AdminProvider",
		constant.VERSION_KEY:   "1.0.0",
	}
	setAttachment(rpcInvocation, attachment)
	request.Data = rpcInvocation
	request.Event = false
	request.TwoWay = false

	pkgWriteHandler := NewRpcClientPackageHandler(client)
	pkgBytes, err := pkgWriteHandler.Write(nil, request)
	assert.NoError(t, err)
	pkgReadHandler := NewRpcServerPackageHandler(svr)
	_, pkgLen, err := pkgReadHandler.Read(nil, pkgBytes)
	assert.NoError(t, err)
	assert.Equal(t, pkgLen, len(pkgBytes))

	// simulate incomplete tcp package
	incompletePkgLen := len(pkgBytes) - 10
	assert.True(t, incompletePkgLen >= impl.HEADER_LENGTH, "header buffer too short")
	incompletePkg := pkgBytes[0 : incompletePkgLen-1]
	pkg, pkgLen, err := pkgReadHandler.Read(nil, incompletePkg)
	assert.NoError(t, err)
	assert.Equal(t, pkg, nil)
	assert.Equal(t, pkgLen, 0)
}

//////////////////////////////////
// provider
//////////////////////////////////

type AdminProvider struct {
}

func (a *AdminProvider) GetAdmin(ctx context.Context, req []interface{}, rsp *User) error {
	rsp.Id = req[0].(string)
	rsp.Name = req[1].(string)
	return nil
}

func (a *AdminProvider) Reference() string {
	return "AdminProvider"
}
