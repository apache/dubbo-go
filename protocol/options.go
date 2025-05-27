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

package protocol

import (
	"strconv"
)

import (
// "github.com/dubbogo/gost/log/logger"
)

import (
	// "dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple"
)

type ClientOption interface {
	applyToClient(*global.ProtocolClientConfig)
}

type ServerOption interface {
	applyToServer(*global.ProtocolConfig)
}

type Option interface {
	ClientOption
	ServerOption
}

// WithClientOptions composes multiple ClientOptions into one.
func WithClientOptions(options ...ClientOption) ClientOption {
	return &clientOptionsOption{options}
}

type clientOptionsOption struct {
	options []ClientOption
}

func (o *clientOptionsOption) applyToClient(config *global.ProtocolClientConfig) {
	for _, option := range o.options {
		option.applyToClient(config)
	}
}

type ServerOptionsOption struct {
	options []ServerOption
}

func (o *ServerOptionsOption) applyToServer(config *global.ProtocolConfig) {
	for _, option := range o.options {
		option.applyToServer(config)
	}
}

type optionsOption struct {
	options []Option
}

func (o *optionsOption) applyToClient(config *global.ProtocolClientConfig) {
	for _, option := range o.options {
		option.applyToClient(config)
	}
}

func (o *optionsOption) applyToServer(config *global.ProtocolConfig) {
	for _, option := range o.options {
		option.applyToServer(config)
	}
}

type ServerOptions struct {
	Protocol *global.ProtocolConfig

	ID string
}

func defaultServerOptions() *ServerOptions {
	return &ServerOptions{Protocol: global.DefaultProtocolConfig()}
}

func NewServerOptions(opts ...ServerOption) *ServerOptions {
	defOpts := defaultServerOptions()
	for _, opt := range opts {
		opt.applyToServer(defOpts.Protocol)
	}

	if defOpts.ID == "" {
		if defOpts.Protocol.Name == "" {
			// should be the same as default value of config.ProtocolConfig.Protocol
			defOpts.ID = "tri"
		} else {
			defOpts.ID = defOpts.Protocol.Name
		}
	}

	return defOpts
}

type ClientOptions struct {
	ProtocolClient *global.ProtocolClientConfig

	ID string
}

func defaultClientOptions() *ClientOptions {
	return &ClientOptions{ProtocolClient: global.DefaultProtocolClientConfig()}
}

func NewClientOptions(opts ...ClientOption) *ClientOptions {
	defOpts := defaultClientOptions()
	for _, opt := range opts {
		opt.applyToClient(defOpts.ProtocolClient)
	}

	if defOpts.ID == "" {
		if defOpts.ProtocolClient.Name == "" {
			// should be the same as default value of config.ProtocolConfig.Protocol
			defOpts.ID = "tri"
		} else {
			defOpts.ID = defOpts.ProtocolClient.Name
		}
	}

	return defOpts
}

type tripleOption struct {
	triOpts triple.Options
}

func (o *tripleOption) applyToClient(config *global.ProtocolClientConfig) {
	config = global.DefaultProtocolClientConfig()
}

func (o *tripleOption) applyToServer(config *global.ProtocolConfig) {
	config = global.DefaultProtocolConfig()
}

func WithTriple(opts ...triple.Option) Option {
	triSrvOpts := triple.NewOptions(opts...)

	return &tripleOption{
		triOpts: *triSrvOpts,
	}
}

// type Option func(*ServerOptions)
//
// func WithDubbo() Option {
// 	return func(opts *ServerOptions) {
// 		opts.Protocol.Name = "dubbo"
// 	}
// }
//
// func WithJSONRPC() Option {
// 	return func(opts *ServerOptions) {
// 		opts.Protocol.Name = "jsonrpc"
// 	}
// }
//
// func WithREST() Option {
// 	return func(opts *ServerOptions) {
// 		opts.Protocol.Name = "rest"
// 	}
// }

// func WithProtocol(p string) Option {
// 	return func(opts *ServerOptions) {
// 		opts.Protocol.Name = p
// 	}
// }

// WithID specifies the id of protocol.Options. Then you could configure server.WithProtocolIDs and
// server.WithServer_ProtocolIDs to specify which protocol you need to use in multi-protocols scenario.
// func WithID(id string) Option {
// 	return func(opts *ServerOptions) {
// 		opts.ID = id
// 	}
// }
//
// func WithIp(ip string) Option {
// 	return func(opts *ServerOptions) {
// 		opts.Protocol.Ip = ip
// 	}
// }

type portOption struct {
	Port string
}

func (o *portOption) applyToServer(config *global.ProtocolConfig) {
	config.Port = o.Port
}

func WithPort(port int) ServerOption {
	return &portOption{strconv.Itoa(port)}
}

// func WithParams(params any) Option {
// 	return func(opts *ServerOptions) {
// 		opts.Protocol.Params = params
// 	}
// }
//
// func WithMaxServerSendMsgSize(size int) Option {
// 	return func(opts *ServerOptions) {
// 		opts.Protocol.MaxServerSendMsgSize = strconv.Itoa(size)
// 	}
// }
//
// func WithMaxServerRecvMsgSize(size int) Option {
// 	return func(opts *ServerOptions) {
// 		opts.Protocol.MaxServerRecvMsgSize = strconv.Itoa(size)
// 	}
// }
