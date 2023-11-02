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
)

import (
	"github.com/creasty/defaults"

	"github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/metrics"
)

// MetricConfig This is the config struct for all metrics implementation
type MetricConfig struct {
	Enable             *bool             `default:"false" yaml:"enable" json:"enable,omitempty" property:"enable"`
	Port               string            `default:"9090" yaml:"port" json:"port,omitempty" property:"port"`
	Path               string            `default:"/metrics" yaml:"path" json:"path,omitempty" property:"path"`
	Protocol           string            `default:"prometheus" yaml:"protocol" json:"protocol,omitempty" property:"protocol"`
	EnableMetadata     *bool             `default:"false" yaml:"enable-metadata" json:"enable-metadata,omitempty" property:"enable-metadata"`
	EnableRegistry     *bool             `default:"false" yaml:"enable-registry" json:"enable-registry,omitempty" property:"enable-registry"`
	EnableConfigCenter *bool             `default:"false" yaml:"enable-config-center" json:"enable-config-center,omitempty" property:"enable-config-center"`
	Prometheus         *PrometheusConfig `yaml:"prometheus" json:"prometheus" property:"prometheus"`
	Aggregation        *AggregateConfig  `yaml:"aggregation" json:"aggregation" property:"aggregation"`
	rootConfig         *RootConfig
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
	Enabled *bool `default:"true" yaml:"enabled" json:"enabled,omitempty" property:"enabled"`
}

type PushgatewayConfig struct {
	Enabled      *bool  `default:"false" yaml:"enabled" json:"enabled,omitempty" property:"enabled"`
	BaseUrl      string `default:"" yaml:"base-url" json:"base-url,omitempty" property:"base-url"`
	Job          string `default:"default_dubbo_job" yaml:"job" json:"job,omitempty" property:"job"`
	Username     string `default:"" yaml:"username" json:"username,omitempty" property:"username"`
	Password     string `default:"" yaml:"password" json:"password,omitempty" property:"password"`
	PushInterval int    `default:"30" yaml:"push-interval" json:"push-interval,omitempty" property:"push-interval"`
}

func (mc *MetricConfig) ToReporterConfig() *metrics.ReporterConfig {
	defaultMetricsReportConfig := metrics.NewReporterConfig()

	defaultMetricsReportConfig.Enable = *mc.Enable
	defaultMetricsReportConfig.Port = mc.Port
	defaultMetricsReportConfig.Path = mc.Path
	defaultMetricsReportConfig.Protocol = mc.Protocol
	return defaultMetricsReportConfig
}

func (mc *MetricConfig) Init(rc *RootConfig) error {
	if mc == nil {
		return errors.New("metrics config is null")
	}
	if err := defaults.Set(mc); err != nil {
		return err
	}
	if err := verify(mc); err != nil {
		return err
	}
	mc.rootConfig = rc
	if *mc.Enable {
		metrics.Init(mc.toURL())
	}
	return nil
}

type MetricConfigBuilder struct {
	metricConfig *MetricConfig
}

func NewMetricConfigBuilder() *MetricConfigBuilder {
	return &MetricConfigBuilder{metricConfig: &MetricConfig{}}
}

func (mcb *MetricConfigBuilder) SetMetadataEnabled(enabled bool) *MetricConfigBuilder {
	mcb.metricConfig.EnableMetadata = &enabled
	return mcb
}

func (mcb *MetricConfigBuilder) SetRegistryEnabled(enabled bool) *MetricConfigBuilder {
	mcb.metricConfig.EnableRegistry = &enabled
	return mcb
}

func (mcb *MetricConfigBuilder) SetConfigCenterEnabled(enabled bool) *MetricConfigBuilder {
	mcb.metricConfig.EnableConfigCenter = &enabled
	return mcb
}

func (mcb *MetricConfigBuilder) Build() *MetricConfig {
	return mcb.metricConfig
}

// DynamicUpdateProperties dynamically update properties.
func (mc *MetricConfig) DynamicUpdateProperties(newMetricConfig *MetricConfig) {
	// TODO update
}

// prometheus://localhost:9090?&histogram.enabled=false&prometheus.exporter.enabled=false
func (mc *MetricConfig) toURL() *common.URL {
	url, _ := common.NewURL("localhost", common.WithProtocol(mc.Protocol))
	url.SetParam(constant.PrometheusExporterMetricsPortKey, mc.Port)
	url.SetParam(constant.PrometheusExporterMetricsPathKey, mc.Path)
	url.SetParam(constant.ApplicationKey, mc.rootConfig.Application.Name)
	url.SetParam(constant.AppVersionKey, mc.rootConfig.Application.Version)
	url.SetParam(constant.RpcEnabledKey, strconv.FormatBool(*mc.Enable))
	url.SetParam(constant.MetadataEnabledKey, strconv.FormatBool(*mc.EnableMetadata))
	url.SetParam(constant.RegistryEnabledKey, strconv.FormatBool(*mc.EnableRegistry))
	url.SetParam(constant.ConfigCenterEnabledKey, strconv.FormatBool(*mc.EnableConfigCenter))
	if mc.Aggregation != nil {
		url.SetParam(constant.AggregationEnabledKey, strconv.FormatBool(*mc.Aggregation.Enabled))
		url.SetParam(constant.AggregationBucketNumKey, strconv.Itoa(mc.Aggregation.BucketNum))
		url.SetParam(constant.AggregationTimeWindowSecondsKey, strconv.Itoa(mc.Aggregation.TimeWindowSeconds))
	}
	if mc.Prometheus != nil {
		if mc.Prometheus.Exporter != nil {
			exporter := mc.Prometheus.Exporter
			url.SetParam(constant.PrometheusExporterEnabledKey, strconv.FormatBool(*exporter.Enabled))
		}
		if mc.Prometheus.Pushgateway != nil {
			pushGateWay := mc.Prometheus.Pushgateway
			url.SetParam(constant.PrometheusPushgatewayEnabledKey, strconv.FormatBool(*pushGateWay.Enabled))
			url.SetParam(constant.PrometheusPushgatewayBaseUrlKey, pushGateWay.BaseUrl)
			url.SetParam(constant.PrometheusPushgatewayUsernameKey, pushGateWay.Username)
			url.SetParam(constant.PrometheusPushgatewayPasswordKey, pushGateWay.Password)
			url.SetParam(constant.PrometheusPushgatewayPushIntervalKey, strconv.Itoa(pushGateWay.PushInterval))
			url.SetParam(constant.PrometheusPushgatewayJobKey, pushGateWay.Job)
		}
	}
	return url
}
