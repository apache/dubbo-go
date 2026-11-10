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
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/rpcerror"
)

func TestObservationExpiryPreservesLateUserCallback(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, o := rpcerror.WithObservation(ctx)
	expired := make(chan rpcerror.Completion, 1)
	o.OnComplete(func(c rpcerror.Completion) { expired <- c })
	o.BeginAsync(ctx, time.Minute)
	p := NewPendingResponse(SequenceID())
	p.Observation = o
	var calls atomic.Int32
	p.Callback = func(common.CallbackResponse) { calls.Add(1) }
	AddPendingResponse(p)
	o.FinishInvoke(nil, true)
	cancel()
	select {
	case c := <-expired:
		if c.Known {
			t.Fatal("invented result")
		}
	case <-time.After(time.Second):
		t.Fatal("no cleanup")
	}
	if GetPendingResponse(SequenceType(p.seq)) != p {
		t.Fatal("observation deleted business waiter")
	}
	(&Response{ID: p.seq}).Handle()
	if calls.Load() != 1 {
		t.Fatal("suppressed late user callback")
	}
	select {
	case <-expired:
		t.Fatal("repeated metrics completion")
	default:
	}
}

func TestCallbackResponseSnapshotRace(t *testing.T) {
	p := NewPendingResponse(SequenceID())
	p.Callback = func(common.CallbackResponse) {}
	AddPendingResponse(p)
	var wg sync.WaitGroup
	wg.Go(func() { (&Response{ID: p.seq}).Handle() })
	wg.Go(func() {
		for range 100 {
			if p.getResponse() == nil {
				t.Error("lost response snapshot")
			}
		}
	})
	wg.Wait()
}
