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

package router

import (
	"dubbo.apache.org/dubbo-go/v3/global"
)

type Options struct {
	Router *global.RouterConfig
}

func defaultOptions() *Options {
	return &Options{
		Router: global.DefaultRouterConfig(),
	}
}

func NewOptions(opts ...Option) *Options {
	defOpts := defaultOptions()
	for _, opt := range opts {
		opt(defOpts)
	}
	return defOpts
}

type Option func(*Options)

func WithScope(scope string) Option {
	return func(opts *Options) {
		opts.Router.Scope = scope
	}
}

func WithKey(key string) Option {
	return func(opts *Options) {
		opts.Router.Key = key
	}
}

func WithForce(force bool) Option {
	return func(opts *Options) {
		opts.Router.Force = &force
	}
}

func WithRuntime(runtime bool) Option {
	return func(opts *Options) {
		opts.Router.Force = &runtime
	}
}

func WithEnabled(enabled bool) Option {
	return func(opts *Options) {
		opts.Router.Force = &enabled
	}
}

func WithValid(valid bool) Option {
	return func(opts *Options) {
		opts.Router.Valid = &valid
	}
}

func WithPriority(priority int) Option {
	return func(opts *Options) {
		opts.Router.Priority = priority
	}
}

func WithConditions(conditions []string) Option {
	return func(opts *Options) {
		opts.Router.Conditions = conditions
	}
}

func WithTags(tags []global.Tag) Option {
	return func(opts *Options) {
		opts.Router.Tags = tags
	}

}
