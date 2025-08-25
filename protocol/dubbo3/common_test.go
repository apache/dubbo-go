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

package dubbo3

import (
	"context"
	"fmt"
)

import (
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/protocol/dubbo3/internal"
)

// TODO: After the config is removed, remove the test
// userd dubbo3 biz service
func addService() {
	config.SetProviderService(newGreeterProvider())
}

type greeterProvider struct {
	internal.UnimplementedGreeterServer
}

func newGreeterProvider() *greeterProvider {
	return &greeterProvider{}
}

func (g *greeterProvider) SayHello(ctx context.Context, req *internal.HelloRequest) (reply *internal.HelloReply, err error) {
	fmt.Printf("req: %v", req)
	return &internal.HelloReply{Message: "this is message from reply"}, nil
}

// removed unused helper handler
