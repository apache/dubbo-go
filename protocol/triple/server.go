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
	"fmt"
	"reflect"
	"sync"
	"time"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"github.com/dustin/go-humanize"

	"google.golang.org/grpc"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/dubbo3"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/reflection"
	tri "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
	"dubbo.apache.org/dubbo-go/v3/server"
)

// Server is TRIPLE server
type Server struct {
	triServer *tri.Server
	services  map[string]grpc.ServiceInfo
	mu        sync.RWMutex
}

// NewServer creates a new TRIPLE server
func NewServer() *Server {
	return &Server{
		services: make(map[string]grpc.ServiceInfo),
	}
}

// Start TRIPLE server
func (s *Server) Start(invoker protocol.Invoker, info *server.ServiceInfo) {
	var (
		addr string
		URL  *common.URL
	)
	URL = invoker.GetURL()
	addr = URL.Location
	s.triServer = tri.NewServer(addr)
	serialization := URL.GetParam(constant.SerializationKey, constant.ProtobufSerialization)
	switch serialization {
	case constant.ProtobufSerialization:
	case constant.JSONSerialization:
	case constant.Hessian2Serialization:
	default:
		panic(fmt.Sprintf("Unsupported serialization: %s", serialization))
	}
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
	//var cfg *tls.Config
	// todo(DMwangnima): think about a more elegant way to configure tls
	//tlsConfig := config.GetRootConfig().TLSConfig
	//if tlsConfig != nil {
	//	cfg, err = config.GetServerTlsConfig(&config.TLSConfig{
	//		CACertFile:    tlsConfig.CACertFile,
	//		TLSCertFile:   tlsConfig.TLSCertFile,
	//		TLSKeyFile:    tlsConfig.TLSKeyFile,
	//		TLSServerName: tlsConfig.TLSServerName,
	//	})
	//	if err != nil {
	//		return
	//	}
	//	logger.Infof("Triple Server initialized the TLSConfig configuration")
	//}
	//srv.TLSConfig = cfg
	// todo:// move tls config to handleService

	hanOpts := getHanOpts(URL)
	intfName := URL.Interface()
	if info != nil {
		s.handleServiceWithInfo(intfName, invoker, info, hanOpts...)
		s.saveServiceInfo(intfName, info)
	} else {
		s.compatHandleService(intfName, hanOpts...)
	}
	reflection.Register(s)

	go func() {
		if runErr := s.triServer.Run(); runErr != nil {
			logger.Errorf("server serve failed with err: %v", runErr)
		}
	}()
}

// todo(DMwangnima): extract a common function
// RefreshService refreshes Triple Service
func (s *Server) RefreshService(invoker protocol.Invoker, info *server.ServiceInfo) {
	var (
		URL     *common.URL
		hanOpts []tri.HandlerOption
	)
	URL = invoker.GetURL()
	serialization := URL.GetParam(constant.SerializationKey, constant.ProtobufSerialization)
	switch serialization {
	case constant.ProtobufSerialization:
	case constant.JSONSerialization:
	case constant.Hessian2Serialization:
	default:
		panic(fmt.Sprintf("Unsupported serialization: %s", serialization))
	}
	hanOpts = getHanOpts(URL)
	intfName := URL.Interface()
	if info != nil {
		s.handleServiceWithInfo(intfName, invoker, info, hanOpts...)
		s.saveServiceInfo(intfName, info)
	} else {
		s.compatHandleService(intfName, hanOpts...)
	}
}

func getHanOpts(url *common.URL) (hanOpts []tri.HandlerOption) {
	var err error
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

	// todo:// open tracing
	hanOpts = append(hanOpts, tri.WithInterceptors())

	group := url.GetParam(constant.GroupKey, "")
	version := url.GetParam(constant.VersionKey, "")
	hanOpts = append(hanOpts, tri.WithGroup(group), tri.WithVersion(version))
	return hanOpts
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

// *Important*, this function is responsible for being compatible with old triple-gen code
// compatHandleService registers handler based on ServiceConfig and provider service.
func (s *Server) compatHandleService(interfaceName string, opts ...tri.HandlerOption) {
	providerServices := config.GetProviderConfig().Services
	if len(providerServices) == 0 {
		logger.Info("Provider service map is null")
	}
	//waitTripleExporter(providerServices)
	for key, providerService := range providerServices {
		if providerService.Interface != interfaceName {
			continue
		}
		// todo(DMwangnima): judge protocol type
		service := config.GetProviderService(key)
		serviceKey := common.ServiceKey(providerService.Interface, providerService.Group, providerService.Version)
		exporter, _ := tripleProtocol.ExporterMap().Load(serviceKey)
		if exporter == nil {
			continue
			//panic(fmt.Sprintf("no exporter found for servicekey: %v", serviceKey))
		}
		invoker := exporter.(protocol.Exporter).GetInvoker()
		if invoker == nil {
			panic(fmt.Sprintf("no invoker found for servicekey: %v", serviceKey))
		}
		ds, ok := service.(dubbo3.Dubbo3GrpcService)
		if !ok {
			info := createServiceInfoWithReflection(service)
			s.handleServiceWithInfo(interfaceName, invoker, info, opts...)
		}

		// inject invoker, it has all invocation logics
		ds.XXX_SetProxyImpl(invoker)
		s.compatRegisterHandler(interfaceName, ds, opts...)
	}
}

func (s *Server) compatRegisterHandler(interfaceName string, svc dubbo3.Dubbo3GrpcService, opts ...tri.HandlerOption) {
	desc := svc.XXX_ServiceDesc()
	// init unary handlers
	for _, method := range desc.Methods {
		// please refer to protocol/triple/internal/proto/triple_gen/greettriple for procedure examples
		// error could be ignored because base is empty string
		procedure := joinProcedure(interfaceName, method.MethodName)
		_ = s.triServer.RegisterCompatUnaryHandler(procedure, method.MethodName, svc, tri.MethodHandler(method.Handler), opts...)
	}

	// init stream handlers
	for _, stream := range desc.Streams {
		// please refer to protocol/triple/internal/proto/triple_gen/greettriple for procedure examples
		// error could be ignored because base is empty string
		procedure := joinProcedure(interfaceName, stream.StreamName)
		var typ tri.StreamType
		switch {
		case stream.ClientStreams && stream.ServerStreams:
			typ = tri.StreamTypeBidi
		case stream.ClientStreams:
			typ = tri.StreamTypeClient
		case stream.ServerStreams:
			typ = tri.StreamTypeServer
		}
		_ = s.triServer.RegisterCompatStreamHandler(procedure, svc, typ, stream.Handler, opts...)
	}
}

// handleServiceWithInfo injects invoker and create handler based on ServiceInfo
func (s *Server) handleServiceWithInfo(interfaceName string, invoker protocol.Invoker, info *server.ServiceInfo, opts ...tri.HandlerOption) {
	for _, method := range info.Methods {
		m := method
		procedure := joinProcedure(interfaceName, method.Name)
		switch m.Type {
		case constant.CallUnary:
			_ = s.triServer.RegisterUnaryHandler(
				procedure,
				m.ReqInitFunc,
				func(ctx context.Context, req *tri.Request) (*tri.Response, error) {
					var args []interface{}
					args = append(args, req.Msg)
					// todo: inject method.Meta to attachments
					invo := invocation.NewRPCInvocation(m.Name, args, nil)
					res := invoker.Invoke(ctx, invo)
					// todo(DMwangnima): if we do not use MethodInfo.MethodFunc, create Response manually
					return res.Result().(*tri.Response), res.Error()
				},
				opts...,
			)
		case constant.CallClientStream:
			_ = s.triServer.RegisterClientStreamHandler(
				procedure,
				func(ctx context.Context, stream *tri.ClientStream) (*tri.Response, error) {
					var args []interface{}
					args = append(args, m.StreamInitFunc(stream))
					invo := invocation.NewRPCInvocation(m.Name, args, nil)
					res := invoker.Invoke(ctx, invo)
					return res.Result().(*tri.Response), res.Error()
				},
				opts...,
			)
		case constant.CallServerStream:
			_ = s.triServer.RegisterServerStreamHandler(
				procedure,
				m.ReqInitFunc,
				func(ctx context.Context, request *tri.Request, stream *tri.ServerStream) error {
					var args []interface{}
					args = append(args, request.Msg, m.StreamInitFunc(stream))
					invo := invocation.NewRPCInvocation(m.Name, args, nil)
					res := invoker.Invoke(ctx, invo)
					return res.Error()
				},
				opts...,
			)
		case constant.CallBidiStream:
			_ = s.triServer.RegisterBidiStreamHandler(
				procedure,
				func(ctx context.Context, stream *tri.BidiStream) error {
					var args []interface{}
					args = append(args, m.StreamInitFunc(stream))
					invo := invocation.NewRPCInvocation(m.Name, args, nil)
					res := invoker.Invoke(ctx, invo)
					return res.Error()
				},
				opts...,
			)
		}
	}
}

func (s *Server) saveServiceInfo(interfaceName string, info *server.ServiceInfo) {
	ret := grpc.ServiceInfo{}
	ret.Methods = make([]grpc.MethodInfo, 0, len(info.Methods))
	for _, method := range info.Methods {
		md := grpc.MethodInfo{}
		md.Name = method.Name
		switch method.Type {
		case constant.CallUnary:
			md.IsClientStream = false
			md.IsServerStream = false
		case constant.CallBidiStream:
			md.IsClientStream = true
			md.IsServerStream = true
		case constant.CallClientStream:
			md.IsClientStream = true
			md.IsServerStream = false
		case constant.CallServerStream:
			md.IsClientStream = false
			md.IsServerStream = true
		}
		ret.Methods = append(ret.Methods, md)
	}
	ret.Metadata = info
	s.mu.Lock()
	defer s.mu.Unlock()
	s.services[interfaceName] = ret
}

func (s *Server) GetServiceInfo() map[string]grpc.ServiceInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	res := make(map[string]grpc.ServiceInfo, len(s.services))
	for k, v := range s.services {
		res[k] = v
	}
	return res
}

// Stop TRIPLE server
func (s *Server) Stop() {
	_ = s.triServer.Stop()
}

// GracefulStop TRIPLE server
func (s *Server) GracefulStop() {
	_ = s.triServer.GracefulStop(context.Background())
}

func createServiceInfoWithReflection(svc common.RPCService) *server.ServiceInfo {
	var info *server.ServiceInfo
	val := reflect.ValueOf(svc)
	typ := reflect.TypeOf(svc)
	methodNum := val.NumMethod()
	methodInfos := make([]server.MethodInfo, methodNum)
	for i := 0; i < methodNum; i++ {
		methodType := typ.Method(i)
		if methodType.Name == "Reference" {
			continue
		}
		paramsNum := methodType.Type.NumIn()
		// ignore ctx
		paramsTypes := make([]reflect.Type, paramsNum-1)
		for j := 1; j < paramsNum; j++ {
			paramsTypes[j-1] = methodType.Type.In(j)
		}
		methodInfo := server.MethodInfo{
			Name: methodType.Name,
			Type: constant.CallUnary,
			ReqInitFunc: func() interface{} {
				params := make([]interface{}, len(paramsTypes))
				for i, paramType := range paramsTypes {
					params[i] = reflect.New(paramType).Interface()
				}
				return params
			},
		}
		methodInfos[i] = methodInfo
	}

	return info
}
