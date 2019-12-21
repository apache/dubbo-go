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
	//"context"
	"reflect"
	"testing"

	"github.com/bmizerany/assert"
	"google.golang.org/grpc"

	"github.com/apache/dubbo-go/protocol/grpc/internal"
)

//type GrpcGreeterImpl struct {
//	SayHello func(ctx context.Context, in *internal.HelloRequest, out *internal.HelloReply) error
//}
//
//func (u *GrpcGreeterImpl) Reference() string {
//	return "GrpcGreeterImpl"
//}
//
//func (u *GrpcGreeterImpl) GetDubboStub(cc *grpc.ClientConn) internal.GreeterClient {
//	return internal.NewGreeterClient(cc)
//}

func TestGetInvoker(t *testing.T) {
	var conn *grpc.ClientConn
	var impl *internal.GrpcGreeterImpl
	invoker := getInvoker(impl, conn)

	i := reflect.TypeOf(invoker)
	expected := reflect.TypeOf(internal.NewGreeterClient(nil))
	assert.Equal(t, i, expected)
}
