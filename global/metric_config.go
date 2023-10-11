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

// MetricConfig This is the config struct for all metrics implementation
type MetricConfig struct {
	Enable      *bool             `default:"false" yaml:"enable" json:"enable,omitempty" property:"enable"`
	Port        string            `default:"9090" yaml:"port" json:"port,omitempty" property:"port"`
	Path        string            `default:"/metrics" yaml:"path" json:"path,omitempty" property:"path"`
	Protocol    string            `default:"prometheus" yaml:"protocol" json:"protocol,omitempty" property:"protocol"`
	Prometheus  *PrometheusConfig `yaml:"prometheus" json:"prometheus" property:"prometheus"`
	Aggregation *AggregateConfig  `yaml:"aggregation" json:"aggregation" property:"aggregation"`
}

type AggregateConfig struct {
	Enabled           *bool `default:"false" yaml:"enabled" json:"enabled,omitempty" property:"enabled"`
	BucketNum         int   `default:"10" yaml:"bucket-num" json:"bucket-num,omitempty" property:"bucket-num"`
	TimeWindowSeconds int   `default:"120" yaml:"time-window-seconds" json:"time-window-seconds,omitempty" property:"time-window-seconds"`
}

type PrometheusConfig struct {
	Exporter    *Exporter          `yaml:"exporter" json:"exporter,omitempty" property:"exporter"`
	Pushgateway *PushgatewayConfig `yaml:"pushgateway" json:"pushgateway,omitempty" property:"pushgateway"`
}

type Exporter struct {
	Enabled *bool `default:"false" yaml:"enabled" json:"enabled,omitempty" property:"enabled"`
}

type PushgatewayConfig struct {
	Enabled      *bool  `default:"false" yaml:"enabled" json:"enabled,omitempty" property:"enabled"`
	BaseUrl      string `default:"" yaml:"base-url" json:"base-url,omitempty" property:"base-url"`
	Job          string `default:"default_dubbo_job" yaml:"job" json:"job,omitempty" property:"job"`
	Username     string `default:"" yaml:"username" json:"username,omitempty" property:"username"`
	Password     string `default:"" yaml:"password" json:"password,omitempty" property:"password"`
	PushInterval int    `default:"30" yaml:"push-interval" json:"push-interval,omitempty" property:"push-interval"`
}

func DefaultMetricConfig() *MetricConfig {
	// return a new config without setting any field means there is not any default value for initialization
	return &MetricConfig{}
}

func defaultPrometheusConfig() *PrometheusConfig {
	return &PrometheusConfig{Exporter: &Exporter{}, Pushgateway: &PushgatewayConfig{}}
}

type MetricOption func(*MetricConfig)

func WithMetric_AggregateEnabled() MetricOption {
	return func(cfg *MetricConfig) {
		if cfg.Aggregation == nil {
			cfg.Aggregation = &AggregateConfig{}
		}
		enabled := true
		cfg.Aggregation.Enabled = &enabled
	}
}

func WithMetric_AggregateBucketNum(num int) MetricOption {
	return func(cfg *MetricConfig) {
		if cfg.Aggregation == nil {
			cfg.Aggregation = &AggregateConfig{}
		}
		cfg.Aggregation.BucketNum = num
	}
}

func WithMetric_AggregateTimeWindowSeconds(seconds int) MetricOption {
	return func(cfg *MetricConfig) {
		if cfg.Aggregation == nil {
			cfg.Aggregation = &AggregateConfig{}
		}
		cfg.Aggregation.TimeWindowSeconds = seconds
	}
}

func WithMetric_PrometheusEnabled() MetricOption {
	return func(cfg *MetricConfig) {
		if cfg.Prometheus == nil {
			cfg.Prometheus.Exporter = &Exporter{}
		}
		enabled := true
		cfg.Prometheus.Exporter.Enabled = &enabled
	}
}

func WithMetric_PrometheusGatewayUrl(url string) MetricOption {
	return func(cfg *MetricConfig) {
		if cfg.Prometheus == nil {
			cfg.Prometheus = defaultPrometheusConfig()
		}
		cfg.Prometheus.Pushgateway.BaseUrl = url
	}
}

func WithMetric_PrometheusGatewayJob(job string) MetricOption {
	return func(cfg *MetricConfig) {
		if cfg.Prometheus == nil {
			cfg.Prometheus = defaultPrometheusConfig()
		}
		cfg.Prometheus.Pushgateway.Job = job
	}
}

func WithMetric_PrometheusGatewayUsername(username string) MetricOption {
	return func(cfg *MetricConfig) {
		if cfg.Prometheus == nil {
			cfg.Prometheus = defaultPrometheusConfig()
		}
		cfg.Prometheus.Pushgateway.Username = username
	}
}

func WithMetric_PrometheusGatewayPassword(password string) MetricOption {
	return func(cfg *MetricConfig) {
		if cfg.Prometheus == nil {
			cfg.Prometheus = defaultPrometheusConfig()
		}
		cfg.Prometheus.Pushgateway.Password = password
	}
}
func WithMetric_PrometheusGatewayInterval(interval int) MetricOption {
	return func(cfg *MetricConfig) {
		if cfg.Prometheus == nil {
			cfg.Prometheus = defaultPrometheusConfig()
		}
		cfg.Prometheus.Pushgateway.PushInterval = interval
	}
}

func WithMetric_Enable(enable bool) MetricOption {
	return func(cfg *MetricConfig) {
		cfg.Enable = &enable
	}
}

func WithMetric_Port(port string) MetricOption {
	return func(cfg *MetricConfig) {
		cfg.Port = port
	}
}

func WithMetric_Path(path string) MetricOption {
	return func(cfg *MetricConfig) {
		cfg.Path = path
	}
}

func WithMetric_Protocol(protocol string) MetricOption {
	return func(cfg *MetricConfig) {
		cfg.Protocol = protocol
	}
}
