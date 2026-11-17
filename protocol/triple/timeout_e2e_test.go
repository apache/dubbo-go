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
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/require"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"google.golang.org/protobuf/types/known/wrapperspb"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	tri "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
)

type timeoutObservation struct {
	hasDeadline bool
	remaining   time.Duration
	grpcTimeout string
}

func startTimeoutE2EServer(t *testing.T) (addr string, seen *timeoutObservation, done func()) {
	t.Helper()
	obs := &timeoutObservation{}
	var mu sync.Mutex
	pingHandler := tri.NewUnaryHandler(
		"/connect.ping.v1.PingService/Ping",
		func() any { return &wrapperspb.StringValue{} },
		func(ctx context.Context, req *tri.Request) (*tri.Response, error) {
			mu.Lock()
			defer mu.Unlock()
			if dl, ok := ctx.Deadline(); ok {
				obs.hasDeadline = true
				obs.remaining = time.Until(dl)
			}
			return tri.NewResponse(&wrapperspb.StringValue{Value: req.Any().(*wrapperspb.StringValue).Value}), nil
		},
	)
	wrapped := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/grpc.health.v1.Health/") {
			mu.Lock()
			obs.grpcTimeout = r.Header.Get("Grpc-Timeout")
			mu.Unlock()
		}
		pingHandler.ServeHTTP(w, r)
	})
	server := &http.Server{Handler: h2c.NewHandler(wrapped, &http2.Server{})}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	go func() { _ = server.Serve(ln) }()
	return ln.Addr().String(), obs, func() { _ = server.Close() }
}

func referTimeoutE2E(t *testing.T, addr string, urlTimeout string) base.Invoker {
	t.Helper()
	opts := []common.Option{
		common.WithMethods([]string{"Ping"}),
		common.WithProtocol(TRIPLE),
		common.WithParamsValue(constant.IDLMode, constant.NONIDL),
	}
	if urlTimeout != "" {
		opts = append(opts, common.WithParamsValue(constant.TimeoutKey, urlTimeout))
	}
	url, err := common.NewURL("tri://"+addr+"/connect.ping.v1.PingService", opts...)
	require.NoError(t, err)
	invoker := GetProtocol().Refer(url)
	require.NotNil(t, invoker)
	t.Cleanup(invoker.Destroy)
	return invoker
}

func pingOnce(t *testing.T, urlTimeout, callTimeout string, parent context.Context) *timeoutObservation {
	t.Helper()
	addr, obs, stop := startTimeoutE2EServer(t)
	t.Cleanup(stop)
	invoker := referTimeoutE2E(t, addr, urlTimeout)
	if parent == nil {
		parent = context.Background()
	}
	resp := &wrapperspb.StringValue{}
	inv := invocation.NewRPCInvocationWithOptions(
		invocation.WithMethodName("Ping"),
		invocation.WithParameterRawValues([]any{&wrapperspb.StringValue{Value: "hello"}, resp}),
	)
	inv.SetAttribute(constant.CallTypeKey, constant.CallUnary)
	if callTimeout != "" {
		inv.SetAttachment(constant.TimeoutKey, callTimeout)
	}
	res := invoker.Invoke(parent, inv)
	require.NoError(t, res.Error())
	require.Equal(t, "hello", resp.Value)
	return obs
}

func requireRemainingNear(t *testing.T, obs *timeoutObservation, want time.Duration) {
	t.Helper()
	require.True(t, obs.hasDeadline, "handler ctx must have a deadline")
	require.Greater(t, obs.remaining, time.Duration(0), "handler deadline already expired")
	require.LessOrEqual(t, obs.remaining, want)
	// Distinguish 100ms vs 1s vs 5s without treating this as a wall-clock-only test:
	// the remaining time must be in the same order as the expected timeout, and
	// the gRPC timeout header must have been set on the production request.
	require.NotEmpty(t, obs.grpcTimeout, "production request must carry a grpc-timeout header")
	lo := want / 2
	if lo < 20*time.Millisecond {
		lo = 20 * time.Millisecond
	}
	require.Greater(t, obs.remaining, lo, "handler remaining %s is too far from %s (grpc-timeout=%q)", obs.remaining, want, obs.grpcTimeout)
}

// TestTripleTimeoutOverProductionRPC verifies the two Triple timeout defects
// on the production Refer/Invoke path against a real h2c server and handler
// context / grpc-timeout header, not only package-level helpers.
func TestTripleTimeoutOverProductionRPC(t *testing.T) {
	t.Run("unitless millisecond call timeout", func(t *testing.T) {
		// URL fallback would be 1s; unitless "5000" must be 5s.
		obs := pingOnce(t, "1s", "5000", nil)
		requireRemainingNear(t, obs, 5*time.Second)
		require.Greater(t, obs.remaining, 2*time.Second, "must not fall back to the 1s URL timeout (grpc-timeout=%q)", obs.grpcTimeout)
	})

	t.Run("call-level timeout overrides URL timeout", func(t *testing.T) {
		obs := pingOnce(t, "5s", "200ms", nil)
		requireRemainingNear(t, obs, 200*time.Millisecond)
		require.Less(t, obs.remaining, 2*time.Second, "must not be overwritten by the 5s URL timeout (grpc-timeout=%q)", obs.grpcTimeout)
	})

	t.Run("existing context deadline is not extended", func(t *testing.T) {
		parent, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		t.Cleanup(cancel)
		obs := pingOnce(t, "5s", "10s", parent)
		requireRemainingNear(t, obs, 200*time.Millisecond)
		require.Less(t, obs.remaining, time.Second, "existing deadline must not be extended to the 10s attachment (grpc-timeout=%q)", obs.grpcTimeout)
	})

	t.Run("URL timeout remains the no-attachment fallback", func(t *testing.T) {
		obs := pingOnce(t, "3s", "", nil)
		requireRemainingNear(t, obs, 3*time.Second)
	})

	t.Run("empty URL timeout keeps GetParamDuration legal default", func(t *testing.T) {
		// GetParamDuration("", "") returns 3s; that is existing production
		// behavior and is not changed by this patch.
		obs := pingOnce(t, "", "", nil)
		requireRemainingNear(t, obs, 3*time.Second)
	})
}
