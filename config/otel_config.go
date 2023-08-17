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
	"github.com/dubbogo/gost/log/logger"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
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
	Exporter    string  `default:"jaeger" yaml:"exporter" json:"exporter,omitempty" property:"exporter"` // jaeger, zipkin, OTLP
	Endpoint    string  `default:"http://localhost:14268/api/traces" yaml:"endpoint" json:"endpoint,omitempty" property:"endpoint"`
	Propagator  string  `default:"w3c" yaml:"propagator" json:"propagator,omitempty" property:"propagator"`       // one of w3c(standard), b3(for zipkin),
	SampleMode  string  `default:"ratio" yaml:"sample-mode" json:"sample-mode,omitempty" property:"sample-mode"`  // one of always, never, ratio
	SampleRatio float64 `default:"0.5" yaml:"sample-ratio" json:"sample-ratio,omitempty" property:"sample-ratio"` // [0.0, 1.0]
}

func (oc *OtelConfig) Init(appConfig *ApplicationConfig) error {
	if oc == nil {
		return errors.New("otel config is nil")
	}
	if err := defaults.Set(oc); err != nil {
		return err
	}
	if err := verify(oc); err != nil {
		return err
	}
	if *oc.TraceConfig.Enable {
		return oc.TraceConfig.init(appConfig)
	}

	return nil
}

func (c *OtelTraceConfig) init(appConfig *ApplicationConfig) error {
	exporter, err := extension.GetTraceExporter(c.Exporter, c.toTraceProviderConfig(appConfig))
	if err != nil {
		return err
	}
	otel.SetTracerProvider(exporter.GetTracerProvider())
	otel.SetTextMapPropagator(exporter.GetPropagator())

	// print trace exporter configuration
	logger.Infof("%s trace provider with endpoint: %s, propagator: %s", c.Exporter, c.Endpoint, c.Propagator)
	logger.Infof("sample mode: %s", c.SampleMode)
	if c.SampleMode == "ratio" {
		logger.Infof("sample ratio: %.2f", c.SampleRatio)
	}

	return nil
}

func (c *OtelTraceConfig) toTraceProviderConfig(a *ApplicationConfig) *trace.ExporterConfig {
	tpc := &trace.ExporterConfig{
		Exporter:         c.Exporter,
		Endpoint:         c.Endpoint,
		SampleMode:       c.SampleMode,
		SampleRatio:      c.SampleRatio,
		Propagator:       c.Propagator,
		ServiceNamespace: a.Organization,
		ServiceName:      a.Name,
		ServiceVersion:   a.Version,
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
