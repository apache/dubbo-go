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
	"fmt"
	"net"
	"reflect"
)

import (
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/opentracing/opentracing-go"
	"google.golang.org/grpc"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/protocol"
)

// Server is a gRPC server
type Server struct {
	grpcServer *grpc.Server
	bufferSize int
}

// NewServer creates a new server
func NewServer() *Server {
	return &Server{}
}

// DubboGrpcService is gRPC service
type DubboGrpcService interface {
	// SetProxyImpl sets proxy.
	SetProxyImpl(impl protocol.Invoker)
	// GetProxyImpl gets proxy.
	GetProxyImpl() protocol.Invoker
	// ServiceDesc gets an RPC service's specification.
	ServiceDesc() *grpc.ServiceDesc
}

func (s *Server) SetBufferSize(n int) {
	s.bufferSize = n
}

// Start gRPC server with @url
func (s *Server) Start(url *common.URL) {
	var (
		addr string
		err  error
	)
	addr = url.Location
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}

	// if global trace instance was set, then server tracer instance can be get. If not , will return Nooptracer
	tracer := opentracing.GlobalTracer()
	server := grpc.NewServer(
		grpc.UnaryInterceptor(otgrpc.OpenTracingServerInterceptor(tracer)),
		grpc.MaxRecvMsgSize(1024*1024*s.bufferSize),
		grpc.MaxSendMsgSize(1024*1024*s.bufferSize))

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

// Stop gRPC server
func (s *Server) Stop() {
	s.grpcServer.Stop()
}
