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

package base_test

import (
	"context"
	"errors"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

func TestIsStreamingCall(t *testing.T) {
	inv := invocation.NewRPCInvocation("Call", nil, nil)
	assert.False(t, base.IsStreamingCall(inv))
	for callType, streaming := range map[string]bool{
		constant.CallUnary:        false,
		constant.CallClientStream: true,
		constant.CallServerStream: true,
		constant.CallBidiStream:   true,
	} {
		inv.SetAttribute(constant.CallTypeKey, callType)
		assert.Equal(t, streaming, base.IsStreamingCall(inv), callType)
	}
}

func TestClaimStreamWatches(t *testing.T) {
	inv := invocation.NewRPCInvocation("Call", nil, nil)
	var ended []string
	ctx, outerClaimed := base.WatchStream(context.Background(), inv, func(error) { ended = append(ended, "outer") })
	ctx, innerClaimed := base.WatchStream(ctx, inv, func(error) { ended = append(ended, "inner") })

	assert.Nil(t, base.ClaimStreamWatches(ctx, invocation.NewRPCInvocation("Other", nil, nil)),
		"another invocation must not claim the watches")
	assert.False(t, outerClaimed())
	assert.False(t, innerClaimed())

	end := base.ClaimStreamWatches(ctx, inv)
	require.NotNil(t, end)
	assert.True(t, outerClaimed())
	assert.True(t, innerClaimed())
	assert.Nil(t, base.ClaimStreamWatches(ctx, inv), "a watch is claimed only once")
	assert.Empty(t, ended, "claiming does not end the stream")

	end(errors.New("stream failed"))
	assert.Equal(t, []string{"inner", "outer"}, ended)
}

func TestClaimStreamWatchesWithoutWatch(t *testing.T) {
	assert.Nil(t, base.ClaimStreamWatches(context.Background(), invocation.NewRPCInvocation("Call", nil, nil)))
}
