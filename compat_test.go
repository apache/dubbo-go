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

package dubbo

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/global"
)

// TestCompatLoggerConfigWithTraceIntegration tests the conversion functions for LoggerConfig with TraceIntegration
func TestCompatLoggerConfigWithTraceIntegration(t *testing.T) {
	t.Run("global_to_config_with_trace_integration", func(t *testing.T) {
		enabled := true
		recordError := false
		globalCfg := &global.LoggerConfig{
			Driver:   "zap",
			Level:    "info",
			Format:   "json",
			Appender: "console",
			TraceIntegration: &global.TraceIntegrationConfig{
				Enabled:           &enabled,
				RecordErrorToSpan: &recordError,
			},
		}

		configCfg := compatLoggerConfig(globalCfg)

		assert.NotNil(t, configCfg)
		assert.Equal(t, globalCfg.Driver, configCfg.Driver)
		assert.Equal(t, globalCfg.Level, configCfg.Level)
		assert.Equal(t, globalCfg.Format, configCfg.Format)
		assert.Equal(t, globalCfg.Appender, configCfg.Appender)
		assert.NotNil(t, configCfg.TraceIntegration)
		assert.Equal(t, *globalCfg.TraceIntegration.Enabled, *configCfg.TraceIntegration.Enabled)
		assert.Equal(t, *globalCfg.TraceIntegration.RecordErrorToSpan, *configCfg.TraceIntegration.RecordErrorToSpan)
	})

	t.Run("global_to_config_with_nil_trace_integration", func(t *testing.T) {
		globalCfg := &global.LoggerConfig{
			Driver:           "zap",
			Level:            "info",
			TraceIntegration: nil,
		}

		configCfg := compatLoggerConfig(globalCfg)

		assert.NotNil(t, configCfg)
		assert.Nil(t, configCfg.TraceIntegration)
	})

	t.Run("global_to_config_nil", func(t *testing.T) {
		var globalCfg *global.LoggerConfig
		configCfg := compatLoggerConfig(globalCfg)
		assert.Nil(t, configCfg)
	})

	t.Run("config_to_global_with_trace_integration", func(t *testing.T) {
		enabled := true
		recordError := false
		configCfg := &config.LoggerConfig{
			Driver:   "zap",
			Level:    "info",
			Format:   "json",
			Appender: "console",
			TraceIntegration: &config.TraceIntegrationConfig{
				Enabled:           &enabled,
				RecordErrorToSpan: &recordError,
			},
		}

		globalCfg := compatGlobalLoggerConfig(configCfg)

		assert.NotNil(t, globalCfg)
		assert.Equal(t, configCfg.Driver, globalCfg.Driver)
		assert.Equal(t, configCfg.Level, globalCfg.Level)
		assert.Equal(t, configCfg.Format, globalCfg.Format)
		assert.Equal(t, configCfg.Appender, globalCfg.Appender)
		assert.NotNil(t, globalCfg.TraceIntegration)
		assert.Equal(t, *configCfg.TraceIntegration.Enabled, *globalCfg.TraceIntegration.Enabled)
		assert.Equal(t, *configCfg.TraceIntegration.RecordErrorToSpan, *globalCfg.TraceIntegration.RecordErrorToSpan)
	})

	t.Run("config_to_global_with_nil_trace_integration", func(t *testing.T) {
		configCfg := &config.LoggerConfig{
			Driver:           "zap",
			Level:            "info",
			TraceIntegration: nil,
		}

		globalCfg := compatGlobalLoggerConfig(configCfg)

		assert.NotNil(t, globalCfg)
		assert.Nil(t, globalCfg.TraceIntegration)
	})

	t.Run("config_to_global_nil", func(t *testing.T) {
		var configCfg *config.LoggerConfig
		globalCfg := compatGlobalLoggerConfig(configCfg)
		assert.Nil(t, globalCfg)
	})

	t.Run("round_trip_conversion", func(t *testing.T) {
		enabled := true
		recordError := false
		originalGlobal := &global.LoggerConfig{
			Driver: "zap",
			Level:  "debug",
			TraceIntegration: &global.TraceIntegrationConfig{
				Enabled:           &enabled,
				RecordErrorToSpan: &recordError,
			},
		}

		// global -> config
		intermediateConfig := compatLoggerConfig(originalGlobal)
		assert.NotNil(t, intermediateConfig.TraceIntegration)

		// config -> global
		convertedGlobal := compatGlobalLoggerConfig(intermediateConfig)
		assert.NotNil(t, convertedGlobal.TraceIntegration)

		// Verify values are preserved
		assert.Equal(t, originalGlobal.Driver, convertedGlobal.Driver)
		assert.Equal(t, originalGlobal.Level, convertedGlobal.Level)
		assert.Equal(t, *originalGlobal.TraceIntegration.Enabled, *convertedGlobal.TraceIntegration.Enabled)
		assert.Equal(t, *originalGlobal.TraceIntegration.RecordErrorToSpan, *convertedGlobal.TraceIntegration.RecordErrorToSpan)
	})
}

// TestCompatTraceIntegrationConfig tests the helper conversion functions
func TestCompatTraceIntegrationConfig(t *testing.T) {
	t.Run("compatTraceIntegrationConfig_with_values", func(t *testing.T) {
		enabled := true
		recordError := false
		globalCfg := &global.TraceIntegrationConfig{
			Enabled:           &enabled,
			RecordErrorToSpan: &recordError,
		}

		configCfg := compatTraceIntegrationConfig(globalCfg)

		assert.NotNil(t, configCfg)
		assert.Equal(t, *globalCfg.Enabled, *configCfg.Enabled)
		assert.Equal(t, *globalCfg.RecordErrorToSpan, *configCfg.RecordErrorToSpan)
	})

	t.Run("compatTraceIntegrationConfig_nil", func(t *testing.T) {
		var globalCfg *global.TraceIntegrationConfig
		configCfg := compatTraceIntegrationConfig(globalCfg)
		assert.Nil(t, configCfg)
	})

	t.Run("compatGlobalTraceIntegrationConfig_with_values", func(t *testing.T) {
		enabled := true
		recordError := false
		configCfg := &config.TraceIntegrationConfig{
			Enabled:           &enabled,
			RecordErrorToSpan: &recordError,
		}

		globalCfg := compatGlobalTraceIntegrationConfig(configCfg)

		assert.NotNil(t, globalCfg)
		assert.Equal(t, *configCfg.Enabled, *globalCfg.Enabled)
		assert.Equal(t, *configCfg.RecordErrorToSpan, *globalCfg.RecordErrorToSpan)
	})

	t.Run("compatGlobalTraceIntegrationConfig_nil", func(t *testing.T) {
		var configCfg *config.TraceIntegrationConfig
		globalCfg := compatGlobalTraceIntegrationConfig(configCfg)
		assert.Nil(t, globalCfg)
	})
}
