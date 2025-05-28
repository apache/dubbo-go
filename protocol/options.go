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
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
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

	logger.Warnf("defOpts.Protocol: %+v", defOpts.Protocol)

	if defOpts.ID == "" {
		if defOpts.Protocol.Name == "" {
			// should be the same as default value of config.ProtocolConfig.Protocol
			defOpts.ID = constant.TriProtocol
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
			defOpts.ID = constant.TriProtocol
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
	config.TripleConfig = o.triOpts.Triple
}

func (o *tripleOption) applyToServer(config *global.ProtocolConfig) {
	config.TripleConfig = o.triOpts.Triple
}

func WithTriple(opts ...triple.Option) Option {
	triSrvOpts := triple.NewOptions(opts...)

	return &tripleOption{
		triOpts: *triSrvOpts,
	}
}

type dubboOption struct{}

func (o *dubboOption) applyToClient(config *global.ProtocolClientConfig) {
	config.Name = constant.DubboProtocol
}

func (o *dubboOption) applyToServer(config *global.ProtocolConfig) {
	config.Name = constant.DubboProtocol
}

// TODO: Maybe we need configure dubbo protocol future.
func WithDubbo() Option {
	return &dubboOption{}
}

type jsonRPCOption struct{}

func (o *jsonRPCOption) applyToClient(config *global.ProtocolClientConfig) {
	config.Name = constant.JSONRPCProtocol
}

func (o *jsonRPCOption) applyToServer(config *global.ProtocolConfig) {
	config.Name = constant.JSONRPCProtocol
}

// TODO: Maybe we need configure jsonRPC protocol future.
func WithJSONRPC() Option {
	return &jsonRPCOption{}
}

type restOption struct{}

func (o *restOption) applyToClient(config *global.ProtocolClientConfig) {
	config.Name = constant.RESTProtocol
}

func (o *restOption) applyToServer(config *global.ProtocolConfig) {
	config.Name = constant.RESTProtocol
}

// TODO: Maybe we need configure REST protocol future.
func WithREST() Option {
	return &restOption{}
}

type protocolNameOption struct {
	Name string
}

func (o *protocolNameOption) applyToClient(config *global.ProtocolClientConfig) {
	config.Name = o.Name
}

func (o *protocolNameOption) applyToServer(config *global.ProtocolConfig) {
	config.Name = o.Name
}

// NOTE: This option can't be configured freely.
func WithProtocol(p string) Option {
	return &protocolNameOption{p}
}

// TODO: protocol config struct don't have ID field, we need to deal with it.
//
// WithID specifies the id of protocol.Options. Then you could configure server.WithProtocolIDs and
// server.WithServer_ProtocolIDs to specify which protocol you need to use in multi-protocols scenario.
// func WithID(id string) Option {
// 	return func(opts *ServerOptions) {
// 		opts.ID = id
// 	}
// }

type ipOption struct {
	Ip string
}

func (o *ipOption) applyToServer(config *global.ProtocolConfig) {
	config.Ip = o.Ip
}

func WithIp(ip string) ServerOption {
	return &ipOption{ip}
}

type portOption struct {
	Port string
}

func (o *portOption) applyToServer(config *global.ProtocolConfig) {
	config.Port = o.Port
}

func WithPort(port int) ServerOption {
	return &portOption{strconv.Itoa(port)}
}

type paramsOption struct {
	Params any
}

func (o *paramsOption) applyToServer(config *global.ProtocolConfig) {
	config.Params = o.Params
}

func WithParams(params any) ServerOption {
	return &paramsOption{params}
}

// Deprecated：use triple.WithMaxServerSendMsgSize()
//
// TODO: remove MaxServerSendMsgSize and MaxServerRecvMsgSize when version 4.0.0
type maxServerSendMsgSizeOption struct {
	MaxServerSendMsgSize string
}

// Deprecated：use triple.WithMaxServerSendMsgSize()
//
// TODO: remove MaxServerSendMsgSize and MaxServerRecvMsgSize when version 4.0.0
func (o *maxServerSendMsgSizeOption) applyToServer(config *global.ProtocolConfig) {
	config.MaxServerSendMsgSize = o.MaxServerSendMsgSize
}

// Deprecated：use triple.WithMaxServerSendMsgSize()
//
// TODO: remove MaxServerSendMsgSize and MaxServerRecvMsgSize when version 4.0.0
func WithMaxServerSendMsgSize(size string) ServerOption {
	return &maxServerSendMsgSizeOption{size}
}

// Deprecated：use triple.WithMaxServerRecvMsgSize()
//
// TODO: remove MaxServerSendMsgSize and MaxServerRecvMsgSize when version 4.0.0
type maxServerRecvMsgSize struct {
	MaxServerRecvMsgSize string
}

// Deprecated：use triple.WithMaxServerRecvMsgSize()
//
// TODO: remove MaxServerSendMsgSize and MaxServerRecvMsgSize when version 4.0.0
func (o *maxServerRecvMsgSize) applyToServer(config *global.ProtocolConfig) {
	config.MaxServerRecvMsgSize = o.MaxServerRecvMsgSize
}

// Deprecated：use triple.WithMaxServerRecvMsgSize()
//
// TODO: remove MaxServerSendMsgSize and MaxServerRecvMsgSize when version 4.0.0
func WithMaxServerRecvMsgSize(size string) ServerOption {
	return &maxServerRecvMsgSize{size}
}
