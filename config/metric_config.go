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

	"github.com/dubbogo/gost/log/logger"

	"github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/metrics"
)

// MetricConfig This is the config struct for all metrics implementation
type MetricConfig struct {
	Mode               string `default:"pull" yaml:"mode" json:"mode,omitempty" property:"mode"` // push or pull,
	Namespace          string `default:"dubbo" yaml:"namespace" json:"namespace,omitempty" property:"namespace"`
	Enable             *bool  `default:"false" yaml:"enable" json:"enable,omitempty" property:"enable"`
	Port               string `default:"9090" yaml:"port" json:"port,omitempty" property:"port"`
	Path               string `default:"/metrics" yaml:"path" json:"path,omitempty" property:"path"`
	PushGatewayAddress string `default:"" yaml:"push-gateway-address" json:"push-gateway-address,omitempty" property:"push-gateway-address"`
	SummaryMaxAge      int64  `default:"600000000000" yaml:"summary-max-age" json:"summary-max-age,omitempty" property:"summary-max-age"`
	Protocol           string `default:"prometheus" yaml:"protocol" json:"protocol,omitempty" property:"protocol"`
	rootConfig         *RootConfig
}

func (mc *MetricConfig) ToReporterConfig() *metrics.ReporterConfig {
	defaultMetricsReportConfig := metrics.NewReporterConfig()
	if mc.Mode == metrics.ReportModePush {
		defaultMetricsReportConfig.Mode = metrics.ReportModePush
	}
	if mc.Namespace != "" {
		defaultMetricsReportConfig.Namespace = mc.Namespace
	}

	defaultMetricsReportConfig.Enable = *mc.Enable
	defaultMetricsReportConfig.Port = mc.Port
	defaultMetricsReportConfig.Path = mc.Path
	defaultMetricsReportConfig.PushGatewayAddress = mc.PushGatewayAddress
	defaultMetricsReportConfig.SummaryMaxAge = mc.SummaryMaxAge
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
	config := mc.ToReporterConfig()
	extension.GetMetricReporter(mc.Protocol, config)
	metrics.Init(mc.toURL())
	return nil
}

type MetricConfigBuilder struct {
	metricConfig *MetricConfig
}

func NewMetricConfigBuilder() *MetricConfigBuilder {
	return &MetricConfigBuilder{metricConfig: &MetricConfig{}}
}

func (mcb *MetricConfigBuilder) Build() *MetricConfig {
	return mcb.metricConfig
}

// DynamicUpdateProperties dynamically update properties.
func (mc *MetricConfig) DynamicUpdateProperties(newMetricConfig *MetricConfig) {
	if newMetricConfig != nil {
		if newMetricConfig.Enable != mc.Enable {
			mc.Enable = newMetricConfig.Enable
			logger.Infof("MetricConfig's Enable was dynamically updated, new value:%v", mc.Enable)

			extension.GetMetricReporter(mc.Protocol, mc.ToReporterConfig())
		}
	}
}

// prometheus://localhost:9090?&histogram.enabled=false&prometheus.exporter.enabled=false
func (mc *MetricConfig) toURL() *common.URL {
	url, _ := common.NewURL("localhost", common.WithProtocol(mc.Protocol))
	url.SetParam(constant.PrometheusExporterEnabledKey, strconv.FormatBool(*mc.Enable))
	url.SetParam(constant.PrometheusExporterMetricsPortKey, mc.Port)
	url.SetParam(constant.PrometheusExporterMetricsPathKey, mc.Path)
	url.SetParam(constant.ApplicationKey, mc.rootConfig.Application.Name)
	url.SetParam(constant.AppVersionKey, mc.rootConfig.Application.Version)
	return url
}
