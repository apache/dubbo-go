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
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"

	"github.com/dubbogo/gost/log/logger"

	grpc_go "github.com/dubbogo/grpc-go"

	"github.com/dustin/go-humanize"

	"google.golang.org/grpc"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/internal"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/dubbo3"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	tri "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
	dubbotls "dubbo.apache.org/dubbo-go/v3/tls"
)

// Server is TRIPLE adaptation layer representation. It makes use of tri.Server to
// provide functionality.
type Server struct {
	triServer          *tri.Server
	cfg                *global.TripleConfig
	mu                 sync.RWMutex
	services           map[string]grpc.ServiceInfo
	transportStarted   bool
	mountedHTTPHandler http.Handler
	transportSettings  *transportSettings
}

type transportSettings struct {
	location     string
	callProtocol string
	tripleConfig *global.TripleConfig
	tlsConfig    *tls.Config
	rawTLSConfig *global.TLSConfig
}

// NewServer creates a new TRIPLE server.
func NewServer(cfg *global.TripleConfig) *Server {
	return &Server{
		cfg:      cfg,
		services: make(map[string]grpc.ServiceInfo),
	}
}

// Start prepares the Triple transport for one exported service. It resolves the
// listener configuration, ensures the underlying tri.Server exists, registers
// the service handlers, and finally starts the shared transport.
func (s *Server) Start(invoker base.Invoker, info *common.ServiceInfo) error {
	url := invoker.GetURL()

	settings, err := resolveServerTransport(url)
	if err != nil {
		return fmt.Errorf("TRIPLE server init failed: %w", err)
	}

	if err := s.ensureTriServer(settings); err != nil {
		return err
	}
	if err := s.refreshService(invoker, info); err != nil {
		return err
	}
	s.startTransport(settings.callProtocol, settings.tlsConfig)
	return nil
}

// resolveServerTransport extracts transport-level Triple configuration from the
// export URL and converts it into the concrete listener settings used by the
// adaptation layer.
func resolveServerTransport(url *common.URL) (*transportSettings, error) {
	if url == nil {
		return nil, fmt.Errorf("triple server url must not be nil")
	}

	tripleConf, err := resolveTripleConfig(url)
	if err != nil {
		return nil, err
	}

	callProtocol := constant.CallHTTP2
	if tripleConf != nil && tripleConf.Http3 != nil && tripleConf.Http3.Enable {
		callProtocol = constant.CallHTTP2AndHTTP3
	}

	// todo: support opentracing interceptor

	// TODO: move tls config to handleService

	rawTLSConfig, err := resolveRawTLSConfig(url)
	if err != nil {
		return nil, err
	}

	var tlsConf *tls.Config
	if dubbotls.IsServerTLSValid(rawTLSConfig) {
		tlsConf, err = dubbotls.GetServerTlSConfig(rawTLSConfig)
		if err != nil {
			return nil, fmt.Errorf("TRIPLE server initialized the TLSConfig configuration failed: %w", err)
		}
		logger.Infof("TRIPLE Server initialized the TLSConfig configuration")
	}

	return &transportSettings{
		location:     url.Location,
		callProtocol: callProtocol,
		tripleConfig: tripleConf,
		tlsConfig:    tlsConf,
		rawTLSConfig: rawTLSConfig,
	}, nil
}

// ensureTriServer lazily creates the shared tri.Server for one listener
// address. Service export and mount-first startup both converge here so they
// share the same transport lifecycle and reflection registration.
func (s *Server) ensureTriServer(settings *transportSettings) error {
	if err := s.validateTransportSettings(settings); err != nil {
		return err
	}

	if s.triServer != nil {
		return nil
	}

	s.mu.Lock()
	s.transportSettings = settings
	s.cfg = settings.tripleConfig
	s.mu.Unlock()

	s.triServer = tri.NewServer(settings.location, settings.tripleConfig)
	// MountHTTPHandler may initialize transport before any Triple service is
	// exported, so the first tri.Server instance must inherit the root handler.
	s.mu.RLock()
	handler := s.mountedHTTPHandler
	s.mu.RUnlock()
	if handler != nil {
		s.triServer.SetFallbackHTTPHandler(handler)
	}
	internal.ReflectionRegister(s)
	return nil
}

// MountHTTPHandler attaches a single root HTTP fallback to the Triple server.
// The mounted handler follows the same single-root semantics as server.Server:
// once a handler has been attached for one listener, later mounts must fail
// fast instead of silently replacing the existing fallback.
func (s *Server) MountHTTPHandler(handler http.Handler) error {
	if handler == nil {
		return fmt.Errorf("mounted HTTP handler must not be nil")
	}

	s.mu.Lock()
	if s.mountedHTTPHandler != nil {
		s.mu.Unlock()
		return fmt.Errorf("an HTTP handler has already been mounted")
	}
	s.mountedHTTPHandler = handler
	triServer := s.triServer
	s.mu.Unlock()

	if triServer != nil {
		triServer.SetFallbackHTTPHandler(handler)
	}
	return nil
}

func (s *Server) ValidateTransportURL(url *common.URL) error {
	settings, err := resolveServerTransport(url)
	if err != nil {
		return err
	}
	return s.validateTransportSettings(settings)
}

// StartHTTPTransport starts the shared Triple listener without requiring any
// exported service handlers first. This is used by mount-first and HTTP-only
// flows to bring up the listener before service export happens later.
func (s *Server) StartHTTPTransport(url *common.URL) error {
	settings, err := resolveServerTransport(url)
	if err != nil {
		return err
	}

	// Reuse the same transport bootstrap path as normal service export so mount-
	// first and export-first flows converge on one listener lifecycle.
	if err := s.ensureTriServer(settings); err != nil {
		return err
	}
	s.startTransport(settings.callProtocol, settings.tlsConfig)
	return nil
}

func (s *Server) startTransport(callProtocol string, tlsConf *tls.Config) {
	if s.transportStarted {
		return
	}
	s.transportStarted = true

	go func() {
		if runErr := s.triServer.Run(callProtocol, tlsConf); runErr != nil {
			logger.Errorf("server serve failed with err: %v", runErr)
		}
	}()
}

func (s *Server) registerServiceHandlers(invoker base.Invoker, info *common.ServiceInfo, hanOpts []tri.HandlerOption) {
	url := invoker.GetURL()

	// IDLMode means that this will only be set when
	// the new triple is started in non-IDL mode.
	// TODO: remove IDLMode when config package is removed
	IDLMode := url.GetParam(constant.IDLMode, "")

	var service common.RPCService
	if IDLMode == constant.NONIDL {
		service, _ = url.GetAttribute(constant.RpcServiceKey)
	}

	intfName := url.Interface()
	//OpenAPI group
	var openapiGroup string
	if g, ok := url.GetAttribute(constant.OpenAPIMetaKeyOpenAPIGroup); ok {
		if gs, ok := g.(string); ok && gs != "" {
			openapiGroup = gs
		}
	}

	if info != nil {
		// new triple idl mode
		s.handleServiceWithInfo(intfName, invoker, info, hanOpts...)
		s.saveServiceInfo(intfName, info, openapiGroup, url.Group(), url.Version())
	} else if IDLMode == constant.NONIDL {
		// new triple non-idl mode
		reflectInfo := createServiceInfoWithReflection(service)
		s.handleServiceWithInfo(intfName, invoker, reflectInfo, hanOpts...)
		s.saveServiceInfo(intfName, reflectInfo, openapiGroup, url.Group(), url.Version())
	} else {
		s.compatHandleService(url, intfName, url.Group(), url.Version(), hanOpts...)
	}
}

// RefreshService refreshes Triple service.
// The exported wrapper preserves the historical panic-on-invalid-export behavior
// for direct callers, while internal paths use refreshService to surface errors
// so export and mount flows can share one explicit failure path.
func (s *Server) RefreshService(invoker base.Invoker, info *common.ServiceInfo) {
	if err := s.refreshService(invoker, info); err != nil {
		panic(err)
	}
}

func (s *Server) refreshService(invoker base.Invoker, info *common.ServiceInfo) error {
	url := invoker.GetURL()

	hanOpts, err := resolveHandlerOptions(url)
	if err != nil {
		return err
	}

	s.registerServiceHandlers(invoker, info, hanOpts)
	return nil
}

// validateTransportSettings only guards listener-fixed settings. Handler-level
// Triple options may still vary per exported service, but protocol and TLS
// configuration must remain stable once one listener has been created.
func (s *Server) validateTransportSettings(next *transportSettings) error {
	if next == nil {
		return nil
	}

	s.mu.RLock()
	current := s.transportSettings
	s.mu.RUnlock()
	if current == nil {
		return nil
	}

	if current.callProtocol != next.callProtocol {
		return fmt.Errorf("triple transport at %s already uses protocol %s, cannot switch to %s", next.location, current.callProtocol, next.callProtocol)
	}
	if !reflect.DeepEqual(current.rawTLSConfig, next.rawTLSConfig) {
		return fmt.Errorf("triple transport at %s already uses different TLS settings", next.location)
	}
	return nil
}

func resolveTripleConfig(url *common.URL) (*global.TripleConfig, error) {
	if url == nil {
		return nil, fmt.Errorf("triple server url must not be nil")
	}

	tripleConfRaw, ok := url.GetAttribute(constant.TripleConfigKey)
	if !ok || tripleConfRaw == nil {
		return nil, nil
	}

	tripleConf, ok := tripleConfRaw.(*global.TripleConfig)
	if !ok {
		return nil, fmt.Errorf("invalid triple config type %T", tripleConfRaw)
	}
	return tripleConf.Clone(), nil
}

func resolveRawTLSConfig(url *common.URL) (*global.TLSConfig, error) {
	if url == nil {
		return nil, fmt.Errorf("triple server url must not be nil")
	}

	tlsConfRaw, ok := url.GetAttribute(constant.TLSConfigKey)
	if !ok || tlsConfRaw == nil {
		return nil, nil
	}

	tlsConf, ok := tlsConfRaw.(*global.TLSConfig)
	if !ok {
		return nil, fmt.Errorf("invalid tls config type %T", tlsConfRaw)
	}
	return tlsConf.Clone(), nil
}

func resolveHandlerOptions(url *common.URL) ([]tri.HandlerOption, error) {
	if url == nil {
		return nil, fmt.Errorf("triple server url must not be nil")
	}

	serialization := url.GetParam(constant.SerializationKey, constant.ProtobufSerialization)
	switch serialization {
	case constant.ProtobufSerialization:
	case constant.JSONSerialization:
	case constant.Hessian2Serialization:
	case constant.MsgpackSerialization:
	default:
		return nil, fmt.Errorf("unsupported serialization: %s", serialization)
	}

	tripleConf, err := resolveTripleConfig(url)
	if err != nil {
		return nil, err
	}

	hanOpts := getHanOpts(url, tripleConf)
	hanOpts = append(hanOpts, tri.WithExpectedCodecName(serialization))
	return hanOpts, nil
}

func getHanOpts(url *common.URL, tripleConf *global.TripleConfig) (hanOpts []tri.HandlerOption) {
	group := url.GetParam(constant.GroupKey, "")
	version := url.GetParam(constant.VersionKey, "")
	hanOpts = append(hanOpts, tri.WithGroup(group), tri.WithVersion(version))

	// Deprecated：use TripleConfig
	// TODO: remove MaxServerSendMsgSize and MaxServerRecvMsgSize when version 4.0.0
	maxServerRecvMsgSize := constant.DefaultMaxServerRecvMsgSize
	if recvMsgSize, convertErr := humanize.ParseBytes(url.GetParam(constant.MaxServerRecvMsgSize, "")); convertErr == nil && recvMsgSize != 0 {
		maxServerRecvMsgSize = int(recvMsgSize)
	}
	hanOpts = append(hanOpts, tri.WithReadMaxBytes(maxServerRecvMsgSize))

	// Deprecated：use TripleConfig
	// TODO: remove MaxServerSendMsgSize and MaxServerRecvMsgSize when version 4.0.0
	maxServerSendMsgSize := constant.DefaultMaxServerSendMsgSize
	if sendMsgSize, convertErr := humanize.ParseBytes(url.GetParam(constant.MaxServerSendMsgSize, "")); convertErr == nil && sendMsgSize != 0 {
		maxServerSendMsgSize = int(sendMsgSize)
	}
	hanOpts = append(hanOpts, tri.WithSendMaxBytes(maxServerSendMsgSize))

	if tripleConf == nil {
		return hanOpts
	}

	if tripleConf.MaxServerRecvMsgSize != "" {
		logger.Debugf("MaxServerRecvMsgSize: %v", tripleConf.MaxServerRecvMsgSize)
		if recvMsgSize, convertErr := humanize.ParseBytes(tripleConf.MaxServerRecvMsgSize); convertErr == nil && recvMsgSize != 0 {
			maxServerRecvMsgSize = int(recvMsgSize)
		}
		hanOpts = append(hanOpts, tri.WithReadMaxBytes(maxServerRecvMsgSize))
	}

	if tripleConf.MaxServerSendMsgSize != "" {
		logger.Debugf("MaxServerSendMsgSize: %v", tripleConf.MaxServerSendMsgSize)
		if sendMsgSize, convertErr := humanize.ParseBytes(tripleConf.MaxServerSendMsgSize); convertErr == nil && sendMsgSize != 0 {
			maxServerSendMsgSize = int(sendMsgSize)
		}
		hanOpts = append(hanOpts, tri.WithSendMaxBytes(maxServerSendMsgSize))
	}

	// todo:// open tracing

	// CORS configuration
	if tripleConf.Cors != nil && len(tripleConf.Cors.AllowOrigins) > 0 {
		hanOpts = append(hanOpts, tri.WithCORS(&tri.CorsConfig{
			AllowOrigins:     tripleConf.Cors.AllowOrigins,
			AllowMethods:     tripleConf.Cors.AllowMethods,
			AllowHeaders:     tripleConf.Cors.AllowHeaders,
			ExposeHeaders:    tripleConf.Cors.ExposeHeaders,
			AllowCredentials: tripleConf.Cors.AllowCredentials,
			MaxAge:           tripleConf.Cors.MaxAge,
		}))
	}

	return hanOpts
}

// *Important*, this function is responsible for being compatible with old triple-gen code and non-idl code
// compatHandleService registers handler based on ServiceConfig and provider service.
func (s *Server) compatHandleService(url *common.URL, interfaceName string, group, version string, opts ...tri.HandlerOption) {
	var providerServices map[string]*global.ServiceConfig
	if providerConfRaw, ok := url.GetAttribute(constant.ProviderConfigKey); ok {
		if providerConf, ok := providerConfRaw.(*global.ProviderConfig); ok && providerConf != nil {
			providerServices = providerConf.Services
		}
	}
	if len(providerServices) == 0 {
		logger.Info("Provider service map is null, please register ProviderServices")
		return
	}
	for key, providerService := range providerServices {
		if providerService.Interface != interfaceName || providerService.Group != group || providerService.Version != version {
			continue
		}
		service, _ := url.GetAttribute(constant.RpcServiceKey)
		if service == nil {
			logger.Warnf("no rpc service found for key: %v", key)
			continue
		}
		serviceKey := common.ServiceKey(providerService.Interface, providerService.Group, providerService.Version)
		exporter, _ := tripleProtocol.ExporterMap().Load(serviceKey)
		if exporter == nil {
			logger.Warnf("no exporter found for serviceKey: %v", serviceKey)
			continue
		}
		invoker := exporter.(base.Exporter).GetInvoker()
		if invoker == nil {
			panic(fmt.Sprintf("no invoker found for servicekey: %v", serviceKey))
		}
		ds, ok := service.(dubbo3.Dubbo3GrpcService)
		if !ok {
			info := createServiceInfoWithReflection(service)
			s.handleServiceWithInfo(interfaceName, invoker, info, opts...)
			s.saveServiceInfo(interfaceName, info, "", "", "")
			continue
		}
		s.compatSaveServiceInfo(ds.XXX_ServiceDesc())
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

// handleServiceWithInfo injects invoker and creates handlers based on ServiceInfo.
// Each method is registered once under its canonical procedure path. Triple's
// transport-layer route mux performs case-insensitive fallback matching.
func (s *Server) handleServiceWithInfo(interfaceName string, invoker base.Invoker, info *common.ServiceInfo, opts ...tri.HandlerOption) {
	for _, method := range info.Methods {
		m := method
		procedure := joinProcedure(interfaceName, method.Name)
		s.registerMethodHandler(procedure, m, invoker, opts...)
	}
}

// registerMethodHandler registers a single method handler for the given procedure path.
func (s *Server) registerMethodHandler(procedure string, m common.MethodInfo, invoker base.Invoker, opts ...tri.HandlerOption) {
	switch m.Type {
	case constant.CallUnary:
		s.registerUnaryMethodHandler(procedure, m, invoker, opts...)
	case constant.CallClientStream:
		s.registerClientStreamMethodHandler(procedure, m, invoker, opts...)
	case constant.CallServerStream:
		s.registerServerStreamMethodHandler(procedure, m, invoker, opts...)
	case constant.CallBidiStream:
		s.registerBidiStreamMethodHandler(procedure, m, invoker, opts...)
	}
}

func (s *Server) registerUnaryMethodHandler(procedure string, m common.MethodInfo, invoker base.Invoker, opts ...tri.HandlerOption) {
	_ = s.triServer.RegisterUnaryHandler(
		procedure,
		m.ReqInitFunc,
		func(ctx context.Context, req *tri.Request) (*tri.Response, error) {
			args := extractUnaryInvocationArgs(req.Msg)
			attachments := generateAttachments(req.Header())
			// inject attachments
			ctx = context.WithValue(ctx, constant.AttachmentKey, attachments)
			invo := invocation.NewRPCInvocation(m.Name, args, attachments)
			res := invoker.Invoke(ctx, invo)
			// todo(DMwangnima): modify InfoInvoker to get a unified processing logic
			// please refer to server/InfoInvoker.Invoke()
			triResp := wrapTripleResponse(res.Result())
			appendTripleOutgoingAttachments(ctx, res.Attachments())
			return triResp, res.Error()
		},
		opts...,
	)
}

func (s *Server) registerClientStreamMethodHandler(procedure string, m common.MethodInfo, invoker base.Invoker, opts ...tri.HandlerOption) {
	_ = s.triServer.RegisterClientStreamHandler(
		procedure,
		func(ctx context.Context, stream *tri.ClientStream) (*tri.Response, error) {
			args := []any{m.StreamInitFunc(stream)}
			attachments := generateAttachments(stream.RequestHeader())
			// inject attachments
			ctx = context.WithValue(ctx, constant.AttachmentKey, attachments)
			invo := invocation.NewRPCInvocation(m.Name, args, attachments)
			res := invoker.Invoke(ctx, invo)
			return wrapTripleResponse(res.Result()), res.Error()
		},
		opts...,
	)
}

func (s *Server) registerServerStreamMethodHandler(procedure string, m common.MethodInfo, invoker base.Invoker, opts ...tri.HandlerOption) {
	_ = s.triServer.RegisterServerStreamHandler(
		procedure,
		m.ReqInitFunc,
		func(ctx context.Context, req *tri.Request, stream *tri.ServerStream) error {
			args := []any{req.Msg, m.StreamInitFunc(stream)}
			attachments := generateAttachments(req.Header())
			// inject attachments
			ctx = context.WithValue(ctx, constant.AttachmentKey, attachments)
			invo := invocation.NewRPCInvocation(m.Name, args, attachments)
			res := invoker.Invoke(ctx, invo)
			return res.Error()
		},
		opts...,
	)
}

func (s *Server) registerBidiStreamMethodHandler(procedure string, m common.MethodInfo, invoker base.Invoker, opts ...tri.HandlerOption) {
	_ = s.triServer.RegisterBidiStreamHandler(
		procedure,
		func(ctx context.Context, stream *tri.BidiStream) error {
			args := []any{m.StreamInitFunc(stream)}
			attachments := generateAttachments(stream.RequestHeader())
			// inject attachments
			ctx = context.WithValue(ctx, constant.AttachmentKey, attachments)
			invo := invocation.NewRPCInvocation(m.Name, args, attachments)
			res := invoker.Invoke(ctx, invo)
			return res.Error()
		},
		opts...,
	)
}

func extractUnaryInvocationArgs(msg any) []any {
	if argsRaw, ok := msg.([]any); ok {
		args := make([]any, 0, len(argsRaw))
		// non-idl mode, req.Msg consists of many arguments
		for _, argRaw := range argsRaw {
			// refer to createServiceInfoWithReflection, in ReqInitFunc, argRaw is a pointer to real arg.
			// so we have to invoke Elem to get the real arg.
			args = append(args, reflect.ValueOf(argRaw).Elem().Interface())
		}
		return args
	}
	// triple idl mode and old triple idl mode
	return []any{msg}
}

func wrapTripleResponse(result any) *tri.Response {
	if existingResp, ok := result.(*tri.Response); ok {
		return existingResp
	}
	// please refer to proxy/proxy_factory/ProxyInvoker.Invoke
	return tri.NewResponse([]any{result})
}

func appendTripleOutgoingAttachments(ctx context.Context, attachments map[string]any) {
	for k, v := range attachments {
		switch val := v.(type) {
		case string:
			tri.AppendToOutgoingContext(ctx, k, val)
		case []string:
			for _, item := range val {
				tri.AppendToOutgoingContext(ctx, k, item)
			}
		}
	}
}

func (s *Server) saveServiceInfo(interfaceName string, info *common.ServiceInfo, openapiGroup string, dubboGroup string, dubboVersion string) {
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
	// todo(DMwangnima): using interfaceName is not enough, we need to consider group and version
	s.services[interfaceName] = ret

	if s.triServer != nil {
		s.triServer.RegisterOpenAPIService(interfaceName, info, openapiGroup, dubboGroup, dubboVersion)
	}
}

func (s *Server) compatSaveServiceInfo(desc *grpc_go.ServiceDesc) {
	ret := grpc.ServiceInfo{}
	ret.Methods = make([]grpc.MethodInfo, 0, len(desc.Streams)+len(desc.Methods))
	for _, method := range desc.Methods {
		md := grpc.MethodInfo{
			Name:           method.MethodName,
			IsClientStream: false,
			IsServerStream: false,
		}
		ret.Methods = append(ret.Methods, md)
	}
	for _, stream := range desc.Streams {
		md := grpc.MethodInfo{
			Name:           stream.StreamName,
			IsClientStream: stream.ClientStreams,
			IsServerStream: stream.ServerStreams,
		}
		ret.Methods = append(ret.Methods, md)
	}
	ret.Metadata = desc.Metadata
	s.mu.Lock()
	defer s.mu.Unlock()
	s.services[desc.ServiceName] = ret
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
	shutdownCtx, cancel := context.WithTimeout(context.Background(), constant.DefaultGracefulShutdownTimeout)
	defer cancel()

	if err := s.triServer.GracefulStop(shutdownCtx); err != nil {
		logger.Errorf("Triple server shutdown error: %v", err)
	}
}

// createServiceInfoWithReflection is for non-idl scenario.
// It makes use of reflection to extract method parameters information and create ServiceInfo.
// As a result, Server could use this ServiceInfo to register.
func createServiceInfoWithReflection(svc common.RPCService) *common.ServiceInfo {
	var info common.ServiceInfo
	svcType := reflect.TypeOf(svc)
	methodNum := svcType.NumMethod()

	// +1 for generic call method
	methodInfos := make([]common.MethodInfo, 0, methodNum+1)

	for i := range methodNum {
		methodType := svcType.Method(i)
		if methodType.Name == "Reference" {
			continue
		}
		methodInfo := buildMethodInfoWithReflection(methodType)
		if methodInfo != nil {
			methodInfos = append(methodInfos, *methodInfo)
		}
	}

	// Add $invoke method for generic call support
	methodInfos = append(methodInfos, buildGenericMethodInfo())

	info.Methods = methodInfos
	return &info
}

// buildMethodInfoWithReflection creates MethodInfo for a single method using reflection.
func buildMethodInfoWithReflection(methodType reflect.Method) *common.MethodInfo {
	paramsNum := methodType.Type.NumIn()
	// the first param is receiver itself, the second param is ctx
	if paramsNum < 2 {
		logger.Fatalf("TRIPLE does not support %s method that does not have any parameter", methodType.Name)
		return nil
	}

	// Extract parameter types (skip receiver and context)
	paramsTypes := make([]reflect.Type, paramsNum-2)
	for j := 2; j < paramsNum; j++ {
		paramsTypes[j-2] = methodType.Type.In(j)
	}

	// Extract return types for OpenAPI schema generation.
	// Only record response.type when the signature is a reliable unary shape:
	// exactly 2 return values where the second implements error.
	// This avoids:
	//   - methods returning only error getting a synthetic response type
	//   - non-standard signatures producing misleading OpenAPI schemas
	returnsNum := methodType.Type.NumOut()
	var respType reflect.Type
	if returnsNum == 2 {
		errorType := reflect.TypeOf((*error)(nil)).Elem()
		if methodType.Type.Out(1).Implements(errorType) {
			respType = methodType.Type.Out(0)
		}
	}

	// Build Meta for OpenAPI schema generation.
	// Only set request.type / response.type when they can be determined reliably.
	meta := make(map[string]any)
	if len(paramsTypes) == 1 {
		meta["request.type"] = paramsTypes[0]
	}
	if respType != nil {
		meta["response.type"] = respType
	}

	// Capture method for closure
	method := methodType
	return &common.MethodInfo{
		Name: methodType.Name,
		Type: constant.CallUnary, // only support Unary invocation now
		Meta: meta,
		ReqInitFunc: func() any {
			params := make([]any, len(paramsTypes))
			for k, paramType := range paramsTypes {
				params[k] = reflect.New(paramType).Interface()
			}
			return params
		},
		MethodFunc: func(ctx context.Context, args []any, handler any) (any, error) {
			return callMethodByReflection(ctx, method, handler, args)
		},
	}
}

// buildGenericMethodInfo creates MethodInfo for $invoke generic call method.
func buildGenericMethodInfo() common.MethodInfo {
	return common.MethodInfo{
		Name: constant.Generic,
		Type: constant.CallUnary,
		ReqInitFunc: func() any {
			return []any{
				func(s string) *string { return &s }(""), // methodName *string
				&[]string{},                              // types *[]string
				&[]hessian.Object{},                      // args *[]hessian.Object
			}
		},
	}
}

// isReflectValueNil safely checks if a reflect.Value is nil.
// It first checks if the value's kind supports nil checking to avoid panic.
func isReflectValueNil(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice, reflect.UnsafePointer:
		return v.IsNil()
	default:
		return false
	}
}

func callMethodByReflection(ctx context.Context, method reflect.Method, handler any, args []any) (any, error) {
	in := []reflect.Value{reflect.ValueOf(handler)}
	in = append(in, reflect.ValueOf(ctx))
	for _, arg := range args {
		in = append(in, reflect.ValueOf(arg))
	}

	var returnValues []reflect.Value
	if shouldUseGenericVariadicCallSlice(ctx, method, args) {
		returnValues = method.Func.CallSlice(in)
	} else {
		returnValues = method.Func.Call(in)
	}

	if len(returnValues) == 1 {
		if isReflectValueNil(returnValues[0]) {
			return nil, nil
		}
		if err, ok := returnValues[0].Interface().(error); ok {
			return nil, err
		}
		return nil, nil
	}
	var result any
	var err error
	if !isReflectValueNil(returnValues[0]) {
		result = returnValues[0].Interface()
	}
	if len(returnValues) > 1 && !isReflectValueNil(returnValues[1]) {
		if e, ok := returnValues[1].Interface().(error); ok {
			err = e
		}
	}
	return result, err
}

// shouldUseGenericVariadicCallSlice mirrors the ServiceInfo reflection gate for
// Triple's reflection-based method dispatch.
func shouldUseGenericVariadicCallSlice(ctx context.Context, method reflect.Method, args []any) bool {
	if !method.Type.IsVariadic() || len(args) == 0 || len(args) != method.Type.NumIn()-2 {
		return false
	}

	value, ok := ctx.Value(constant.DubboCtxKey(constant.GenericVariadicCallSliceKey)).(bool)
	if !ok || !value {
		return false
	}

	lastArg := args[len(args)-1]
	if lastArg == nil {
		return false
	}

	lastArgType := reflect.TypeOf(lastArg)
	variadicSliceType := method.Type.In(method.Type.NumIn() - 1)
	return lastArgType.AssignableTo(variadicSliceType) || lastArgType.ConvertibleTo(variadicSliceType)
}

// generateAttachments transfer http.Header to map[string]any and make all keys lowercase
func generateAttachments(header http.Header) map[string]any {
	attachments := make(map[string]any, len(header))
	for key, val := range header {
		lowerKey := strings.ToLower(key)
		attachments[lowerKey] = val
	}

	return attachments
}
