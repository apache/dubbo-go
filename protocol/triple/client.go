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
	"errors"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"github.com/dustin/go-humanize"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"

	"golang.org/x/net/http2"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
	tri "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
	dubbotls "dubbo.apache.org/dubbo-go/v3/tls"
)

const (
	httpPrefix  string = "http://"
	httpsPrefix string = "https://"
)

// clientManager wraps triple clients and is responsible for find concrete triple client to invoke
// callUnary, callClientStream, callServerStream, callBidiStream.
// A Reference has a clientManager.
type clientManager struct {
	isIDL bool
	// triple_protocol clients, key is method name
	triClients map[string]*tri.Client
}

// TODO: code a triple client between clientManager and triple_protocol client
// TODO: write a NewClient for triple client

func (cm *clientManager) getClient(method string) (*tri.Client, error) {
	triClient, ok := cm.triClients[method]
	if !ok {
		return nil, fmt.Errorf("missing triple client for method: %s", method)
	}
	return triClient, nil
}

func (cm *clientManager) callUnary(ctx context.Context, method string, req, resp any) error {
	triClient, err := cm.getClient(method)
	if err != nil {
		return err
	}
	triReq := tri.NewRequest(req)
	triResp := tri.NewResponse(resp)
	if err := triClient.CallUnary(ctx, triReq, triResp); err != nil {
		return err
	}
	return nil
}

func (cm *clientManager) callClientStream(ctx context.Context, method string) (any, error) {
	triClient, err := cm.getClient(method)
	if err != nil {
		return nil, err
	}
	stream, err := triClient.CallClientStream(ctx)
	if err != nil {
		return nil, err
	}
	return stream, nil
}

func (cm *clientManager) callServerStream(ctx context.Context, method string, req any) (any, error) {
	triClient, err := cm.getClient(method)
	if err != nil {
		return nil, err
	}
	triReq := tri.NewRequest(req)
	stream, err := triClient.CallServerStream(ctx, triReq)
	if err != nil {
		return nil, err
	}
	return stream, nil
}

func (cm *clientManager) callBidiStream(ctx context.Context, method string) (any, error) {
	triClient, err := cm.getClient(method)
	if err != nil {
		return nil, err
	}
	stream, err := triClient.CallBidiStream(ctx)
	if err != nil {
		return nil, err
	}
	return stream, nil
}

func (cm *clientManager) close() error {
	// There is no need to release resources right now.
	// But we leave this function here for future use.
	return nil
}

// newClientManager extracts configurations from url and builds clientManager
func newClientManager(url *common.URL) (*clientManager, error) {
	var cliOpts []tri.ClientOption
	var isIDL bool

	// set serialization
	serialization := url.GetParam(constant.SerializationKey, constant.ProtobufSerialization)
	switch serialization {
	case constant.ProtobufSerialization:
		isIDL = true
	case constant.JSONSerialization:
		isIDL = true
		cliOpts = append(cliOpts, tri.WithProtoJSON())
	case constant.Hessian2Serialization:
		cliOpts = append(cliOpts, tri.WithHessian2())
	case constant.MsgpackSerialization:
		cliOpts = append(cliOpts, tri.WithMsgPack())
	default:
		panic(fmt.Sprintf("Unsupported serialization: %s", serialization))
	}

	// set timeout
	timeout := url.GetParamDuration(constant.TimeoutKey, "")
	cliOpts = append(cliOpts, tri.WithTimeout(timeout))

	// set service group and version
	group := url.GetParam(constant.GroupKey, "")
	version := url.GetParam(constant.VersionKey, "")
	cliOpts = append(cliOpts, tri.WithGroup(group), tri.WithVersion(version))

	// todo(DMwangnima): support opentracing

	// handle tls
	var (
		tlsFlag bool
		tlsConf *global.TLSConfig
		cfg     *tls.Config
		err     error
	)

	tlsConfRaw, ok := url.GetAttribute(constant.TLSConfigKey)
	if ok {
		tlsConf, ok = tlsConfRaw.(*global.TLSConfig)
		if !ok {
			return nil, errors.New("TRIPLE clientManager initialized the TLSConfig configuration failed")
		}
	}
	if dubbotls.IsClientTLSValid(tlsConf) {
		cfg, err = dubbotls.GetClientTlSConfig(tlsConf)
		if err != nil {
			return nil, err
		}
		if cfg != nil {
			logger.Infof("TRIPLE clientManager initialized the TLSConfig configuration")
			tlsFlag = true
		}
	}

	var tripleConf *global.TripleConfig

	tripleConfRaw, ok := url.GetAttribute(constant.TripleConfigKey)
	if ok {
		tripleConf = tripleConfRaw.(*global.TripleConfig)
	}

	// handle keepalive options
	cliKeepAliveOpts, keepAliveInterval, keepAliveTimeout, genKeepAliveOptsErr := genKeepAliveOptions(url, tripleConf)
	if genKeepAliveOptsErr != nil {
		logger.Errorf("genKeepAliveOpts err: %v", genKeepAliveOptsErr)
		return nil, genKeepAliveOptsErr
	}
	cliOpts = append(cliOpts, cliKeepAliveOpts...)

	// handle http transport of triple protocol
	var transport http.RoundTripper

	var callProtocol string
	if tripleConf != nil && tripleConf.Http3 != nil && tripleConf.Http3.Enable {
		callProtocol = constant.CallHTTP2AndHTTP3
	} else {
		// HTTP default type is HTTP/2.
		callProtocol = constant.CallHTTP2
	}

	switch callProtocol {
	// This case might be for backward compatibility,
	// it's not useful for the Triple protocol, HTTP/1 lacks trailer functionality.
	// Triple protocol only supports HTTP/2 and HTTP/3.
	case constant.CallHTTP:
		transport = &http.Transport{
			TLSClientConfig: cfg,
		}
		cliOpts = append(cliOpts, tri.WithTriple())
	case constant.CallHTTP2:
		// TODO: Enrich the http2 transport config for triple protocol.
		if tlsFlag {
			transport = &http2.Transport{
				TLSClientConfig: cfg,
				ReadIdleTimeout: keepAliveInterval,
				PingTimeout:     keepAliveTimeout,
			}
		} else {
			transport = &http2.Transport{
				DialTLSContext: func(_ context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
					return net.Dial(network, addr)
				},
				AllowHTTP:       true,
				ReadIdleTimeout: keepAliveInterval,
				PingTimeout:     keepAliveTimeout,
			}
		}
	case constant.CallHTTP3:
		if !tlsFlag {
			return nil, fmt.Errorf("TRIPLE http3 client must have TLS config, but TLS config is nil")
		}

		// TODO: Enrich the http3 transport config for triple protocol.
		transport = &http3.Transport{
			TLSClientConfig: cfg,
			QUICConfig: &quic.Config{
				// ref: https://quic-go.net/docs/quic/connection/#keeping-a-connection-alive
				KeepAlivePeriod: keepAliveInterval,
				// ref: https://quic-go.net/docs/quic/connection/#idle-timeout
				MaxIdleTimeout: keepAliveTimeout,
			},
		}

		logger.Infof("Triple http3 client transport init successfully")
	case constant.CallHTTP2AndHTTP3:
		if !tlsFlag {
			return nil, fmt.Errorf("TRIPLE HTTP/2 and HTTP/3 client must have TLS config, but TLS config is nil")
		}

		// Create a dual transport that can handle both HTTP/2 and HTTP/3
		transport = newDualTransport(cfg, keepAliveInterval, keepAliveTimeout)
		logger.Infof("Triple HTTP/2 and HTTP/3 client transport init successfully")
	default:
		return nil, fmt.Errorf("unsupported http protocol: %s", callProtocol)
	}

	httpClient := &http.Client{
		Transport: transport,
	}

	var baseTriURL string
	baseTriURL = strings.TrimPrefix(url.Location, httpPrefix)
	baseTriURL = strings.TrimPrefix(baseTriURL, httpsPrefix)
	if tlsFlag {
		baseTriURL = httpsPrefix + baseTriURL
	} else {
		baseTriURL = httpPrefix + baseTriURL
	}

	triClients := make(map[string]*tri.Client)

	if len(url.Methods) != 0 {
		for _, method := range url.Methods {
			triURL, err := joinPath(baseTriURL, url.Interface(), method)
			if err != nil {
				return nil, fmt.Errorf("JoinPath failed for base %s, interface %s, method %s", baseTriURL, url.Interface(), method)
			}
			triClient := tri.NewClient(httpClient, triURL, cliOpts...)
			triClients[method] = triClient
		}
	} else {
		// This branch is for the non-IDL mode, where we pass in the service solely
		// for the purpose of using reflection to obtain all methods of the service.
		// There might be potential for optimization in this area later on.
		service, ok := url.GetAttribute(constant.RpcServiceKey)
		if !ok {
			return nil, fmt.Errorf("triple clientmanager can't get methods")
		}

		serviceType := reflect.TypeOf(service)
		for i := range serviceType.NumMethod() {
			methodName := serviceType.Method(i).Name
			triURL, err := joinPath(baseTriURL, url.Interface(), methodName)
			if err != nil {
				return nil, fmt.Errorf("JoinPath failed for base %s, interface %s, method %s", baseTriURL, url.Interface(), methodName)
			}
			triClient := tri.NewClient(httpClient, triURL, cliOpts...)
			triClients[methodName] = triClient
		}
	}

	return &clientManager{
		isIDL:      isIDL,
		triClients: triClients,
	}, nil
}

func genKeepAliveOptions(url *common.URL, tripleConf *global.TripleConfig) ([]tri.ClientOption, time.Duration, time.Duration, error) {
	var cliKeepAliveOpts []tri.ClientOption

	// set max send and recv msg size
	maxCallRecvMsgSize := constant.DefaultMaxCallRecvMsgSize
	if recvMsgSize, err := humanize.ParseBytes(url.GetParam(constant.MaxCallRecvMsgSize, "")); err == nil && recvMsgSize > 0 {
		maxCallRecvMsgSize = int(recvMsgSize)
	}
	cliKeepAliveOpts = append(cliKeepAliveOpts, tri.WithReadMaxBytes(maxCallRecvMsgSize))
	maxCallSendMsgSize := constant.DefaultMaxCallSendMsgSize
	if sendMsgSize, err := humanize.ParseBytes(url.GetParam(constant.MaxCallSendMsgSize, "")); err == nil && sendMsgSize > 0 {
		maxCallSendMsgSize = int(sendMsgSize)
	}
	cliKeepAliveOpts = append(cliKeepAliveOpts, tri.WithSendMaxBytes(maxCallSendMsgSize))

	// set keepalive interval and keepalive timeout
	// Deprecated：use tripleconfig
	// TODO: remove KeepAliveInterval and KeepAliveInterval in version 4.0.0
	keepAliveInterval := url.GetParamDuration(constant.KeepAliveInterval, constant.DefaultKeepAliveInterval)
	keepAliveTimeout := url.GetParamDuration(constant.KeepAliveTimeout, constant.DefaultKeepAliveTimeout)

	if tripleConf == nil {
		return cliKeepAliveOpts, keepAliveInterval, keepAliveTimeout, nil
	}

	var parseErr error

	if tripleConf.KeepAliveInterval != "" {
		keepAliveInterval, parseErr = time.ParseDuration(tripleConf.KeepAliveInterval)
		if parseErr != nil {
			return nil, 0, 0, parseErr
		}
	}
	if tripleConf.KeepAliveTimeout != "" {
		keepAliveTimeout, parseErr = time.ParseDuration(tripleConf.KeepAliveTimeout)
		if parseErr != nil {
			return nil, 0, 0, parseErr
		}
	}

	return cliKeepAliveOpts, keepAliveInterval, keepAliveTimeout, nil
}

// dualTransport is a transport that can handle both HTTP/2 and HTTP/3
// It uses HTTP Alternative Services (Alt-Svc) for protocol negotiation
type dualTransport struct {
	http2Transport *http2.Transport
	http3Transport *http3.Transport
	// Cache for alternative services to avoid repeated lookups
	altSvcCache map[string]*altSvcInfo
	// Mutex for thread-safe access to altSvcCache
	altSvcMu sync.RWMutex
}

// altSvcInfo represents cached alternative service information
type altSvcInfo struct {
	protocol string // "h3" for HTTP/3, "h2" for HTTP/2
	host     string
	port     string
	expires  time.Time
}

// newDualTransport creates a new dual transport that supports both HTTP/2 and HTTP/3
func newDualTransport(tlsConfig *tls.Config, keepAliveInterval, keepAliveTimeout time.Duration) http.RoundTripper {
	http2Transport := &http2.Transport{
		TLSClientConfig: tlsConfig,
		ReadIdleTimeout: keepAliveInterval,
		PingTimeout:     keepAliveTimeout,
	}

	http3Transport := &http3.Transport{
		TLSClientConfig: tlsConfig,
		QUICConfig: &quic.Config{
			KeepAlivePeriod: keepAliveInterval,
			MaxIdleTimeout:  keepAliveTimeout,
		},
	}

	return &dualTransport{
		http2Transport: http2Transport,
		http3Transport: http3Transport,
		altSvcCache:    make(map[string]*altSvcInfo),
	}
}

// RoundTrip implements http.RoundTripper interface with HTTP Alternative Services support
func (dt *dualTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Check if we have cached alternative service information
	dt.altSvcMu.RLock()
	cachedAltSvc, exists := dt.altSvcCache[req.URL.Host]
	dt.altSvcMu.RUnlock()

	// If we have valid cached alt-svc info and it's for HTTP/3, try HTTP/3 first
	if exists && cachedAltSvc.expires.After(time.Now()) && cachedAltSvc.protocol == "h3" {
		logger.Debugf("Using cached HTTP/3 alternative service for %s", req.URL.String())
		resp, err := dt.http3Transport.RoundTrip(req)
		if err == nil {
			// Update alt-svc cache from response headers
			dt.updateAltSvcCache(req.URL.Host, resp.Header)
			return resp, nil
		}
		logger.Debugf("Cached HTTP/3 request failed to %s, falling back to HTTP/2: %v", req.URL.String(), err)
	}

	// Start with HTTP/2 to get alternative service information
	logger.Debugf("Making initial HTTP/2 request to %s to discover alternative services", req.URL.String())
	resp, err := dt.http2Transport.RoundTrip(req)
	if err != nil {
		logger.Errorf("HTTP/2 request failed to %s: %v", req.URL.String(), err)
		return nil, err
	}

	// Check for alternative services in the response
	dt.updateAltSvcCache(req.URL.Host, resp.Header)

	// If the response indicates HTTP/3 is available, try HTTP/3 for future requests
	if altSvc := dt.getAltSvcForHost(req.URL.Host); altSvc != nil && altSvc.protocol == "h3" {
		logger.Debugf("Server %s supports HTTP/3, will use HTTP/3 for future requests", req.URL.Host)
	}

	return resp, nil
}

// updateAltSvcCache updates the alternative service cache based on response headers
func (dt *dualTransport) updateAltSvcCache(host string, headers http.Header) {
	altSvcHeader := headers.Get("Alt-Svc")
	if altSvcHeader == "" {
		return
	}

	// Parse Alt-Svc header according to RFC 7838
	// Example: "h3=\":443\"; ma=86400, h3-29=\":443\"; ma=86400"
	altSvcs := dt.parseAltSvcHeader(altSvcHeader)
	if len(altSvcs) == 0 {
		return
	}

	// Prefer HTTP/3 over HTTP/2
	var preferredAltSvc *altSvcInfo
	for _, altSvc := range altSvcs {
		if altSvc.protocol == "h3" {
			preferredAltSvc = altSvc
			break
		} else if altSvc.protocol == "h2" && preferredAltSvc == nil {
			preferredAltSvc = altSvc
		}
	}

	if preferredAltSvc != nil {
		dt.altSvcMu.Lock()
		dt.altSvcCache[host] = preferredAltSvc
		dt.altSvcMu.Unlock()
		logger.Debugf("Updated alt-svc cache for %s: %s", host, preferredAltSvc.protocol)
	}
}

// parseAltSvcHeader parses the Alt-Svc header according to RFC 7838
func (dt *dualTransport) parseAltSvcHeader(altSvcHeader string) []*altSvcInfo {
	var altSvcs []*altSvcInfo

	// Split by comma to get individual alternative services
	parts := strings.Split(altSvcHeader, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Parse protocol and authority
		// Format: protocol="authority"; ma=max_age
		// Example: h3=":443"; ma=86400
		altSvc := dt.parseAltSvcPart(part)
		if altSvc != nil {
			altSvcs = append(altSvcs, altSvc)
		}
	}

	return altSvcs
}

// parseAltSvcPart parses a single alternative service part
func (dt *dualTransport) parseAltSvcPart(part string) *altSvcInfo {
	// Find the protocol (before the first '=')
	eqIndex := strings.Index(part, "=")
	if eqIndex == -1 {
		return nil
	}

	protocol := strings.TrimSpace(part[:eqIndex])
	if protocol != "h3" && protocol != "h2" {
		return nil
	}

	// Find the authority (between quotes)
	quoteStart := strings.Index(part, "\"")
	if quoteStart == -1 {
		return nil
	}

	quoteEnd := strings.LastIndex(part, "\"")
	if quoteEnd == -1 || quoteEnd <= quoteStart {
		return nil
	}

	authority := part[quoteStart+1 : quoteEnd]

	// Parse authority to get host and port
	host, port := dt.parseAuthority(authority)

	// Parse max_age if present
	maxAge := 24 * time.Hour // Default to 24 hours
	if maIndex := strings.Index(part, "ma="); maIndex != -1 {
		maPart := part[maIndex+3:]
		if semicolonIndex := strings.Index(maPart, ";"); semicolonIndex != -1 {
			maPart = maPart[:semicolonIndex]
		}
		if age, err := strconv.Atoi(strings.TrimSpace(maPart)); err == nil && age > 0 {
			maxAge = time.Duration(age) * time.Second
		}
	}

	return &altSvcInfo{
		protocol: protocol,
		host:     host,
		port:     port,
		expires:  time.Now().Add(maxAge),
	}
}

// parseAuthority parses the authority part of an alternative service
func (dt *dualTransport) parseAuthority(authority string) (host, port string) {
	// Authority format can be:
	// - ":port" (host is the same as original)
	// - "host:port"
	// - "host" (port is the same as original)

	if strings.HasPrefix(authority, ":") {
		// Just port specified
		port = authority[1:]
		return "", port
	}

	if colonIndex := strings.Index(authority, ":"); colonIndex != -1 {
		// Both host and port specified
		host = authority[:colonIndex]
		port = authority[colonIndex+1:]
		return host, port
	}

	// Just host specified
	host = authority
	return host, ""
}

// getAltSvcForHost returns the cached alternative service for a host
func (dt *dualTransport) getAltSvcForHost(host string) *altSvcInfo {
	dt.altSvcMu.RLock()
	defer dt.altSvcMu.RUnlock()

	altSvc, exists := dt.altSvcCache[host]
	if !exists || altSvc.expires.Before(time.Now()) {
		return nil
	}

	return altSvc
}
