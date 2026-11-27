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
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestPendingRejectsKindEvidence(t *testing.T) {
	for range 100 {
		invocation := new(int)
		ctx, o := WithInvocationObservation(context.Background(), invocation)
		o.BeginAsync(ctx, time.Minute)
		o.Complete(errors.New("response"))
		var wg sync.WaitGroup
		for range 10 {
			wg.Go(func() { o.RecordKind(KindBusiness); RecordKindFor(ctx, invocation, KindLimit) })
		}
		wg.Wait()
		o.OnComplete(func(c Completion) {
			if c.Kind != KindUnknown {
				t.Errorf("pending snapshot changed: %v", c.Kind)
			}
		})
		o.FinishInvoke(nil, true)
	}
}

func TestConcurrentSubscribersAndLateReplay(t *testing.T) {
	_, o := WithObservation(context.Background())
	var count atomic.Int32
	o.OnComplete(func(Completion) { panic("monitor failure") })
	var wg sync.WaitGroup
	for range 100 {
		wg.Go(func() {
			o.OnComplete(func(c Completion) {
				if !c.Known || c.Kind != KindBusiness {
					t.Errorf("bad snapshot: %+v", c)
				}
				count.Add(1)
			})
		})
	}
	o.RecordKind(KindBusiness)
	wg.Go(func() { o.FinishInvoke(errors.New("business"), true) })
	wg.Wait()
	o.FinishInvoke(nil, true)
	o.Complete(nil)
	if count.Load() != 100 {
		t.Fatalf("not exactly once: %d", count.Load())
	}
}

func TestEnsureObservationOwnership(t *testing.T) {
	invocation := new(int)
	ctx, first, owner := EnsureObservation(context.Background(), invocation, true)
	if !owner {
		t.Fatal("missing owner")
	}
	_, second, owner := EnsureObservation(ctx, invocation, true)
	if owner || first != second {
		t.Fatal("same attempt replaced")
	}
	_, child, owner := EnsureObservation(ctx, new(int), true)
	if !owner || child == first {
		t.Fatal("inherited observation reused by downstream call")
	}
	first.FinishInvoke(nil, true)
	_, retry, owner := EnsureObservation(ctx, invocation, true)
	if !owner || retry == first {
		t.Fatal("retry reused frozen observation")
	}
}

func TestClaimReleasesFilterInvocationIdentity(t *testing.T) {
	invocation := new(int)
	ctx, o, _ := EnsureObservation(context.Background(), invocation, true)
	if ClaimObservation(ctx, invocation) != o {
		t.Fatal("claim failed")
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.identity != nil || o.invocation != nil {
		t.Fatal("callback monitoring retained mutable invocation")
	}
}
