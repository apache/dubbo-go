/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package grpc

import (
	"context"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/protocol/grpc/internal"
)

func TestGrpcProtocol_Export(t *testing.T) {
	// Export
	addService()

	proto := GetProtocol()
	url, err := common.NewURL(context.Background(), "grpc://127.0.0.1:40000/GrpcGreeterImpl?accesslog=&app.version=0.0.1&application=BDTService&bean.name=GrpcGreeterImpl&cluster=failover&environment=dev&execute.limit=&execute.limit.rejected.handler=&group=&interface=io.grpc.examples.helloworld.GreeterGrpc%24IGreeter&loadbalance=random&methods.SayHello.loadbalance=random&methods.SayHello.retries=1&methods.SayHello.tps.limit.interval=&methods.SayHello.tps.limit.rate=&methods.SayHello.tps.limit.strategy=&methods.SayHello.weight=0&module=dubbogo+say-hello+client&name=BDTService&organization=ikurento.com&owner=ZX&registry.role=3&retries=&service.filter=echo%2Ctoken%2Caccesslog%2Ctps%2Cexecute%2Cpshutdown&timestamp=1576923717&tps.limit.interval=&tps.limit.rate=&tps.limit.rejected.handler=&tps.limit.strategy=&tps.limiter=&version=&warmup=100")
	assert.NoError(t, err)
	exporter := proto.Export(protocol.NewBaseInvoker(url))
	time.Sleep(time.Second)

	// make sure url
	eq := exporter.GetInvoker().GetUrl().URLEqual(url)
	assert.True(t, eq)

	// make sure exporterMap after 'Unexport'
	_, ok := proto.(*GrpcProtocol).ExporterMap().Load(url.ServiceKey())
	assert.True(t, ok)
	exporter.Unexport()
	_, ok = proto.(*GrpcProtocol).ExporterMap().Load(url.ServiceKey())
	assert.False(t, ok)

	// make sure serverMap after 'Destroy'
	_, ok = proto.(*GrpcProtocol).serverMap[url.Location]
	assert.True(t, ok)
	proto.Destroy()
	_, ok = proto.(*GrpcProtocol).serverMap[url.Location]
	assert.False(t, ok)
}

func TestGrpcProtocol_Refer(t *testing.T) {
	go internal.InitGrpcServer()
	defer internal.ShutdownGrpcServer()
	time.Sleep(time.Second)

	proto := GetProtocol()
	url, err := common.NewURL(context.Background(), "grpc://127.0.0.1:30000/GrpcGreeterImpl?accesslog=&anyhost=true&app.version=0.0.1&application=BDTService&async=false&bean.name=GrpcGreeterImpl&category=providers&cluster=failover&dubbo=dubbo-provider-golang-2.6.0&environment=dev&execute.limit=&execute.limit.rejected.handler=&generic=false&group=&interface=io.grpc.examples.helloworld.GreeterGrpc%24IGreeter&ip=192.168.1.106&loadbalance=random&methods.SayHello.loadbalance=random&methods.SayHello.retries=1&methods.SayHello.tps.limit.interval=&methods.SayHello.tps.limit.rate=&methods.SayHello.tps.limit.strategy=&methods.SayHello.weight=0&module=dubbogo+say-hello+client&name=BDTService&organization=ikurento.com&owner=ZX&pid=49427&reference.filter=cshutdown&registry.role=3&remote.timestamp=1576923717&retries=&service.filter=echo%2Ctoken%2Caccesslog%2Ctps%2Cexecute%2Cpshutdown&side=provider&timestamp=1576923740&tps.limit.interval=&tps.limit.rate=&tps.limit.rejected.handler=&tps.limit.strategy=&tps.limiter=&version=&warmup=100!")
	assert.NoError(t, err)
	invoker := proto.Refer(url)

	// make sure url
	eq := invoker.GetUrl().URLEqual(url)
	assert.True(t, eq)

	// make sure invokers after 'Destroy'
	invokersLen := len(proto.(*GrpcProtocol).Invokers())
	assert.Equal(t, 1, invokersLen)
	proto.Destroy()
	invokersLen = len(proto.(*GrpcProtocol).Invokers())
	assert.Equal(t, 0, invokersLen)
}
