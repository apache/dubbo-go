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
