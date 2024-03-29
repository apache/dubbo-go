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
	"dubbo.apache.org/dubbo-go/v3/global"
)

type Options struct {
	Otel *global.OtelConfig
}

func defaultOptions() *Options {
	return &Options{Otel: global.DefaultOtelConfig()}
}

func NewOptions(opts ...Option) *Options {
	defOpts := defaultOptions()
	for _, opt := range opts {
		opt(defOpts)
	}
	return defOpts
}

type Option func(*Options)

func WithEnabled() Option {
	return func(opts *Options) {
		b := true
		opts.Otel.TracingConfig.Enable = &b
	}
}

func WithStdoutExporter() Option {
	return func(opts *Options) {
		opts.Otel.TracingConfig.Exporter = "stdout"
	}
}

func WithJaegerExporter() Option {
	return func(opts *Options) {
		opts.Otel.TracingConfig.Exporter = "jaeger"
	}
}

func WithZipkinExporter() Option {
	return func(opts *Options) {
		opts.Otel.TracingConfig.Exporter = "zipkin"
	}
}

func WithOtlpHttpExporter() Option {
	return func(opts *Options) {
		opts.Otel.TracingConfig.Exporter = "otlp-http"
	}
}

func WithOtlpGrpcExporter() Option {
	return func(opts *Options) {
		opts.Otel.TracingConfig.Exporter = "otlp-grpc"
	}
}

func WithExporter(exporter string) Option {
	return func(opts *Options) {
		opts.Otel.TracingConfig.Exporter = exporter
	}
}

// WithW3cPropagator w3c(standard)
func WithW3cPropagator() Option {
	return func(opts *Options) {
		opts.Otel.TracingConfig.Propagator = "w3c"
	}
}

// WithB3Propagator b3(for zipkin)
func WithB3Propagator() Option {
	return func(opts *Options) {
		opts.Otel.TracingConfig.Propagator = "b3"
	}
}

func WithPropagator(propagator string) Option {
	return func(opts *Options) {
		opts.Otel.TracingConfig.Propagator = propagator
	}
}

// WithRatio only takes effect when WithRatioMode is set
func WithRatio(ratio float64) Option {
	return func(opts *Options) {
		opts.Otel.TracingConfig.SampleRatio = ratio
	}
}

func WithRatioMode() Option {
	return func(opts *Options) {
		opts.Otel.TracingConfig.SampleMode = "ratio"
	}
}

func WithAlwaysMode() Option {
	return func(opts *Options) {
		opts.Otel.TracingConfig.SampleMode = "always"
	}
}

func WithNeverMode() Option {
	return func(opts *Options) {
		opts.Otel.TracingConfig.SampleMode = "never"
	}
}

func WithMode(mode string) Option {
	return func(opts *Options) {
		opts.Otel.TracingConfig.SampleMode = mode
	}
}

func WithEndpoint(endpoint string) Option {
	return func(opts *Options) {
		opts.Otel.TracingConfig.Endpoint = endpoint
	}
}
