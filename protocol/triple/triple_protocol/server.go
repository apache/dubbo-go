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

package triple_protocol

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"github.com/dubbogo/grpc-go"

	"github.com/quic-go/quic-go/http3"

	uatomic "go.uber.org/atomic"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"golang.org/x/sync/errgroup"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/internal/http3config"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/openapi"
)

// netListen and netListenPacket create the pre-bound sockets in
// startHttp2AndHttp3. Tests override them to simulate Serve failures.
var (
	netListen       = net.Listen
	netListenPacket = net.ListenPacket
)

type Server struct {
	addr               string
	mux                *methodRouteMux
	handlers           map[string]*Handler
	httpSrv            uatomic.Pointer[http.Server]
	http3Srv           uatomic.Pointer[http3.Server]
	stopCount          uatomic.Uint32
	tripleConfig       *global.TripleConfig // Configuration for the triple protocol
	openapiIntegration *openapi.OpenAPIIntegration
}

func (s *Server) RegisterUnaryHandler(
	procedure string,
	reqInitFunc func() any,
	unary func(context.Context, *Request) (*Response, error),
	options ...HandlerOption,
) error {
	hdl, ok := s.handlers[procedure]
	if !ok {
		hdl = NewUnaryHandler(procedure, reqInitFunc, unary, options...)
		s.handlers[procedure] = hdl
		s.mux.Handle(procedure, hdl)
	} else {
		config := newHandlerConfig(procedure, options)
		implementation := generateUnaryHandlerFunc(procedure, reqInitFunc, unary, config.Interceptor)
		hdl.processImplementation(getIdentifier(config.Group, config.Version), implementation)
	}

	return nil
}

func (s *Server) RegisterClientStreamHandler(
	procedure string,
	stream func(context.Context, *ClientStream) (*Response, error),
	options ...HandlerOption,
) error {
	hdl, ok := s.handlers[procedure]
	if !ok {
		hdl = NewClientStreamHandler(procedure, stream, options...)
		s.handlers[procedure] = hdl
		s.mux.Handle(procedure, hdl)
	} else {
		config := newHandlerConfig(procedure, options)
		implementation := generateClientStreamHandlerFunc(procedure, stream, config.Interceptor)
		hdl.processImplementation(getIdentifier(config.Group, config.Version), implementation)
	}

	return nil
}

func (s *Server) RegisterServerStreamHandler(
	procedure string,
	reqInitFunc func() any,
	stream func(context.Context, *Request, *ServerStream) error,
	options ...HandlerOption,
) error {
	hdl, ok := s.handlers[procedure]
	if !ok {
		hdl = NewServerStreamHandler(procedure, reqInitFunc, stream, options...)
		s.handlers[procedure] = hdl
		s.mux.Handle(procedure, hdl)
	} else {
		config := newHandlerConfig(procedure, options)
		implementation := generateServerStreamHandlerFunc(procedure, reqInitFunc, stream, config.Interceptor)
		hdl.processImplementation(getIdentifier(config.Group, config.Version), implementation)
	}

	return nil
}

func (s *Server) RegisterBidiStreamHandler(
	procedure string,
	stream func(context.Context, *BidiStream) error,
	options ...HandlerOption,
) error {
	hdl, ok := s.handlers[procedure]
	if !ok {
		hdl = NewBidiStreamHandler(procedure, stream, options...)
		s.handlers[procedure] = hdl
		s.mux.Handle(procedure, hdl)
	} else {
		config := newHandlerConfig(procedure, options)
		implementation := generateBidiStreamHandlerFunc(procedure, stream, config.Interceptor)
		hdl.processImplementation(getIdentifier(config.Group, config.Version), implementation)
	}

	return nil
}

func (s *Server) RegisterCompatUnaryHandler(
	procedure string,
	method string,
	srv any,
	unary MethodHandler,
	options ...HandlerOption,
) error {
	hdl, ok := s.handlers[procedure]
	if !ok {
		hdl = NewCompatUnaryHandler(procedure, method, srv, unary, options...)
		s.handlers[procedure] = hdl
		s.mux.Handle(procedure, hdl)
	} else {
		config := newHandlerConfig(procedure, options)
		implementation := generateCompatUnaryHandlerFunc(procedure, method, srv, unary, config.Interceptor)
		hdl.processImplementation(getIdentifier(config.Group, config.Version), implementation)
	}

	return nil
}

func (s *Server) RegisterCompatStreamHandler(
	procedure string,
	srv any,
	typ StreamType,
	streamFunc func(srv any, stream grpc.ServerStream) error,
	options ...HandlerOption,
) error {
	hdl, ok := s.handlers[procedure]
	if !ok {
		hdl = NewCompatStreamHandler(procedure, srv, typ, streamFunc, options...)
		s.handlers[procedure] = hdl
		s.mux.Handle(procedure, hdl)
	} else {
		config := newHandlerConfig(procedure, options)
		implementation := generateCompatStreamHandlerFunc(procedure, srv, streamFunc, config.Interceptor)
		hdl.processImplementation(getIdentifier(config.Group, config.Version), implementation)
	}

	return nil
}

func (s *Server) SetFallbackHTTPHandler(h http.Handler) {
	if s.mux == nil {
		return
	}
	s.mux.SetFallbackHandler(h)
}

// Start starts the server for the given protocol without blocking. It
// snapshots the startup epoch synchronously before the transport goroutine
// runs, so run's checkpoint detects a Stop that completes before the
// goroutine executes. Serve errors are logged; use Run when the error must
// be returned synchronously.
func (s *Server) Start(callProtocol string, tlsConf *tls.Config) {
	epoch := s.beginStart()
	go func() {
		if runErr := s.run(callProtocol, tlsConf, epoch); runErr != nil {
			logger.Errorf("[Triple][Server] server serve failed, err=%v", runErr)
		}
	}()
}

// beginStart snapshots the startup epoch before the transport goroutine
// runs. It must be called synchronously on the start path so run's
// checkpoint can detect a Stop that completes before run reads the counter.
func (s *Server) beginStart() uint32 {
	return s.stopCount.Load()
}

// Run starts the server for the given protocol and blocks until the server
// is closed. It keeps the pre-3640 two-argument signature: the startup epoch
// is snapshotted here, so a Stop that completes before this call executes
// cannot be detected. Callers that need that guarantee use Start.
func (s *Server) Run(callProtocol string, tlsConf *tls.Config) error {
	return s.run(callProtocol, tlsConf, s.stopCount.Load())
}

func (s *Server) run(callProtocol string, tlsConf *tls.Config, epoch uint32) error {
	// A Stop that completed after the synchronous epoch snapshot but before
	// this checkpoint aborts the startup here, before any socket is bound or
	// served.
	if s.stopCount.Load() != epoch {
		return nil
	}

	// Support for starting HTTP/2 and HTTP/3 servers simultaneously.
	switch callProtocol {
	case constant.CallHTTP2:
		return s.startHttp2(tlsConf, epoch)
	case constant.CallHTTP3:
		return s.startHttp3(tlsConf, epoch)
	case constant.CallHTTP2AndHTTP3:
		return s.startHttp2AndHttp3(tlsConf, epoch)
	default:
		return fmt.Errorf("unsupported protocol: %s, only http2, http3, or http2-and-http3 are supported", callProtocol)
	}
}

func (s *Server) startHttp2(tlsConf *tls.Config, epoch uint32) error {
	s.httpSrv.Store(&http.Server{
		Addr:      s.addr,
		Handler:   h2c.NewHandler(s.mux, &http2.Server{}),
		TLSConfig: tlsConf,
	})

	// A Stop that landed after the entry checkpoint but before this server
	// was published closed nothing; abort so no listener is served.
	if s.stopCount.Load() != epoch {
		return nil
	}

	logger.Debugf("[Triple][Server] triple HTTP/2 Server starting on %v", s.addr)

	srv := s.httpSrv.Load()

	var err error
	if tlsConf != nil {
		err = srv.ListenAndServeTLS("", "")
	} else {
		err = srv.ListenAndServe()
	}
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) startHttp3(tlsConf *tls.Config, epoch uint32) error {
	if tlsConf == nil {
		return fmt.Errorf("TRIPLE HTTP/3 Server must have TLS config, but TLS config is nil")
	}

	var http3Config *global.Http3Config
	if s.tripleConfig != nil {
		http3Config = s.tripleConfig.Http3
	}

	quicConfig, err := http3config.NewQUICConfig(http3Config, nil)
	if err != nil {
		return err
	}

	s.http3Srv.Store(&http3.Server{
		Addr:    s.addr,
		Handler: s.mux,
		// Adapt and enhance a generic tls.Config object into a configuration
		// specifically for HTTP/3 services.
		// ref: https://quic-go.net/docs/http3/server/#setting-up-a-http3server
		TLSConfig:  http3.ConfigureTLSConfig(tlsConf),
		QUICConfig: quicConfig,
	})

	// A Stop that landed after the entry checkpoint but before this server
	// was published closed nothing; abort so no listener is served.
	if s.stopCount.Load() != epoch {
		return nil
	}

	logger.Debugf("[Triple][Server] triple HTTP/3 Server starting on %v", s.addr)

	err = s.http3Srv.Load().ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) startHttp2AndHttp3(tlsConf *tls.Config, epoch uint32) error {
	// Check if TLS config is provided for HTTP/3
	if tlsConf == nil {
		return fmt.Errorf("TRIPLE HTTP/2 and HTTP/3 Server must have TLS config, but TLS config is nil")
	}

	var http3Config *global.Http3Config
	if s.tripleConfig != nil {
		http3Config = s.tripleConfig.Http3
	}

	quicConfig, err := http3config.NewQUICConfig(http3Config, nil)
	if err != nil {
		return err
	}

	if len(tlsConf.Certificates) == 0 &&
		tlsConf.GetCertificate == nil &&
		tlsConf.GetConfigForClient == nil {
		return fmt.Errorf("TRIPLE HTTP/2 and HTTP/3 Server must have a TLS certificate configured, but none of Certificates/GetCertificate/GetConfigForClient is set")
	}

	// Pre-bind the TCP (HTTP/2) listener before serving any request:
	// fail fast with the bind error when the port is occupied.
	tcpLn, err := netListen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("HTTP/2 server bind error: %w", err)
	}
	defer tcpLn.Close()

	// Pre-bind the UDP (HTTP/3) socket as well; on failure close the
	// already-bound TCP listener and return, no request has been served yet.
	udpConn, err := netListenPacket("udp", s.addr)
	if err != nil {
		return fmt.Errorf("HTTP/3 server bind error: %w", err)
	}
	defer udpConn.Close()

	// Start HTTP/3 server first to get its configuration
	s.http3Srv.Store(&http3.Server{
		Addr:       s.addr,
		Handler:    s.mux,
		TLSConfig:  http3.ConfigureTLSConfig(tlsConf),
		QUICConfig: quicConfig,
	})

	// Create Alt-Svc handler wrapper for HTTP/2 server
	var negotiation bool
	if s.tripleConfig != nil && s.tripleConfig.Http3 != nil {
		negotiation = s.tripleConfig.Http3.Negotiation
	}
	altSvcHandler := NewAltSvcHandler(s.mux, s.http3Srv.Load(), negotiation)

	// Start HTTP/2 server with Alt-Svc handler wrapper
	s.httpSrv.Store(&http.Server{
		Addr:      s.addr,
		Handler:   h2c.NewHandler(altSvcHandler, &http2.Server{}),
		TLSConfig: tlsConf,
	})

	// A Stop during the bind or store steps closed nothing; abort so the
	// deferred closes release the sockets.
	if s.stopCount.Load() != epoch {
		return nil
	}

	logger.Debugf("[Triple][Server] triple HTTP/2 and HTTP/3 Server starting on %v", s.addr)

	// Use errgroup to manage concurrent server startup
	eg, _ := errgroup.WithContext(context.Background())

	// Start HTTP/2 server in a goroutine
	eg.Go(func() error {
		if err := s.httpSrv.Load().ServeTLS(tcpLn, "", ""); err != nil && err != http.ErrServerClosed {
			// Close the HTTP/3 server so its Serve call returns and
			// eg.Wait does not block on the still-listening UDP socket.
			_ = s.http3Srv.Load().Close()
			return fmt.Errorf("HTTP/2 server error: %w", err)
		}
		return nil
	})

	// Start HTTP/3 server in a goroutine
	eg.Go(func() error {
		if err := s.http3Srv.Load().Serve(udpConn); err != nil && err != http.ErrServerClosed {
			// Close the HTTP/2 server so its Serve call returns and
			// eg.Wait does not block on the still-listening TCP listener.
			_ = s.httpSrv.Load().Close()
			return fmt.Errorf("HTTP/3 server error: %w", err)
		}
		return nil
	})

	// Wait for the first error from either server
	return eg.Wait()
}

// Stop the Triple server for both HTTP/2 and HTTP/3.
func (s *Server) Stop() error {
	// Record the stop first so an in-flight startup aborts at its checkpoint.
	s.stopCount.Add(1)

	eg, _ := errgroup.WithContext(context.Background())

	// stop HTTP server
	if srv := s.httpSrv.Load(); srv != nil {
		eg.Go(func() error {
			if err := srv.Close(); err != nil {
				return fmt.Errorf("http server close failed: %w", err)
			}
			return nil
		})
	}

	// stop HTTP/3 server
	if srv3 := s.http3Srv.Load(); srv3 != nil {
		eg.Go(func() error {
			if err := srv3.Close(); err != nil {
				return fmt.Errorf("http3 server close failed: %w", err)
			}
			return nil
		})
	}

	// Wait for all goroutines to complete and collect any errors
	return eg.Wait()
}

// Gracefulstop shutdown the Triple server for both HTTP/2 and HTTP/3 gracefully.
func (s *Server) GracefulStop(ctx context.Context) error {
	// Record the stop first so an in-flight startup aborts at its checkpoint.
	s.stopCount.Add(1)

	eg, ctx := errgroup.WithContext(ctx)

	// shutdown HTTP server
	if srv := s.httpSrv.Load(); srv != nil {
		eg.Go(func() error {
			if err := srv.Shutdown(ctx); err != nil {
				return fmt.Errorf("http server shutdown failed: %w", err)
			}
			return nil
		})
	}

	// shutdown HTTP/3 server
	if srv3 := s.http3Srv.Load(); srv3 != nil {
		eg.Go(func() error {
			if err := srv3.Shutdown(ctx); err != nil {
				return fmt.Errorf("http3 server shutdown failed: %w", err)
			}
			return nil
		})
	}

	// Wait for all goroutines to complete and collect any errors
	return eg.Wait()
}

func NewServer(addr string, tripleConf *global.TripleConfig) *Server {
	s := &Server{
		mux:          newMethodRouteMux(),
		addr:         addr,
		handlers:     make(map[string]*Handler),
		tripleConfig: tripleConf,
	}

	var openapiIntegration *openapi.OpenAPIIntegration
	if tripleConf != nil && tripleConf.OpenAPI != nil && tripleConf.OpenAPI.Enabled {
		openapiIntegration = openapi.NewOpenAPIIntegration(tripleConf.OpenAPI)
	}
	s.openapiIntegration = openapiIntegration

	openapiHandler := openapi.NewHTTPHandler(openapiIntegration)
	basePath := "/dubbo/openapi"
	if tripleConf != nil && tripleConf.OpenAPI != nil && tripleConf.OpenAPI.Path != "" {
		basePath = tripleConf.OpenAPI.Path
	}
	s.mux.Handle(basePath, openapiHandler)
	s.mux.Handle(basePath+"/", openapiHandler)
	s.mux.Handle(basePath+"/swagger-ui/", openapiHandler)
	s.mux.Handle(basePath+"/redoc/", openapiHandler)
	s.mux.Handle(basePath+"/openapi.json", openapiHandler)
	s.mux.Handle(basePath+"/openapi.yaml", openapiHandler)
	s.mux.Handle(basePath+"/openapi.yml", openapiHandler)
	s.mux.Handle(basePath+"/api-docs/", openapiHandler)

	return s
}

func (s *Server) RegisterOpenAPIService(interfaceName string, info *common.ServiceInfo, openapiGroup string, dubboGroup string, dubboVersion string) {
	if s.openapiIntegration != nil {
		s.openapiIntegration.RegisterService(interfaceName, info, openapiGroup, dubboGroup, dubboVersion)
	}
}
