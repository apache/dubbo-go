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

package failover

import (
	"context"
	"errors"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.uber.org/mock/gomock"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/directory/static"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/mock"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

// TestFailoverMarksRetryAttempts checks the mark the metrics filter reads to
// count consumer retries.
func TestFailoverMarksRetryAttempts(t *testing.T) {
	failed := &result.RPCResult{Err: errors.New("unavailable")}
	cases := []struct {
		name           string
		enclosingRetry bool
		results        []*result.RPCResult
		want           []bool
	}{
		{"first attempt succeeds", false, []*result.RPCResult{{}}, []bool{false}},
		{"first attempt fails", false, []*result.RPCResult{failed, {}}, []bool{false, true}},
		{"two attempts fail", false, []*result.RPCResult{failed, failed, {}}, []bool{false, true, true}},
		{"inside an enclosing retry", true, []*result.RPCResult{failed, {}}, []bool{true, true}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			invoker := mock.NewMockInvoker(ctrl)
			invoker.EXPECT().IsAvailable().Return(true).AnyTimes()
			var marks []bool
			invoker.EXPECT().Invoke(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, inv base.Invocation) result.Result {
				marks = append(marks, base.IsRetryAttempt(ctx, inv))
				return c.results[len(marks)-1]
			}).Times(len(c.results))
			cluster := registerFailover(invoker)

			inv := &invocation.RPCInvocation{}
			ctx := context.Background()
			if c.enclosingRetry {
				ctx = base.WithRetryAttempt(ctx, inv)
			}
			res := cluster.Invoke(ctx, inv)
			require.NoError(t, res.Error())
			assert.Equal(t, c.want, marks)
		})
	}
}

// TestNestedFailoverMarksRetryAttempts covers a group failover inside a
// multi-group failover: every leaf attempt of the outer retry is a retry.
func TestNestedFailoverMarksRetryAttempts(t *testing.T) {
	ctrl := gomock.NewController(t)
	leaf := mock.NewMockInvoker(ctrl)
	leaf.EXPECT().IsAvailable().Return(true).AnyTimes()
	results := []*result.RPCResult{}
	for range constant.DefaultRetriesInt + 1 { // the first inner call fails
		results = append(results, &result.RPCResult{Err: errors.New("unavailable")})
	}
	results = append(results, &result.RPCResult{}) // the outer retry succeeds
	var marks []bool
	leaf.EXPECT().Invoke(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, inv base.Invocation) result.Result {
		marks = append(marks, base.IsRetryAttempt(ctx, inv))
		return results[len(marks)-1]
	}).Times(len(results))
	outer := newFailoverClusterInvoker(static.NewDirectory([]base.Invoker{registerFailover(leaf)}))

	res := outer.Invoke(context.Background(), &invocation.RPCInvocation{})
	require.NoError(t, res.Error())
	want := []bool{false} // then the inner retries and the outer retry
	for range constant.DefaultRetriesInt + 1 {
		want = append(want, true)
	}
	assert.Equal(t, want, marks)
}
