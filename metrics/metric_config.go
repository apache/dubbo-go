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
}

func DefaultMetricConfig() *MetricConfig {
	// return a new config without setting any field means there is not any default value for initialization
	return &MetricConfig{}
}

type MetricOption func(*MetricConfig)

func WithMode(mode string) MetricOption {
	return func(cfg *MetricConfig) {
		cfg.Mode = mode
	}
}

func WithNamespace(namespace string) MetricOption {
	return func(cfg *MetricConfig) {
		cfg.Namespace = namespace
	}
}

func WithEnable(enable bool) MetricOption {
	return func(cfg *MetricConfig) {
		cfg.Enable = &enable
	}
}

func WithPort(port string) MetricOption {
	return func(cfg *MetricConfig) {
		cfg.Port = port
	}
}

func WithPath(path string) MetricOption {
	return func(cfg *MetricConfig) {
		cfg.Path = path
	}
}

func WithPushGatewayAddress(address string) MetricOption {
	return func(cfg *MetricConfig) {
		cfg.PushGatewayAddress = address
	}
}

func WithSummaryMaxAge(age int64) MetricOption {
	return func(cfg *MetricConfig) {
		cfg.SummaryMaxAge = age
	}
}

func WithProtocol(protocol string) MetricOption {
	return func(cfg *MetricConfig) {
		cfg.Protocol = protocol
	}
}
