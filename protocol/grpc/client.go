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
	"reflect"
	"strconv"
)

import (
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/opentracing/opentracing-go"
	"google.golang.org/grpc"
	"gopkg.in/yaml.v2"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/config"
)

var (
	clientConf *ClientConfig
)

func init() {
	// load clientconfig from consumer_config
	consumerConfig := config.GetConsumerConfig()

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

	if consumerConfig.ApplicationConfig == nil {
		return
	}
	protocolConf := config.GetConsumerConfig().ProtocolConf

	if protocolConf == nil {
		logger.Info("protocol_conf default use dubbo config")
	} else {
		grpcConf := protocolConf.(map[interface{}]interface{})[GRPC]
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

}

// Client is gRPC client include client connection and invoker
type Client struct {
	*grpc.ClientConn
	invoker reflect.Value
}

// NewClient creates a new gRPC client.
func NewClient(url *common.URL) *Client {
	// if global trace instance was set , it means trace function enabled. If not , will return Nooptracer
	tracer := opentracing.GlobalTracer()
	dailOpts := make([]grpc.DialOption, 0, 4)
	maxMessageSize, _ := strconv.Atoi(url.GetParam(constant.MESSAGE_SIZE_KEY, "4"))
	dailOpts = append(dailOpts, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithUnaryInterceptor(
		otgrpc.OpenTracingClientInterceptor(tracer, otgrpc.LogPayloads())),
		grpc.WithDefaultCallOptions(
			grpc.CallContentSubtype(clientConf.ContentSubType),
			grpc.MaxCallRecvMsgSize(1024*1024*maxMessageSize),
			grpc.MaxCallSendMsgSize(1024*1024*maxMessageSize)))
	conn, err := grpc.Dial(url.Location, dailOpts...)
	if err != nil {
		panic(err)
	}

	key := url.GetParam(constant.BEAN_NAME_KEY, "")
	impl := config.GetConsumerService(key)
	invoker := getInvoker(impl, conn)

	return &Client{
		ClientConn: conn,
		invoker:    reflect.ValueOf(invoker),
	}
}

func getInvoker(impl interface{}, conn *grpc.ClientConn) interface{} {
	var in []reflect.Value
	in = append(in, reflect.ValueOf(conn))
	method := reflect.ValueOf(impl).MethodByName("GetDubboStub")
	res := method.Call(in)
	return res[0].Interface()
}
