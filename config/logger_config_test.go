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
		err := Load(WithPath("./testdata/config/logger/empty_log.yml"))
		assert.Nil(t, err)
		assert.NotNil(t, rootConfig)
		loggerConfig := rootConfig.Logger
		assert.NotNil(t, loggerConfig)
		logger.Info("hello")
	})

	t.Run("use config", func(t *testing.T) {
		err := Load(WithPath("./testdata/config/logger/log.yml"))
		assert.Nil(t, err)
		loggerConfig := rootConfig.Logger
		assert.NotNil(t, loggerConfig)
		// default
		assert.Equal(t, "debug", loggerConfig.Level)
		assert.Equal(t, "message", loggerConfig.EncoderConfig.MessageKey)
		assert.Equal(t, "stacktrace", loggerConfig.EncoderConfig.StacktraceKey)
		logger.Info("hello")
	})
}
