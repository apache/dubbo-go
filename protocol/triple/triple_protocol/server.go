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
	"net/http"
	"sync"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"github.com/dubbogo/grpc-go"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/quic-go/quic-go/http3"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

type Server struct {
	mu       sync.Mutex
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

func (s *Server) Run(httpType string) error {
	switch httpType {
	case constant.CallHTTP2:
		return s.startHttp2()
	case constant.CallHTTP3:
		return s.startHttp3()
	default:
		panic("Unsupport http type, only support http2 or http3")
	}
}

func (s *Server) startHttp2() error {
	logger.Debug("Triple HTTP/2 server starting")
	s.httpSrv.Handler = h2c.NewHandler(s.mux, &http2.Server{})

	if s.httpSrv.TLSConfig != nil {
		return s.httpSrv.ListenAndServeTLS("", "")
	}
	return s.httpSrv.ListenAndServe()
}

func (s *Server) startHttp3() error {
	logger.Debug("Triple HTTP/3 server starting")
	// TODO: Find an ideal way to initialize the http3 server handler.
	s.http3Srv.Handler = s.mux

	return s.http3Srv.ListenAndServe()
}

// TODO: Find an ideal way to set TLSConfig.
func (s *Server) SetTLSConfig(c *tls.Config, httpType string) {
	switch httpType {
	case constant.CallHTTP2:
		s.httpSrv.TLSConfig = c
	case constant.CallHTTP3:
		// Adapt and enhance a generic tls.Config object into a configuration
		// specifically for HTTP/3 services.
		// ref: https://quic-go.net/docs/http3/server/#setting-up-a-http3server
		s.http3Srv.TLSConfig = http3.ConfigureTLSConfig(c)
	}
}

func (s *Server) Stop() error {
	var err error

	if s.httpSrv != nil {
		err = s.httpSrv.Close()
	}

	if s.http3Srv != nil {
		err = s.http3Srv.Close()
	}

	return err
}

func (s *Server) GracefulStop(ctx context.Context) error {
	var err error

	if s.httpSrv != nil {
		err = s.httpSrv.Shutdown(ctx)
	}

	// TODO: triple http3 GracefulStop

	// FIXME: There is something wrong when http3 work with GracefulStop.
	// if s.http3Srv != nil {
	// 	err = s.http3Srv.Shutdown(ctx)
	// }

	return err
}

func NewServer(addr string, httpType string) *Server {
	var srv *Server

	switch httpType {
	case constant.CallHTTP2:
		srv = &Server{
			mux:      http.NewServeMux(),
			handlers: make(map[string]*Handler),
			httpSrv:  &http.Server{Addr: addr},
		}
	case constant.CallHTTP3:
		srv = &Server{
			mux:      http.NewServeMux(),
			handlers: make(map[string]*Handler),
			http3Srv: &http3.Server{Addr: addr},
		}
	}

	return srv
}
