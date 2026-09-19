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

package logger_test

import (
	"bytes"
	"testing"
)

import (
	dubbogoLogger "github.com/dubbogo/gost/log/logger"

	"github.com/sirupsen/logrus"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

import (
	"dubbo.apache.org/dubbo-go/v3/logger"
	logruslogger "dubbo.apache.org/dubbo-go/v3/logger/core/logrus"
	zaplogger "dubbo.apache.org/dubbo-go/v3/logger/core/zap"
)

func TestFacadeSetLoggerLevel_ZapAndLogrus(t *testing.T) {
	prev := logger.GetLogger()
	t.Cleanup(func() {
		logger.SetLogger(prev)
	})

	t.Run("zap", func(t *testing.T) {
		var buf bytes.Buffer
		atomicLevel := zap.NewAtomicLevelAt(zapcore.InfoLevel)
		encoder := zapcore.NewJSONEncoder(zapcore.EncoderConfig{
			MessageKey:  "msg",
			LevelKey:    "level",
			EncodeLevel: zapcore.LowercaseLevelEncoder,
		})
		core := zapcore.NewCore(encoder, zapcore.AddSync(&buf), atomicLevel)
		sugar := zap.New(core).Sugar()
		t.Cleanup(func() { _ = sugar.Sync() })
		ctxLogger := zaplogger.NewZapCtxLogger(&dubbogoLogger.DubboLogger{
			Logger:       sugar,
			DynamicLevel: atomicLevel,
		}, false)

		logger.SetLogger(ctxLogger)
		ctxLogger.Debug("facade-debug-info")
		ctxLogger.Info("facade-info-info")
		_ = sugar.Sync()
		assert.NotContains(t, buf.String(), "facade-debug-info")
		assert.Contains(t, buf.String(), "facade-info-info")

		require.True(t, logger.SetLoggerLevel("debug"))
		buf.Reset()
		ctxLogger.Debug("facade-debug-after")
		_ = sugar.Sync()
		assert.Contains(t, buf.String(), "facade-debug-after")

		require.False(t, logger.SetLoggerLevel("not-a-level"))
		require.True(t, logger.SetLoggerLevel("warn"))
		buf.Reset()
		ctxLogger.Debug("facade-debug-warn")
		ctxLogger.Info("facade-info-warn")
		ctxLogger.Warn("facade-warn-warn")
		_ = sugar.Sync()
		out := buf.String()
		assert.NotContains(t, out, "facade-debug-warn")
		assert.NotContains(t, out, "facade-info-warn")
		assert.Contains(t, out, "facade-warn-warn")
	})

	t.Run("logrus", func(t *testing.T) {
		var buf bytes.Buffer
		lg := logrus.New()
		lg.SetOutput(&buf)
		lg.SetFormatter(&logrus.JSONFormatter{})
		lg.SetLevel(logrus.InfoLevel)
		ctxLogger := logruslogger.NewLogrusCtxLogger(&dubbogoLogger.DubboLogger{Logger: lg}, false)

		logger.SetLogger(ctxLogger)
		ctxLogger.Debug("facade-debug-info")
		ctxLogger.Info("facade-info-info")
		assert.NotContains(t, buf.String(), "facade-debug-info")
		assert.Contains(t, buf.String(), "facade-info-info")

		require.True(t, logger.SetLoggerLevel("debug"))
		buf.Reset()
		ctxLogger.Debug("facade-debug-after")
		assert.Contains(t, buf.String(), "facade-debug-after")

		require.False(t, logger.SetLoggerLevel("not-a-level"))
		assert.Equal(t, logrus.DebugLevel, lg.GetLevel())
		require.True(t, logger.SetLoggerLevel("warn"))
		buf.Reset()
		ctxLogger.Debug("facade-debug-warn")
		ctxLogger.Info("facade-info-warn")
		ctxLogger.Warn("facade-warn-warn")
		out := buf.String()
		assert.NotContains(t, out, "facade-debug-warn")
		assert.NotContains(t, out, "facade-info-warn")
		assert.Contains(t, out, "facade-warn-warn")
	})
}
