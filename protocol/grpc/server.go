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
	"strconv"
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

// DubboGrpcService is gRPC service
type DubboGrpcService interface {
	// SetProxyImpl sets proxy.
	SetProxyImpl(impl protocol.Invoker)
	// GetProxyImpl gets proxy.
	GetProxyImpl() protocol.Invoker
	// ServiceDesc gets an RPC service's specification.
	ServiceDesc() *grpc.ServiceDesc
}

// Server is a gRPC server
type Server struct {
	listener   net.Listener
	grpcServer *grpc.Server
}

// NewServer creates a new server
func NewServer(url *common.URL) *Server {
	var (
		err error
	)

	listener, err := net.Listen("tcp", url.Location)
	if err != nil {
		panic(err)
	}

	// If global trace instance was set, then server tracer
	// instance can be get. If not, will return NoopTracer.
	tracer := opentracing.GlobalTracer()
	grpcMessageSize, _ := strconv.Atoi(url.GetParam(constant.MESSAGE_SIZE_KEY, "4"))
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(otgrpc.OpenTracingServerInterceptor(tracer)),
		grpc.StreamInterceptor(otgrpc.OpenTracingStreamServerInterceptor(tracer)),
		grpc.MaxRecvMsgSize(1024*1024*grpcMessageSize),
		grpc.MaxSendMsgSize(1024*1024*grpcMessageSize),
	)

	server := Server{
		listener:   listener,
		grpcServer: grpcServer,
	}

	return &server
}

// Start gRPC server with @url
func (s *Server) Start(url *common.URL) {
	var (
		err  error
	)

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

	s.grpcServer.RegisterService(ds.ServiceDesc(), service)

	go func() {
		if err = s.grpcServer.Serve(s.listener); err != nil {
			logger.Errorf("server serve failed with err: %v", err)
		}
	}()
}

// Stop gRPC server
func (s *Server) Stop() {
	s.grpcServer.Stop()
}
