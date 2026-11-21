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
	"path/filepath"
	"runtime"
	"testing"
)

import (
	dubbogoLogger "github.com/dubbogo/gost/log/logger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

import (
	"dubbo.apache.org/dubbo-go/v3/logger"
)

// installObservedZapCtxLogger installs a ZapCtxLogger built like instantiate
// does (AddCaller, AddCallerSkip(1)) as the global logger.
func installObservedZapCtxLogger(t *testing.T, traceEnabled bool) *observer.ObservedLogs {
	t.Helper()
	core, logs := observer.New(zapcore.DebugLevel)
	sugar := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1)).Sugar()
	base := &dubbogoLogger.DubboLogger{Logger: sugar, DynamicLevel: zap.NewAtomicLevelAt(zapcore.DebugLevel)}
	prev := dubbogoLogger.GetLogger()
	dubbogoLogger.SetLogger(newZapCtxLogger(base, traceEnabled, false))
	t.Cleanup(func() { dubbogoLogger.SetLogger(prev) })
	return logs
}

// nextLine returns the line number of the statement after the call.
func nextLine() int {
	_, _, l, _ := runtime.Caller(1)
	return l + 1
}

// TestCtxLoggingReportsCallSite checks the caller reported for each way the
// framework and user code log, with trace integration on and off.
func TestCtxLoggingReportsCallSite(t *testing.T) {
	traceCtx, wantTraceID, _, _ := mustZapTraceContext(t)

	calls := []struct {
		name string
		log  func(ctx context.Context) int
	}{
		{"package Ctx function", func(ctx context.Context) int {
			l := nextLine()
			logger.CtxWarnf(ctx, "msg")
			return l
		}},
		{"CtxLogger method", func(ctx context.Context) int {
			l := nextLine()
			logger.GetCtxLogger().CtxWarnf(ctx, "msg")
			return l
		}},
		{"plain gost function", func(context.Context) int {
			l := nextLine()
			dubbogoLogger.Warnf("msg")
			return l
		}},
	}
	for _, traceEnabled := range []bool{true, false} {
		for _, c := range calls {
			t.Run(fmt.Sprintf("%s/trace=%t", c.name, traceEnabled), func(t *testing.T) {
				logs := installObservedZapCtxLogger(t, traceEnabled)
				wantLine := c.log(traceCtx)

				entries := logs.All()
				require.Len(t, entries, 1)
				caller := entries[0].Caller
				require.True(t, caller.Defined)
				assert.Equal(t, "ctx_caller_test.go", filepath.Base(caller.File))
				assert.Equal(t, wantLine, caller.Line)

				fields := entries[0].ContextMap()
				if traceEnabled && c.name != "plain gost function" {
					assert.Equal(t, wantTraceID, fields["trace_id"])
				} else {
					assert.NotContains(t, fields, "trace_id")
				}
			})
		}
	}
}
