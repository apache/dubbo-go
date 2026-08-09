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
	"time"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"github.com/dustin/go-humanize"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"

	"golang.org/x/net/http2"

	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
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
	isIDL        bool
	triClient    *Client
	healthClient *Client
}

type Client struct {
	delegate *tri.Client
}

func NewClient(httpClient tri.HTTPClient, url string, opts ...tri.ClientOption) *Client {
	return &Client{delegate: tri.NewClient(httpClient, url, opts...)}
}

func (c *Client) CallUnary(ctx context.Context, method string, req, resp any, responseHeader, responseTrailer *http.Header) error {
	triReq := tri.NewRequest(req)
	triResp := tri.NewResponse(resp)
	err := c.delegate.CallUnary(ctx, triReq, method, triResp)
	if responseHeader != nil {
		*responseHeader = triResp.Header().Clone()
	}
	if responseTrailer != nil {
		*responseTrailer = triResp.Trailer().Clone()
	}
	return err
}

func (c *Client) CallClientStream(ctx context.Context, method string) (*tri.ClientStreamForClient, error) {
	return c.delegate.CallClientStream(ctx, method)
}

func (c *Client) CallServerStream(ctx context.Context, method string, req any) (*tri.ServerStreamForClient, error) {
	return c.delegate.CallServerStream(ctx, tri.NewRequest(req), method)
}

func (c *Client) CallBidiStream(ctx context.Context, method string) (*tri.BidiStreamForClient, error) {
	return c.delegate.CallBidiStream(ctx, method)
}

func (cm *clientManager) callUnary(ctx context.Context, method string, req, resp any, responseHeader, responseTrailer *http.Header) error {
	return cm.triClient.CallUnary(ctx, method, req, resp, responseHeader, responseTrailer)
}

func (cm *clientManager) callClientStream(ctx context.Context, method string) (any, error) {
	return cm.triClient.CallClientStream(ctx, method)
}

func (cm *clientManager) callServerStream(ctx context.Context, method string, req any) (any, error) {
	return cm.triClient.CallServerStream(ctx, method, req)
}

func (cm *clientManager) callBidiStream(ctx context.Context, method string) (any, error) {
	return cm.triClient.CallBidiStream(ctx, method)
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

	// Set serialization
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

	// Set timeout
	timeout := url.GetParamDuration(constant.TimeoutKey, "")
	cliOpts = append(cliOpts, tri.WithTimeout(timeout))

	// Set service group and version
	group := url.GetParam(constant.GroupKey, "")
	version := url.GetParam(constant.VersionKey, "")
	cliOpts = append(cliOpts, tri.WithGroup(group), tri.WithVersion(version))

	// TODO(DMwangnima): support OpenTracing

	// Handle TLS
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
			logger.Info("[Triple][Client] triple clientManager initialized the TLSConfig configuration")
			tlsFlag = true
		}
	}

	var tripleConf *global.TripleConfig

	tripleConfRaw, ok := url.GetAttribute(constant.TripleConfigKey)
	if ok {
		tripleConf = tripleConfRaw.(*global.TripleConfig)
	}

	maxRecvBytes, maxSendBytes := resolveSizeLimits(url)
	cliOpts = append(cliOpts, tri.WithReadMaxBytes(maxRecvBytes), tri.WithSendMaxBytes(maxSendBytes))
	keepAliveInterval, keepAliveTimeout, err := resolveKeepAlive(url, tripleConf)
	if err != nil {
		logger.Errorf("[Triple][Client] resolve keepalive failed, err=%v", err)
		return nil, err
	}

	var callProtocol string
	if tripleConf != nil && tripleConf.Http3 != nil && tripleConf.Http3.Enable {
		callProtocol = constant.CallHTTP2AndHTTP3
	} else {
		// HTTP default type is HTTP/2.
		callProtocol = constant.CallHTTP2
	}

	if callProtocol == constant.CallHTTP {
		cliOpts = append(cliOpts, tri.WithTriple())
	}
	transport, err := newClientTransport(callProtocol, cfg, keepAliveInterval, keepAliveTimeout)
	if err != nil {
		return nil, err
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

	triURL, err := joinPath(baseTriURL, url.Interface())
	if err != nil {
		return nil, fmt.Errorf("JoinPath failed for base %s, interface %s", baseTriURL, url.Interface())
	}

	triClient := NewClient(httpClient, triURL, cliOpts...)
	healthURL, err := joinPath(baseTriURL, constant.HealthCheckServiceInterface)
	if err != nil {
		return nil, fmt.Errorf("JoinPath failed for base %s, health interface %s", baseTriURL, constant.HealthCheckServiceInterface)
	}
	healthClient := NewClient(httpClient, healthURL, tri.WithTimeout(timeout))

	return &clientManager{
		isIDL:        isIDL,
		triClient:    triClient,
		healthClient: healthClient,
	}, nil
}

func (cm *clientManager) callHealthWatch(ctx context.Context, service string) (*tri.ServerStreamForClient, error) {
	if cm.healthClient == nil {
		return nil, errors.New("triple health client is not initialized")
	}
	return cm.healthClient.CallServerStream(ctx, "Watch", &grpc_health_v1.HealthCheckRequest{Service: service})
}

func newClientTransport(callProtocol string, cfg *tls.Config, keepAliveInterval, keepAliveTimeout time.Duration) (http.RoundTripper, error) {
	switch callProtocol {
	case constant.CallHTTP:
		return &http.Transport{TLSClientConfig: cfg}, nil
	case constant.CallHTTP2:
		transport := &http2.Transport{
			TLSClientConfig: cfg,
			ReadIdleTimeout: keepAliveInterval,
			PingTimeout:     keepAliveTimeout,
		}
		if cfg != nil {
			transport.DialTLSContext = func(ctx context.Context, network, addr string, tlsConfig *tls.Config) (net.Conn, error) {
				return (&tls.Dialer{Config: tlsConfig}).DialContext(ctx, network, addr)
			}
		} else {
			transport.AllowHTTP = true
			transport.DialTLSContext = func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, network, addr)
			}
		}
		return transport, nil
	case constant.CallHTTP3:
		if cfg == nil {
			return nil, fmt.Errorf("TRIPLE http3 client must have TLS config, but TLS config is nil")
		}
		logger.Info("[Triple][Client] triple http3 client transport init successfully")
		return &http3.Transport{
			TLSClientConfig: cfg,
			QUICConfig: &quic.Config{
				KeepAlivePeriod: keepAliveInterval,
				MaxIdleTimeout:  keepAliveTimeout,
			},
		}, nil
	case constant.CallHTTP2AndHTTP3:
		if cfg == nil {
			return nil, fmt.Errorf("TRIPLE HTTP/2 and HTTP/3 client must have TLS config, but TLS config is nil")
		}
		logger.Info("[Triple][Client] triple HTTP/2 and HTTP/3 client transport init successfully")
		return newDualTransport(cfg, keepAliveInterval, keepAliveTimeout), nil
	default:
		return nil, fmt.Errorf("unsupported http protocol: %s", callProtocol)
	}
}

func resolveSizeLimits(url *common.URL) (int, int) {
	maxCallRecvMsgSize := constant.DefaultMaxCallRecvMsgSize
	if recvMsgSize, err := humanize.ParseBytes(url.GetParam(constant.MaxCallRecvMsgSize, "")); err == nil && recvMsgSize > 0 {
		maxCallRecvMsgSize = int(recvMsgSize)
	}
	maxCallSendMsgSize := constant.DefaultMaxCallSendMsgSize
	if sendMsgSize, err := humanize.ParseBytes(url.GetParam(constant.MaxCallSendMsgSize, "")); err == nil && sendMsgSize > 0 {
		maxCallSendMsgSize = int(sendMsgSize)
	}
	return maxCallRecvMsgSize, maxCallSendMsgSize
}

func resolveKeepAlive(url *common.URL, tripleConf *global.TripleConfig) (time.Duration, time.Duration, error) {
	// Compatibility: read the legacy URL keepalive parameters.
	// TODO: remove KeepAliveInterval and KeepAliveTimeout in version 4.0.0.
	keepAliveInterval := url.GetParamDuration(constant.KeepAliveInterval, constant.DefaultKeepAliveInterval)
	keepAliveTimeout := url.GetParamDuration(constant.KeepAliveTimeout, constant.DefaultKeepAliveTimeout)

	if tripleConf == nil {
		return keepAliveInterval, keepAliveTimeout, nil
	}

	var parseErr error

	if tripleConf.KeepAliveInterval != "" {
		keepAliveInterval, parseErr = time.ParseDuration(tripleConf.KeepAliveInterval)
		if parseErr != nil {
			return 0, 0, parseErr
		}
	}
	if tripleConf.KeepAliveTimeout != "" {
		keepAliveTimeout, parseErr = time.ParseDuration(tripleConf.KeepAliveTimeout)
		if parseErr != nil {
			return 0, 0, parseErr
		}
	}

	return keepAliveInterval, keepAliveTimeout, nil
}
