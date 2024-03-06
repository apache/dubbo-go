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

package metrics

import (
	"strconv"
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/global"
)

type Options struct {
	Metrics *global.MetricsConfig
}

func defaultOptions() *Options {
	return &Options{Metrics: global.DefaultMetricsConfig()}
}

func NewOptions(opts ...Option) *Options {
	MetricOptions := defaultOptions()
	for _, opt := range opts {
		opt(MetricOptions)
	}
	return MetricOptions
}

type Option func(*Options)

func WithAggregationEnabled() Option {
	return func(opts *Options) {
		enabled := true
		opts.Metrics.Aggregation.Enabled = &enabled
	}
}

func WithAggregationBucketNum(num int) Option {
	return func(opts *Options) {
		opts.Metrics.Aggregation.BucketNum = num
	}
}

func WithAggregationTimeWindowSeconds(seconds int) Option {
	return func(opts *Options) {
		opts.Metrics.Aggregation.TimeWindowSeconds = seconds
	}
}

func WithPrometheus() Option {
	return func(opts *Options) {
		opts.Metrics.Protocol = "prometheus"
	}
}

func WithPrometheusExporterEnabled() Option {
	return func(opts *Options) {
		enabled := true
		opts.Metrics.Prometheus.Exporter.Enabled = &enabled
	}
}

func WithPrometheusPushgatewayEnabled() Option {
	return func(opts *Options) {
		enabled := true
		opts.Metrics.Prometheus.Pushgateway.Enabled = &enabled
	}
}

func WithPrometheusGatewayUrl(url string) Option {
	return func(opts *Options) {
		opts.Metrics.Prometheus.Pushgateway.BaseUrl = url
	}
}

func WithPrometheusGatewayJob(job string) Option {
	return func(opts *Options) {
		opts.Metrics.Prometheus.Pushgateway.Job = job
	}
}

func WithPrometheusGatewayUsername(username string) Option {
	return func(opts *Options) {
		opts.Metrics.Prometheus.Pushgateway.Username = username
	}
}

func WithPrometheusGatewayPassword(password string) Option {
	return func(opts *Options) {
		opts.Metrics.Prometheus.Pushgateway.Password = password
	}
}
func WithPrometheusGatewayInterval(interval time.Duration) Option {
	return func(opts *Options) {
		opts.Metrics.Prometheus.Pushgateway.PushInterval = int(interval.Seconds())
	}
}

func WithConfigCenterEnabled() Option {
	return func(opts *Options) {
		b := true
		opts.Metrics.EnableConfigCenter = &b
	}
}

func WithMetadataEnabled() Option {
	return func(opts *Options) {
		b := true
		opts.Metrics.EnableMetadata = &b
	}
}

func WithRegistryEnabled() Option {
	return func(opts *Options) {
		b := true
		opts.Metrics.EnableRegistry = &b
	}
}

// WithEnabled this will enable rpc and tracing by default, config-center, metadata and registry metrics will still be in disable state.
func WithEnabled() Option {
	return func(opts *Options) {
		b := true
		opts.Metrics.Enable = &b
	}
}

func WithPort(port int) Option {
	return func(opts *Options) {
		opts.Metrics.Port = strconv.Itoa(port)
	}
}

func WithPath(path string) Option {
	return func(opts *Options) {
		opts.Metrics.Path = path
	}
}
