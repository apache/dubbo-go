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

package remoting

import (
	"context"
	"errors"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/rpcerror"
)

// The async callback must receive the decoded error unchanged, while metrics
// still see the kind declared by the response.
func TestHandleAsyncKeepsCallbackErrorAndReportsKind(t *testing.T) {
	raw := errors.New("payload limit exceeded")
	_, o := rpcerror.WithObservation(context.Background())
	got := make(chan rpcerror.Completion, 1)
	o.OnComplete(func(c rpcerror.Completion) { got <- c })
	o.BeginAsync(context.Background(), time.Minute)
	o.FinishInvoke(nil, true)

	p := NewPendingResponse(SequenceID())
	p.Observation = o
	var callbackErr error
	p.Callback = func(r common.CallbackResponse) {
		callbackErr = r.(AsyncCallbackResponse).Reply.(*Response).Error
	}
	AddPendingResponse(p)
	(&Response{ID: p.seq, Error: raw, Kind: rpcerror.KindCodec}).Handle()

	require.Same(t, raw, callbackErr, "callback must see the decoded error type")
	select {
	case c := <-got:
		assert.True(t, c.Known)
		assert.Equal(t, rpcerror.KindCodec, rpcerror.KindOf(c.Err))
		require.ErrorIs(t, c.Err, raw)
	case <-time.After(time.Second):
		t.Fatal("observation did not complete")
	}
}

// A synchronous waiter gets the raw error plus the kind; the requester decides
// whether to record it, because it may return a timeout instead.
func TestHandleSyncPublishesKindBeforeDone(t *testing.T) {
	raw := errors.New("biz")
	p := NewPendingResponse(SequenceID())
	AddPendingResponse(p)
	(&Response{ID: p.seq, Error: raw, Kind: rpcerror.KindBusiness}).Handle()

	<-p.Done
	require.Same(t, raw, p.Err)
	assert.Equal(t, rpcerror.KindBusiness, p.Kind)
}
