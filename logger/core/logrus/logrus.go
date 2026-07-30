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

package logrus

import (
	"io"
	"os"
	"strings"
)

import (
	dubbogoLogger "github.com/dubbogo/gost/log/logger"

	"github.com/mattn/go-colorable"

	"github.com/sirupsen/logrus"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/logger"
	"dubbo.apache.org/dubbo-go/v3/logger/core"
)

func init() {
	extension.SetLogger("logrus", instantiate)
}

func instantiate(config *common.URL) (log logger.Logger, err error) {
	var (
		level     string
		writer    []io.Writer
		lv        logrus.Level
		appender  []string
		formatter logrus.Formatter
		lg        *logrus.Logger
	)
	lg = logrus.New()
	level = config.GetParam(constant.LoggerLevelKey, constant.LoggerLevel)

	if lv, err = logrus.ParseLevel(level); err != nil {
		lg.SetLevel(logrus.InfoLevel)
	} else {
		lg.SetLevel(lv)
	}

	appender = strings.Split(config.GetParam(constant.LoggerAppenderKey, constant.LoggerAppender), ",")
	for _, apt := range appender {
		switch apt {
		case "console":
			writer = append(writer, os.Stdout)
		case "file":
			file := core.FileConfig(config)
			writer = append(writer, colorable.NewNonColorable(file))
		}
	}
	lg.SetOutput(io.MultiWriter(writer...))

	format := config.GetParam(constant.LoggerFormatKey, constant.LoggerFormat)
	switch strings.ToLower(format) {
	case "text":
		formatter = &logrus.TextFormatter{}
	case "json":
		formatter = &logrus.JSONFormatter{}
	default:
		formatter = &logrus.TextFormatter{}
	}
	lg.SetFormatter(formatter)

	baseLogger := &dubbogoLogger.DubboLogger{Logger: lg}
	traceEnabled := config.GetParamBool(constant.LoggerTraceEnabledKey, false)
	if traceEnabled {
		recordErrorToSpan := config.GetParamBool(constant.LoggerTraceRecordErrorKey, true)
		return NewLogrusCtxLogger(baseLogger, recordErrorToSpan), err
	}

	return baseLogger, err
}
