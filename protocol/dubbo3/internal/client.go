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

package internal

import (
	"context"
)

import (
	"github.com/dubbogo/triple/pkg/triple"
)

import (
	"dubbo.apache.org/dubbo-go/v3/config"
)

func init() {
	config.SetConsumerService(&GrpcGreeterImpl{})
}

// GrpcGreeterImpl
//used for dubbo3 biz client
type GrpcGreeterImpl struct {
	SayHello func(ctx context.Context, in *HelloRequest, out *HelloReply) error
}

// Reference ...
func (u *GrpcGreeterImpl) Reference() string {
	return "DubboGreeterImpl"
}

// GetDubboStub ...
func (u *GrpcGreeterImpl) GetDubboStub(cc *triple.TripleConn) GreeterClient {
	return NewGreeterDubbo3Client(cc)
}
