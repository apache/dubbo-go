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
	"dubbo.apache.org/dubbo-go/v3/global"
)

type Options struct {
	Protocol *global.ProtocolConfig
}

func DefaultOptions() *Options {
	return &Options{Protocol: global.DefaultProtocolConfig()}
}

type Option func(*Options)

func WithDubbo() Option {
	return func(opts *Options) {
		opts.Protocol.Name = "dubbo"
	}
}

func WithGRPC() Option {
	return func(opts *Options) {
		opts.Protocol.Name = "grpc"
	}
}

func WithJSONRPC() Option {
	return func(opts *Options) {
		opts.Protocol.Name = "jsonrpc"
	}
}

func WithREST() Option {
	return func(opts *Options) {
		opts.Protocol.Name = "rest"
	}
}

func WithTriple() Option {
	return func(opts *Options) {
		opts.Protocol.Name = "tri"
	}
}

func WithIp(ip string) Option {
	return func(opts *Options) {
		opts.Protocol.Ip = ip
	}
}

func WithPort(port int) Option {
	return func(opts *Options) {
		opts.Protocol.Port = strconv.Itoa(port)
	}
}

func WithParams(params interface{}) Option {
	return func(opts *Options) {
		opts.Protocol.Params = params
	}
}

func WithMaxServerSendMsgSize(size int) Option {
	return func(opts *Options) {
		opts.Protocol.MaxServerSendMsgSize = strconv.Itoa(size)
	}
}

func WithMaxServerRecvMsgSize(size int) Option {
	return func(opts *Options) {
		opts.Protocol.MaxServerRecvMsgSize = strconv.Itoa(size)
	}
}
