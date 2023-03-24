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
	log "github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/config"
	_ "dubbo.apache.org/dubbo-go/v3/filter/filter_impl"
	_ "dubbo.apache.org/dubbo-go/v3/metrics/prometheus"
	_ "dubbo.apache.org/dubbo-go/v3/proxy/proxy_factory"
)

// server is used to implement helloworld.GreeterServer.
type Server struct {
	UnimplementedGreeterServer
}

// SayHello implements helloworld.GreeterServer
func (s *Server) SayHello(ctx context.Context, in *HelloRequest) (*HelloReply, error) {
	log.Infof("Received: %v", in.GetName())
	return &HelloReply{Message: "Hello " + in.GetName()}, nil
}

// InitDubboServer creates global gRPC server.
func InitDubboServer() {
	serviceConfig := config.NewServiceConfigBuilder().
		SetInterface("org.apache.dubbo.DubboGreeterImpl").
		SetProtocolIDs("tripleKey").Build()

	providerConfig := config.NewProviderConfigBuilder().SetServices(map[string]*config.ServiceConfig{
		common.GetReference(&Server{}): serviceConfig,
	}).Build()

	protocolConfig := config.NewProtocolConfigBuilder().SetName("tri").SetPort("20003").Build()

	rootConfig := config.NewRootConfigBuilder().SetProvider(providerConfig).SetProtocols(map[string]*config.ProtocolConfig{
		"tripleKey": protocolConfig,
	}).Build()

	config.SetProviderService(&Server{})
	if err := rootConfig.Init(); err != nil {
		panic(err)
	}
	rootConfig.Start()
}
