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
	"google.golang.org/grpc"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/protocol/grpc/internal"
)

func TestGetInvoker(t *testing.T) {
	var conn *grpc.ClientConn
	var impl *internal.GrpcGreeterImpl
	invoker := getInvoker(impl, conn)

	i := reflect.TypeOf(invoker)
	expected := reflect.TypeOf(internal.NewGreeterClient(nil))
	assert.Equal(t, i, expected)
}

func TestNewClient(t *testing.T) {
	server, err := internal.NewServer("127.0.0.1:30000")
	assert.NoError(t, err)
	go server.Start()
	defer server.Stop()

	url, err := common.NewURL(mockGrpcCommonUrl)
	assert.NoError(t, err)

	cli, err := NewClient(url)
	assert.NoError(t, err)

	impl := &internal.GreeterClientImpl{}
	client := impl.GetDubboStub(cli.ClientConn)
	result, err := client.SayHello(context.Background(), &internal.HelloRequest{Name: "request name"})
	assert.NoError(t, err)
	assert.Equal(t, &internal.HelloReply{Message: "Hello request name"}, result)
}
