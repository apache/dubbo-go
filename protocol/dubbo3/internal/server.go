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
	"dubbo.apache.org/dubbo-go/v3/common"
	_ "dubbo.apache.org/dubbo-go/v3/common/proxy/proxy_factory"
	"dubbo.apache.org/dubbo-go/v3/config"
	_ "dubbo.apache.org/dubbo-go/v3/filter/filter_impl"
	_ "dubbo.apache.org/dubbo-go/v3/metrics/prometheus"
)

// server is used to implement helloworld.GreeterServer.
type Server struct {
	GreeterProviderBase
}

// SayHello implements helloworld.GreeterServer
func (s *Server) SayHello(ctx context.Context, in *HelloRequest) (*HelloReply, error) {
	log.Printf("Received: %v", in.GetName())
	return &HelloReply{Message: "Hello " + in.GetName()}, nil
}

// InitDubboServer creates global gRPC server.
func InitDubboServer() {
	serviceConfig := config.NewServiceConfig(
		config.WithServiceInterface("org.apache.dubbo.DubboGreeterImpl"),
		config.WithServiceProtocolKeys("tripleKey"),
	)

	providerConfig := config.GetProviderInstance(
		config.WithProviderService(common.GetReference(&Server{}), serviceConfig),
	)

	protocolConfig := config.NewProtocolConfig(
		config.WithProtocolName("tri"),
		config.WithProtocolPort("20003"),
	)

	rootConfig := config.GetInstance(
		config.WithRootProviderConfig(providerConfig),
		config.WithRootProtocolConfig("tripleKey", protocolConfig),
	)

	config.SetProviderService(&Server{})
	rootConfig.Init()
	rootConfig.Start()
}
