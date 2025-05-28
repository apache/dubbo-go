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
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/global"
)

type Options struct {
	Triple *global.TripleConfig
}

func defaultOptions() *Options {
	return &Options{Triple: global.DefaultTripleConfig()}
}

func NewOptions(opts ...Option) *Options {
	defSrvOpts := defaultOptions()
	for _, opt := range opts {
		opt(defSrvOpts)
	}
	return defSrvOpts
}

type Option func(*Options)

// WithKeepAlive sets the keep-alive interval and timeout for the Triple protocol.
// interval: The duration between keep-alive pings.
// timeout: The duration to wait for a keep-alive response before considering the connection dead.
// If not set, default interval is 10s, default timeout is 20s.
func WithKeepAlive(interval, timeout time.Duration) Option {
	return func(opts *Options) {
		opts.Triple.KeepAliveInterval = interval.String()
		opts.Triple.KeepAliveTimeout = timeout.String()
	}
}

// WithKeepAliveInterval sets the keep-alive interval for the Triple protocol.
// interval: The duration between keep-alive pings.
// If not set, default interval is 10s.
func WithKeepAliveInterval(interval time.Duration) Option {
	return func(opts *Options) {
		opts.Triple.KeepAliveInterval = interval.String()
	}
}

// WithKeepAliveTimeout sets the keep-alive timeout for the Triple protocol.
// timeout: The duration to wait for a keep-alive response before considering the connection dead.
// If not set, default timeout is 20s.
func WithKeepAliveTimeout(timeout time.Duration) Option {
	return func(opts *Options) {
		opts.Triple.KeepAliveTimeout = timeout.String()
	}
}

// WithMaxServerSendMsgSize sets the maximum size of messages that the server can send.
// size: The maximum message size in bytes, specified as a string (e.g., "4MB").
// If not set, default value is 2147MB (math.MaxInt32).
func WithMaxServerSendMsgSize(size string) Option {
	return func(opts *Options) {
		opts.Triple.MaxServerSendMsgSize = size
	}
}

// WithMaxServerRecvMsgSize sets the maximum size of messages that the server can receive.
// size: The maximum message size in bytes, specified as a string (e.g., "4MB").
// If not set, default value is 4MB (4194304 bytes).
func WithMaxServerRecvMsgSize(size string) Option {
	return func(opts *Options) {
		opts.Triple.MaxServerRecvMsgSize = size
	}
}
