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
package trace

import (
	"context"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

// TestServerSpanOfGenericCallUsesActualMethod checks that a $invoke call gets
// the actual method in both the span name and rpc.method. The server filter
// runs before the generic service filter rewrites the invocation.
func TestServerSpanOfGenericCallUsesActualMethod(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })

	url := common.NewURLWithOptions(common.WithParamsValue(constant.InterfaceKey, "com.example.UserService"))
	inv := invocation.NewRPCInvocation(constant.Generic,
		[]any{"GetUser", []string{"java.lang.String"}, []any{"u-1"}}, map[string]any{})
	require.True(t, inv.IsGenericInvocation())

	(&otelServerFilter{Propagators: propagation.TraceContext{}, TracerProvider: provider}).
		Invoke(context.Background(), base.NewBaseInvoker(url), inv)

	spans := recorder.Ended()
	require.Len(t, spans, 1)
	assert.Equal(t, "GetUser", spans[0].Name())
	assert.Contains(t, spans[0].Attributes(), semconv.RPCMethod("GetUser"))
}
