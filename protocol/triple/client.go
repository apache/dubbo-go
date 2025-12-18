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
	isIDL    bool
	triGuard *sync.RWMutex
	// triple_protocol clients
	triClients *TriClientPool
}

// TODO: code a triple client between clientManager and triple_protocol client
// TODO: write a NewClient for triple client

func (cm *clientManager) getClient(method string) (*tri.Client, error) {
	cm.triGuard.RLock()
	defer cm.triGuard.RUnlock()

	if cm.triClients == nil {
		return nil, fmt.Errorf("no client pool found for method: %s", method)
	}
	// Wait up to defaultClientGetTimeout for a pooled client; if none becomes available we return the
	// fallback client and record a timeout which is used to drive autoscaling.
	triClient, err := cm.triClients.Get(constant.DefaultClientGetTimeout)
	if err != nil {
		return nil, fmt.Errorf("missing triple client for method: %s", method)
	}
	if triClient == nil {
		return nil, fmt.Errorf("missing triple client for method: %s", method)
	}
	return triClient, nil
}

func (cm *clientManager) putClient(method string, c *tri.Client) {
	if c == nil {
		return
	}

	cm.triGuard.RLock()
	defer cm.triGuard.RUnlock()

	if cm.triClients == nil {
		logger.Warnf("no client pool found for method %s, dropping client", method)
		return
	}
	cm.triClients.Put(c)
}

func (cm *clientManager) callUnary(ctx context.Context, method string, req, resp any) error {
	triClient, err := cm.getClient(method)
	if err != nil {
		return err
	}
	defer cm.putClient(method, triClient)
	triReq := tri.NewRequest(req)
	triResp := tri.NewResponse(resp)
	if err := triClient.CallUnary(ctx, triReq, triResp, method); err != nil {
		return err
	}
	return nil
}

func (cm *clientManager) callClientStream(ctx context.Context, method string) (any, error) {
	triClient, err := cm.getClient(method)
	if err != nil {
		return nil, err
	}
	defer cm.putClient(method, triClient)
	stream, err := triClient.CallClientStream(ctx, method)
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
	defer cm.putClient(method, triClient)
	triReq := tri.NewRequest(req)
	stream, err := triClient.CallServerStream(ctx, triReq, method)
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
	defer cm.putClient(method, triClient)
	stream, err := triClient.CallBidiStream(ctx, method)
	if err != nil {
		return nil, err
	}
	return stream, nil
}

func (cm *clientManager) close() error {
	cm.triGuard.Lock()
	defer cm.triGuard.Unlock()
	if cm.triClients == nil {
		return ErrTriClientPoolCloseWhenEmpty
	}
	cm.triClients.Close()
	cm.triClients = nil

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

	var callProtocol string
	if tripleConf != nil && tripleConf.Http3 != nil && tripleConf.Http3.Enable {
		callProtocol = constant.CallHTTP2AndHTTP3
	} else {
		// HTTP default type is HTTP/2.
		callProtocol = constant.CallHTTP2
	}

	transport, err := buildTransport(callProtocol, cfg, keepAliveInterval, keepAliveTimeout, tlsFlag)
	if err != nil {
		return nil, err
	}
	httpClient := &http.Client{
		Transport: transport,
	}
	perClientTransport := url.GetParamBool(constant.TriCliPoolPerClientTrans, false)

	var baseTriURL string
	baseTriURL = strings.TrimPrefix(url.Location, httpPrefix)
	baseTriURL = strings.TrimPrefix(baseTriURL, httpsPrefix)
	if tlsFlag {
		baseTriURL = httpsPrefix + baseTriURL
	} else {
		baseTriURL = httpPrefix + baseTriURL
	}

	triURL, err := joinPath(baseTriURL, url.Interface())
	if err != nil {
		return nil, fmt.Errorf("JoinPath failed for base %s, interface %s", baseTriURL, url.Interface())
	}

	pool, err := NewTriClientPool(constant.DefaultClientWarmingUp, constant.TriClientPoolMaxSize, func() *tri.Client {
		if perClientTransport {
			dedicatedTransport, buildErr := buildTransport(callProtocol, cfg, keepAliveInterval, keepAliveTimeout, tlsFlag)
			if buildErr != nil {
				return tri.NewClient(httpClient, triURL, cliOpts...)
			}
			return tri.NewClient(&http.Client{Transport: dedicatedTransport}, triURL, cliOpts...)
		}
		return tri.NewClient(httpClient, triURL, cliOpts...)
	})

	if err != nil {
		return nil, err
	}

	return &clientManager{
		isIDL:      isIDL,
		triGuard:   &sync.RWMutex{},
		triClients: pool,
	}, nil
}

func buildTransport(callProtocol string, cfg *tls.Config, keepAliveInterval, keepAliveTimeout time.Duration, tlsFlag bool) (http.RoundTripper, error) {
	switch callProtocol {
	// This case might be for backward compatibility,
	// it's not useful for the Triple protocol, HTTP/1 lacks trailer functionality.
	// Triple protocol only supports HTTP/2 and HTTP/3.
	case constant.CallHTTP:
		return &http.Transport{
			TLSClientConfig: cfg,
		}, nil
	case constant.CallHTTP2:
		// TODO: Enrich the http2 transport config for triple protocol.
		if tlsFlag {
			return &http2.Transport{
				TLSClientConfig: cfg,
				ReadIdleTimeout: keepAliveInterval,
				PingTimeout:     keepAliveTimeout,
			}, nil
		}
		return &http2.Transport{
			DialTLSContext: func(_ context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
				return net.Dial(network, addr)
			},
			AllowHTTP:       true,
			ReadIdleTimeout: keepAliveInterval,
			PingTimeout:     keepAliveTimeout,
		}, nil
	case constant.CallHTTP3:
		if !tlsFlag {
			return nil, fmt.Errorf("TRIPLE http3 client must have TLS config, but TLS config is nil")
		}
		logger.Infof("Triple http3 client transport init successfully")
		// TODO: Enrich the http3 transport config for triple protocol.
		return &http3.Transport{
			TLSClientConfig: cfg,
			QUICConfig: &quic.Config{
				// ref: https://quic-go.net/docs/quic/connection/#keeping-a-connection-alive
				KeepAlivePeriod: keepAliveInterval,
				// ref: https://quic-go.net/docs/quic/connection/#idle-timeout
				MaxIdleTimeout: keepAliveTimeout,
			},
		}, nil
	case constant.CallHTTP2AndHTTP3:
		if !tlsFlag {
			return nil, fmt.Errorf("TRIPLE HTTP/2 and HTTP/3 client must have TLS config, but TLS config is nil")
		}
		logger.Infof("Triple HTTP/2 and HTTP/3 client transport init successfully")
		// Create a dual transport that can handle both HTTP/2 and HTTP/3
		return newDualTransport(cfg, keepAliveInterval, keepAliveTimeout), nil
	default:
		return nil, fmt.Errorf("unsupported http protocol: %s", callProtocol)
	}
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
	altSvcCache *tri.AltSvcCache
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
		altSvcCache:    tri.NewAltSvcCache(),
	}
}

// RoundTrip implements http.RoundTripper interface with HTTP Alternative Services support
func (dt *dualTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Check if we have cached alternative service information
	cachedAltSvc := dt.altSvcCache.Get(req.URL.Host)

	// If we have valid cached alt-svc info and it's for HTTP/3, try HTTP/3 first
	// Check if the cached information is still valid (not expired)
	if cachedAltSvc != nil && cachedAltSvc.Protocol == "h3" {
		logger.Debugf("Using cached HTTP/3 alternative service for %s", req.URL.String())
		resp, err := dt.http3Transport.RoundTrip(req)
		if err == nil {
			// Update alt-svc cache from response headers
			dt.altSvcCache.UpdateFromHeaders(req.URL.Host, resp.Header)
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
	dt.altSvcCache.UpdateFromHeaders(req.URL.Host, resp.Header)

	// If the response indicates HTTP/3 is available, try HTTP/3 for future requests
	if altSvc := dt.altSvcCache.Get(req.URL.Host); altSvc != nil && altSvc.Protocol == "h3" {
		logger.Debugf("Server %s supports HTTP/3, will use HTTP/3 for future requests", req.URL.Host)
	}

	return resp, nil
}
