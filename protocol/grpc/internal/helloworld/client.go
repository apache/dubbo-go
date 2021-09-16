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

package helloworld

import (
	"context"
)

import (
	"google.golang.org/grpc"
)

import (
	"dubbo.apache.org/dubbo-go/v3/config"
)

func init() {
	config.SetConsumerServiceByInterfaceName("io.grpc.examples.helloworld.GreeterGrpc$IGreeter", &GrpcGreeterImpl{})
	config.SetConsumerService(&GrpcGreeterImpl{})
}

// GrpcGreeterImpl
// used for dubbo-grpc biz client
type GrpcGreeterImpl struct {
	SayHello func(ctx context.Context, in *HelloRequest, out *HelloReply) error
}

// Reference ...
func (u *GrpcGreeterImpl) Reference() string {
	return "GrpcGreeterImpl"
}

// GetDubboStub ...
func (u *GrpcGreeterImpl) GetDubboStub(cc *grpc.ClientConn) GreeterClient {
	return NewGreeterClient(cc)
}
