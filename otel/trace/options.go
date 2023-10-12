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

package trace

import (
	"fmt"
)

import (
	"dubbo.apache.org/dubbo-go/v3/global"
)

type Options struct {
	Tracing *global.TracingConfig

	ID string
}

func defaultOptions() *Options {
	return &Options{Tracing: &global.TracingConfig{}}
}

func NewOptions(opts ...Option) *Options {
	defOpts := defaultOptions()
	for _, opt := range opts {
		opt(defOpts)
	}

	if defOpts.Tracing.Name == "" {
		panic(fmt.Sprintf("Please specify the tracing system to use, eg. WithZipkin()"))
	}
	if defOpts.ID == "" {
		defOpts.ID = defOpts.Tracing.Name
	}

	return defOpts
}

type Option func(*Options)

func WithID(id string) Option {
	return func(opts *Options) {
		opts.ID = id
	}
}

func WithZipkin() Option {
	return func(opts *Options) {
		opts.Tracing.Name = "zipkin"
	}
}

func WithJaeger() Option {
	return func(opts *Options) {
		opts.Tracing.Name = "jaeger"
	}
}

func WithOltp() Option {
	return func(opts *Options) {
		opts.Tracing.Name = "oltp"
	}
}

func WithStdout() Option {
	return func(opts *Options) {
		opts.Tracing.Name = "stdout"
	}
}

func WithServiceName(name string) Option {
	return func(opts *Options) {
		opts.Tracing.ServiceName = name
	}
}

func WithAddress(address string) Option {
	return func(opts *Options) {
		opts.Tracing.Address = address
	}
}

func WithUseAgent() Option {
	return func(opts *Options) {
		b := true
		opts.Tracing.UseAgent = &b
	}
}
