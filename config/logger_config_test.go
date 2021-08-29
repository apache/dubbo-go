package config

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/logger"
)

func TestLoggerInit(t *testing.T) {

	t.Run("empty use default", func(t *testing.T) {
		err := Load(WithPath("./testdata/config/logger/empty_log.yaml"))
		assert.Nil(t, err)
		assert.NotNil(t, rootConfig)
		loggerConfig := rootConfig.Logger
		assert.NotNil(t, loggerConfig)
		logger.Info("hello")
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
