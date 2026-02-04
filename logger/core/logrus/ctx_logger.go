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
	"context"
	"errors"
	"fmt"
)

import (
	dubbogoLogger "github.com/dubbogo/gost/log/logger"

	"github.com/sirupsen/logrus"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

import (
	"dubbo.apache.org/dubbo-go/v3/logger"
)

// LogrusCtxLogger wraps DubboLogger with context-aware logging support.
type LogrusCtxLogger struct {
	*dubbogoLogger.DubboLogger
	recordErrorToSpan bool
}

var _ logger.CtxLogger = (*LogrusCtxLogger)(nil)

// NewLogrusCtxLogger creates a new LogrusCtxLogger.
func NewLogrusCtxLogger(base *dubbogoLogger.DubboLogger, recordErrorToSpan bool) *LogrusCtxLogger {
	return &LogrusCtxLogger{
		DubboLogger:       base,
		recordErrorToSpan: recordErrorToSpan,
	}
}

// withTraceFields injects trace information from context into logger.
func (l *LogrusCtxLogger) withTraceFields(ctx context.Context) *logrus.Entry {
	logrusLogger, ok := l.Logger.(*logrus.Logger)
	if !ok {
		return nil
	}

	fields := logger.ExtractTraceFields(ctx)
	if fields.TraceID == "" {
		return logrus.NewEntry(logrusLogger) // No trace information
	}

	return logrusLogger.WithFields(logrus.Fields{
		"trace_id":    fields.TraceID,
		"span_id":     fields.SpanID,
		"trace_flags": fields.TraceFlags,
	})
}

func (l *LogrusCtxLogger) CtxDebugf(ctx context.Context, template string, args ...any) {
	if entry := l.withTraceFields(ctx); entry != nil {
		entry.Debugf(template, args...)
	} else {
		l.Debugf(template, args...)
	}
}

func (l *LogrusCtxLogger) CtxDebug(ctx context.Context, args ...any) {
	if entry := l.withTraceFields(ctx); entry != nil {
		entry.Debug(args...)
	} else {
		l.Debug(args...)
	}
}

func (l *LogrusCtxLogger) CtxInfof(ctx context.Context, template string, args ...any) {
	if entry := l.withTraceFields(ctx); entry != nil {
		entry.Infof(template, args...)
	} else {
		l.Infof(template, args...)
	}
}

func (l *LogrusCtxLogger) CtxInfo(ctx context.Context, args ...any) {
	if entry := l.withTraceFields(ctx); entry != nil {
		entry.Info(args...)
	} else {
		l.Info(args...)
	}
}

func (l *LogrusCtxLogger) CtxWarnf(ctx context.Context, template string, args ...any) {
	if entry := l.withTraceFields(ctx); entry != nil {
		entry.Warnf(template, args...)
	} else {
		l.Warnf(template, args...)
	}
}

func (l *LogrusCtxLogger) CtxWarn(ctx context.Context, args ...any) {
	if entry := l.withTraceFields(ctx); entry != nil {
		entry.Warn(args...)
	} else {
		l.Warn(args...)
	}
}

// CtxErrorf logs error and optionally records to span.
func (l *LogrusCtxLogger) CtxErrorf(ctx context.Context, template string, args ...any) {
	if entry := l.withTraceFields(ctx); entry != nil {
		entry.Errorf(template, args...)
	} else {
		l.Errorf(template, args...)
	}

	if l.recordErrorToSpan {
		l.recordErrorToSpanIfPresent(ctx, template, args...)
	}
}

// CtxError logs error and optionally records to span.
func (l *LogrusCtxLogger) CtxError(ctx context.Context, args ...any) {
	if entry := l.withTraceFields(ctx); entry != nil {
		entry.Error(args...)
	} else {
		l.Error(args...)
	}

	if l.recordErrorToSpan {
		l.recordErrorToSpanIfPresent(ctx, "%s", fmt.Sprint(args...))
	}
}

func (l *LogrusCtxLogger) recordErrorToSpanIfPresent(ctx context.Context, template string, args ...any) {
	if ctx == nil {
		return
	}

	span := trace.SpanFromContext(ctx)
	if span == nil || !span.IsRecording() {
		return
	}

	msg := fmt.Sprintf(template, args...)
	span.SetStatus(codes.Error, msg)
	span.RecordError(errors.New(msg))
}
