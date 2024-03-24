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
	"github.com/dubbogo/gost/log/logger"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"net"
	"net/http"
	"strconv"
	"sync"
)

import (
	"github.com/dubbogo/grpc-go"
)

type TLSConfigProvider func() (*tls.Config, error)

type Server struct {
	mu                sync.Mutex
	mux               *http.ServeMux
	handlers          map[string]*Handler
	httpSrv           *http.Server
	httpsSrv          *http.Server
	httpLn            net.Listener
	httpsLn           net.Listener
	addr              string
	tlsConfigProvider TLSConfigProvider
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
	method string,
	srv interface{},
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

func (s *Server) getHTTPSAddress(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		// Handle error
	}
	portNum, _ := strconv.Atoi(port)
	portNum++
	return net.JoinHostPort(host, strconv.Itoa(portNum))
}

func (s *Server) Run() error {
	// todo(DMwangnima): deal with TLS
	// Check if both listeners are nil
	// todo http and https port can be different based on mutual tls mode and tls config provider existed or not
	httpAddr := s.addr
	httpsAddr := s.getHTTPSAddress(s.addr)
	httpOn := true
	httpsOn := false
	if s.tlsConfigProvider != nil {
		httpsOn = true
	}

	handler := h2c.NewHandler(s.mux, &http2.Server{})

	setHTTPHeaders := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Set http scheme header
			r.Header.Set(":x-scheme", "http")
			r.Header.Set(":x-host", r.Host)
			r.Header.Set(":x-path", r.RequestURI)
			r.Header.Set(":x-method", r.Method)
			h.ServeHTTP(w, r)
		})
	}

	setHTTPSHeaders := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Set http scheme header
			r.Header.Set(":x-scheme", "https")
			r.Header.Set(":x-host", r.Host)
			r.Header.Set(":x-path", r.RequestURI)
			r.Header.Set(":x-method", r.Method)
			certs := r.TLS.PeerCertificates
			if len(certs) > 0 {
				peerCert := certs[0]
				if len(peerCert.URIs) > 0 {
					spiffeURI := peerCert.URIs[0].String()
					// Set spiffe scheme header
					r.Header.Set(":x-spiffe", spiffeURI)
				}
			}
			h.ServeHTTP(w, r)
		})
	}

	if s.httpLn == nil && httpOn {
		httpLn, err := net.Listen("tcp", httpAddr)
		if err != nil {
			httpLn.Close()
			return err
		}
		s.httpLn = httpLn
		s.httpSrv = &http.Server{Handler: setHTTPHeaders(handler)}

	}
	if s.httpsLn == nil && httpsOn {
		tlsCfg, err := s.tlsConfigProvider()
		if err != nil {
			logger.Error("can not get tls config")
		}
		httpsLn, err := tls.Listen("tcp", httpsAddr, tlsCfg)
		if err != nil {
			httpsLn.Close()
			return err
		}
		s.httpsLn = httpsLn
		s.httpsSrv = &http.Server{Handler: setHTTPSHeaders(handler)}
	}
	if httpsOn {
		go s.httpsSrv.Serve(s.httpsLn)
	}
	// http should be on now
	if err := s.httpSrv.Serve(s.httpLn); err != nil {
		return err
	}
	return nil
}

func (s *Server) Stop() error {
	return s.httpSrv.Close()
}

func (s *Server) GracefulStop(ctx context.Context) error {
	if s.httpsSrv != nil {
		s.httpsSrv.Shutdown(ctx)
	}
	return s.httpSrv.Shutdown(ctx)
}

func NewServer(addr string, provider TLSConfigProvider) *Server {
	return &Server{
		mux:               http.NewServeMux(),
		handlers:          make(map[string]*Handler),
		addr:              addr,
		tlsConfigProvider: provider,
	}
}
