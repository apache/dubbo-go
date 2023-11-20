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

package global

import (
	"strconv"
	"time"
)

// MethodConfig defines method config
type MethodConfig struct {
	InterfaceId                 string
	InterfaceName               string
	Name                        string `yaml:"name"  json:"name,omitempty" property:"name"`
	Retries                     string `yaml:"retries"  json:"retries,omitempty" property:"retries"`
	LoadBalance                 string `yaml:"loadbalance"  json:"loadbalance,omitempty" property:"loadbalance"`
	Weight                      int64  `yaml:"weight"  json:"weight,omitempty" property:"weight"`
	TpsLimitInterval            string `yaml:"tps.limit.interval" json:"tps.limit.interval,omitempty" property:"tps.limit.interval"`
	TpsLimitRate                string `yaml:"tps.limit.rate" json:"tps.limit.rate,omitempty" property:"tps.limit.rate"`
	TpsLimitStrategy            string `yaml:"tps.limit.strategy" json:"tps.limit.strategy,omitempty" property:"tps.limit.strategy"`
	ExecuteLimit                string `yaml:"execute.limit" json:"execute.limit,omitempty" property:"execute.limit"`
	ExecuteLimitRejectedHandler string `yaml:"execute.limit.rejected.handler" json:"execute.limit.rejected.handler,omitempty" property:"execute.limit.rejected.handler"`
	Sticky                      bool   `yaml:"sticky"   json:"sticky,omitempty" property:"sticky"`
	RequestTimeout              string `yaml:"timeout"  json:"timeout,omitempty" property:"timeout"`
}

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
	Method *MethodConfig
}

func defaultMethodOptions() *MethodOptions {
	return &MethodOptions{Method: &MethodConfig{}}
}

func NewMethodOptions(opts ...MethodOption) *MethodOptions {
	defOpts := defaultMethodOptions()
	for _, opt := range opts {
		opt(defOpts)
	}
	return defOpts
}
