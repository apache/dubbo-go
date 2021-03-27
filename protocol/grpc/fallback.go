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
	"reflect"
	"strings"
	"sync"
)

import (
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

import (
	"github.com/apache/dubbo-go/common/logger"
)

// serviceInfo wrap the service, which is used by
// FallbackHandler internally.
type serviceInfo struct {
	serviceImpl interface{}
	methods     map[string]*grpc.MethodDesc
	streams     map[string]*grpc.StreamDesc
	mdata       interface{}
}

// FallbackHandler route the request to service by
// service name.
//
// For example:
//
//     handler := NewFallbackHandler()
//     go func() {
//         time.Sleep(25 * time.Second)
//         handler.RegisterService(sd0, ss0)
//     }()
//
//     listener, err := net.Listen("tcp", port)
//     if err != nil {
//         log.Fatalf("failed to listen: %v", err)
//     }
//
//     server := grpc.NewServer(grpc.UnknownServiceHandler(handler.Handle))
//     server.RegisterService(sd0, ss0)
//     if err := server.Serve(listener); err != nil {
//         log.Fatalf("failed to serve: %v", err)
//     }
//
// Based on https://github.com/grpc/grpc-go/blob/bce1cded4b05db45e02a87b94b75fa5cb07a76a5/server.go,
// there are three differences between server.RegisterService and handler.RegisterService:
// 1. server.RegisterService must happen before server.Serve because server.go#L593
//    will use serve flag to prevent register success after server start, but
//    there is no restriction on handler.RegisterService.
// 2. Service registered by server handle request at server.go#L1537-L1547,
//    while service registered by handler do at server.go#L1548-L1552 by a fallback
//    logic.
// 3. Service registered by server can handle unary and stream rpc respectively,
//    while service registered by handler handle unary rpc by a stream way. So
//    the unary interceptor will not work on the latter.
type FallbackHandler struct {
	mu       sync.Mutex
	services map[string]*serviceInfo
}

// NewFallbackHandler return FallbackHandler pointer.
func NewFallbackHandler() *FallbackHandler {
	return &FallbackHandler{
		services: make(map[string]*serviceInfo),
	}
}

// RegisterService is used to register service definition
// and implementation. It check whether ss implement sd
// at first.
// This method make FallbackHandler implement ServiceRegistrar
// interface like grpc server.
func (handler *FallbackHandler) RegisterService(sd *grpc.ServiceDesc, ss interface{}) {
	if ss != nil {
		ht := reflect.TypeOf(sd.HandlerType).Elem()
		st := reflect.TypeOf(ss)
		if !st.Implements(ht) {
			logger.Errorf("grpc: Server.RegisterService found the handler of type %v that does not satisfy %v", st, ht)
			return
		}
	}
	handler.register(sd, ss)
}

// register is used to register service definition
// and implementation.
func (handler *FallbackHandler) register(sd *grpc.ServiceDesc, ss interface{}) {
	handler.mu.Lock()
	defer handler.mu.Unlock()
	if _, ok := handler.services[sd.ServiceName]; ok {
		logger.Errorf("grpc: Server.RegisterService found duplicate service registration for %q", sd.ServiceName)
		return
	}
	info := &serviceInfo{
		serviceImpl: ss,
		methods:     make(map[string]*grpc.MethodDesc),
		streams:     make(map[string]*grpc.StreamDesc),
		mdata:       sd.Metadata,
	}
	for i := range sd.Methods {
		d := &sd.Methods[i]
		info.methods[d.MethodName] = d
	}
	for i := range sd.Streams {
		d := &sd.Streams[i]
		info.streams[d.StreamName] = d
	}
	handler.services[sd.ServiceName] = info
}

// parseName split http2 path into service name
// and method name.
func parseName(sm string) (string, string, error) {
	if sm != "" && sm[0] == '/' {
		sm = sm[1:]
	}

	pos := strings.LastIndex(sm, "/")
	if pos == -1 {
		return "", "", errors.Errorf("no / in %s", sm)
	}

	return sm[:pos], sm[pos+1:], nil
}

// Handle process stream by service.
func (handler *FallbackHandler) Handle(srv interface{}, stream grpc.ServerStream) error {
	method, ok := grpc.MethodFromServerStream(stream)
	if !ok {
		return errors.Errorf("cat no retrieve method because of no transport stream ctx in server stream")
	}

	service, method, err := parseName(method)
	if err != nil {
		return err
	}

	info, knownService := handler.services[service]
	if knownService {
		if md, ok := info.methods[method]; ok {
			return processUnaryRPC(stream, info, md)
		}
		if sd, ok := info.streams[method]; ok {
			return processStreamingRPC(stream, info, sd)
		}
	}

	if !knownService {
		return errors.Errorf("unknown service %v", service)
	} else {
		return errors.Errorf("unknown method %v for service %v", method, service)
	}
}

// processUnaryRPC process unary rpc.
//
// Refer:
// 1. https://github.com/grpc/grpc-go/issues/1801#issuecomment-358379067
func processUnaryRPC(stream grpc.ServerStream, info *serviceInfo, md *grpc.MethodDesc) error {
	df := func(v interface{}) error {
		return stream.RecvMsg(v)
	}
	result, err := md.Handler(info.serviceImpl, stream.Context(), df, nil)
	if err != nil {
		return err
	}
	return stream.SendMsg(result)
}

// processStreamRPC process stream rpc.
func processStreamingRPC(stream grpc.ServerStream, info *serviceInfo, sd *grpc.StreamDesc) error {
	return sd.Handler(info.serviceImpl, stream)
}
