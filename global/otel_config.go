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

// OtelConfig is the configuration of the tracing.
type OtelConfig struct {
	TracingConfig *OtelTraceConfig `yaml:"tracing" json:"trace,omitempty" property:"trace"`
}

type OtelTraceConfig struct {
	Enable      *bool   `default:"false" yaml:"enable" json:"enable,omitempty" property:"enable"`
	Exporter    string  `default:"stdout" yaml:"exporter" json:"exporter,omitempty" property:"exporter"` // stdout, jaeger, zipkin, otlp-http, otlp-grpc
	Endpoint    string  `default:"" yaml:"endpoint" json:"endpoint,omitempty" property:"endpoint"`
	Propagator  string  `default:"w3c" yaml:"propagator" json:"propagator,omitempty" property:"propagator"`       // one of w3c(standard), b3(for zipkin),
	SampleMode  string  `default:"ratio" yaml:"sample-mode" json:"sample-mode,omitempty" property:"sample-mode"`  // one of always, never, ratio
	SampleRatio float64 `default:"0.5" yaml:"sample-ratio" json:"sample-ratio,omitempty" property:"sample-ratio"` // [0.0, 1.0]
}

func DefaultOtelConfig() *OtelConfig {
	return &OtelConfig{
		TracingConfig: &OtelTraceConfig{},
	}
}

// Clone a new OtelConfig
func (c *OtelConfig) Clone() *OtelConfig {
	return &OtelConfig{
		TracingConfig: c.TracingConfig.Clone(),
	}
}

// Clone a new OtelTraceConfig
func (c *OtelTraceConfig) Clone() *OtelTraceConfig {
	var newEnable *bool
	if c.Enable != nil {
		newEnable = new(bool)
		*newEnable = *c.Enable
	}

	return &OtelTraceConfig{
		Enable:      newEnable,
		Exporter:    c.Exporter,
		Endpoint:    c.Endpoint,
		Propagator:  c.Propagator,
		SampleMode:  c.SampleMode,
		SampleRatio: c.SampleRatio,
	}
}
