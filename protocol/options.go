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

// WithClientOptions composes ClientOption values for NewClientOptions and applies
// each option to the same ClientOptions instance in order. Use it to package a reusable client
// transport policy, such as Triple keepalive and message-size settings, into one option.
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

// WithServerOptions composes ServerOption values for NewServerOptions and applies
// each option to the same ServerOptions instance in order. Use it to package a reusable server
// endpoint policy before passing it to server.WithProtocol or server.WithServerProtocol.
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

// WithOptions composes options that apply to both ClientOptions and ServerOptions.
// Use it when the same transport settings must be shared by clients and servers, for example
// common Triple message limits in an application that both consumes and provides services.
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

// ========== Protocol selection ==========

type tripleOption struct {
	triOpts triple.Options
}

func (o *tripleOption) applyToClient(config *ClientOptions) {
	config.ProtocolClient.TripleConfig = o.triOpts.Triple
}

func (o *tripleOption) applyToServer(config *ServerOptions) {
	config.Protocol.TripleConfig = o.triOpts.Triple
}

// WithTriple applies Triple transport settings, such as message limits or keepalive behavior,
// to both client and server protocol configurations. The default protocol is Triple; when
// combining options explicitly, use it with a matching protocol selection. Choose Triple for
// HTTP/2, streaming, or gRPC-compatible interoperability.
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

// WithDubbo makes clients create Dubbo transports and servers expose Dubbo endpoints. Choose it
// when interoperating with existing services that use the classic Dubbo protocol.
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

// WithJSONRPC makes clients create JSON-RPC transports and servers expose JSON-RPC endpoints.
// Choose it when integrating with systems that speak JSON-RPC rather than Dubbo or Triple.
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

// WithREST makes clients create REST transports and servers expose REST endpoints. Choose it
// when the service contract is exposed as HTTP resources for REST clients.
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

// WithProtocol selects a registered transport extension for both clients and servers. Protocol
// initialization fails when the supplied name has no matching extension; prefer the built-in
// selection helpers when applicable. Use this option only for a custom protocol registered by
// the application or another module.
func WithProtocol(p string) Option {
	return &protocolNameOption{p}
}

// ========== Server protocol configuration ==========

type idOption struct {
	ID string
}

func (o *idOption) applyToServer(config *ServerOptions) {
	config.ID = o.ID
}

// WithID names this server protocol configuration so services can select it with
// server.WithProtocolIDs or server.WithServerProtocolIDs. IDs must be distinct when multiple
// endpoints use the same protocol on different addresses or ports.
func WithID(id string) ServerOption {
	return &idOption{id}
}

type ipOption struct {
	Ip string
}

func (o *ipOption) applyToServer(config *ServerOptions) {
	config.Protocol.Ip = o.Ip
}

// WithIp binds this protocol endpoint to the supplied local IP and publishes that address to
// registries. Use it on multi-homed hosts to choose the correct network interface, and ensure
// the published address is reachable by consumers.
func WithIp(ip string) ServerOption {
	return &ipOption{ip}
}

type portOption struct {
	Port string
}

func (o *portOption) applyToServer(config *ServerOptions) {
	config.Protocol.Port = o.Port
}

// WithPort binds and publishes this protocol endpoint on the supplied port. When omitted, the
// server allocates an available random port during export. Set a stable port for production,
// firewall rules, or direct clients; omit it when an ephemeral test port is acceptable.
func WithPort(port int) ServerOption {
	return &portOption{strconv.Itoa(port)}
}

type paramsOption struct {
	Params any
}

func (o *paramsOption) applyToServer(config *ServerOptions) {
	config.Protocol.Params = o.Params
}

// WithParams supplies transport-specific server settings consumed by the selected protocol,
// such as its underlying remoting configuration. The expected value type depends on that
// protocol implementation. Use it only when the protocol documents a concrete parameter type;
// prefer typed options such as WithTriple when they are available.
func WithParams(params any) ServerOption {
	return &paramsOption{params}
}

// ========== Deprecated options ==========

// WithMaxServerSendMsgSize is retained for compatibility and panics when applied.
//
// Deprecated: use triple.WithMaxServerSendMsgSize with WithTriple instead.
func WithMaxServerSendMsgSize(size string) ServerOption {
	panic("use triple.WithMaxServerSendMsgSize()")
}

// WithMaxServerRecvMsgSize is retained for compatibility and panics when applied.
//
// Deprecated: use triple.WithMaxServerRecvMsgSize with WithTriple instead.
func WithMaxServerRecvMsgSize(size string) ServerOption {
	panic("use triple.WithMaxServerRecvMsgSize()")
}
