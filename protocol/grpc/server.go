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
	"sync"
	"time"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"

	"github.com/opentracing/opentracing-go"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/protocol"
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
	grpcServer *grpc.Server
	bufferSize int
}

// NewServer creates a new server
func NewServer() *Server {
	return &Server{}
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

	// If global trace instance was set, then server tracer instance
	// can be get. If not, will return NoopTracer.
	tracer := opentracing.GlobalTracer()
	server := grpc.NewServer(
		grpc.UnaryInterceptor(otgrpc.OpenTracingServerInterceptor(tracer)),
		grpc.StreamInterceptor(otgrpc.OpenTracingStreamServerInterceptor(tracer)),
		grpc.MaxRecvMsgSize(1024*1024*s.bufferSize),
		grpc.MaxSendMsgSize(1024*1024*s.bufferSize),
	)
	s.grpcServer = server

	go func() {
		providerServices := config.GetProviderConfig().Services

		if len(providerServices) == 0 {
			panic("provider service map is null")
		}
		// wait all exporter ready , then set proxy impl and grpc registerService
		waitGrpcExporter(providerServices)
		registerService(providerServices, server)
		reflection.Register(server)

		if err = server.Serve(lis); err != nil {
			logger.Errorf("server serve failed with err: %v", err)
		}
	}()
}

// getSyncMapLen get sync map len
func getSyncMapLen(m *sync.Map) int {
	length := 0

	m.Range(func(_, _ interface{}) bool {
		length++
		return true
	})
	return length
}

// waitGrpcExporter wait until len(providerServices) = len(ExporterMap)
func waitGrpcExporter(providerServices map[string]*config.ServiceConfig) {
	t := time.NewTicker(50 * time.Millisecond)
	defer t.Stop()
	pLen := len(providerServices)
	ta := time.NewTimer(10 * time.Second)
	defer ta.Stop()

	for {
		select {
		case <-t.C:
			mLen := getSyncMapLen(grpcProtocol.ExporterMap())
			if pLen == mLen {
				return
			}
		case <-ta.C:
			panic("wait grpc exporter timeout when start grpc server")
		}
	}
}

// registerService SetProxyImpl invoker and grpc service
func registerService(providerServices map[string]*config.ServiceConfig, server *grpc.Server) {
	for key, providerService := range providerServices {
		service := config.GetProviderService(key)
		ds, ok := service.(DubboGrpcService)
		if !ok {
			panic("illegal service type registered")
		}

		serviceKey := common.ServiceKey(providerService.Interface, providerService.Group, providerService.Version)
		exporter, _ := grpcProtocol.ExporterMap().Load(serviceKey)
		if exporter == nil {
			panic(fmt.Sprintf("no exporter found for servicekey: %v", serviceKey))
		}
		invoker := exporter.(protocol.Exporter).GetInvoker()
		if invoker == nil {
			panic(fmt.Sprintf("no invoker found for servicekey: %v", serviceKey))
		}

		ds.SetProxyImpl(invoker)
		server.RegisterService(ds.ServiceDesc(), service)
	}
}

// Stop gRPC server
func (s *Server) Stop() {
	s.grpcServer.Stop()
}

// GracefulStop gRPC server
func (s *Server) GracefulStop() {
	s.grpcServer.GracefulStop()
}
