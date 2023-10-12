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
	"dubbo.apache.org/dubbo-go/v3/global"
	"time"
)

type Options struct {
	Metric *global.MetricConfig
}

func defaultOptions() *Options {
	return &Options{Metric: global.DefaultMetricConfig()}
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
		opts.Metric.Aggregation.Enabled = &enabled
	}
}

func WithAggregationBucketNum(num int) Option {
	return func(opts *Options) {
		opts.Metric.Aggregation.BucketNum = num
	}
}

func WithAggregationTimeWindowSeconds(seconds int) Option {
	return func(opts *Options) {
		opts.Metric.Aggregation.TimeWindowSeconds = seconds
	}
}

func WithPrometheus() Option {
	return func(opts *Options) {
		opts.Metric.Protocol = "prometheus"
	}
}

func WithPrometheusExporterEnabled() Option {
	return func(opts *Options) {
		enabled := true
		opts.Metric.Prometheus.Exporter.Enabled = &enabled
	}
}

func WithPrometheusGatewayUrl(url string) Option {
	return func(opts *Options) {
		opts.Metric.Prometheus.Pushgateway.BaseUrl = url
	}
}

func WithPrometheusGatewayJob(job string) Option {
	return func(opts *Options) {
		opts.Metric.Prometheus.Pushgateway.Job = job
	}
}

func WithPrometheusGatewayUsername(username string) Option {
	return func(opts *Options) {
		opts.Metric.Prometheus.Pushgateway.Username = username
	}
}

func WithPrometheusGatewayPassword(password string) Option {
	return func(opts *Options) {
		opts.Metric.Prometheus.Pushgateway.Password = password
	}
}
func WithPrometheusGatewayInterval(interval time.Duration) Option {
	return func(opts *Options) {
		opts.Metric.Prometheus.Pushgateway.PushInterval = int(interval.Seconds())
	}
}

func WithEnabled() Option {
	return func(opts *Options) {
		b := true
		opts.Metric.Enable = &b
	}
}

func WithPort(port string) Option {
	return func(opts *Options) {
		opts.Metric.Port = port
	}
}

func WithPath(path string) Option {
	return func(opts *Options) {
		opts.Metric.Path = path
	}
}
