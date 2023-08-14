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

package triple

import (
	"context"
	"crypto/tls"
	"net/http"
	"path"
	"sync"
	"time"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"github.com/dustin/go-humanize"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	tri "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
	"dubbo.apache.org/dubbo-go/v3/server"
)

// Server is TRIPLE server
type Server struct {
	httpServer *http.Server
}

// NewServer creates a new TRIPLE server
func NewServer() *Server {
	return &Server{}
}

// Start TRIPLE server
func (s *Server) Start(invoker protocol.Invoker, info *server.ServiceInfo) {
	var (
		addr    string
		err     error
		url     *common.URL
		hanOpts []tri.HandlerOption
	)
	url = invoker.GetURL()
	addr = url.Location
	srv := &http.Server{
		Addr: addr,
	}

	maxServerRecvMsgSize := constant.DefaultMaxServerRecvMsgSize
	if recvMsgSize, convertErr := humanize.ParseBytes(url.GetParam(constant.MaxServerRecvMsgSize, "")); convertErr == nil && recvMsgSize != 0 {
		maxServerRecvMsgSize = int(recvMsgSize)
	}
	hanOpts = append(hanOpts, tri.WithReadMaxBytes(maxServerRecvMsgSize))

	maxServerSendMsgSize := constant.DefaultMaxServerSendMsgSize
	if sendMsgSize, convertErr := humanize.ParseBytes(url.GetParam(constant.MaxServerSendMsgSize, "")); err == convertErr && sendMsgSize != 0 {
		maxServerSendMsgSize = int(sendMsgSize)
	}
	hanOpts = append(hanOpts, tri.WithSendMaxBytes(maxServerSendMsgSize))

	// todo: implement interceptor
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
		logger.Infof("Triple Server initialized the TLSConfig configuration")
	}
	srv.TLSConfig = cfg

	// todo:// open tracing
	hanOpts = append(hanOpts, tri.WithInterceptors())
	// todo:// move tls config to handleService
	s.httpServer = srv

	go func() {
		//providerServices := config.GetProviderConfig().Services
		//
		//if len(providerServices) == 0 {
		//	panic("provider service map is null")
		//}
		// todo: remove this logic?
		// wait all exporter ready , then set proxy impl and grpc registerService
		//waitTripleExporter(providerServices)
		mux := http.NewServeMux()
		if info != nil {
			handleServiceWithInfo(invoker, info, mux)
		}
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

// getSyncMapLen gets sync map len
func getSyncMapLen(m *sync.Map) int {
	length := 0

	m.Range(func(_, _ interface{}) bool {
		length++
		return true
	})
	return length
}

// waitTripleExporter wait until len(providerServices) = len(ExporterMap)
func waitTripleExporter(providerServices map[string]*config.ServiceConfig) {
	t := time.NewTicker(50 * time.Millisecond)
	defer t.Stop()
	pLen := len(providerServices)
	ta := time.NewTimer(10 * time.Second)
	defer ta.Stop()

	for {
		select {
		case <-t.C:
			mLen := getSyncMapLen(tripleProtocol.ExporterMap())
			if pLen == mLen {
				return
			}
		case <-ta.C:
			panic("wait Triple exporter timeout when start GRPC_NEW server")
		}
	}
}

//// handleService injects invoker and creates handler based on ServiceConfig and provider service.
//func handleService(providerServices map[string]*config.ServiceConfig, mux *http.ServeMux, opts ...tri.HandlerOption) {
//	for key, providerService := range providerServices {
//		service := config.GetProviderService(key)
//		ds, ok := service.(TripleService)
//		if !ok {
//			panic("illegal service type registered")
//		}
//
//		serviceKey := common.ServiceKey(providerService.Interface, providerService.Group, providerService.Version)
//		exporter, _ := tripleProtocol.ExporterMap().Load(serviceKey)
//		if exporter == nil {
//			panic(fmt.Sprintf("no exporter found for servicekey: %v", serviceKey))
//		}
//		invoker := exporter.(protocol.Exporter).GetInvoker()
//		if invoker == nil {
//			panic(fmt.Sprintf("no invoker found for servicekey: %v", serviceKey))
//		}
//
//		// inject invoker, it has all invocation logics
//		ds.SetProxyImpl(invoker)
//		path, handler := ds.BuildHandler(service, opts...)
//		mux.Handle(path, tri.New)
//		mux.Handle(path, handler)
//	}
//}

// handleServiceWithInfo injects invoker and create handler based on ServiceInfo
func handleServiceWithInfo(invoker protocol.Invoker, info *server.ServiceInfo, mux *http.ServeMux, opts ...tri.HandlerOption) {
	for _, method := range info.Methods {
		var handler http.Handler
		procedure := path.Join(info.InterfaceName, method.Name)
		switch method.Type {
		case constant.CallUnary:
			handler = tri.NewUnaryHandler(
				procedure,
				method.ReqInitFunc,
				func(ctx context.Context, req *tri.Request) (*tri.Response, error) {
					var args []interface{}
					args = append(args, req.Msg)
					// todo: inject method.Meta to attachments
					invo := invocation.NewRPCInvocation(method.Name, args, nil)
					res := invoker.Invoke(ctx, invo)
					return res.Result().(*tri.Response), res.Error()
				},
				opts...,
			)
		case constant.CallClientStream:
			handler = tri.NewClientStreamHandler(
				procedure,
				func(ctx context.Context, stream *tri.ClientStream) (*tri.Response, error) {
					var args []interface{}
					args = append(args, method.StreamInitFunc(stream))
					invo := invocation.NewRPCInvocation(method.Name, args, nil)
					res := invoker.Invoke(ctx, invo)
					return res.Result().(*tri.Response), res.Error()
				},
			)
		case constant.CallServerStream:
			handler = tri.NewServerStreamHandler(
				procedure,
				method.ReqInitFunc,
				func(ctx context.Context, request *tri.Request, stream *tri.ServerStream) error {
					var args []interface{}
					args = append(args, request.Msg, method.StreamInitFunc(stream))
					invo := invocation.NewRPCInvocation(method.Name, args, nil)
					res := invoker.Invoke(ctx, invo)
					return res.Error()
				},
			)
		case constant.CallBidiStream:
			handler = tri.NewBidiStreamHandler(
				procedure,
				func(ctx context.Context, stream *tri.BidiStream) error {
					var args []interface{}
					args = append(args, method.StreamInitFunc(stream))
					invo := invocation.NewRPCInvocation(method.Name, args, nil)
					res := invoker.Invoke(ctx, invo)
					return res.Error()
				},
			)
		}
		mux.Handle(procedure, handler)
	}
}

// Stop TRIPLE server
func (s *Server) Stop() {
	// todo: process error
	s.httpServer.Close()
}

// GracefulStop TRIPLE server
func (s *Server) GracefulStop() {
	// todo: process error and use timeout
	s.httpServer.Shutdown(context.Background())
}
