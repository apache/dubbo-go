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
)

// This is the config struct for all metrics implementation
type MetricConfig struct {
	Reporters        []string          `yaml:"reporters" json:"reporters,omitempty"`
}

// parse the config from yml
func (c *MetricConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	if err := defaults.Set(c); err != nil {
		return err
	}
	type plain MetricConfig
	if err := unmarshal((*plain)(c)); err != nil {
		return err
	}
	return nil
}

// find the MetricConfig
// if it is nil, create a new one
func GetMetricConfig() *MetricConfig {
	if metricConfig == nil {
		metricConfig = &MetricConfig{}
	}
	return metricConfig
}


