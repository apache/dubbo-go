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

package logger

import (
	"github.com/apache/dubbo-getty"

	"github.com/natefinch/lumberjack"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger Logger

// init a default logger
func init() {
	InitLogger(nil)
}

// nolint
type DubboLogger struct {
	Logger
	dynamicLevel zap.AtomicLevel
}

type Config struct {
	LumberjackConfig *lumberjack.Logger `yaml:"lumberjack-config"`
	ZapConfig        *zap.Config        `yaml:"zap-config"`
}

// Logger is the interface for Logger types
type Logger interface {
	Info(args ...interface{})
	Warn(args ...interface{})
	Error(args ...interface{})
	Debug(args ...interface{})
	Fatal(args ...interface{})

	Infof(fmt string, args ...interface{})
	Warnf(fmt string, args ...interface{})
	Errorf(fmt string, args ...interface{})
	Debugf(fmt string, args ...interface{})
	Fatalf(fmt string, args ...interface{})
}

// InitLogger use for init logger by @conf
func InitLogger(conf *Config) {
	var (
		zapLogger *zap.Logger
		config    = &Config{}
	)
	if conf == nil || conf.ZapConfig == nil {
		zapLoggerEncoderConfig := zapcore.EncoderConfig{
			TimeKey:        "time",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "message",
			StacktraceKey:  "stacktrace",
			EncodeLevel:    zapcore.CapitalColorLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		}
		config.ZapConfig = &zap.Config{
			Level:            zap.NewAtomicLevelAt(zap.DebugLevel),
			Development:      false,
			Encoding:         "console",
			EncoderConfig:    zapLoggerEncoderConfig,
			OutputPaths:      []string{"stderr"},
			ErrorOutputPaths: []string{"stderr"},
		}
	} else {
		config.ZapConfig = conf.ZapConfig
	}

	if conf == nil || conf.LumberjackConfig == nil {
		zapLogger, _ = config.ZapConfig.Build(zap.AddCallerSkip(1))
	} else {
		config.LumberjackConfig = conf.LumberjackConfig
		zapLogger = initZapLoggerWithSyncer(config)
	}

	logger = &DubboLogger{Logger: zapLogger.Sugar(), dynamicLevel: config.ZapConfig.Level}

	// set getty log
	getty.SetLogger(logger)
}

// SetLogger sets logger for dubbo and getty
func SetLogger(log Logger) {
	logger = log
	getty.SetLogger(logger)
}

// GetLogger gets the logger
func GetLogger() Logger {
	return logger
}

// SetLoggerLevel use for set logger level
func SetLoggerLevel(level string) bool {
	if l, ok := logger.(OpsLogger); ok {
		l.SetLoggerLevel(level)
		return true
	}
	return false
}

// OpsLogger use for the SetLoggerLevel
type OpsLogger interface {
	Logger
	SetLoggerLevel(level string)
}

// SetLoggerLevel use for set logger level
func (dl *DubboLogger) SetLoggerLevel(level string) {
	l := new(zapcore.Level)
	if err := l.Set(level); err == nil {
		dl.dynamicLevel.SetLevel(*l)
	}
}

// initZapLoggerWithSyncer init zap Logger with syncer
func initZapLoggerWithSyncer(conf *Config) *zap.Logger {
	core := zapcore.NewCore(
		conf.getEncoder(),
		conf.getLogWriter(),
		zap.NewAtomicLevelAt(zap.DebugLevel),
	)

	return zap.New(core, zap.AddCallerSkip(1))
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
