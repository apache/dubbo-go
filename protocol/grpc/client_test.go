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
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/protocol/grpc/internal/helloworld"
	"dubbo.apache.org/dubbo-go/v3/protocol/grpc/internal/routeguide"
)

func TestUnaryClient(t *testing.T) {
	server, err := helloworld.NewServer("127.0.0.1:30000")
	assert.NoError(t, err)
	go server.Start()
	defer server.Stop()

	url, err := common.NewURL(helloworldURL)
	assert.NoError(t, err)

	cli, err := NewClient(url)
	assert.NoError(t, err)

	impl := &helloworld.GreeterClientImpl{}
	client := impl.GetDubboStub(cli.ClientConn)
	result, err := client.SayHello(context.Background(), &helloworld.HelloRequest{Name: "request name"})
	assert.NoError(t, err)
	assert.Equal(t, &helloworld.HelloReply{Message: "Hello request name"}, result)
}

func TestStreamClient(t *testing.T) {
	server, err := routeguide.NewServer("127.0.0.1:30000")
	assert.NoError(t, err)
	go server.Start()
	defer server.Stop()

	url, err := common.NewURL(routeguideURL)
	assert.NoError(t, err)

	cli, err := NewClient(url)
	assert.NoError(t, err)

	impl := &routeguide.RouteGuideClientImpl{}
	client := impl.GetDubboStub(cli.ClientConn)

	result, err := client.GetFeature(context.Background(), &routeguide.Point{Latitude: 409146138, Longitude: -746188906})
	assert.NoError(t, err)
	assert.Equal(t, &routeguide.Feature{
		Name:     "Berkshire Valley Management Area Trail, Jefferson, NJ, USA",
		Location: &routeguide.Point{Latitude: 409146138, Longitude: -746188906},
	}, result)

	listFeaturesStream, err := client.ListFeatures(context.Background(), &routeguide.Rectangle{
		Lo: &routeguide.Point{Latitude: 400000000, Longitude: -750000000},
		Hi: &routeguide.Point{Latitude: 420000000, Longitude: -730000000},
	})
	assert.NoError(t, err)
	routeguide.PrintFeatures(listFeaturesStream)

	recordRouteStream, err := client.RecordRoute(context.Background())
	assert.NoError(t, err)
	routeguide.RunRecordRoute(recordRouteStream)

	routeChatStream, err := client.RouteChat(context.Background())
	assert.NoError(t, err)
	routeguide.RunRouteChat(routeChatStream)
}
