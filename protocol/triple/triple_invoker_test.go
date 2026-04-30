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

package triple

import (
	"context"
	"net/http"
	"sync"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	tri "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
)

func Test_parseInvocation(t *testing.T) {
	tests := []struct {
		desc   string
		ctx    func() context.Context
		url    *common.URL
		invo   func() base.Invocation
		expect func(t *testing.T, callType string, inRaw []any, methodName string, err error)
	}{
		{
			desc: "miss callType",
			ctx: func() context.Context {
				return context.Background()
			},
			url: common.NewURLWithOptions(),
			invo: func() base.Invocation {
				return invocation.NewRPCInvocationWithOptions()
			},
			expect: func(t *testing.T, callType string, inRaw []any, methodName string, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "miss CallType")
			},
		},
		{
			desc: "wrong callType",
			ctx: func() context.Context {
				return context.Background()
			},
			url: common.NewURLWithOptions(),
			invo: func() base.Invocation {
				iv := invocation.NewRPCInvocationWithOptions()
				iv.SetAttribute(constant.CallTypeKey, 1)
				return iv
			},
			expect: func(t *testing.T, callType string, inRaw []any, methodName string, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "CallType should be string")
			},
		},
		{
			desc: "empty methodName",
			ctx: func() context.Context {
				return context.Background()
			},
			url: common.NewURLWithOptions(),
			invo: func() base.Invocation {
				iv := invocation.NewRPCInvocationWithOptions()
				iv.SetAttribute(constant.CallTypeKey, constant.CallUnary)
				return iv
			},
			expect: func(t *testing.T, callType string, inRaw []any, methodName string, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "miss MethodName")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			callType, inRaw, methodName, err := parseInvocation(test.ctx(), test.url, test.invo())
			test.expect(t, callType, inRaw, methodName, err)
		})
	}
}

func Test_parseAttachments(t *testing.T) {
	tests := []struct {
		desc   string
		ctx    func() context.Context
		url    *common.URL
		invo   func() base.Invocation
		expect func(t *testing.T, ctx context.Context, err error)
	}{
		{
			desc: "url has pre-defined keys in triAttachmentKeys",
			ctx: func() context.Context {
				return context.Background()
			},
			url: common.NewURLWithOptions(
				common.WithInterface("interface"),
				common.WithToken("token"),
			),
			invo: func() base.Invocation {
				return invocation.NewRPCInvocationWithOptions()
			},
			expect: func(t *testing.T, ctx context.Context, err error) {
				require.NoError(t, err)
				header := http.Header(tri.ExtractFromOutgoingContext(ctx))
				assert.NotNil(t, header)
				assert.Equal(t, "interface", header.Get(constant.InterfaceKey))
				assert.Equal(t, "token", header.Get(constant.TokenKey))
			},
		},
		{
			desc: "user passed-in legal attachments",
			ctx: func() context.Context {
				userDefined := make(map[string]any)
				userDefined["key1"] = "val1"
				userDefined["key2"] = []string{"key2_1", "key2_2"}
				return context.WithValue(context.Background(), constant.AttachmentKey, userDefined)
			},
			url: common.NewURLWithOptions(),
			invo: func() base.Invocation {
				return invocation.NewRPCInvocationWithOptions()
			},
			expect: func(t *testing.T, ctx context.Context, err error) {
				require.NoError(t, err)
				header := http.Header(tri.ExtractFromOutgoingContext(ctx))
				assert.NotNil(t, header)
				assert.Equal(t, "val1", header.Get("key1"))
				assert.Equal(t, []string{"key2_1", "key2_2"}, header.Values("key2"))
			},
		},
		{
			desc: "user passed-in illegal attachments",
			ctx: func() context.Context {
				userDefined := make(map[string]any)
				userDefined["key1"] = 1
				return context.WithValue(context.Background(), constant.AttachmentKey, userDefined)
			},
			url: common.NewURLWithOptions(),
			invo: func() base.Invocation {
				return invocation.NewRPCInvocationWithOptions()
			},
			expect: func(t *testing.T, ctx context.Context, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			ctx := test.ctx()
			inv := test.invo()
			parseAttachments(ctx, test.url, inv)
			ctx, err := mergeAttachmentToOutgoing(ctx, inv)
			test.expect(t, ctx, err)
		})
	}
}

// newTestTripleInvoker creates a TripleInvoker for testing without network connection
func newTestTripleInvoker(url *common.URL, cm *clientManager) *TripleInvoker {
	return &TripleInvoker{
		BaseInvoker:   *base.NewBaseInvoker(url),
		quitOnce:      sync.Once{},
		clientGuard:   &sync.RWMutex{},
		clientManager: cm,
	}
}

func TestTripleInvoker_SetGetClientManager(t *testing.T) {
	url := common.NewURLWithOptions()
	ti := newTestTripleInvoker(url, nil)

	// initially nil
	assert.Nil(t, ti.getClientManager())

	// set clientManager
	cm := &clientManager{
		isIDL:     true,
		triClient: nil,
	}
	ti.setClientManager(cm)
	assert.Equal(t, cm, ti.getClientManager())

	// set to nil
	ti.setClientManager(nil)
	assert.Nil(t, ti.getClientManager())
}

func TestTripleInvoker_IsAvailable(t *testing.T) {
	tests := []struct {
		desc          string
		clientManager *clientManager
		expect        bool
	}{
		{
			desc:          "clientManager is nil",
			clientManager: nil,
			expect:        false,
		},
		{
			desc: "clientManager is not nil",
			clientManager: &clientManager{
				isIDL:     true,
				triClient: nil,
			},
			expect: true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			url := common.NewURLWithOptions()
			ti := newTestTripleInvoker(url, test.clientManager)
			assert.Equal(t, test.expect, ti.IsAvailable())
		})
	}
}

func TestTripleInvoker_IsDestroyed(t *testing.T) {
	tests := []struct {
		desc          string
		clientManager *clientManager
		destroyed     bool
		expect        bool
	}{
		{
			desc:          "clientManager is nil",
			clientManager: nil,
			destroyed:     false,
			expect:        false,
		},
		{
			desc: "clientManager is not nil and not destroyed",
			clientManager: &clientManager{
				isIDL:     true,
				triClient: nil,
			},
			destroyed: false,
			expect:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			url := common.NewURLWithOptions()
			ti := newTestTripleInvoker(url, test.clientManager)
			assert.Equal(t, test.expect, ti.IsDestroyed())
		})
	}
}

func TestTripleInvoker_Destroy(t *testing.T) {
	t.Run("destroy with clientManager", func(t *testing.T) {
		url := common.NewURLWithOptions()
		cm := &clientManager{
			isIDL:     true,
			triClient: nil,
		}
		ti := newTestTripleInvoker(url, cm)

		assert.True(t, ti.IsAvailable())
		assert.NotNil(t, ti.getClientManager())

		ti.Destroy()

		assert.False(t, ti.IsAvailable())
		assert.Nil(t, ti.getClientManager())
	})

	t.Run("destroy without clientManager", func(t *testing.T) {
		url := common.NewURLWithOptions()
		ti := newTestTripleInvoker(url, nil)

		ti.Destroy()

		assert.False(t, ti.IsAvailable())
		assert.Nil(t, ti.getClientManager())
	})

	t.Run("destroy called multiple times", func(t *testing.T) {
		url := common.NewURLWithOptions()
		cm := &clientManager{
			isIDL:     true,
			triClient: nil,
		}
		ti := newTestTripleInvoker(url, cm)

		// first destroy
		ti.Destroy()
		assert.Nil(t, ti.getClientManager())

		// second destroy should not panic
		ti.Destroy()
		assert.Nil(t, ti.getClientManager())
	})
}

func TestTripleInvoker_Invoke(t *testing.T) {
	tests := []struct {
		desc         string
		setup        func() (*TripleInvoker, base.Invocation)
		expectErr    error
		expectErrMsg string
	}{
		{
			desc: "invoker is destroyed",
			setup: func() (*TripleInvoker, base.Invocation) {
				url := common.NewURLWithOptions()
				ti := newTestTripleInvoker(url, &clientManager{
					isIDL:     true,
					triClient: nil,
				})
				ti.Destroy()
				inv := invocation.NewRPCInvocationWithOptions()
				return ti, inv
			},
			expectErr: base.ErrDestroyedInvoker,
		},
		{
			desc: "clientManager is nil",
			setup: func() (*TripleInvoker, base.Invocation) {
				url := common.NewURLWithOptions()
				ti := newTestTripleInvoker(url, nil)
				inv := invocation.NewRPCInvocationWithOptions()
				return ti, inv
			},
			expectErr: base.ErrClientClosed,
		},
		{
			desc: "parseInvocation error - miss callType",
			setup: func() (*TripleInvoker, base.Invocation) {
				url := common.NewURLWithOptions()
				ti := newTestTripleInvoker(url, &clientManager{
					isIDL:     true,
					triClient: nil,
				})
				inv := invocation.NewRPCInvocationWithOptions()
				return ti, inv
			},
			expectErrMsg: "miss CallType",
		},
		{
			desc: "mergeAttachmentToOutgoing error - invalid attachment",
			setup: func() (*TripleInvoker, base.Invocation) {
				url := common.NewURLWithOptions()
				ti := newTestTripleInvoker(url, &clientManager{
					isIDL:     true,
					triClient: nil,
				})
				inv := invocation.NewRPCInvocationWithOptions(
					invocation.WithMethodName("TestMethod"),
					invocation.WithAttachment("invalid_key", 123), // invalid attachment type
				)
				inv.SetAttribute(constant.CallTypeKey, constant.CallUnary)
				return ti, inv
			},
			expectErrMsg: "invalid",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			ti, inv := test.setup()
			result := ti.Invoke(context.Background(), inv)
			if test.expectErr != nil {
				assert.Equal(t, test.expectErr, result.Error())
			}
			if test.expectErrMsg != "" {
				require.Error(t, result.Error())
				assert.Contains(t, result.Error().Error(), test.expectErrMsg)
			}
		})
	}
}

func TestTripleInvoker_Invoke_Concurrent(t *testing.T) {
	url := common.NewURLWithOptions()
	ti := newTestTripleInvoker(url, &clientManager{
		isIDL:     true,
		triClient: nil,
	})

	var wg sync.WaitGroup
	concurrency := 10

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			inv := invocation.NewRPCInvocationWithOptions()
			_ = ti.Invoke(context.Background(), inv)
		}()
	}

	wg.Wait()
}

func Test_mergeAttachmentToOutgoing(t *testing.T) {
	tests := []struct {
		desc   string
		ctx    context.Context
		invo   func() base.Invocation
		expect func(t *testing.T, ctx context.Context, err error)
	}{
		{
			desc: "with timeout attachment",
			ctx:  context.Background(),
			invo: func() base.Invocation {
				inv := invocation.NewRPCInvocationWithOptions(
					invocation.WithAttachment(constant.TimeoutKey, "5000"),
				)
				return inv
			},
			expect: func(t *testing.T, ctx context.Context, err error) {
				require.NoError(t, err)
				timeout := ctx.Value(tri.TimeoutKey{})
				assert.Equal(t, "5000", timeout)
			},
		},
		{
			desc: "with string attachment",
			ctx:  context.Background(),
			invo: func() base.Invocation {
				inv := invocation.NewRPCInvocationWithOptions(
					invocation.WithAttachment("custom-key", "custom-value"),
				)
				return inv
			},
			expect: func(t *testing.T, ctx context.Context, err error) {
				require.NoError(t, err)
				header := http.Header(tri.ExtractFromOutgoingContext(ctx))
				assert.Equal(t, "custom-value", header.Get("custom-key"))
			},
		},
		{
			desc: "with string slice attachment",
			ctx:  context.Background(),
			invo: func() base.Invocation {
				inv := invocation.NewRPCInvocationWithOptions(
					invocation.WithAttachment("multi-key", []string{"val1", "val2"}),
				)
				return inv
			},
			expect: func(t *testing.T, ctx context.Context, err error) {
				require.NoError(t, err)
				header := http.Header(tri.ExtractFromOutgoingContext(ctx))
				assert.Equal(t, []string{"val1", "val2"}, header.Values("multi-key"))
			},
		},
		{
			desc: "with invalid attachment type",
			ctx:  context.Background(),
			invo: func() base.Invocation {
				inv := invocation.NewRPCInvocationWithOptions(
					invocation.WithAttachment("invalid-key", 12345),
				)
				return inv
			},
			expect: func(t *testing.T, ctx context.Context, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid")
			},
		},
		{
			desc: "with empty attachments",
			ctx:  context.Background(),
			invo: func() base.Invocation {
				return invocation.NewRPCInvocationWithOptions()
			},
			expect: func(t *testing.T, ctx context.Context, err error) {
				assert.NoError(t, err)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			inv := test.invo()
			ctx, err := mergeAttachmentToOutgoing(test.ctx, inv)
			test.expect(t, ctx, err)
		})
	}
}

// TestParseAttachments_TraceparentNotOverwritten verifies that parseAttachments does NOT
// copy traceparent/tracestate from ctx attachments into the invocation, preserving the
// values injected by the OTEL filter. (Regression test for issue #3240, Root Cause 1)
func TestParseAttachments_TraceparentNotOverwritten(t *testing.T) {
	// Simulate: OTEL filter has already injected the correct traceparent into invocation
	otelTraceparent := "00-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa-bbbbbbbbbbbbbbbb-01"
	otelTracestate := "vendor=opaque"

	inv := invocation.NewRPCInvocationWithOptions(
		invocation.WithAttachment("traceparent", otelTraceparent),
		invocation.WithAttachment("tracestate", otelTracestate),
	)

	// Simulate: upstream request ctx carries a DIFFERENT (stale) traceparent
	upstreamTraceparent := "00-cccccccccccccccccccccccccccccccc-dddddddddddddddd-01"
	upstreamTracestate := "vendor=stale"
	userAtta := map[string]any{
		"traceparent": upstreamTraceparent,
		"tracestate":  upstreamTracestate,
		"user-key":    "user-val", // normal user key should still be copied
	}
	ctx := context.WithValue(context.Background(), constant.AttachmentKey, userAtta)

	url := common.NewURLWithOptions()
	parseAttachments(ctx, url, inv)

	// traceparent and tracestate must remain as the OTEL filter set them
	tp, _ := inv.GetAttachment("traceparent")
	assert.Equal(t, otelTraceparent, tp, "traceparent must not be overwritten by ctx attachment")

	ts, _ := inv.GetAttachment("tracestate")
	assert.Equal(t, otelTracestate, ts, "tracestate must not be overwritten by ctx attachment")

	// normal user key must still be copied
	uk, _ := inv.GetAttachment("user-key")
	assert.Equal(t, "user-val", uk, "non-trace user attachment must still be copied")
}

// TestMergeAttachmentToOutgoing_HeaderIsolation verifies that serial calls to
// mergeAttachmentToOutgoing from the same parent context produce independent outgoing
// headers. (Regression test for issue #3240, Root Cause 2)
func TestMergeAttachmentToOutgoing_HeaderIsolation(t *testing.T) {
	parentCtx := context.Background()

	// First RPC: traceparent for call B
	invB := invocation.NewRPCInvocationWithOptions(
		invocation.WithAttachment("traceparent", "00-aaaa-bbbb-01"),
		invocation.WithAttachment("shared-key", "valueB"),
	)
	ctxB, err := mergeAttachmentToOutgoing(parentCtx, invB)
	require.NoError(t, err)

	headerB := http.Header(tri.ExtractFromOutgoingContext(ctxB))
	assert.Equal(t, "00-aaaa-bbbb-01", headerB.Get("traceparent"))
	assert.Equal(t, "valueB", headerB.Get("shared-key"))

	// Second RPC: traceparent for call C — uses same parentCtx
	invC := invocation.NewRPCInvocationWithOptions(
		invocation.WithAttachment("traceparent", "00-cccc-dddd-01"),
		invocation.WithAttachment("shared-key", "valueC"),
	)
	ctxC, err := mergeAttachmentToOutgoing(parentCtx, invC)
	require.NoError(t, err)

	headerC := http.Header(tri.ExtractFromOutgoingContext(ctxC))
	assert.Equal(t, "00-cccc-dddd-01", headerC.Get("traceparent"),
		"second RPC must carry its own traceparent, not the first RPC's value")
	assert.Equal(t, "valueC", headerC.Get("shared-key"))

	// Verify the two headers are completely independent
	assert.NotEqual(t, headerB.Get("traceparent"), headerC.Get("traceparent"),
		"serial RPCs must have independent traceparent values")

	// Verify headerB was not mutated by the second call
	assert.Equal(t, "00-aaaa-bbbb-01", headerB.Get("traceparent"),
		"first RPC's header must not be mutated by second RPC")
	// Verify no accumulation: each header should have exactly one traceparent value
	assert.Len(t, headerB.Values("traceparent"), 1, "headerB must have exactly one traceparent")
	assert.Len(t, headerC.Values("traceparent"), 1, "headerC must have exactly one traceparent")
}
