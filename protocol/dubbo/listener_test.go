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

package dubbo

import (
	"testing"
)

import (
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/protocol/invocation"
)

// test rebuild the ctx
func TestRebuildCtx(t *testing.T) {
	opentracing.SetGlobalTracer(mocktracer.New())
	attach := make(map[string]interface{}, 10)
	attach[constant.VERSION_KEY] = "1.0"
	attach[constant.GROUP_KEY] = "MyGroup"
	inv := invocation.NewRPCInvocation("MethodName", []interface{}{"OK", "Hello"}, attach)

	// attachment doesn't contains any tracing key-value pair,
	ctx := rebuildCtx(inv)
	assert.NotNil(t, ctx)
	assert.Nil(t, ctx.Value(constant.TRACING_REMOTE_SPAN_CTX))

	span, ctx := opentracing.StartSpanFromContext(ctx, "Test-Client")

	injectTraceCtx(span, inv)
	// rebuild the context success
	inv = invocation.NewRPCInvocation("MethodName", []interface{}{"OK", "Hello"}, attach)
	ctx = rebuildCtx(inv)
	span.Finish()
	assert.NotNil(t, ctx)
	assert.NotNil(t, ctx.Value(constant.TRACING_REMOTE_SPAN_CTX))
}
