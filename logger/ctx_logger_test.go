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
	"fmt"
	"testing"
)

import (
	gostLogger "github.com/dubbogo/gost/log/logger"

	"github.com/stretchr/testify/assert"
)

// plainRecorder is a custom logger without context support.
type plainRecorder struct {
	gostLogger.Logger
	lines []string
}

func (r *plainRecorder) Warnf(format string, args ...any) {
	r.lines = append(r.lines, "WARN "+fmt.Sprintf(format, args...))
}

func (r *plainRecorder) Error(args ...any) {
	r.lines = append(r.lines, "ERROR "+fmt.Sprint(args...))
}

func TestCtxLoggingWithCustomLoggerWithoutContextSupport(t *testing.T) {
	prev := gostLogger.GetLogger()
	rec := &plainRecorder{Logger: prev}
	gostLogger.SetLogger(rec)
	t.Cleanup(func() { gostLogger.SetLogger(prev) })

	_, ok := GetCtxLogger().(plainCtxLogger)
	assert.True(t, ok, "a logger without Ctx methods is adapted")

	ctx := context.Background()
	CtxWarnf(ctx, "rejected %s", "req-1")
	GetCtxLogger().CtxError(ctx, "failed")

	assert.Equal(t, []string{"WARN rejected req-1", "ERROR failed"}, rec.lines)
}
