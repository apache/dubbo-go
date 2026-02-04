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

package zap

import (
	"context"
	"fmt"
)

import (
	dubbogoLogger "github.com/dubbogo/gost/log/logger"

	"go.opentelemetry.io/otel/codes"

	"go.opentelemetry.io/otel/trace"

	"go.uber.org/zap"
)

import (
	"dubbo.apache.org/dubbo-go/v3/logger"
)

// ZapCtxLogger wraps DubboLogger with context-aware logging support.
type ZapCtxLogger struct {
	*dubbogoLogger.DubboLogger
	recordErrorToSpan bool
}

var _ logger.CtxLogger = (*ZapCtxLogger)(nil)

func NewZapCtxLogger(base *dubbogoLogger.DubboLogger, recordErrorToSpan bool) *ZapCtxLogger {
	return &ZapCtxLogger{
		DubboLogger:       base,
		recordErrorToSpan: recordErrorToSpan,
	}
}

// withTraceFields injects trace information from context into logger.
func (l *ZapCtxLogger) withTraceFields(ctx context.Context) *zap.SugaredLogger {
	zapLogger, ok := l.Logger.(*zap.SugaredLogger)
	if !ok {
		return nil
	}

	fields := logger.ExtractTraceFields(ctx)
	if fields.TraceID == "" {
		return zapLogger
	}

	return zapLogger.With(
		"trace_id", fields.TraceID,
		"span_id", fields.SpanID,
		"trace_flags", fields.TraceFlags,
	)
}

func (l *ZapCtxLogger) CtxDebugf(ctx context.Context, template string, args ...any) {
	if ctxLogger := l.withTraceFields(ctx); ctxLogger != nil {
		ctxLogger.Debugf(template, args...)
	} else {
		l.Debugf(template, args...)
	}
}

func (l *ZapCtxLogger) CtxDebug(ctx context.Context, args ...any) {
	if ctxLogger := l.withTraceFields(ctx); ctxLogger != nil {
		ctxLogger.Debug(args...)
	} else {
		l.Debug(args...)
	}
}

func (l *ZapCtxLogger) CtxInfof(ctx context.Context, template string, args ...any) {
	if ctxLogger := l.withTraceFields(ctx); ctxLogger != nil {
		ctxLogger.Infof(template, args...)
	} else {
		l.Infof(template, args...)
	}
}

func (l *ZapCtxLogger) CtxInfo(ctx context.Context, args ...any) {
	if ctxLogger := l.withTraceFields(ctx); ctxLogger != nil {
		ctxLogger.Info(args...)
	} else {
		l.Info(args...)
	}
}

func (l *ZapCtxLogger) CtxWarnf(ctx context.Context, template string, args ...any) {
	if ctxLogger := l.withTraceFields(ctx); ctxLogger != nil {
		ctxLogger.Warnf(template, args...)
	} else {
		l.Warnf(template, args...)
	}
}

func (l *ZapCtxLogger) CtxWarn(ctx context.Context, args ...any) {
	if ctxLogger := l.withTraceFields(ctx); ctxLogger != nil {
		ctxLogger.Warn(args...)
	} else {
		l.Warn(args...)
	}
}

// CtxErrorf logs error and optionally records to span.
func (l *ZapCtxLogger) CtxErrorf(ctx context.Context, template string, args ...any) {
	if ctxLogger := l.withTraceFields(ctx); ctxLogger != nil {
		ctxLogger.Errorf(template, args...)
	} else {
		l.Errorf(template, args...)
	}

	if l.recordErrorToSpan {
		l.recordErrorToSpanIfPresent(ctx, template, args...)
	}
}

// CtxError logs error and optionally records to span.
func (l *ZapCtxLogger) CtxError(ctx context.Context, args ...any) {
	if ctxLogger := l.withTraceFields(ctx); ctxLogger != nil {
		ctxLogger.Error(args...)
	} else {
		l.Error(args...)
	}

	if l.recordErrorToSpan {
		l.recordErrorToSpanIfPresent(ctx, "%s", fmt.Sprint(args...))
	}
}

func (l *ZapCtxLogger) recordErrorToSpanIfPresent(ctx context.Context, template string, args ...any) {
	if ctx == nil {
		return
	}

	span := trace.SpanFromContext(ctx)
	if span == nil || !span.IsRecording() {
		return
	}

	msg := fmt.Sprintf(template, args...)
	span.SetStatus(codes.Error, msg)
	span.RecordError(fmt.Errorf("%s", msg))
}
