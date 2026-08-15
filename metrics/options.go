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

// Options holds the configuration of the metrics module.
// It wraps the global MetricsConfig, and can be built programmatically
// with NewOptions and a set of Option functions.
//
// The metrics module is disabled by default; use WithEnabled to turn it on.
//
// NewOptions returns zero values for untouched fields; the defaults
// documented on the With* functions (e.g. port 9090, path "/metrics") only
// take effect at instance (client/server) initialization.
type Options struct {
	Metrics *global.MetricsConfig
}

// defaultOptions wraps a fresh global.MetricsConfig; no option is applied,
// so every field keeps its zero value until instance initialization.
func defaultOptions() *Options {
	return &Options{Metrics: global.DefaultMetricsConfig()}
}

// NewOptions wraps a fresh global.MetricsConfig and applies the given
// options in order; later options win on conflicting fields. Fields left
// untouched are zero-valued until filled with the defaults during instance
// (client/server) initialization.
func NewOptions(opts ...Option) *Options {
	MetricOptions := defaultOptions()
	for _, opt := range opts {
		opt(MetricOptions)
	}
	return MetricOptions
}

// Option is a functional option used to customize the metrics Options.
type Option func(*Options)

// WithAggregationEnabled enables metrics aggregation, such as the
// time-window based aggregation for counters and rt metrics.
//
// Aggregation is disabled by default (applied at instance initialization).
func WithAggregationEnabled() Option {
	return func(opts *Options) {
		enabled := true
		opts.Metrics.Aggregation.Enabled = &enabled
	}
}

// WithAggregationBucketNum sets the number of buckets used by metrics
// aggregation. A larger bucket count gives finer time resolution inside
// the time window at the cost of more memory.
//
// The default (10) is applied at instance initialization.
func WithAggregationBucketNum(num int) Option {
	return func(opts *Options) {
		opts.Metrics.Aggregation.BucketNum = num
	}
}

// WithAggregationTimeWindowSeconds sets the time window, in seconds,
// of the metrics aggregation. Metrics older than the window are
// discarded from the aggregation result.
//
// The default (120 seconds) is applied at instance initialization.
func WithAggregationTimeWindowSeconds(seconds int) Option {
	return func(opts *Options) {
		opts.Metrics.Aggregation.TimeWindowSeconds = seconds
	}
}

// WithPrometheus sets the metrics protocol to prometheus.
//
// Prometheus is the default protocol (applied at instance initialization),
// so this option is only needed to switch back to it after another
// protocol has been configured.
func WithPrometheus() Option {
	return func(opts *Options) {
		opts.Metrics.Protocol = "prometheus"
	}
}

// WithPrometheusExporterEnabled enables the prometheus exporter, which
// exposes the collected metrics over the http endpoint configured by
// WithPort and WithPath.
//
// The exporter is enabled by default (applied at instance initialization).
func WithPrometheusExporterEnabled() Option {
	return func(opts *Options) {
		enabled := true
		opts.Metrics.Prometheus.Exporter.Enabled = &enabled
	}
}

// WithPrometheusPushgatewayEnabled enables pushing metrics to the
// prometheus pushgateway, so that they can be scraped by prometheus
// even if the instance is short-lived or unreachable directly.
//
// Pushgateway is disabled by default (applied at instance initialization).
func WithPrometheusPushgatewayEnabled() Option {
	return func(opts *Options) {
		enabled := true
		opts.Metrics.Prometheus.Pushgateway.Enabled = &enabled
	}
}

// WithPrometheusGatewayUrl sets the base url of the prometheus
// pushgateway, e.g. "http://pushgateway:9091".
//
// The option has no default value: it is not filled in at instance
// initialization and must be set when pushgateway is enabled.
func WithPrometheusGatewayUrl(url string) Option {
	return func(opts *Options) {
		opts.Metrics.Prometheus.Pushgateway.BaseUrl = url
	}
}

// WithPrometheusGatewayJob sets the job name reported to the prometheus
// pushgateway. It is used to group the pushed metrics in prometheus.
//
// The default ("default_dubbo_job") is applied at instance initialization.
func WithPrometheusGatewayJob(job string) Option {
	return func(opts *Options) {
		opts.Metrics.Prometheus.Pushgateway.Job = job
	}
}

// WithPrometheusGatewayUsername sets the username for basic
// authentication with the prometheus pushgateway.
//
// The default (empty username, i.e. no authentication) is applied at
// instance initialization.
func WithPrometheusGatewayUsername(username string) Option {
	return func(opts *Options) {
		opts.Metrics.Prometheus.Pushgateway.Username = username
	}
}

// WithPrometheusGatewayPassword sets the password for basic
// authentication with the prometheus pushgateway.
//
// The default (empty password, i.e. no authentication) is applied at
// instance initialization.
func WithPrometheusGatewayPassword(password string) Option {
	return func(opts *Options) {
		opts.Metrics.Prometheus.Pushgateway.Password = password
	}
}

// WithPrometheusGatewayInterval sets the interval at which metrics are
// pushed to the prometheus pushgateway.
//
// The default (30 seconds) is applied at instance initialization.
func WithPrometheusGatewayInterval(interval time.Duration) Option {
	return func(opts *Options) {
		opts.Metrics.Prometheus.Pushgateway.PushInterval = int(interval.Seconds())
	}
}

// WithConfigCenterEnabled enables the config-center metrics, which
// report the state of the dynamic configuration center (e.g. the
// configuration that the instance has loaded or subscribed).
//
// Config-center metrics are disabled by default (applied at instance initialization).
func WithConfigCenterEnabled() Option {
	return func(opts *Options) {
		b := true
		opts.Metrics.EnableConfigCenter = &b
	}
}

// WithMetadataEnabled enables the metadata metrics, which report the
// operations of the metadata center (e.g. store provider metadata).
//
// Metadata metrics are disabled by default (applied at instance initialization).
func WithMetadataEnabled() Option {
	return func(opts *Options) {
		b := true
		opts.Metrics.EnableMetadata = &b
	}
}

// WithRegistryEnabled enables the registry metrics, which report the
// interactions with the service registry (e.g. register, subscribe).
//
// Registry metrics are disabled by default (applied at instance initialization).
func WithRegistryEnabled() Option {
	return func(opts *Options) {
		b := true
		opts.Metrics.EnableRegistry = &b
	}
}

// WithEnabled enables the metrics module. It enables the rpc metrics
// by default, while config-center, metadata and registry metrics
// are still in disable state and need WithConfigCenterEnabled,
// WithMetadataEnabled and WithRegistryEnabled respectively.
//
// The metrics module is disabled by default (applied at instance initialization).
func WithEnabled() Option {
	return func(opts *Options) {
		b := true
		opts.Metrics.Enable = &b
	}
}

// WithPort sets the port on which the metrics are exposed.
//
// The default (9090) is applied at instance initialization.
func WithPort(port int) Option {
	return func(opts *Options) {
		opts.Metrics.Port = strconv.Itoa(port)
	}
}

// WithPath sets the http path on which the metrics are exposed.
//
// The default ("/metrics") is applied at instance initialization.
func WithPath(path string) Option {
	return func(opts *Options) {
		opts.Metrics.Path = path
	}
}

// WithProbeEnabled enables the health probe endpoints
// (liveness, readiness and startup), which are typically used
// by Kubernetes for container health checks.
//
// Probe endpoints are disabled by default (applied at instance initialization).
func WithProbeEnabled() Option {
	return func(opts *Options) {
		b := true
		opts.Metrics.Probe.Enabled = &b
	}
}

// WithProbePort sets the port on which the probe endpoints are served.
//
// The default (22222) is applied at instance initialization.
func WithProbePort(port int) Option {
	return func(opts *Options) {
		opts.Metrics.Probe.Port = strconv.Itoa(port)
	}
}

// WithProbeLivenessPath sets the http path of the liveness probe.
//
// The default ("/live") is applied at instance initialization.
func WithProbeLivenessPath(path string) Option {
	return func(opts *Options) {
		opts.Metrics.Probe.LivenessPath = path
	}
}

// WithProbeReadinessPath sets the http path of the readiness probe.
//
// The default ("/ready") is applied at instance initialization.
func WithProbeReadinessPath(path string) Option {
	return func(opts *Options) {
		opts.Metrics.Probe.ReadinessPath = path
	}
}

// WithProbeStartupPath sets the http path of the startup probe.
//
// The default ("/startup") is applied at instance initialization.
func WithProbeStartupPath(path string) Option {
	return func(opts *Options) {
		opts.Metrics.Probe.StartupPath = path
	}
}

// WithProbeUseInternalState sets whether the probe endpoints report the
// internal state of the framework (e.g. whether the dubbo server has
// started up and is ready to serve) as the probe result, instead of
// answering only with the result of the registered custom checks.
//
// The default (true) is applied at instance initialization.
func WithProbeUseInternalState(use bool) Option {
	return func(opts *Options) {
		opts.Metrics.Probe.UseInternalState = &use
	}
}
