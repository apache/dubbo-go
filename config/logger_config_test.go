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

import (
	cfgcenter "dubbo.apache.org/dubbo-go/v3/config_center"
	dlogger "dubbo.apache.org/dubbo-go/v3/logger"
)

func TestLoggerInit(t *testing.T) {
	t.Run("empty use default", func(t *testing.T) {
		err := Load(WithPath("./testdata/config/logger/empty_log.yaml"))
		assert.Nil(t, err)
		assert.NotNil(t, rootConfig)
		loggerConfig := rootConfig.Logger
		assert.NotNil(t, loggerConfig)
	})

	t.Run("use config", func(t *testing.T) {
		err := Load(WithPath("./testdata/config/logger/log.yaml"))
		assert.Nil(t, err)
		loggerConfig := rootConfig.Logger
		assert.NotNil(t, loggerConfig)
		// default
		logger.Info("hello")
	})

	t.Run("use config with file", func(t *testing.T) {
		err := Load(WithPath("./testdata/config/logger/file_log.yaml"))
		assert.Nil(t, err)
		loggerConfig := rootConfig.Logger
		assert.NotNil(t, loggerConfig)
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
		SetDriver("zap").
		SetLevel("info").
		SetFileName("dubbo.log").
		SetFileMaxAge(10).
		Build()

	assert.NotNil(t, config)

	assert.Equal(t, config.File.Name, "dubbo.log")
	assert.Equal(t, config.Driver, "zap")
	assert.Equal(t, config.Level, "info")
	assert.Equal(t, config.File.MaxAge, 10)

	// default value
	assert.Equal(t, config.Appender, "console")
	assert.Equal(t, config.Format, "text")
	assert.Equal(t, config.File.MaxSize, 100)
	assert.Equal(t, *config.File.Compress, true)
	assert.Equal(t, config.File.MaxBackups, 5)
}

func TestLoggerDynamicUpdateLevel(t *testing.T) {
	// load initial config from bytes
	initialYAML := `
  dubbo:
    logger:
      driver: zap
      level: info
  `
	err := Load(WithBytes([]byte(initialYAML)))
	assert.Nil(t, err)
	assert.NotNil(t, rootConfig)
	assert.NotNil(t, rootConfig.Logger)
	assert.Equal(t, "info", rootConfig.Logger.Level)

	// if runtime logger doesn't support dynamic level, skip this test
	if !dlogger.SetLoggerLevel(rootConfig.Logger.Level) {
		t.Skip("logger doesn't support dynamic level change in current environment")
	}

	// simulate config center change: update level -> debug
	updatedYAML := `
  dubbo:
    logger:
      level: debug
  `
	evt := &cfgcenter.ConfigChangeEvent{Key: "test", Value: updatedYAML}
	rootConfig.Process(evt)

	// expect level updated in place
	assert.Equal(t, "debug", rootConfig.Logger.Level)
}

func TestLoggerDynamicUpdateInvalidLevel(t *testing.T) {
	// load initial config from bytes
	initialYAML := `
  dubbo:
    logger:
      driver: zap
      level: info
  `
	err := Load(WithBytes([]byte(initialYAML)))
	assert.Nil(t, err)
	assert.NotNil(t, rootConfig)
	assert.NotNil(t, rootConfig.Logger)
	assert.Equal(t, "info", rootConfig.Logger.Level)

	// simulate config center change with invalid level
	updatedYAML := `
  dubbo:
    logger:
      level: invalid-level
  `
	evt := &cfgcenter.ConfigChangeEvent{Key: "test", Value: updatedYAML}
	rootConfig.Process(evt)

	// expect level unchanged when runtime doesn't support the invalid level
	assert.Equal(t, "info", rootConfig.Logger.Level)
}
