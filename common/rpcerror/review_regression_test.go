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

package rpcerror

import (
	"context"
	"testing"
	"time"
)

func TestEnsureObservationUpgradesConsumerBinding(t *testing.T) {
	inv := new(int)
	ctx, o, owner := EnsureObservation(context.Background(), inv, false)
	if !owner {
		t.Fatal("outer observer must own lifecycle")
	}
	ctx, reused, owner := EnsureObservation(ctx, inv, true)
	if reused != o || owner {
		t.Fatal("consumer split observation")
	}
	RecordKindFor(ctx, inv, KindLimit)
	if ClaimObservation(ctx, inv) != o {
		t.Fatal("consumer upgrade cannot claim")
	}
	if ClaimObservation(ctx, inv) != nil {
		t.Fatal("claimed twice")
	}
	o.OnComplete(func(c Completion) {
		if c.Kind != KindLimit {
			t.Errorf("kind lost: %+v", c)
		}
	})
	o.FinishInvoke(nil, true)
}

func TestRebindObservationIdentityAndClaim(t *testing.T) {
	old, translated := new(int), new(int)
	ctx, o, _ := EnsureObservation(context.Background(), old, true)
	RebindObservation(ctx, new(int), translated)
	if o.identity != old {
		t.Fatal("unrelated invocation rebound observation")
	}
	RebindObservation(ctx, old, translated)
	_, reused, owner := EnsureObservation(ctx, translated, true)
	if reused != o || owner {
		t.Fatal("translation split observation")
	}
	if ClaimObservation(ctx, old) != nil || ClaimObservation(ctx, translated) != o {
		t.Fatal("claim binding not transferred")
	}
	RebindObservation(ctx, translated, old)
	if o.invocation != nil || o.identity != nil || !o.claimed {
		t.Fatal("claimed observation rebound")
	}
	o.FinishInvoke(nil, true)
	RebindObservation(ctx, translated, old)
	if o.identity != nil {
		t.Fatal("frozen observation rebound")
	}
}

func TestClaimedObservationCannotUpgradeAgain(t *testing.T) {
	inv := new(int)
	ctx, o, _ := EnsureObservation(context.Background(), inv, true)
	ClaimObservation(ctx, inv)
	// A retained identity must not authorize rebinding an already claimed request.
	o.mu.Lock()
	o.identity = inv
	o.mu.Unlock()
	_, reused, owner := EnsureObservation(ctx, inv, true)
	if reused != o || owner || o.invocation != nil {
		t.Fatal("consumer upgrade reclaimed transport observation")
	}
	o.FinishInvoke(nil, true)
}

func TestObservationNilInputs(t *testing.T) {
	var nilCtx context.Context
	ctx, o := WithObservation(nilCtx)
	if ctx == nil || o == nil {
		t.Fatal("nil context not normalized")
	}
	ctx, o, _ = EnsureObservation(nilCtx, new(int), true)
	if ctx == nil || o == nil {
		t.Fatal("EnsureObservation returned nil")
	}
	if OwnsSpan(nilCtx, "span") {
		t.Fatal("nil context owns span")
	}
	if !OwnsSpan(WithSpanOwner(nilCtx, "span"), "span") {
		t.Fatal("span owner missing")
	}
	(*Observation)(nil).RecordTraceID("trace")
	RebindObservation(nilCtx, new(int), new(int))
	o.FinishInvoke(nil, false)
}

type nilCompletionError struct{}

func (*nilCompletionError) Error() string { panic("typed nil must not be formatted") }

func TestTypedNilCompletionIsSuccess(t *testing.T) {
	var err *nilCompletionError
	for _, async := range []bool{false, true} {
		_, o := WithObservation(context.Background())
		got := make(chan Completion, 1)
		o.OnComplete(func(c Completion) { got <- c })
		if async {
			o.BeginAsync(context.Background(), time.Minute)
			o.Complete(err)
		}
		o.FinishInvoke(err, true)
		select {
		case c := <-got:
			if c.Err != nil || !c.Known {
				t.Fatalf("typed nil misclassified: %+v", c)
			}
		default:
			t.Fatal("typed nil left async completion pending")
		}
	}
}

func TestUnknownCompletionReleasesAsyncResources(t *testing.T) {
	ctx := t.Context()
	_, o := WithObservation(ctx)
	o.BeginAsync(ctx, time.Minute)
	o.FinishInvoke(nil, false)
	o.mu.Lock()
	defer o.mu.Unlock()
	if !o.frozen || o.timer != nil || o.stopContext != nil {
		t.Fatal("panic fallback retained monitoring resources")
	}
}
