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
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"github.com/dubbogo/grpc-go"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

type Server struct {
	mu       sync.Mutex
	addr     string
	mux      *http.ServeMux
	handlers map[string]*Handler
	httpSrv  *http.Server
	http3Srv *http3.Server
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

func (s *Server) Run(callProtocol string, tlsConf *tls.Config) error {
	// TODO: Refactor to support starting HTTP/2 and HTTP/3 servers simultaneously.
	// The current switch logic is mutually exclusive. Future work should allow enabling
	// both protocols, likely based on configuration, and run them concurrently.
	switch callProtocol {
	case constant.CallHTTP2:
		return s.startHttp2(tlsConf)
	case constant.CallHTTP3:
		return s.startHttp3(tlsConf)
	default:
		return fmt.Errorf("unsupported protocol: %s, only http2 or http3 are supported", callProtocol)
	}
}

func (s *Server) startHttp2(tlsConf *tls.Config) error {
	s.httpSrv = &http.Server{
		Addr:      s.addr,
		Handler:   h2c.NewHandler(s.mux, &http2.Server{}),
		TLSConfig: tlsConf,
	}

	logger.Debugf("TRIPLE HTTP/2 Server starting on %v", s.addr)

	var err error

	if tlsConf != nil {
		err = s.httpSrv.ListenAndServeTLS("", "")
	} else {
		err = s.httpSrv.ListenAndServe()
	}

	return err
}

func (s *Server) startHttp3(tlsConf *tls.Config) error {
	if tlsConf == nil {
		return fmt.Errorf("TRIPLE HTTP/3 Server must have TLS config, but TLS config is nil")
	}

	s.http3Srv = &http3.Server{
		Addr:    s.addr,
		Handler: s.mux,
		// Adapt and enhance a generic tls.Config object into a configuration
		// specifically for HTTP/3 services.
		// ref: https://quic-go.net/docs/http3/server/#setting-up-a-http3server
		TLSConfig: http3.ConfigureTLSConfig(tlsConf),
		// TODO: Detailed QUIC configuration.
		QUICConfig: &quic.Config{},
	}

	logger.Debugf("TRIPLE HTTP/3 Server starting on %v", s.addr)

	return s.http3Srv.ListenAndServe()
}

// Stop the Triple server for both HTTP/2 and HTTP/3.
// Because stop is very fast, there is no need to parallelize stop.
func (s *Server) Stop() error {
	var errs []error

	// stop HTTP server
	if s.httpSrv != nil {
		if err := s.httpSrv.Close(); err != nil {
			errs = append(errs, fmt.Errorf("http server close failed: %w", err))
		}
	}

	// stop HTTP/3 server
	if s.http3Srv != nil {
		if err := s.http3Srv.Close(); err != nil {
			errs = append(errs, fmt.Errorf("http3 server close failed: %w", err))
		}
	}

	switch len(errs) {
	case 0:
		return nil
	case 1:
		return errs[0]
	default:
		var sb strings.Builder
		sb.WriteString("multiple errors occurred during stop:")
		for _, err := range errs {
			// Newline and indent for easier reading
			sb.WriteString("\n\t- ")
			sb.WriteString(err.Error())
		}
		return errors.New(sb.String())
	}
}

// Gracefulstop shutdown the Triple server for both HTTP/2 and HTTP/3 gracefully.
// Because graceful shutdown is slow, I adopted concurrent processing.
func (s *Server) GracefulStop(ctx context.Context) error {
	var (
		wg      sync.WaitGroup
		errChan = make(chan error, 2)
	)

	// shutdown HTTP server
	if s.httpSrv != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.httpSrv.Shutdown(ctx); err != nil {
				errChan <- fmt.Errorf("http server shutdown failed: %w", err)
			}
		}()
	}

	// shutdown HTTP/3 server
	if s.http3Srv != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.http3Srv.Shutdown(ctx); err != nil {
				errChan <- fmt.Errorf("http3 server shutdown failed: %w", err)
			}
		}()
	}

	wg.Wait()
	close(errChan)

	// Error Collection and Handling.
	// Collect all errors into a slice.
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	switch len(errs) {
	case 0:
		return nil
	case 1:
		return errs[0]
	default:
		var sb strings.Builder
		sb.WriteString("multiple errors occurred during graceful stop:")
		for _, err := range errs {
			// Newline and indent for easier reading
			sb.WriteString("\n\t- ")
			sb.WriteString(err.Error())
		}
		return errors.New(sb.String())
	}
}

func NewServer(addr string) *Server {
	return &Server{
		mux:      http.NewServeMux(),
		addr:     addr,
		handlers: make(map[string]*Handler),
	}
}
