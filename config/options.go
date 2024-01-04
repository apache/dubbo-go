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

package config

import (
	"strconv"
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/global"
)

type MethodOption func(*MethodOptions)

func WithInterfaceId(id string) MethodOption {
	return func(opts *MethodOptions) {
		opts.Method.InterfaceId = id
	}
}

func WithInterfaceName(name string) MethodOption {
	return func(opts *MethodOptions) {
		opts.Method.InterfaceName = name
	}
}

func WithName(name string) MethodOption {
	return func(opts *MethodOptions) {
		opts.Method.Name = name
	}
}

func WithRetries(retries int) MethodOption {
	return func(opts *MethodOptions) {
		opts.Method.Retries = strconv.Itoa(retries)
	}
}

func WithLoadBalance(lb string) MethodOption {
	return func(opts *MethodOptions) {
		opts.Method.LoadBalance = lb
	}
}

func WithWeight(weight int64) MethodOption {
	return func(opts *MethodOptions) {
		opts.Method.Weight = weight
	}
}

func WithTpsLimitInterval(interval int) MethodOption {
	return func(opts *MethodOptions) {
		opts.Method.TpsLimitInterval = strconv.Itoa(interval)
	}
}

func WithTpsLimitRate(rate int) MethodOption {
	return func(opts *MethodOptions) {
		opts.Method.TpsLimitRate = strconv.Itoa(rate)
	}
}

func WithTpsLimitStrategy(strategy string) MethodOption {
	return func(opts *MethodOptions) {
		opts.Method.TpsLimitStrategy = strategy
	}
}

func WithExecuteLimit(limit int) MethodOption {
	return func(opts *MethodOptions) {
		opts.Method.ExecuteLimit = strconv.Itoa(limit)
	}
}

func WithExecuteLimitRejectedHandler(handler string) MethodOption {
	return func(opts *MethodOptions) {
		opts.Method.ExecuteLimitRejectedHandler = handler
	}
}

func WithSticky() MethodOption {
	return func(opts *MethodOptions) {
		opts.Method.Sticky = true
	}
}

func WithRequestTimeout(millSeconds time.Duration) MethodOption {
	return func(opts *MethodOptions) {
		opts.Method.RequestTimeout = millSeconds.String()
	}
}

type MethodOptions struct {
	Method *global.MethodConfig
}

func defaultMethodOptions() *MethodOptions {
	return &MethodOptions{Method: &global.MethodConfig{}}
}

func NewMethodOptions(opts ...MethodOption) *MethodOptions {
	defOpts := defaultMethodOptions()
	for _, opt := range opts {
		opt(defOpts)
	}
	return defOpts
}
