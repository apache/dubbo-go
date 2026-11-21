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
	"context"
)

import (
	gostLogger "github.com/dubbogo/gost/log/logger"
)

// GetCtxLogger returns the global logger, the one behind gost's logger
// package functions, as a CtxLogger. The built-in zap and logrus loggers
// implement CtxLogger and add trace fields when trace integration is enabled.
// A custom logger that does not implement CtxLogger is adapted: its Ctx
// methods drop the context and call the plain method.
//
// Call the Ctx method on the returned value at the log site, for example
// logger.GetCtxLogger().CtxWarnf(ctx, ...), so that caller reporting still
// points at the log site.
func GetCtxLogger() CtxLogger {
	l := gostLogger.GetLogger()
	if ctxLogger, ok := l.(CtxLogger); ok {
		return ctxLogger
	}
	return plainCtxLogger{l}
}

// callerSkipper is implemented by loggers that report the caller. The
// package-level Ctx functions log through CallerSkipped so that the reported
// caller is their call site, not the function itself.
type callerSkipper interface {
	CallerSkipped() CtxLogger
}

func pkgCtxLogger() CtxLogger {
	l := GetCtxLogger()
	if s, ok := l.(callerSkipper); ok {
		return s.CallerSkipped()
	}
	return l
}

// CtxDebug logs at debug level through the global logger, with trace fields from ctx.
func CtxDebug(ctx context.Context, args ...any) { pkgCtxLogger().CtxDebug(ctx, args...) }

// CtxDebugf logs at debug level through the global logger, with trace fields from ctx.
func CtxDebugf(ctx context.Context, template string, args ...any) {
	pkgCtxLogger().CtxDebugf(ctx, template, args...)
}

// CtxInfo logs at info level through the global logger, with trace fields from ctx.
func CtxInfo(ctx context.Context, args ...any) { pkgCtxLogger().CtxInfo(ctx, args...) }

// CtxInfof logs at info level through the global logger, with trace fields from ctx.
func CtxInfof(ctx context.Context, template string, args ...any) {
	pkgCtxLogger().CtxInfof(ctx, template, args...)
}

// CtxWarn logs at warn level through the global logger, with trace fields from ctx.
func CtxWarn(ctx context.Context, args ...any) { pkgCtxLogger().CtxWarn(ctx, args...) }

// CtxWarnf logs at warn level through the global logger, with trace fields from ctx.
func CtxWarnf(ctx context.Context, template string, args ...any) {
	pkgCtxLogger().CtxWarnf(ctx, template, args...)
}

// CtxError logs at error level through the global logger, with trace fields from ctx.
func CtxError(ctx context.Context, args ...any) { pkgCtxLogger().CtxError(ctx, args...) }

// CtxErrorf logs at error level through the global logger, with trace fields from ctx.
func CtxErrorf(ctx context.Context, template string, args ...any) {
	pkgCtxLogger().CtxErrorf(ctx, template, args...)
}

// plainCtxLogger adapts a Logger without context support to CtxLogger.
type plainCtxLogger struct {
	gostLogger.Logger
}

func (l plainCtxLogger) CtxDebug(_ context.Context, args ...any) { l.Debug(args...) }
func (l plainCtxLogger) CtxDebugf(_ context.Context, template string, args ...any) {
	l.Debugf(template, args...)
}
func (l plainCtxLogger) CtxInfo(_ context.Context, args ...any) { l.Info(args...) }
func (l plainCtxLogger) CtxInfof(_ context.Context, template string, args ...any) {
	l.Infof(template, args...)
}
func (l plainCtxLogger) CtxWarn(_ context.Context, args ...any) { l.Warn(args...) }
func (l plainCtxLogger) CtxWarnf(_ context.Context, template string, args ...any) {
	l.Warnf(template, args...)
}
func (l plainCtxLogger) CtxError(_ context.Context, args ...any) { l.Error(args...) }
func (l plainCtxLogger) CtxErrorf(_ context.Context, template string, args ...any) {
	l.Errorf(template, args...)
}
