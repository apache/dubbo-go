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

package dubbo

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/proxy/proxy_factory"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/remoting/getty"
)

const (
	mockCommonUrl = "dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?anyhost=true&" +
		"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&" +
		"environment=dev&interface=com.ikurento.user.UserProvider&ip=192.168.56.1&methods=GetUser%2C&" +
		"module=dubbogo+user-info+server&org=ikurento.com&owner=ZX&pid=1447&revision=0.0.1&" +
		"side=provider&timeout=3000&timestamp=1556509797245"
)

func initDubboInvokerTest() {
	getty.SetServerConfig(getty.ServerConfig{
		SessionNumber:  700,
		SessionTimeout: "20s",
		GettySessionParam: getty.GettySessionParam{
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
	getty.SetClientConf(getty.ClientConfig{
		ConnectionNum:   1,
		HeartbeatPeriod: "3s",
		SessionTimeout:  "20s",
		PoolTTL:         600,
		PoolSize:        64,
		GettySessionParam: getty.GettySessionParam{
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
}

func TestDubboProtocol_Export(t *testing.T) {
	initDubboInvokerTest()
	srvCfg := getty.GetDefaultServerConfig()
	getty.SetServerConfig(srvCfg)
	// Export
	proto := GetProtocol()
	url, err := common.NewURL(mockCommonUrl)
	assert.NoError(t, err)
	exporter := proto.Export(protocol.NewBaseInvoker(url))
	// make sure url
	eq := exporter.GetInvoker().GetUrl().URLEqual(url)
	assert.True(t, eq)

	// second service: the same path and the different version
	url2, err := common.NewURL(mockCommonUrl, common.WithParamsValue(constant.VERSION_KEY, "v1.1"))
	assert.NoError(t, err)
	exporter2 := proto.Export(protocol.NewBaseInvoker(url2))
	// make sure url
	eq2 := exporter2.GetInvoker().GetUrl().URLEqual(url2)
	assert.True(t, eq2)

	// make sure exporterMap after 'Unexport'
	_, ok := proto.(*DubboProtocol).ExporterMap().Load(url2.ServiceKey())
	assert.True(t, ok)
	exporter2.Unexport()
	_, ok = proto.(*DubboProtocol).ExporterMap().Load(url2.ServiceKey())
	assert.False(t, ok)

	// make sure serverMap after 'Destroy'
	_, ok = proto.(*DubboProtocol).serverMap[url.Location]
	assert.True(t, ok)
	proto.Destroy()
	_, ok = proto.(*DubboProtocol).serverMap[url.Location]
	assert.False(t, ok)
}

func TestDubboProtocolReferNoConnect(t *testing.T) {
	// Refer
	initDubboInvokerTest()
	proto := GetProtocol()
	url, err := common.NewURL(mockCommonUrl)
	assert.NoError(t, err)
	invoker := proto.Refer(url)
	assert.Nil(t, invoker)
}

func TestDubboProtocol_Refer(t *testing.T) {
	initDubboInvokerTest()
	cliCfg := getty.GetDefaultClientConfig()
	getty.SetClientConf(cliCfg)
	// Refer
	proto := GetProtocol()

	url, err := common.NewURL(mockCommonUrl)
	proto.Export(&proxy_factory.ProxyInvoker{
		BaseInvoker: *protocol.NewBaseInvoker(url),
	})
	assert.NoError(t, err)
	invoker := proto.Refer(url)
	// make sure url
	eq := invoker.GetUrl().URLEqual(url)
	assert.True(t, eq)

	// make sure invokers after 'Destroy'
	invokersLen := len(proto.(*DubboProtocol).Invokers())
	assert.Equal(t, 1, invokersLen)
	proto.Destroy()
	invokersLen = len(proto.(*DubboProtocol).Invokers())
	assert.Equal(t, 0, invokersLen)
}
