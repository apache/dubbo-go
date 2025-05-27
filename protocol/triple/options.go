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

func WithTestOption(test string) Option {
	return func(opts *Options) {
		opts.Triple.Test = test
	}
}

func WithKeepAlive(interval, timeout string) Option {
	return func(opts *Options) {
		opts.Triple.KeepAliveInterval = interval
		opts.Triple.KeepAliveTimeout = timeout
	}
}

func WithKeepAliveInterval(interval string) Option {
	return func(opts *Options) {
		opts.Triple.KeepAliveInterval = interval
	}
}

func WithKeepAliveTimeout(timeout string) Option {
	return func(opts *Options) {
		opts.Triple.KeepAliveTimeout = timeout
	}
}
