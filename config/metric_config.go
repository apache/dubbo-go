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
	"github.com/creasty/defaults"
	"github.com/pkg/errors"
)

// MetricConfig This is the config struct for all metrics implementation
type MetricConfig struct {
	Reporters   []string           `default:"[\"prometheus\"]" yaml:"reporters" json:"reporters,omitempty"`
	Mode        string             `default:"pull" yaml:"mode" json:"mode,omitempty" property:"mode"` // push or pull,
	Namespace   string             `default:"dubbo" yaml:"namespace" json:"namespace,omitempty" property:"namespace"`
	Enable      bool               `default:"false" yaml:"enable" json:"enable,omitempty" property:"enable"`
	Port        string             `default:"9090" yaml:"port" json:"port,omitempty" property:"port"`
	Path        string             `default:"/metrics" yaml:"path" json:"path,omitempty" property:"path"`
	Pushgateway *PushGateWayConfig `yaml:"pushgateway" json:"pushgateway,omitempty"`
}

type PushGateWayConfig struct {
	Enabled     bool              `default:"false" yaml:"enabled",json:"enabled,omitempty"`
	BaseUrl     string            `yaml:"base-url" json:"base-url,omitempty"`
	PushRate    string            `default:"60s" yaml:"push-rate" json:"push-rate,omitempty"`
	GroupingKey map[string]string `yaml:"grouping-key" json:"grouping-key,omitempty"`
	Job         string            `yaml:"job" json:"job,omitempty"`
	BasicAuth   bool              `yaml:"basicAuth" json:"basicAuth,omitempty"`
	Username    string            `yaml:"username" json:"username,omitempty"`
	Password    string            `yaml:"password" json:"password,omitempty"`
}

func (c *PushGateWayConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	if err := defaults.Set(c); err != nil {
		return err
	}
	type plain PushGateWayConfig
	return unmarshal((*plain)(c))
}

// nolint
func (mc *MetricConfig) Init() error {
	if mc == nil {
		return errors.New("metrics config is null")
	}
	if err := defaults.Set(mc); err != nil {
		return err
	}
	if err := verify(mc); err != nil {
		return err
	}
	//extension.GetMetricReporter("prometheus")
	return nil
}

type MetricConfigBuilder struct {
	metricConfig *MetricConfig
}

// nolint
func NewMetricConfigBuilder() *MetricConfigBuilder {
	return &MetricConfigBuilder{metricConfig: &MetricConfig{}}
}

// nolint
func (mcb *MetricConfigBuilder) Build() *MetricConfig {
	return mcb.metricConfig
}
