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
	"errors"
	"reflect"
	"sync"
	"time"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"github.com/dustin/go-humanize"

	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"

	"github.com/opentracing/opentracing-go"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"gopkg.in/yaml.v2"
)

import (
	"dubbo.apache.org/dubbo-go/v3"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/global"
	dubbotls "dubbo.apache.org/dubbo-go/v3/tls"
)

var clientConf *ClientConfig
var clientConfInitOnce sync.Once

// Client is gRPC client include client connection and invoker
type Client struct {
	*grpc.ClientConn
	invoker reflect.Value
}

// NewClient creates a new gRPC client.
func NewClient(url *common.URL) (*Client, error) {
	clientConfInitOnce.Do(func() {
		clientInit(url)
	})

	// If global trace instance was set, it means trace function enabled.
	// If not, will return NoopTracer.
	tracer := opentracing.GlobalTracer()
	dialOpts := make([]grpc.DialOption, 0, 4)

	// set max send and recv msg size
	maxCallRecvMsgSize := constant.DefaultMaxCallRecvMsgSize
	if recvMsgSize, err := humanize.ParseBytes(url.GetParam(constant.MaxCallRecvMsgSize, "")); err == nil && recvMsgSize > 0 {
		maxCallRecvMsgSize = int(recvMsgSize)
	}
	maxCallSendMsgSize := constant.DefaultMaxCallSendMsgSize
	if sendMsgSize, err := humanize.ParseBytes(url.GetParam(constant.MaxCallSendMsgSize, "")); err == nil && sendMsgSize > 0 {
		maxCallSendMsgSize = int(sendMsgSize)
	}

	// consumer config client connectTimeout
	//connectTimeout := config.GetConsumerConfig().ConnectTimeout

	dialOpts = append(dialOpts,
		grpc.WithBlock(),
		// todo config network timeout
		grpc.WithTimeout(time.Second*3),
		grpc.WithUnaryInterceptor(otgrpc.OpenTracingClientInterceptor(tracer, otgrpc.LogPayloads())),
		grpc.WithStreamInterceptor(otgrpc.OpenTracingStreamClientInterceptor(tracer, otgrpc.LogPayloads())),
		grpc.WithDefaultCallOptions(
			grpc.CallContentSubtype(clientConf.ContentSubType),
			grpc.MaxCallRecvMsgSize(maxCallRecvMsgSize),
			grpc.MaxCallSendMsgSize(maxCallSendMsgSize),
		),
	)

	// TODO: remove config TLSConfig
	// delete this branch
	tlsConfig := config.GetRootConfig().TLSConfig
	if tlsConfig != nil {
		cfg, err := config.GetClientTlsConfig(&config.TLSConfig{
			CACertFile:    tlsConfig.CACertFile,
			TLSCertFile:   tlsConfig.TLSCertFile,
			TLSKeyFile:    tlsConfig.TLSKeyFile,
			TLSServerName: tlsConfig.TLSServerName,
		})
		logger.Infof("Grpc Client initialized the TLSConfig configuration")
		if err != nil {
			return nil, err
		}
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(credentials.NewTLS(cfg)))
	} else if tlsConfRaw, ok := url.GetAttribute(constant.TLSConfigKey); ok {
		// use global TLSConfig handle tls
		tlsConf, ok := tlsConfRaw.(*global.TLSConfig)
		if !ok {
			logger.Errorf("Grpc Client initialized the TLSConfig configuration failed")
			return nil, errors.New("DUBBO3 Client initialized the TLSConfig configuration failed")
		}

		if dubbotls.IsClientTLSValid(tlsConf) {
			cfg, err := dubbotls.GetClientTlSConfig(tlsConf)
			if err != nil {
				return nil, err
			}
			if cfg != nil {
				logger.Infof("Grpc Client initialized the TLSConfig configuration")
				dialOpts = append(dialOpts, grpc.WithTransportCredentials(credentials.NewTLS(cfg)))
			}
		} else {
			dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
		}
	} else {
		// TODO: remove this else
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	conn, err := grpc.Dial(url.Location, dialOpts...)
	if err != nil {
		logger.Errorf("grpc dial error: %v", err)
		return nil, err
	}

	key := url.GetParam(constant.InterfaceKey, "")
	//TODO: Temporary compatibility with old APIs, can be removed later
	consumerService := config.GetConsumerServiceByInterfaceName(key)
	if consumerService == nil {
		if rpcService, ok := url.GetAttribute(constant.RpcServiceKey); ok {
			consumerService = rpcService
		}
	}
	invoker := getInvoker(consumerService, conn)

	return &Client{
		ClientConn: conn,
		invoker:    reflect.ValueOf(invoker),
	}, nil
}

func clientInit(url *common.URL) {
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
		app := url.GetParam(constant.ApplicationKey, "")
		if len(app) == 0 {
			return
		}
	}

	//TODO: Temporary compatibility with old APIs, can be removed later
	protocolConf := dubbo.CompatGlobalProtocolConfigMap(rootConfig.Protocols)
	if protocolConf == nil {
		if protocolConfRaw, ok := url.GetAttribute(constant.ProtocolConfigKey); ok {
			protocolConfig, ok := protocolConfRaw.(map[string]*global.ProtocolConfig)
			if !ok {
				logger.Warnf("protocolConfig assert failed")
				return
			}
			if protocolConfig == nil {
				logger.Warnf("protocolConfig is nil")
				return
			}
			protocolConf = protocolConfig
		}
	}
	grpcConf := protocolConf[GRPC]
	if grpcConf == nil {
		logger.Warnf("grpcConf is nil")
		return
	}
	grpcConfByte, err := yaml.Marshal(grpcConf)
	if err != nil {
		panic(err)
	}
	err = yaml.Unmarshal(grpcConfByte, clientConf)
	if err != nil {
		panic(err)
	}
}

func getInvoker(impl any, conn *grpc.ClientConn) any {
	if impl == nil {
		return nil
	}

	implValue := reflect.ValueOf(impl)
	if !implValue.IsValid() {
		return nil
	}

	method := implValue.MethodByName("GetDubboStub")
	if !method.IsValid() {
		return nil
	}

	res := method.Call([]reflect.Value{reflect.ValueOf(conn)})
	if len(res) == 0 {
		return nil
	}

	return res[0].Interface()
}
