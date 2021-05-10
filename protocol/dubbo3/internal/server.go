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
	"log"
)

import (
	_ "dubbo.apache.org/dubbo-go/v3/common/proxy/proxy_factory"
	"dubbo.apache.org/dubbo-go/v3/config"
	_ "dubbo.apache.org/dubbo-go/v3/filter/filter_impl"
)

// server is used to implement helloworld.GreeterServer.
type Server struct {
	*GreeterProviderBase
}

// SayHello implements helloworld.GreeterServer
func (s *Server) SayHello(ctx context.Context, in *HelloRequest) (*HelloReply, error) {
	log.Printf("Received: %v", in.GetName())
	return &HelloReply{Message: "Hello " + in.GetName()}, nil
}

func (s *Server) Reference() string {
	return "DubboGreeterImpl"
}

// InitDubboServer creates global gRPC server.
func InitDubboServer() {
	providerConfig := config.NewProviderConfig(
		config.WithProviderAppConfig(config.NewDefaultApplicationConfig()),
		config.WithProviderProtocol("tri", "tri", "20003"), // protocol and port
		config.WithProviderServices("DubboGreeterImpl", config.NewServiceConfigByAPI(
			config.WithServiceProtocol("tri"),                                // export protocol
			config.WithServiceInterface("org.apache.dubbo.DubboGreeterImpl"), // interface id
			config.WithServiceLoadBalance("random"),                          // lb
			config.WithServiceWarmUpTime("100"),
			config.WithServiceCluster("failover"),
		)),
	)
	config.SetProviderConfig(*providerConfig) // set to providerConfig ptr

	config.SetProviderService(&Server{
		GreeterProviderBase: &GreeterProviderBase{},
	})
	config.Load()
}
