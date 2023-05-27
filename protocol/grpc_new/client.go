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
	"crypto/tls"
	connect "dubbo.apache.org/dubbo-go/v3/protocol/grpc_new/connect"
	"golang.org/x/net/http2"
	"net/http"
	"reflect"
	"sync"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"github.com/dustin/go-humanize"

	"gopkg.in/yaml.v2"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config"
)

var clientConf *ClientConfig
var clientConfInitOnce sync.Once

// Client is gRPC client include client connection and invoker
type Client struct {
	*http.Client
	invoker reflect.Value
}

// NewClient creates a new gRPC client.
func NewClient(url *common.URL) (*Client, error) {
	clientConfInitOnce.Do(clientInit)

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
		transport = &http2.Transport{
			TLSClientConfig: cfg,
		}
		cliOpts = append(cliOpts, connect.WithGRPC())
	// todo: process triple
	default:
		panic("not support")
	}
	cli := &http.Client{
		Transport: transport,
	}
	invoker := getInvoker(impl, cli, url.Location, cliOpts...)

	return &Client{
		Client:  cli,
		invoker: reflect.ValueOf(invoker),
	}, nil
}

func clientInit() {
	// load rootConfig from runtime
	rootConfig := config.GetRootConfig()

	clientConfig := GetClientConfig()
	clientConf = &clientConfig

	// check client config and decide whether to use the default config
	defer func() {
		if clientConf == nil || len(clientConf.ContentSubType) == 0 {
			defaultClientConfig := GetDefaultClientConfig()
			clientConf = &defaultClientConfig
		}
		if err := clientConf.Validate(); err != nil {
			panic(err)
		}
	}()

	if rootConfig.Application == nil {
		return
	}
	protocolConf := config.GetRootConfig().Protocols

	if protocolConf == nil {
		logger.Info("protocol_conf default use dubbo config")
	} else {
		grpcNewConf := protocolConf[GRPC_NEW]
		if grpcNewConf == nil {
			logger.Warnf("grpcConf is nil")
			return
		}
		grpcNewConfByte, err := yaml.Marshal(grpcNewConf)
		if err != nil {
			panic(err)
		}
		err = yaml.Unmarshal(grpcNewConfByte, clientConf)
		if err != nil {
			panic(err)
		}
	}
}

func getInvoker(impl interface{}, cli connect.HTTPClient, baseURL string, opts ...connect.ClientOption) interface{} {
	var in []reflect.Value
	in = append(in, reflect.ValueOf(cli), reflect.ValueOf(baseURL), reflect.ValueOf(opts))
	method := reflect.ValueOf(impl).MethodByName("GetDubboStub")
	res := method.Call(in)
	return res[0].Interface()
}
