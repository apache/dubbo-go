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
	Enable             *bool             `default:"false" yaml:"enable" json:"enable,omitempty" property:"enable"`
	Port               string            `default:"9090" yaml:"port" json:"port,omitempty" property:"port"`
	Path               string            `default:"/metrics" yaml:"path" json:"path,omitempty" property:"path"`
	Protocol           string            `default:"prometheus" yaml:"protocol" json:"protocol,omitempty" property:"protocol"`
	Prometheus         *PrometheusConfig `yaml:"prometheus" json:"prometheus" property:"prometheus"`
	Aggregation        *AggregateConfig  `yaml:"aggregation" json:"aggregation" property:"aggregation"`
	EnableMetadata     *bool             `default:"true" yaml:"enable-metadata" json:"enable-metadata,omitempty" property:"enable-metadata"`
	EnableRegistry     *bool             `default:"true" yaml:"enable-registry" json:"enable-registry,omitempty" property:"enable-registry"`
	EnableConfigCenter *bool             `default:"true" yaml:"enable-config-center" json:"enable-config-center,omitempty" property:"enable-config-center"`
	EnableRpc          *bool             `default:"true" yaml:"enable-rpc" json:"enable-rpc,omitempty" property:"enable-rpc"`
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
	Enabled  *bool  `default:"false" yaml:"enabled" json:"enabled,omitempty" property:"enabled"`
	BaseUrl  string `default:"" yaml:"base-url" json:"base-url,omitempty" property:"base-url"`
	Job      string `default:"default_dubbo_job" yaml:"job" json:"job,omitempty" property:"job"`
	Username string `default:"" yaml:"username" json:"username,omitempty" property:"username"`
	Password string `default:"" yaml:"password" json:"password,omitempty" property:"password"`
	// seconds
	PushInterval int `default:"30" yaml:"push-interval" json:"push-interval,omitempty" property:"push-interval"`
}

func DefaultMetricConfig() *MetricConfig {
	// return a new config without setting any field means there is not any default value for initialization
	return &MetricConfig{Prometheus: defaultPrometheusConfig(), Aggregation: defaultAggregateConfig()}
}

func defaultPrometheusConfig() *PrometheusConfig {
	return &PrometheusConfig{Exporter: &Exporter{}, Pushgateway: &PushgatewayConfig{}}
}

func defaultAggregateConfig() *AggregateConfig {
	return &AggregateConfig{}
}
