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
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/protocol/grpc/internal/helloworld"
	"dubbo.apache.org/dubbo-go/v3/protocol/grpc/internal/routeguide"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

const (
	helloworldURL = "grpc://127.0.0.1:30000/GrpcGreeterImpl?accesslog=&anyhost=true&app.version=0.0.1&application=BDTService&async=false&bean.name=GrpcGreeterImpl" +
		"&category=providers&cluster=failover&dubbo=dubbo-provider-golang-2.6.0&environment=dev&execute.limit=&execute.limit.rejected.handler=&generic=false&group=&interface=io.grpc.examples.helloworld.GreeterGrpc%24IGreeter" +
		"&ip=192.168.1.106&loadbalance=random&methods.SayHello.loadbalance=random&methods.SayHello.retries=1&methods.SayHello.tps.limit.interval=&methods.SayHello.tps.limit.rate=&methods.SayHello.tps.limit.strategy=" +
		"&methods.SayHello.weight=0&module=dubbogo+say-hello+client&name=BDTService&organization=ikurento.com&owner=ZX&pid=49427&reference.filter=cshutdown&registry.role=3&remote.timestamp=1576923717&retries=" +
		"&service.filter=echo%2Ctoken%2Caccesslog%2Ctps%2Cexecute%2Cpshutdown&side=provider&timestamp=1576923740&tps.limit.interval=&tps.limit.rate=&tps.limit.rejected.handler=&tps.limit.strategy=&tps.limiter=&version=&warmup=100!"

	routeguideURL = "grpc://127.0.0.1:30000/routeGuideImpl?accesslog=&anyhost=true&app.version=0.0.1&application=BDTService&async=false&bean.name=routeGuideImpl" +
		"&category=providers&cluster=failover&dubbo=dubbo-provider-golang-2.6.0&environment=dev&execute.limit=&execute.limit.rejected.handler=&generic=false&group=&interface=io.grpc.examples.helloworld.GreeterGrpc%24IGreeter" +
		"&ip=192.168.1.106&loadbalance=random&methods.SayHello.loadbalance=random&methods.SayHello.retries=1&methods.SayHello.tps.limit.interval=&methods.SayHello.tps.limit.rate=&methods.SayHello.tps.limit.strategy=" +
		"&methods.SayHello.weight=0&module=dubbogo+say-hello+client&name=BDTService&organization=ikurento.com&owner=ZX&pid=49427&reference.filter=cshutdown&registry.role=3&remote.timestamp=1576923717&retries=" +
		"&service.filter=echo%2Ctoken%2Caccesslog%2Ctps%2Cexecute%2Cpshutdown&side=provider&timestamp=1576923740&tps.limit.interval=&tps.limit.rate=&tps.limit.rejected.handler=&tps.limit.strategy=&tps.limiter=&version=&warmup=100!"
)

func TestUnaryInvoke(t *testing.T) {
	server, err := helloworld.NewServer("127.0.0.1:30000")
	assert.NoError(t, err)
	go server.Start()
	defer server.Stop()

	url, err := common.NewURL(helloworldURL)
	assert.NoError(t, err)

	cli, err := NewClient(url)
	assert.NoError(t, err)

	var args []reflect.Value
	args = append(args, reflect.ValueOf(&helloworld.HelloRequest{Name: "request name"}))
	bizReply := &helloworld.HelloReply{}
	invo := invocation.NewRPCInvocationWithOptions(
		invocation.WithMethodName("SayHello"),
		invocation.WithParameterValues(args),
		invocation.WithReply(bizReply),
	)

	invoker := NewGrpcInvoker(url, cli)
	res := invoker.Invoke(context.Background(), invo)
	assert.NoError(t, res.Error())
	assert.Equal(t, &helloworld.HelloReply{Message: "Hello request name"}, res.Result().(reflect.Value).Interface())
	assert.Equal(t, &helloworld.HelloReply{Message: "Hello request name"}, bizReply)
}

func TestStreamInvoke(t *testing.T) {
	server, err := routeguide.NewServer("127.0.0.1:30000")
	assert.NoError(t, err)
	go server.Start()
	defer server.Stop()

	url, err := common.NewURL(routeguideURL)
	assert.NoError(t, err)

	cli, err := NewClient(url)
	assert.NoError(t, err)

	invoker := NewGrpcInvoker(url, cli)

	args := []reflect.Value{
		reflect.ValueOf(&routeguide.Point{Latitude: 409146138, Longitude: -746188906}),
	}
	bizReply := &routeguide.Feature{}
	invo := invocation.NewRPCInvocationWithOptions(
		invocation.WithMethodName("GetFeature"),
		invocation.WithParameterValues(args),
		invocation.WithReply(bizReply),
	)
	res := invoker.Invoke(context.Background(), invo)
	assert.NoError(t, res.Error())
	assert.Equal(t, &routeguide.Feature{
		Name:     "Berkshire Valley Management Area Trail, Jefferson, NJ, USA",
		Location: &routeguide.Point{Latitude: 409146138, Longitude: -746188906},
	}, res.Result().(reflect.Value).Interface())
	assert.Equal(t, &routeguide.Feature{
		Name:     "Berkshire Valley Management Area Trail, Jefferson, NJ, USA",
		Location: &routeguide.Point{Latitude: 409146138, Longitude: -746188906},
	}, bizReply)

	args = []reflect.Value{
		reflect.ValueOf(&routeguide.Rectangle{
			Lo: &routeguide.Point{Latitude: 400000000, Longitude: -750000000},
			Hi: &routeguide.Point{Latitude: 420000000, Longitude: -730000000},
		}),
	}
	invo = invocation.NewRPCInvocationWithOptions(
		invocation.WithMethodName("ListFeatures"),
		invocation.WithParameterValues(args),
	)
	res = invoker.Invoke(context.Background(), invo)
	assert.ErrorIs(t, res.Error(), errNoReply)
	listFeaturesStream := res.Result().(reflect.Value).Interface().(routeguide.RouteGuide_ListFeaturesClient)
	routeguide.PrintFeatures(listFeaturesStream)

	args = make([]reflect.Value, 0)
	invo = invocation.NewRPCInvocationWithOptions(
		invocation.WithMethodName("RecordRoute"),
		invocation.WithParameterValues(args),
	)
	res = invoker.Invoke(context.Background(), invo)
	assert.ErrorIs(t, res.Error(), errNoReply)
	recordRouteStream := res.Result().(reflect.Value).Interface().(routeguide.RouteGuide_RecordRouteClient)
	routeguide.RunRecordRoute(recordRouteStream)

	args = make([]reflect.Value, 0)
	invo = invocation.NewRPCInvocationWithOptions(
		invocation.WithMethodName("RouteChat"),
		invocation.WithParameterValues(args),
	)
	res = invoker.Invoke(context.Background(), invo)
	assert.ErrorIs(t, res.Error(), errNoReply)
	routeChatStream := res.Result().(reflect.Value).Interface().(routeguide.RouteGuide_RouteChatClient)
	routeguide.RunRouteChat(routeChatStream)
}
