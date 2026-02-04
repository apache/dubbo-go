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

	"github.com/natefinch/lumberjack"

	"go.uber.org/zap"
)

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

// CtxLogger extends Logger interface with context-aware logging methods.
type CtxLogger interface {
	Logger
	CtxDebug(ctx context.Context, args ...any)
	CtxDebugf(ctx context.Context, template string, args ...any)
	CtxInfo(ctx context.Context, args ...any)
	CtxInfof(ctx context.Context, template string, args ...any)
	CtxWarn(ctx context.Context, args ...any)
	CtxWarnf(ctx context.Context, template string, args ...any)
	CtxError(ctx context.Context, args ...any)
	CtxErrorf(ctx context.Context, template string, args ...any)
}
