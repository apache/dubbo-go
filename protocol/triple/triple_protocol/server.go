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
	"net/http"
	"sync"
)

import (
	"github.com/dubbogo/grpc-go"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

type Server struct {
	mu       sync.Mutex
	mux      *http.ServeMux
	handlers map[string]*Handler
	httpSrv  *http.Server
}

func (s *Server) RegisterUnaryHandler(
	procedure string,
	reqInitFunc func() interface{},
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
	reqInitFunc func() interface{},
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
	srv interface{},
	unary MethodHandler,
	options ...HandlerOption,
) error {
	hdl, ok := s.handlers[procedure]
	if !ok {
		hdl = NewCompatUnaryHandler(procedure, srv, unary, options...)
		s.handlers[procedure] = hdl
		s.mux.Handle(procedure, hdl)
	} else {
		config := newHandlerConfig(procedure, options)
		implementation := generateCompatUnaryHandlerFunc(procedure, srv, unary, config.Interceptor)
		hdl.processImplementation(getIdentifier(config.Group, config.Version), implementation)
	}

	return nil
}

func (s *Server) RegisterCompatStreamHandler(
	procedure string,
	srv interface{},
	typ StreamType,
	streamFunc func(srv interface{}, stream grpc.ServerStream) error,
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

func (s *Server) Run() error {
	// todo(DMwangnima): deal with TLS
	s.httpSrv.Handler = h2c.NewHandler(s.mux, &http2.Server{})

	if err := s.httpSrv.ListenAndServe(); err != nil {
		return err
	}
	return nil
}

func (s *Server) Stop() error {
	return s.httpSrv.Close()
}

func (s *Server) GracefulStop(ctx context.Context) error {
	return s.httpSrv.Shutdown(ctx)
}

func NewServer(addr string) *Server {
	return &Server{
		mux:      http.NewServeMux(),
		handlers: make(map[string]*Handler),
		httpSrv:  &http.Server{Addr: addr},
	}
}
