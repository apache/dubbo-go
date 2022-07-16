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
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/dubbo/impl"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/proxy/proxy_factory"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

func TestTCPPackageHandle(t *testing.T) {
	svr, url := getServer(t)
	client := getClient(url)
	testDecodeTCPPackage(t, svr, client)
	svr.Stop()
}

func testDecodeTCPPackage(t *testing.T, svr *Server, client *Client) {
	request := remoting.NewRequest("2.0.2")
	ap := &AdminProvider{}
	rpcInvocation := createInvocation("GetAdmin", nil, nil, []interface{}{[]interface{}{"1", "username"}},
		[]reflect.Value{reflect.ValueOf([]interface{}{"1", "username"}), reflect.ValueOf(ap)})
	attachment := map[string]string{
		constant.InterfaceKey: "com.ikurento.user.AdminProvider",
		constant.PathKey:      "AdminProvider",
		constant.VersionKey:   "1.0.0",
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
	assert.Equal(t, err.Error(), "body buffer too short")
	assert.Equal(t, pkg.(*remoting.DecodeResult).Result, nil)
	assert.Equal(t, pkgLen, 0)
}

func getServer(t *testing.T) (*Server, *common.URL) {
	hessian.RegisterPOJO(&User{})
	remoting.RegistryCodec("dubbo", &DubboTestCodec{})

	methods, err := common.ServiceMap.Register("com.ikurento.user.AdminProvider", "dubbo", "", "", &AdminProvider{})
	assert.NoError(t, err)
	assert.Equal(t, "GetAdmin", methods)

	// config
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
		},
	})
	assert.NoError(t, srvConf.CheckValidity())

	url, err := common.NewURL("dubbo://127.0.0.1:20061/com.ikurento.user.AdminProvider?anyhost=true&" +
		"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&" +
		"environment=dev&interface=com.ikurento.user.AdminProvider&ip=127.0.0.1&methods=GetAdmin%2C&" +
		"module=dubbogo+user-info+server&org=ikurento.com&owner=ZX&pid=1447&revision=0.0.1&" +
		"side=provider&timeout=3000&timestamp=1556509797245&bean.name=AdminProvider")
	assert.NoError(t, err)
	// init server
	adminProvider := &AdminProvider{}
	_, err = common.ServiceMap.Register("com.ikurento.user.AdminProvider", url.Protocol, "", "0.0.1", adminProvider)
	assert.NoError(t, err)
	invoker := &proxy_factory.ProxyInvoker{
		BaseInvoker: *protocol.NewBaseInvoker(url),
	}
	handler := func(invocation *invocation.RPCInvocation) protocol.RPCResult {
		// result := protocol.RPCResult{}
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

type AdminProvider struct{}

func (a *AdminProvider) GetAdmin(ctx context.Context, req []interface{}, rsp *User) error {
	rsp.ID = req[0].(string)
	rsp.Name = req[1].(string)
	return nil
}

func (a *AdminProvider) Reference() string {
	return "AdminProvider"
}
