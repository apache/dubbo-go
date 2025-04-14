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
	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
)

// nolint
type DubboLogger struct {
	Logger
	DynamicLevel zap.AtomicLevel
}

type Config struct {
	LumberjackConfig *lumberjack.Logger `yaml:"lumberjack-config"`
	ZapConfig        *zap.Config        `yaml:"zap-config"`
	CallerSkip       int
}

type Logger interface {
	Debug(args ...any)
	Debugf(template string, args ...any)
	Info(args ...any)
	Infof(template string, args ...any)
	Warn(args ...any)
	Warnf(template string, args ...any)
	Error(args ...any)
	Errorf(template string, args ...any)
	Fatal(args ...any)
	Fatalf(fmt string, args ...any)
}

func (l *DubboLogger) Debug(args ...any) {
	l.Logger.Debug(args...)
}

func (l *DubboLogger) Debugf(template string, args ...any) {
	l.Logger.Debugf(template, args...)
}

func (l *DubboLogger) Info(args ...any) {
	l.Logger.Info(args...)
}

func (l *DubboLogger) Infof(template string, args ...any) {
	l.Logger.Infof(template, args...)
}

func (l *DubboLogger) Warn(args ...any) {
	l.Logger.Warn(args...)
}

func (l *DubboLogger) Warnf(template string, args ...any) {
	l.Logger.Warnf(template, args...)
}

func (l *DubboLogger) Error(args ...any) {
	l.Logger.Error(args...)
}

func (l *DubboLogger) Errorf(template string, args ...any) {
	l.Logger.Errorf(template, args...)
}

func (l *DubboLogger) Fatal(args ...any) {
	l.Logger.Fatal(args...)
}

func (l *DubboLogger) Fatalf(fmt string, args ...any) {
	l.Logger.Fatalf(fmt, args...)
}

// Exported functions

func Info(args ...any) {
	logger.Info(args...)
}

func Warn(args ...any) {
	logger.Warn(args...)
}

func Error(args ...any) {
	logger.Error(args...)
}

func Debug(args ...any) {
	logger.Debug(args...)
}

func Infof(fmt string, args ...any) {
	logger.Infof(fmt, args...)
}

func Warnf(fmt string, args ...any) {
	logger.Warnf(fmt, args...)
}

func Errorf(fmt string, args ...any) {
	logger.Errorf(fmt, args...)
}

func Debugf(fmt string, args ...any) {
	logger.Debugf(fmt, args...)
}

func Fatal(args ...any) {
	logger.Fatal(args...)
}

func Fatalf(fmt string, args ...any) {
	logger.Fatalf(fmt, args...)
}
