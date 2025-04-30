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
	"net"
	"net/http"
	"reflect"
	"strings"

	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config"
	"github.com/dubbogo/gost/log/logger"
	"github.com/dustin/go-humanize"
	"golang.org/x/net/http2"

	tri "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
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

	serverAttachments, ok := ctx.Value(constant.AttachmentServerKey).(map[string]any)
	if !ok {
		return nil
	}
	for k, v := range triResp.Trailer() {
		if ok := isFilterHeader(k); ok {
			continue
		}
		if len(v) > 0 {
			serverAttachments[k] = v[0]
		}
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

	// set max send and recv msg size
	maxCallRecvMsgSize := constant.DefaultMaxCallRecvMsgSize
	if recvMsgSize, err := humanize.ParseBytes(url.GetParam(constant.MaxCallRecvMsgSize, "")); err == nil && recvMsgSize > 0 {
		maxCallRecvMsgSize = int(recvMsgSize)
	}
	cliOpts = append(cliOpts, tri.WithReadMaxBytes(maxCallRecvMsgSize))
	maxCallSendMsgSize := constant.DefaultMaxCallSendMsgSize
	if sendMsgSize, err := humanize.ParseBytes(url.GetParam(constant.MaxCallSendMsgSize, "")); err == nil && sendMsgSize > 0 {
		maxCallSendMsgSize = int(sendMsgSize)
	}
	cliOpts = append(cliOpts, tri.WithSendMaxBytes(maxCallSendMsgSize))
	// set keepalive interval and keepalive timeout
	keepAliveInterval := url.GetParamDuration(constant.KeepAliveInterval, constant.DefaultKeepAliveInterval)
	keepAliveTimeout := url.GetParamDuration(constant.KeepAliveTimeout, constant.DefaultKeepAliveTimeout)
	var isIDL bool
	// set serialization
	serialization := url.GetParam(constant.SerializationKey, constant.ProtobufSerialization)
	switch serialization {
	case constant.ProtobufSerialization:
		logger.Warnf("ProtobufSerialzation must IDL")
		isIDL = true
	case constant.JSONSerialization:
		logger.Warnf("JSONSerialization must IDL")
		isIDL = true
		cliOpts = append(cliOpts, tri.WithProtoJSON())
	case constant.Hessian2Serialization:
		logger.Warnf("Hessian2Serialization")
		cliOpts = append(cliOpts, tri.WithHessian2())
	case constant.MsgpackSerialization:
		logger.Warnf("Hessian2Serialization")
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

	// todo(DMwangnima): support TLS in an ideal way
	var cfg *tls.Config
	var tlsFlag bool
	var err error

	// handle tls config
	// TODO: think about a more elegant way to configure tls,
	// Maybe we can try to create a ClientOptions for unified settings,
	// after this function becomes bloated.

	// TODO: Once the global replacement of the config is completed,
	// replace config with global.
	if tlsConfig := config.GetRootConfig().TLSConfig; tlsConfig != nil {
		cfg, err = config.GetClientTlsConfig(&config.TLSConfig{
			CACertFile:    tlsConfig.CACertFile,
			TLSCertFile:   tlsConfig.TLSCertFile,
			TLSKeyFile:    tlsConfig.TLSKeyFile,
			TLSServerName: tlsConfig.TLSServerName,
		})
		if err != nil {
			return nil, err
		}
		logger.Infof("TRIPLE clientManager initialized the TLSConfig configuration")
		tlsFlag = true
	}

	var transport http.RoundTripper
	callType := url.GetParam(constant.CallHTTPTypeKey, constant.CallHTTP2)
	switch callType {
	case constant.CallHTTP:
		transport = &http.Transport{
			TLSClientConfig: cfg,
		}
		cliOpts = append(cliOpts, tri.WithTriple())
	case constant.CallHTTP2:
		if tlsFlag {
			transport = &http2.Transport{
				TLSClientConfig: cfg,
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
	default:
		panic(fmt.Sprintf("Unsupported callType: %s", callType))
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
		service, ok := url.GetAttribute(constant.RpcServiceKey)
		if !ok {
			return nil, fmt.Errorf("Triple clientmanager can't get methods")
		}

		serviceType := reflect.TypeOf(service)
		for i := 0; i < serviceType.NumMethod(); i++ {
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

func isFilterHeader(key string) bool {
	if key != "" && key[0] == ':' {
		return true
	}
	switch key {
	case constant.GrpcHeaderMessage, constant.GrpcHeaderStatus:
		return true
	default:
		return false
	}
}
