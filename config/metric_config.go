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

var defaultHistogramBucket = []float64{10, 50, 100, 200, 500, 1000, 10000}

// MetricConfig This is the config struct for all metrics implementation
type MetricConfig struct {
	Reporters []string `yaml:"reporters" json:"reporters,omitempty"`
	// TODO s?
	HistogramBucket []float64 `yaml:"histogram_bucket" json:"histogram_bucket,omitempty"`
}

// nolint
func (mc *MetricConfig) Init() error {
	return nil
}

// GetHistogramBucket find the histogram bucket
// if it's empty, the default value will be return
func (mc *MetricConfig) GetHistogramBucket() []float64 {
	if len(mc.HistogramBucket) == 0 {
		mc.HistogramBucket = defaultHistogramBucket
	}
	return mc.HistogramBucket
}

type MetricConfigBuilder struct {
	metricConfig *MetricConfig
}

// nolint
func NewMetricConfigBuilder() *MetricConfigBuilder {
	return &MetricConfigBuilder{metricConfig: &MetricConfig{}}
}

// nolint
func (mcb *MetricConfigBuilder) SetReporters(reporters []string) *MetricConfigBuilder {
	mcb.metricConfig.Reporters = reporters
	return mcb
}

// nolint
func (mcb *MetricConfigBuilder) AddReporter(reporter string) *MetricConfigBuilder {
	if mcb.metricConfig.Reporters == nil {
		mcb.metricConfig.Reporters = make([]string, 0)
	}
	mcb.metricConfig.Reporters = append(mcb.metricConfig.Reporters, reporter)
	return mcb
}

// nolint
func (mcb *MetricConfigBuilder) SetHistogramBucket(histogramBucket []float64) *MetricConfigBuilder {
	mcb.metricConfig.HistogramBucket = histogramBucket
	return mcb
}

// nolint
func (mcb *MetricConfigBuilder) AddBucket(bucket float64) *MetricConfigBuilder {
	if mcb.metricConfig.HistogramBucket == nil {
		mcb.metricConfig.HistogramBucket = make([]float64, 0)
	}
	mcb.metricConfig.HistogramBucket = append(mcb.metricConfig.HistogramBucket, bucket)
	return mcb
}

// nolint
func (mcb *MetricConfigBuilder) Build() *MetricConfig {
	if err := mcb.metricConfig.Init(); err != nil {
		panic(err)
	}
	return mcb.metricConfig
}
