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

package cluster

import (
	"context"
	"reflect"
)

// Diagnostics observes serial cluster routing without changing invocation data.
// A consumer interceptor installs an observer only when diagnostics are enabled.
type Diagnostics interface {
	NoProvider()
	Select(attempt int, lb, peer string)
	Retry(attempt int, err error)
}

type diagnosticsKey struct{}

type diagnosticsBinding struct {
	invocation any
	observer   Diagnostics
}

// WithDiagnostics binds an observer to a non-nil invocation pointer. Passing a
// nil observer masks inherited diagnostics; an empty context is left unchanged.
func WithDiagnostics(ctx context.Context, invocation any, observer Diagnostics) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if observer == nil || !validInvocationIdentity(invocation) {
		if binding, _ := ctx.Value(diagnosticsKey{}).(*diagnosticsBinding); binding == nil {
			return ctx
		}
		return context.WithValue(ctx, diagnosticsKey{}, (*diagnosticsBinding)(nil))
	}
	return context.WithValue(ctx, diagnosticsKey{}, &diagnosticsBinding{invocation: invocation, observer: observer})
}

// DiagnosticsFromContext returns diagnostics only for the exact invocation
// pointer that owns them. Inheriting a context alone does not bind a nested RPC.
func DiagnosticsFromContext(ctx context.Context, invocation any) Diagnostics {
	if ctx == nil {
		return nil
	}
	binding, _ := ctx.Value(diagnosticsKey{}).(*diagnosticsBinding)
	if binding == nil || !validInvocationIdentity(invocation) || binding.invocation != invocation {
		return nil
	}
	return binding.observer
}

func validInvocationIdentity(invocation any) bool {
	v := reflect.ValueOf(invocation)
	return v.IsValid() && v.Kind() == reflect.Pointer && !v.IsNil()
}
