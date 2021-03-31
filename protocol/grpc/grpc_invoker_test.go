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

package grpc

import (
	"context"
	"reflect"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/protocol/grpc/internal"
	"github.com/apache/dubbo-go/protocol/invocation"
)

const (
	mockGrpcCommonUrl = "grpc://127.0.0.1:30000/GrpcGreeterImpl?accesslog=&anyhost=true&app.version=0.0.1&application=BDTService&async=false&bean.name=GrpcGreeterImpl" +
		"&category=providers&cluster=failover&dubbo=dubbo-provider-golang-2.6.0&environment=dev&execute.limit=&execute.limit.rejected.handler=&generic=false&group=&interface=io.grpc.examples.helloworld.GreeterGrpc%24IGreeter" +
		"&ip=192.168.1.106&loadbalance=random&methods.SayHello.loadbalance=random&methods.SayHello.retries=1&methods.SayHello.tps.limit.interval=&methods.SayHello.tps.limit.rate=&methods.SayHello.tps.limit.strategy=" +
		"&methods.SayHello.weight=0&module=dubbogo+say-hello+client&name=BDTService&organization=ikurento.com&owner=ZX&pid=49427&reference.filter=cshutdown&registry.role=3&remote.timestamp=1576923717&retries=" +
		"&service.filter=echo%2Ctoken%2Caccesslog%2Ctps%2Cexecute%2Cpshutdown&side=provider&timestamp=1576923740&tps.limit.interval=&tps.limit.rate=&tps.limit.rejected.handler=&tps.limit.strategy=&tps.limiter=&version=&warmup=100!"
)

func TestInvoke(t *testing.T) {
	go internal.InitGrpcServer()
	defer internal.ShutdownGrpcServer()

	url, err := common.NewURL(mockGrpcCommonUrl)
	assert.Nil(t, err)

	cli, err := NewClient(url)
	if err != nil {
		logger.Errorf("grpc new client error %v", err)
	}
	invoker := NewGrpcInvoker(url, cli)

	args := []reflect.Value{}
	args = append(args, reflect.ValueOf(&internal.HelloRequest{Name: "request name"}))
	bizReply := &internal.HelloReply{}
	invo := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("SayHello"),
		invocation.WithParameterValues(args), invocation.WithReply(bizReply))
	res := invoker.Invoke(context.Background(), invo)
	assert.Nil(t, res.Error())
	assert.NotNil(t, res.Result())
	assert.Equal(t, "Hello request name", bizReply.Message)
}
