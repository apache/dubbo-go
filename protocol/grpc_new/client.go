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

package grpc_new

import (
	"context"
	"crypto/tls"
	connect "dubbo.apache.org/dubbo-go/v3/protocol/grpc_new/connect"
	"golang.org/x/net/http2"
	"net"
	"net/http"
	"reflect"
	"strings"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"github.com/dustin/go-humanize"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config"
)

const (
	httpPrefix  string = "http://"
	httpsPrefix string = "https://"
)

// Client is GRPC_NEW client wrapping connection and dubbo invoker
type Client struct {
	*http.Client
	invoker reflect.Value
}

// NewClient creates a new GRPC_NEW client.
func NewClient(url *common.URL) (*Client, error) {
	// If global trace instance was set, it means trace function enabled.
	// If not, will return NoopTracer.
	//tracer := opentracing.GlobalTracer()
	var cliOpts []connect.ClientOption
	// set max send and recv msg size
	maxCallRecvMsgSize := constant.DefaultMaxCallRecvMsgSize
	if recvMsgSize, err := humanize.ParseBytes(url.GetParam(constant.MaxCallRecvMsgSize, "")); err == nil && recvMsgSize > 0 {
		maxCallRecvMsgSize = int(recvMsgSize)
	}
	cliOpts = append(cliOpts, connect.WithReadMaxBytes(maxCallRecvMsgSize))
	maxCallSendMsgSize := constant.DefaultMaxCallSendMsgSize
	if sendMsgSize, err := humanize.ParseBytes(url.GetParam(constant.MaxCallSendMsgSize, "")); err == nil && sendMsgSize > 0 {
		maxCallSendMsgSize = int(sendMsgSize)
	}
	cliOpts = append(cliOpts, connect.WithSendMaxBytes(maxCallSendMsgSize))

	// consumer config client connectTimeout
	//connectTimeout := config.GetConsumerConfig().ConnectTimeout

	//dialOpts = append(dialOpts,
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
	//)
	var cfg *tls.Config
	var tlsFlag bool
	var err error
	tlsConfig := config.GetRootConfig().TLSConfig
	if tlsConfig != nil {
		cfg, err = config.GetClientTlsConfig(&config.TLSConfig{
			CACertFile:    tlsConfig.CACertFile,
			TLSCertFile:   tlsConfig.TLSCertFile,
			TLSKeyFile:    tlsConfig.TLSKeyFile,
			TLSServerName: tlsConfig.TLSServerName,
		})
		logger.Infof("GRPC_NEW Client initialized the TLSConfig configuration")
		if err != nil {
			return nil, err
		}
		tlsFlag = true
	}

	key := url.GetParam(constant.InterfaceKey, "")
	impl := config.GetConsumerServiceByInterfaceName(key)
	conRefs := config.GetConsumerConfig().References
	ref, ok := conRefs[key]
	if !ok {
		panic("no reference")
	}
	// todo: set timeout
	var transport http.RoundTripper
	switch ref.Protocol {
	case "http1":
		transport = &http.Transport{
			TLSClientConfig: cfg,
		}
		cliOpts = append(cliOpts, connect.WithHTTPGet())
	case GRPC_NEW:
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
		cliOpts = append(cliOpts, connect.WithGRPC())
	// todo: process triple
	default:
		panic("not support")
	}
	cli := &http.Client{
		Transport: transport,
	}
	invoker := getInvokerFromStub(impl, cli, url.Location, tlsFlag, cliOpts...)

	return &Client{
		Client:  cli,
		invoker: reflect.ValueOf(invoker),
	}, nil
}

// getInvokerFromStub makes use of GetDubboStub of dubbo clientImpl to generate rpc client
func getInvokerFromStub(impl interface{}, cli connect.HTTPClient, baseURL string, tlsFlag bool, opts ...connect.ClientOption) interface{} {
	baseURL = strings.TrimPrefix(baseURL, httpPrefix)
	baseURL = strings.TrimPrefix(baseURL, httpsPrefix)
	if tlsFlag {
		baseURL = httpsPrefix + baseURL
	} else {
		baseURL = httpPrefix + baseURL
	}
	var in []reflect.Value
	in = append(in, reflect.ValueOf(cli), reflect.ValueOf(baseURL))
	for _, opt := range opts {
		in = append(in, reflect.ValueOf(opt))
	}
	// GetDubboStub(httpClient connect.HTTPClient, baseURL string, opts ...connect.ClientOption)
	method := reflect.ValueOf(impl).MethodByName("GetDubboStub")
	res := method.Call(in)
	return res[0].Interface()
}
