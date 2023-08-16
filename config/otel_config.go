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

import (
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/otel/trace"
)

type OtelConfig struct {
	TraceConfig *OtelTraceConfig `yaml:"trace" json:"trace,omitempty" property:"trace"`
}

type OtelTraceConfig struct {
	Enable      *bool   `default:"false" yaml:"enable" json:"enable,omitempty" property:"enable"`
	Exporter    string  `default:"jaeger" yaml:"exporter" json:"exporter,omitempty" property:"exporter"`
	Endpoint    string  `default:"http://localhost:14268/api/traces" yaml:"endpoint" json:"endpoint,omitempty" property:"endpoint"`
	SampleMode  string  `default:"ratio" yaml:"sample-mode" json:"sample-mode,omitempty" property:"sample-mode"`  // one of always, never, ratio
	SampleRatio float64 `default:"0.5" yaml:"sample-ratio" json:"sample-ratio,omitempty" property:"sample-ratio"` // [0.0, 1.0]
}

func (oc *OtelConfig) Init() error {
	if oc == nil {
		return errors.New("otel config is nil")
	}
	if err := defaults.Set(oc); err != nil {
		return err
	}
	if err := verify(oc); err != nil {
		return err
	}
	if !*oc.TraceConfig.Enable {
		return nil
	}

	return extension.GetTraceProvider(oc.TraceConfig.Exporter, oc.TraceConfig.toTraceProviderConfig())
}

func (oc *OtelTraceConfig) toTraceProviderConfig() *trace.TraceProviderConfig {
	tpc := &trace.TraceProviderConfig{
		Exporter:    oc.Exporter,
		Endpoint:    oc.Endpoint,
		SampleMode:  oc.SampleMode,
		SampleRatio: oc.SampleRatio,
	}
	return tpc
}

type OtelConfigBuilder struct {
	otelConfig *OtelConfig
}

func NewOtelConfigBuilder() *OtelConfigBuilder {
	return &OtelConfigBuilder{
		otelConfig: &OtelConfig{
			TraceConfig: &OtelTraceConfig{},
		},
	}
}

func (ocb *OtelConfigBuilder) Build() *OtelConfig {
	return ocb.otelConfig
}

// TODO: dynamic config
