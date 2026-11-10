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

package rpcerror

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
)

func TestClaimObservationRequiresInvocationIdentity(t *testing.T) {
	owner, other := new(int), new(int)
	ctx, o := WithInvocationObservation(context.Background(), owner)
	var typedNil *int
	for _, invalid := range []any{nil, typedNil, other, *owner, []int{0}, map[string]int{}} {
		if ClaimObservation(ctx, invalid) != nil {
			t.Fatalf("unrelated invocation claimed observation: %T", invalid)
		}
	}
	//nolint:staticcheck // Exercise the intentional nil-context boundary, not normal context usage.
	if ClaimObservation(nil, owner) != nil {
		t.Fatal("nil context claimed an observation")
	}
	if ClaimObservation(context.Background(), owner) != nil {
		t.Fatal("missing observation was claimed")
	}
	if ClaimObservation(ctx, owner) != o || o.invocation != nil {
		t.Fatal("owner did not claim and release its invocation reference")
	}
	if ClaimObservation(ctx, owner) != nil {
		t.Fatal("observation was claimed twice")
	}
}

func TestUnboundObservationShadowsInheritedOwner(t *testing.T) {
	owner := new(int)
	ctx, outer := WithInvocationObservation(context.Background(), owner)
	providerCtx, provider := WithObservation(ctx)
	if ObservationFromContext(providerCtx) != provider || ClaimObservation(providerCtx, owner) != nil {
		t.Fatal("provider observation exposed the inherited consumer owner")
	}
	childOwner := new(int)
	childCtx, child := WithInvocationObservation(providerCtx, childOwner)
	if ClaimObservation(childCtx, owner) != nil || ClaimObservation(childCtx, childOwner) != child || child == provider {
		t.Fatal("nested consumer did not get an independent owner")
	}
	if ClaimObservation(ctx, owner) != outer {
		t.Fatal("nested context changed the outer observation")
	}
}

func TestObservationReleasesUnclaimedInvocation(t *testing.T) {
	for _, freeze := range []bool{false, true} {
		owner := new(int)
		ctx, o := WithInvocationObservation(context.Background(), owner)
		if freeze {
			o.Freeze()
		} else {
			o.FinishInvoke(nil, true)
		}
		if o.invocation != nil || ClaimObservation(ctx, owner) != nil {
			t.Fatal("finished observation retained invocation ownership")
		}
	}
}

func TestClaimObservationRejectsInvalidOwner(t *testing.T) {
	var typedNil *int
	for _, invalid := range []any{nil, typedNil, 0, []int{0}, map[string]int{}} {
		ctx, o := WithInvocationObservation(context.Background(), invalid)
		if o.invocation != nil || ClaimObservation(ctx, invalid) != nil {
			t.Fatalf("invalid owner retained or claimed: %T", invalid)
		}
	}
}

func TestClaimObservationConcurrent(t *testing.T) {
	owner := new(int)
	ctx, o := WithInvocationObservation(context.Background(), owner)
	var claims atomic.Int32
	var wg sync.WaitGroup
	for range 32 {
		wg.Go(func() {
			if ClaimObservation(ctx, owner) == o {
				claims.Add(1)
			}
		})
	}
	wg.Wait()
	if claims.Load() != 1 || o.invocation != nil {
		t.Fatalf("expected one claim and no retained invocation, got %d", claims.Load())
	}
}
