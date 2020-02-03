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
	"fmt"
	"net"
	"reflect"
)

import (
	"google.golang.org/grpc"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/protocol"
)

// Server ...
type Server struct {
	grpcServer *grpc.Server
}

// NewServer ...
func NewServer() *Server {
	return &Server{}
}

// DubboGrpcService ...
type DubboGrpcService interface {
	SetProxyImpl(impl protocol.Invoker)
	GetProxyImpl() protocol.Invoker
	ServiceDesc() *grpc.ServiceDesc
}

// Start ...
func (s *Server) Start(url common.URL) {
	var (
		addr string
		err  error
	)
	addr = url.Location
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}
	server := grpc.NewServer()

	key := url.GetParam(constant.BEAN_NAME_KEY, "")
	service := config.GetProviderService(key)

	ds, ok := service.(DubboGrpcService)
	if !ok {
		panic("illegal service type registered")
	}

	m, ok := reflect.TypeOf(service).MethodByName("SetProxyImpl")
	if !ok {
		panic("method SetProxyImpl is necessary for grpc service")
	}

	exporter, _ := grpcProtocol.ExporterMap().Load(url.ServiceKey())
	if exporter == nil {
		panic(fmt.Sprintf("no exporter found for servicekey: %v", url.ServiceKey()))
	}
	invoker := exporter.(protocol.Exporter).GetInvoker()
	if invoker == nil {
		panic(fmt.Sprintf("no invoker found for servicekey: %v", url.ServiceKey()))
	}
	in := []reflect.Value{reflect.ValueOf(service)}
	in = append(in, reflect.ValueOf(invoker))
	m.Func.Call(in)

	server.RegisterService(ds.ServiceDesc(), service)

	s.grpcServer = server
	go func() {
		if err = server.Serve(lis); err != nil {
			logger.Errorf("server serve failed with err: %v", err)
		}
	}()
}

// Stop ...
func (s *Server) Stop() {
	s.grpcServer.Stop()
}
