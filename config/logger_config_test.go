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
	"testing"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"github.com/stretchr/testify/assert"
)

func TestLoggerInit(t *testing.T) {
	t.Run("empty use default", func(t *testing.T) {
		err := Load(WithPath("./testdata/config/logger/empty_log.yaml"))
		assert.Nil(t, err)
		assert.NotNil(t, rootConfig)
		loggerConfig := rootConfig.Logger
		assert.NotNil(t, loggerConfig)
		assert.Equal(t, []string{"stderr"}, loggerConfig.ZapConfig.OutputPaths)
	})

	t.Run("use config", func(t *testing.T) {
		err := Load(WithPath("./testdata/config/logger/log.yaml"))
		assert.Nil(t, err)
		loggerConfig := rootConfig.Logger
		assert.NotNil(t, loggerConfig)
		// default
		assert.Equal(t, "debug", loggerConfig.ZapConfig.Level)
		assert.Equal(t, "message", loggerConfig.ZapConfig.EncoderConfig.MessageKey)
		assert.Equal(t, "stacktrace", loggerConfig.ZapConfig.EncoderConfig.StacktraceKey)
		logger.Info("hello")
	})

	t.Run("use config with file", func(t *testing.T) {
		err := Load(WithPath("./testdata/config/logger/file_log.yaml"))
		assert.Nil(t, err)
		loggerConfig := rootConfig.Logger
		assert.NotNil(t, loggerConfig)
		// default
		assert.Equal(t, "debug", loggerConfig.ZapConfig.Level)
		assert.Equal(t, "message", loggerConfig.ZapConfig.EncoderConfig.MessageKey)
		assert.Equal(t, "stacktrace", loggerConfig.ZapConfig.EncoderConfig.StacktraceKey)
		logger.Debug("debug")
		logger.Info("info")
		logger.Warn("warn")
		logger.Error("error")
		logger.Debugf("%s", "debug")
		logger.Infof("%s", "info")
		logger.Warnf("%s", "warn")
		logger.Errorf("%s", "error")
	})
}

func TestNewLoggerConfigBuilder(t *testing.T) {
	config := NewLoggerConfigBuilder().
		SetLumberjackConfig(nil).
		SetZapConfig(ZapConfig{}).
		Build()

	assert.NotNil(t, config)
	values := config.getUrlMap()
	assert.NotNil(t, values)
	err := config.check()
	assert.NoError(t, err)
}
