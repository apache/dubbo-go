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
	url_package "net/url"
	"strings"
)

import (
	"github.com/dustin/go-humanize"

	"golang.org/x/net/http2"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
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

func (cm *clientManager) callUnary(ctx context.Context, method string, req, resp interface{}) error {
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

func (cm *clientManager) callClientStream(ctx context.Context, method string) (interface{}, error) {
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

func (cm *clientManager) callServerStream(ctx context.Context, method string, req interface{}) (interface{}, error) {
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

func (cm *clientManager) callBidiStream(ctx context.Context, method string) (interface{}, error) {
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

func newClientManager(url *common.URL) (*clientManager, error) {
	// If global trace instance was set, it means trace function enabled.
	// If not, will return NoopTracer.
	// tracer := opentracing.GlobalTracer()
	var triClientOpts []tri.ClientOption

	// set max send and recv msg size
	maxCallRecvMsgSize := constant.DefaultMaxCallRecvMsgSize
	if recvMsgSize, err := humanize.ParseBytes(url.GetParam(constant.MaxCallRecvMsgSize, "")); err == nil && recvMsgSize > 0 {
		maxCallRecvMsgSize = int(recvMsgSize)
	}
	triClientOpts = append(triClientOpts, tri.WithReadMaxBytes(maxCallRecvMsgSize))
	maxCallSendMsgSize := constant.DefaultMaxCallSendMsgSize
	if sendMsgSize, err := humanize.ParseBytes(url.GetParam(constant.MaxCallSendMsgSize, "")); err == nil && sendMsgSize > 0 {
		maxCallSendMsgSize = int(sendMsgSize)
	}
	triClientOpts = append(triClientOpts, tri.WithSendMaxBytes(maxCallSendMsgSize))

	// todo:// process timeout
	// consumer config client connectTimeout
	//connectTimeout := config.GetConsumerConfig().ConnectTimeout

	// dialOpts = append(dialOpts,
	//
	//	grpc.WithBlock(),
	//	// todo config network timeout
	//	grpc.WithTimeout(time.Second*3),
	//	grpc.WithUnaryInterceptor(otgrpc.OpenTracingClientInterceptor(tracer, otgrpc.LogPayloads())),
	//	grpc.WithStreamInterceptor(otgrpc.OpenTracingStreamClientInterceptor(tracer, otgrpc.LogPayloads())),
	//	grpc.WithDefaultCallOptions(
	//		grpc.CallContentSubtype(clientConf.ContentSubType),
	//		grpc.MaxCallRecvMsgSize(maxCallRecvMsgSize),
	//		grpc.MaxCallSendMsgSize(maxCallSendMsgSize),
	//	),
	//
	// )
	var cfg *tls.Config
	var tlsFlag bool
	//var err error

	// todo: think about a more elegant way to configure tls
	//if tlsConfig := config.GetRootConfig().TLSConfig; tlsConfig != nil {
	//	cfg, err = config.GetClientTlsConfig(&config.TLSConfig{
	//		CACertFile:    tlsConfig.CACertFile,
	//		TLSCertFile:   tlsConfig.TLSCertFile,
	//		TLSKeyFile:    tlsConfig.TLSKeyFile,
	//		TLSServerName: tlsConfig.TLSServerName,
	//	})
	//	if err != nil {
	//		return nil, err
	//	}
	//	logger.Infof("TRIPLE clientManager initialized the TLSConfig configuration successfully")
	//	tlsFlag = true
	//}

	// todo(DMwangnima): this code fragment would be used to be compatible with old triple client
	//key := url.GetParam(constant.InterfaceKey, "")
	//conRefs := config.GetConsumerConfig().References
	//ref, ok := conRefs[key]
	//if !ok {
	//	panic("no reference")
	//}
	// todo: set timeout
	var transport http.RoundTripper
	callType := url.GetParam(constant.CallHTTPTypeKey, constant.CallHTTP2)
	switch callType {
	case constant.CallHTTP:
		transport = &http.Transport{
			TLSClientConfig: cfg,
		}
		triClientOpts = append(triClientOpts, tri.WithTriple())
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
				AllowHTTP: true,
			}
		}
		triClientOpts = append(triClientOpts)
	default:
		panic(fmt.Sprintf("Unsupported type: %s", callType))
	}
	httpClient := &http.Client{
		Transport: transport,
	}

	var baseTriURL string
	baseTriURL = strings.TrimPrefix(url.Location, httpPrefix)
	baseTriURL = strings.TrimPrefix(url.Location, httpsPrefix)
	if tlsFlag {
		baseTriURL = httpsPrefix + baseTriURL
	} else {
		baseTriURL = httpPrefix + baseTriURL
	}
	triClients := make(map[string]*tri.Client)
	for _, method := range url.Methods {
		triURL, err := url_package.JoinPath(baseTriURL, url.Interface(), method)
		if err != nil {
			return nil, fmt.Errorf("JoinPath failed for base %s, interface %s, method %s", baseTriURL, url.Interface(), method)
		}
		triClient := tri.NewClient(httpClient, triURL, triClientOpts...)
		triClients[method] = triClient
	}

	return &clientManager{
		triClients: triClients,
	}, nil
}
