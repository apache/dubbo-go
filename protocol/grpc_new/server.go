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

package grpc_new

import (
	"context"
	"crypto/tls"
	connect "dubbo.apache.org/dubbo-go/v3/protocol/grpc_new/connect"
	"fmt"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"net/http"
	"sync"
	"time"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"github.com/dustin/go-humanize"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

// DubboGrpcNewService is gRPC service
type DubboGrpcNewService interface {
	// SetProxyImpl sets proxy.
	SetProxyImpl(impl protocol.Invoker)
	// GetProxyImpl gets proxy.
	GetProxyImpl() protocol.Invoker
	BuildHandler(impl interface{}, opts ...connect.HandlerOption) (string, http.Handler)
}

// Server is a gRPC server
type Server struct {
	httpServer *http.Server
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
		addr    string
		err     error
		hanOpts []connect.HandlerOption
	)
	addr = url.Location
	srv := &http.Server{
		Addr: addr,
	}

	maxServerRecvMsgSize := constant.DefaultMaxServerRecvMsgSize
	if recvMsgSize, convertErr := humanize.ParseBytes(url.GetParam(constant.MaxServerRecvMsgSize, "")); convertErr == nil && recvMsgSize != 0 {
		maxServerRecvMsgSize = int(recvMsgSize)
	}
	hanOpts = append(hanOpts, connect.WithReadMaxBytes(maxServerRecvMsgSize))

	maxServerSendMsgSize := constant.DefaultMaxServerSendMsgSize
	if sendMsgSize, convertErr := humanize.ParseBytes(url.GetParam(constant.MaxServerSendMsgSize, "")); err == convertErr && sendMsgSize != 0 {
		maxServerSendMsgSize = int(sendMsgSize)
	}
	hanOpts = append(hanOpts, connect.WithSendMaxBytes(maxServerSendMsgSize))

	// If global trace instance was set, then server tracer instance
	// can be get. If not, will return NoopTracer.
	//tracer := opentracing.GlobalTracer()
	//serverOpts = append(serverOpts,
	//	grpc.UnaryInterceptor(otgrpc.OpenTracingServerInterceptor(tracer)),
	//	grpc.StreamInterceptor(otgrpc.OpenTracingStreamServerInterceptor(tracer)),
	//	grpc.MaxRecvMsgSize(maxServerRecvMsgSize),
	//	grpc.MaxSendMsgSize(maxServerSendMsgSize),
	//)
	var cfg *tls.Config
	tlsConfig := config.GetRootConfig().TLSConfig
	if tlsConfig != nil {
		cfg, err = config.GetServerTlsConfig(&config.TLSConfig{
			CACertFile:    tlsConfig.CACertFile,
			TLSCertFile:   tlsConfig.TLSCertFile,
			TLSKeyFile:    tlsConfig.TLSKeyFile,
			TLSServerName: tlsConfig.TLSServerName,
		})
		if err != nil {
			return
		}
		logger.Infof("Grpc Server initialized the TLSConfig configuration")
	}
	srv.TLSConfig = cfg

	// todo:// open tracing
	hanOpts = append(hanOpts, connect.WithInterceptors())
	// todo:// move tls config to handleService
	s.httpServer = srv

	go func() {
		providerServices := config.GetProviderConfig().Services

		if len(providerServices) == 0 {
			panic("provider service map is null")
		}
		// wait all exporter ready , then set proxy impl and grpc registerService
		waitGrpcExporter(providerServices)
		mux := http.NewServeMux()
		handleService(providerServices, mux)
		// todo: figure it out this process
		//reflection.Register(server)
		// todo: without tls
		if cfg == nil {
			srv.Handler = h2c.NewHandler(mux, &http2.Server{})
		} else {
			srv.Handler = mux
		}

		if err = srv.ListenAndServe(); err != nil {
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
			mLen := getSyncMapLen(grpcNewProtocol.ExporterMap())
			if pLen == mLen {
				return
			}
		case <-ta.C:
			panic("wait grpc exporter timeout when start grpc server")
		}
	}
}

// todo: //modify comment
// handleService
func handleService(providerServices map[string]*config.ServiceConfig, mux *http.ServeMux, opts ...connect.HandlerOption) {
	for key, providerService := range providerServices {
		service := config.GetProviderService(key)
		ds, ok := service.(DubboGrpcNewService)
		if !ok {
			panic("illegal service type registered")
		}

		serviceKey := common.ServiceKey(providerService.Interface, providerService.Group, providerService.Version)
		exporter, _ := grpcNewProtocol.ExporterMap().Load(serviceKey)
		if exporter == nil {
			panic(fmt.Sprintf("no exporter found for servicekey: %v", serviceKey))
		}
		invoker := exporter.(protocol.Exporter).GetInvoker()
		if invoker == nil {
			panic(fmt.Sprintf("no invoker found for servicekey: %v", serviceKey))
		}

		ds.SetProxyImpl(invoker)
		// todo: merge service specific options and all options
		path, handler := ds.BuildHandler(service, opts...)
		mux.Handle(path, handler)
	}
}

// Stop gRPC server
func (s *Server) Stop() {
	// todo: process error
	s.httpServer.Close()
}

// GracefulStop gRPC server
func (s *Server) GracefulStop() {
	// todo: process error and use timeout
	s.httpServer.Shutdown(context.Background())
}
