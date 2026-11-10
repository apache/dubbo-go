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

package forking

import (
	"context"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.uber.org/goleak"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/directory/static"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

type contextTestInvoker struct {
	*base.BaseInvoker
	invoke func(context.Context) result.Result
}

func (invoker *contextTestInvoker) Invoke(ctx context.Context, _ base.Invocation) result.Result {
	return invoker.invoke(ctx)
}

func TestForkingInvokeCancelsOutstandingCalls(t *testing.T) {
	for _, tc := range []struct {
		name    string
		success bool
		timeout string
	}{
		{name: "first result", success: true, timeout: "1s"},
		{name: "poll timeout", timeout: "50ms"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
			ctx := t.Context() // Also release the test providers if an assertion fails.

			u := common.NewURLWithOptions(common.WithProtocol("dubbo"))
			u.AddParam(constant.ForksKey, "-1") // Invoke every provider without load balancing.
			u.AddParam(constant.TimeoutKey, tc.timeout)
			const slowCalls = 3
			started := make(chan struct{}, slowCalls)
			stopped := make(chan error, slowCalls)
			var providers []base.Invoker
			for range slowCalls {
				providers = append(providers, &contextTestInvoker{
					BaseInvoker: base.NewBaseInvoker(u),
					invoke: func(ctx context.Context) result.Result {
						started <- struct{}{}
						<-ctx.Done()
						stopped <- ctx.Err()
						return &result.RPCResult{Err: ctx.Err()}
					},
				})
			}
			want := &result.RPCResult{Rest: "first result"}
			if tc.success {
				providers = append(providers, &contextTestInvoker{
					BaseInvoker: base.NewBaseInvoker(u),
					invoke: func(ctx context.Context) result.Result {
						// Ensure all losing calls are in flight before returning the winner.
						for range slowCalls {
							select {
							case <-started:
							case <-ctx.Done():
								return &result.RPCResult{Err: ctx.Err()}
							}
						}
						return want
					},
				})
			}
			clusterInvoker := newForkingClusterInvoker(static.NewDirectory(providers))
			returned := make(chan result.Result, 1)
			go func() {
				returned <- clusterInvoker.Invoke(ctx, &invocation.RPCInvocation{})
			}()

			select {
			case got := <-returned:
				if tc.success {
					assert.Same(t, want, got)
					require.NoError(t, got.Error())
				} else {
					require.Error(t, got.Error())
				}
			case <-time.After(5 * time.Second):
				t.Fatal("forking Invoke did not return")
			}

			timer := time.NewTimer(5 * time.Second)
			defer timer.Stop()
			for range slowCalls {
				select {
				case err := <-stopped:
					require.ErrorIs(t, err, context.Canceled)
				case <-timer.C:
					t.Fatal("forked calls remained blocked after Invoke returned")
				}
			}
			require.NoError(t, ctx.Err(), "fork cancellation must not cancel the caller's context")
		})
	}
}
