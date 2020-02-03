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

// Info ...
func Info(args ...interface{}) {
	logger.Info(args...)
}

// Warn ...
func Warn(args ...interface{}) {
	logger.Warn(args...)
}

// Error ...
func Error(args ...interface{}) {
	logger.Error(args...)
}

// Debug ...
func Debug(args ...interface{}) {
	logger.Debug(args...)
}

// Infof ...
func Infof(fmt string, args ...interface{}) {
	logger.Infof(fmt, args...)
}

// Warnf ...
func Warnf(fmt string, args ...interface{}) {
	logger.Warnf(fmt, args...)
}

// Errorf ...
func Errorf(fmt string, args ...interface{}) {
	logger.Errorf(fmt, args...)
}

// Debugf ...
func Debugf(fmt string, args ...interface{}) {
	logger.Debugf(fmt, args...)
}
