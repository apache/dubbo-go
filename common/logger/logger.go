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
	"flag"
	"io/ioutil"
	"log"
	"os"
	"path"
)

import (
	"github.com/apache/dubbo-getty"
	perrors "github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v2"
)

import (
	"github.com/apache/dubbo-go/common/constant"
)

var (
	logger Logger
)

// nolint
type DubboLogger struct {
	Logger
	dynamicLevel zap.AtomicLevel
}

// Logger is the interface for Logger types
type Logger interface {
	Info(args ...interface{})
	Warn(args ...interface{})
	Error(args ...interface{})
	Debug(args ...interface{})

	Infof(fmt string, args ...interface{})
	Warnf(fmt string, args ...interface{})
	Errorf(fmt string, args ...interface{})
	Debugf(fmt string, args ...interface{})
}

func init() {
	// forbidden to executing twice.
	if logger != nil {
		return
	}

	fs := flag.NewFlagSet("log", flag.ContinueOnError)
	logConfFile := fs.String("logConf", os.Getenv(constant.APP_LOG_CONF_FILE), "default log config path")
	fs.Parse(os.Args[1:])
	for len(fs.Args()) != 0 {
		fs.Parse(fs.Args()[1:])
	}
	err := InitLog(*logConfFile)
	if err != nil {
		log.Printf("[InitLog] warn: %v", err)
	}
}

// InitLog use for init logger by call InitLogger
func InitLog(logConfFile string) error {
	if logConfFile == "" {
		InitLogger(nil)
		return perrors.New("log configure file name is nil")
	}
	if path.Ext(logConfFile) != ".yml" {
		InitLogger(nil)
		return perrors.Errorf("log configure file name{%s} suffix must be .yml", logConfFile)
	}

	confFileStream, err := ioutil.ReadFile(logConfFile)
	if err != nil {
		InitLogger(nil)
		return perrors.Errorf("ioutil.ReadFile(file:%s) = error:%v", logConfFile, err)
	}

	conf := &zap.Config{}
	err = yaml.Unmarshal(confFileStream, conf)
	if err != nil {
		InitLogger(nil)
		return perrors.Errorf("[Unmarshal]init logger error: %v", err)
	}

	InitLogger(conf)

	return nil
}

// InitLogger use for init logger by @conf
func InitLogger(conf *zap.Config) {
	var zapLoggerConfig zap.Config
	if conf == nil {
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
		zapLoggerConfig = zap.Config{
			Level:            zap.NewAtomicLevelAt(zap.DebugLevel),
			Development:      false,
			Encoding:         "console",
			EncoderConfig:    zapLoggerEncoderConfig,
			OutputPaths:      []string{"stderr"},
			ErrorOutputPaths: []string{"stderr"},
		}
	} else {
		zapLoggerConfig = *conf
	}
	zapLogger, _ := zapLoggerConfig.Build(zap.AddCallerSkip(1))
	logger = &DubboLogger{Logger: zapLogger.Sugar(), dynamicLevel: zapLoggerConfig.Level}

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
