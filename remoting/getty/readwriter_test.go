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
	hessian "github.com/apache/dubbo-go-hessian2"
	"github.com/apache/dubbo-go/common"
	. "github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/proxy/proxy_factory"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/protocol/dubbo/impl"
	"github.com/apache/dubbo-go/protocol/invocation"
	"github.com/apache/dubbo-go/remoting"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
	"time"
)

func TestTCPPackageHandle(t *testing.T) {
	svr, url := getServer(t)
	client := getClient(url)
	testDecodeTCPPackage(t, svr, client)
	svr.Stop()
}

func testDecodeTCPPackage(t *testing.T, svr *Server, client *Client) {
	request := remoting.NewRequest("2.0.2")
	up := &UserProvider{}
	invocation := createInvocation("GetUser", nil, nil, []interface{}{[]interface{}{"1", "username"}},
		[]reflect.Value{reflect.ValueOf([]interface{}{"1", "username"}), reflect.ValueOf(up)})
	attachment := map[string]string{INTERFACE_KEY: "com.dubbogo.user.UserProvider",
		PATH_KEY:    "UserProvider",
		VERSION_KEY: "1.0.0",
	}
	setAttachment(invocation, attachment)
	request.Data = invocation
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

func getServer(t *testing.T) (*Server, *common.URL) {

	hessian.RegisterPOJO(&User{})
	remoting.RegistryCodec("dubbo", &DubboTestCodec{})

	methods, err := common.ServiceMap.Register("com.dubbogo.user.UserProvider", "dubbo", "", "", &UserProvider{})
	assert.NoError(t, err)
	assert.Equal(t, "GetBigPkg,GetUser,GetUser0,GetUser1,GetUser2,GetUser3,GetUser4,GetUser5,GetUser6", methods)

	// config
	SetClientConf(ClientConfig{
		ConnectionNum:   2,
		HeartbeatPeriod: "5s",
		SessionTimeout:  "20s",
		PoolTTL:         600,
		PoolSize:        64,
		GettySessionParam: GettySessionParam{
			CompressEncoding: false,
			TcpNoDelay:       true,
			TcpKeepAlive:     true,
			KeepAlivePeriod:  "120s",
			TcpRBufSize:      262144,
			TcpWBufSize:      65536,
			PkgWQSize:        512,
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
			PkgWQSize:        512,
			TcpReadTimeout:   "1s",
			TcpWriteTimeout:  "5s",
			WaitTimeout:      "1s",
			MaxMsgLen:        10240000000,
			SessionName:      "server",
		}})
	assert.NoError(t, srvConf.CheckValidity())

	url, err := common.NewURL("dubbo://127.0.0.1:20061/com.dubbogo.user.UserProvider?anyhost=true&" +
		"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&" +
		"environment=dev&interface=com.ikurento.user.UserProvider&ip=127.0.0.1&methods=GetUser%2C&" +
		"module=dubbogo+user-info+server&org=ikurento.com&owner=ZX&pid=1447&revision=0.0.1&" +
		"side=provider&timeout=3000&timestamp=1556509797245&bean.name=UserProvider")
	assert.NoError(t, err)
	// init server
	userProvider := &UserProvider{}
	_, err = common.ServiceMap.Register("", url.Protocol, "", "0.0.1", userProvider)
	assert.NoError(t, err)
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

	return server, url
}
