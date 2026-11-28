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

package base

import (
	"context"
	"reflect"
)

// A cluster invoker marks an attempt that retries a failed attempt of an
// invocation in the context of that attempt, so that the metrics filter can
// count it. The mark names the invocation: another RPC started with an
// inherited context has its own invocation and does not match it.
type retryAttemptKey struct{}

// WithRetryAttempt marks the attempt of invocation run with the returned
// context as a retry.
func WithRetryAttempt(ctx context.Context, invocation Invocation) context.Context {
	return context.WithValue(ctx, retryAttemptKey{}, invocation)
}

// IsRetryAttempt reports whether ctx marks this attempt of invocation as a
// retry. Identity is compared only for pointers.
func IsRetryAttempt(ctx context.Context, invocation Invocation) bool {
	marked, ok := ctx.Value(retryAttemptKey{}).(Invocation)
	return ok && reflect.ValueOf(marked).Kind() == reflect.Pointer && marked == invocation
}

// TransferRetryAttempt moves the mark of from to to, for a filter that
// forwards a translated invocation in place of from.
func TransferRetryAttempt(ctx context.Context, from, to Invocation) context.Context {
	if IsRetryAttempt(ctx, from) {
		return WithRetryAttempt(ctx, to)
	}
	return ctx
}
