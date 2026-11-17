/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements. See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License. You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package sentinel

import (
	"context"
	"testing"
	"time"
)

import (
	sentinelbase "github.com/alibaba/sentinel-golang/core/base"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/rpcerror"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

func completedKind(t *testing.T, o *rpcerror.Observation, err error) rpcerror.Kind {
	t.Helper()
	got := make(chan rpcerror.Completion, 1)
	o.OnComplete(func(c rpcerror.Completion) { got <- c })
	o.FinishInvoke(err, true)
	select {
	case c := <-got:
		return c.Kind
	case <-time.After(time.Second):
		t.Fatal("observation did not complete")
		return rpcerror.KindUnknown
	}
}

func TestDefaultFallbacksRecordLimitKind(t *testing.T) {
	blockErr := sentinelbase.NewBlockError(sentinelbase.WithBlockType(sentinelbase.BlockTypeFlow))

	t.Run("provider", func(t *testing.T) {
		ctx, o := rpcerror.WithObservation(context.Background())
		inv := invocation.NewRPCInvocation("Call", nil, nil)
		res := getDefaultDubboFallback()(ctx, nil, inv, blockErr)
		require.Same(t, blockErr, res.Error(), "the block error type must not change")
		assert.Equal(t, rpcerror.KindLimit, completedKind(t, o, res.Error()))
	})

	t.Run("consumer owns its observation", func(t *testing.T) {
		inv := invocation.NewRPCInvocation("Call", nil, nil)
		ctx, o := rpcerror.WithInvocationObservation(context.Background(), inv)
		res := getDefaultDubboConsumerFallback()(ctx, nil, inv, blockErr)
		require.Same(t, blockErr, res.Error())
		assert.Equal(t, rpcerror.KindLimit, completedKind(t, o, res.Error()))
	})

	t.Run("consumer does not relabel an enclosing request", func(t *testing.T) {
		ctx, enclosing := rpcerror.WithObservation(context.Background())
		res := getDefaultDubboConsumerFallback()(ctx, nil, invocation.NewRPCInvocation("Downstream", nil, nil), blockErr)
		require.Same(t, blockErr, res.Error())
		assert.Equal(t, rpcerror.KindUnknown, completedKind(t, enclosing, nil))
	})
}
