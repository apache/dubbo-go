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
	"io"
	"os"
)

import (
	"github.com/apache/dubbo-getty"
	"github.com/sirupsen/logrus"
)

var baseLogrusLogger *logrus.Logger

func InitLogrusLog(logConfFile string) error {
	baseLogrusLogger = &logrus.Logger{
		Out: os.Stderr,
		Formatter: &logrus.TextFormatter{
			DisableColors: false,
			FullTimestamp: true,
		},
		Hooks:        make(logrus.LevelHooks),
		Level:        logrus.DebugLevel,
		ExitFunc:     os.Exit,
		ReportCaller: false,
	}
	logger = &DubboLogger{Logger: baseLogrusLogger, dynamicLevel: logrus.DebugLevel.String(), loggerName: "logrus"}
	// set getty log
	getty.SetLogger(logger)
	return nil
}

func SetLogrusFormatter(formatter logrus.Formatter) {
	baseLogrusLogger.SetFormatter(formatter)
}

func SetLogrusLevel(level string) {
	l := new(logrus.Level)
	if err := l.UnmarshalText([]byte(level)); err == nil {
		baseLogrusLogger.SetLevel(*l)
	}
}

func SetLogrusOutput(opt io.Writer) {
	baseLogrusLogger.SetOutput(opt)
}

func SetLogrusReportCaller(reportCaller bool) {
	baseLogrusLogger.SetReportCaller(reportCaller)
}

func SetLogrusNoLock() {
	baseLogrusLogger.SetNoLock()
}
