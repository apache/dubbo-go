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

// Package logger is unified facade provided by Dubbo to work with different logger frameworks, eg, Zapper, Logrus.
package logger

import (
	dubbogoLogger "github.com/dubbogo/gost/log/logger"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger Logger

func init() {
	dubbogoLogger.InitLogger(nil)
}

// SetLogger sets logger for dubbo and getty
func SetLogger(log Logger) {
	logger = log
}

// GetLogger gets the loggerF
func GetLogger() Logger {
	return logger
}

// SetLoggerLevel is used to set the logger level.
func SetLoggerLevel(level string) bool {
	if l, ok := logger.(OpsLogger); ok {
		return l.SetLoggerLevel(level)
	}
	return false
}

// OpsLogger use for the SetLoggerLevel
type OpsLogger interface {
	Logger
	SetLoggerLevel(level string) bool
}

// initZapLoggerWithSyncer init zap Logger with syncer
func initZapLoggerWithSyncer(conf *Config) *zap.Logger {
	core := zapcore.NewCore(
		conf.getEncoder(),
		conf.getLogWriter(),
		zap.NewAtomicLevelAt(conf.ZapConfig.Level.Level()),
	)

	return zap.New(core, zap.AddCaller(), zap.AddCallerSkip(conf.CallerSkip))
}

// getEncoder get encoder by config, zapcore support json and console encoder
func (c *Config) getEncoder() zapcore.Encoder {
	if c.ZapConfig.Encoding == "json" {
		return zapcore.NewJSONEncoder(c.ZapConfig.EncoderConfig)
	} else if c.ZapConfig.Encoding == "console" {
		return zapcore.NewConsoleEncoder(c.ZapConfig.EncoderConfig)
	}
	return nil
}

// getLogWriter get Lumberjack writer by LumberjackConfig
func (c *Config) getLogWriter() zapcore.WriteSyncer {
	return zapcore.AddSync(c.LumberjackConfig)
}
