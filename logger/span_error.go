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

type spanErrorKey struct{}

// WithSpanError returns a copy of ctx with err attached. When record-error-to-span
// is on, CtxError and CtxErrorf record err on the span in place of the log
// message, and the log line keeps the full message. Use it for messages that hold
// data that must stay out of the trace backend, such as a URL with credentials or
// the request arguments. Pass the returned context to that log call only.
func WithSpanError(ctx context.Context, err error) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, spanErrorKey{}, err)
}

// SpanErrorFromContext returns the error attached by WithSpanError, or nil.
func SpanErrorFromContext(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	err, _ := ctx.Value(spanErrorKey{}).(error)
	return err
}
