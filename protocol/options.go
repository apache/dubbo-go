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
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple"
)

type ClientOption interface {
	applyToClient(*ClientOptions)
}

type ServerOption interface {
	applyToServer(*ServerOptions)
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

func (o *clientOptionsOption) applyToClient(config *ClientOptions) {
	for _, option := range o.options {
		option.applyToClient(config)
	}
}

// WithServerOptions composes multiple ServerOptions into one.
func WithServerOptions(options ...ServerOption) ServerOption {
	return &serverOptionsOption{options}
}

type serverOptionsOption struct {
	options []ServerOption
}

func (o *serverOptionsOption) applyToServer(config *ServerOptions) {
	for _, option := range o.options {
		option.applyToServer(config)
	}
}

// WithOptions composes multiple Options into one.
func WithOptions(options ...Option) Option {
	return &optionsOption{options}
}

type optionsOption struct {
	options []Option
}

func (o *optionsOption) applyToClient(config *ClientOptions) {
	for _, option := range o.options {
		option.applyToClient(config)
	}
}

func (o *optionsOption) applyToServer(config *ServerOptions) {
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
		opt.applyToServer(defOpts)
	}

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
	ProtocolClient *global.ClientProtocolConfig

	ID string
}

func defaultClientOptions() *ClientOptions {
	return &ClientOptions{ProtocolClient: global.DefaultClientProtocolConfig()}
}

func NewClientOptions(opts ...ClientOption) *ClientOptions {
	defOpts := defaultClientOptions()
	for _, opt := range opts {
		opt.applyToClient(defOpts)
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

func (o *tripleOption) applyToClient(config *ClientOptions) {
	config.ProtocolClient.TripleConfig = o.triOpts.Triple
}

func (o *tripleOption) applyToServer(config *ServerOptions) {
	config.Protocol.TripleConfig = o.triOpts.Triple
}

func WithTriple(opts ...triple.Option) Option {
	triSrvOpts := triple.NewOptions(opts...)

	return &tripleOption{
		triOpts: *triSrvOpts,
	}
}

type dubboOption struct{}

func (o *dubboOption) applyToClient(config *ClientOptions) {
	config.ProtocolClient.Name = constant.DubboProtocol
}

func (o *dubboOption) applyToServer(config *ServerOptions) {
	config.Protocol.Name = constant.DubboProtocol
}

// TODO: Maybe we need configure dubbo protocol future.
func WithDubbo() Option {
	return &dubboOption{}
}

type jsonRPCOption struct{}

func (o *jsonRPCOption) applyToClient(config *ClientOptions) {
	config.ProtocolClient.Name = constant.JSONRPCProtocol
}

func (o *jsonRPCOption) applyToServer(config *ServerOptions) {
	config.Protocol.Name = constant.JSONRPCProtocol
}

// TODO: Maybe we need configure jsonRPC protocol future.
func WithJSONRPC() Option {
	return &jsonRPCOption{}
}

type restOption struct{}

func (o *restOption) applyToClient(config *ClientOptions) {
	config.ProtocolClient.Name = constant.RESTProtocol
}

func (o *restOption) applyToServer(config *ServerOptions) {
	config.Protocol.Name = constant.RESTProtocol
}

// TODO: Maybe we need configure REST protocol future.
func WithREST() Option {
	return &restOption{}
}

type protocolNameOption struct {
	Name string
}

func (o *protocolNameOption) applyToClient(config *ClientOptions) {
	config.ProtocolClient.Name = o.Name
}

func (o *protocolNameOption) applyToServer(config *ServerOptions) {
	config.Protocol.Name = o.Name
}

// NOTE: This option can't be configured freely.
func WithProtocol(p string) Option {
	return &protocolNameOption{p}
}

type idOption struct {
	ID string
}

func (o *idOption) applyToServer(config *ServerOptions) {
	config.ID = o.ID
}

// WithID specifies the id of protocol.Options. Then you could configure server.WithProtocolIDs and
// server.WithServer_ProtocolIDs to specify which protocol you need to use in multi-protocols scenario.
func WithID(id string) ServerOption {
	return &idOption{id}
}

type ipOption struct {
	Ip string
}

func (o *ipOption) applyToServer(config *ServerOptions) {
	config.Protocol.Ip = o.Ip
}

func WithIp(ip string) ServerOption {
	return &ipOption{ip}
}

type portOption struct {
	Port string
}

func (o *portOption) applyToServer(config *ServerOptions) {
	config.Protocol.Port = o.Port
}

func WithPort(port int) ServerOption {
	return &portOption{strconv.Itoa(port)}
}

type paramsOption struct {
	Params any
}

func (o *paramsOption) applyToServer(config *ServerOptions) {
	config.Protocol.Params = o.Params
}

func WithParams(params any) ServerOption {
	return &paramsOption{params}
}

// TODO: remove MaxServerSendMsgSize and MaxServerRecvMsgSize when version 4.0.0
type maxServerSendMsgSizeOption struct {
	MaxServerSendMsgSize string
}

// TODO: remove MaxServerSendMsgSize and MaxServerRecvMsgSize when version 4.0.0
func (o *maxServerSendMsgSizeOption) applyToServer(config *ServerOptions) {
	config.Protocol.MaxServerSendMsgSize = o.MaxServerSendMsgSize
}

// TODO: remove MaxServerSendMsgSize and MaxServerRecvMsgSize when version 4.0.0
//
// Deprecated：use triple.WithMaxServerSendMsgSize instead.
func WithMaxServerSendMsgSize(size string) ServerOption {
	return &maxServerSendMsgSizeOption{size}
}

// TODO: remove MaxServerSendMsgSize and MaxServerRecvMsgSize when version 4.0.0
type maxServerRecvMsgSize struct {
	MaxServerRecvMsgSize string
}

// TODO: remove MaxServerSendMsgSize and MaxServerRecvMsgSize when version 4.0.0
func (o *maxServerRecvMsgSize) applyToServer(config *ServerOptions) {
	config.Protocol.MaxServerRecvMsgSize = o.MaxServerRecvMsgSize
}

// TODO: remove MaxServerSendMsgSize and MaxServerRecvMsgSize when version 4.0.0
//
// Deprecated: use triple.WithMaxServerRecvMsgSize instead.
func WithMaxServerRecvMsgSize(size string) ServerOption {
	return &maxServerRecvMsgSize{size}
}
