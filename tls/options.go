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

package tls

import (
	"dubbo.apache.org/dubbo-go/v3/global"
)

// The consideration of not placing TLSOption in the global package is
// to prevent users from directly using the global package, so I created
// a new tls directory to allow users to establish config through the tls package.

type Options struct {
	TLSConf *global.TLSConfig
}

func defaultOptions() *Options {
	return &Options{TLSConf: global.DefaultTLSConfig()}
}

func NewOptions(opts ...Option) *Options {
	defOpts := defaultOptions()
	for _, opt := range opts {
		opt(defOpts)
	}
	return defOpts
}

type Option func(*Options)

func WithCACertFile(file string) Option {
	return func(opts *Options) {
		opts.TLSConf.CACertFile = file
	}
}

func WithCertFile(file string) Option {
	return func(opts *Options) {
		opts.TLSConf.TLSCertFile = file
	}
}

func WithKeyFile(file string) Option {
	return func(opts *Options) {
		opts.TLSConf.TLSKeyFile = file
	}
}

func WithServerName(name string) Option {
	return func(opts *Options) {
		opts.TLSConf.TLSServerName = name
	}
}
