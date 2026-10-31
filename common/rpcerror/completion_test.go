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
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestCallbackCompletionHandoff(t *testing.T) {
	for _, early := range []bool{false, true} {
		_, o := WithObservation(context.Background())
		got := make(chan Completion, 4)
		o.OnComplete(func(c Completion) { got <- c })
		o.BeginAsync(context.Background(), time.Minute)
		cause := errors.New("response")
		if early {
			o.Complete(cause)
		}
		o.FinishInvoke(nil, true)
		if !early {
			select {
			case <-got:
				t.Fatal("send success counted as completion")
			default:
			}
			o.Complete(cause)
		}
		c := <-got
		if !c.Known || c.Err != cause {
			t.Fatal(c)
		}
		o.Complete(cause)
		o.FinishInvoke(nil, true)
		select {
		case <-got:
			t.Fatal("duplicate completion")
		default:
		}
		o.mu.Lock()
		if o.notify != nil || o.timer != nil || o.stopContext != nil || o.session != nil || o.invocation != nil {
			t.Fatal("observer resources retained")
		}
		o.mu.Unlock()
	}
}

func TestCallbackSendErrorWinsBufferedResponse(t *testing.T) {
	_, o := WithObservation(context.Background())
	got := make(chan Completion, 1)
	o.OnComplete(func(c Completion) { got <- c })
	o.BeginAsync(context.Background(), time.Minute)
	o.Complete(nil)
	cause := errors.New("send error")
	o.FinishInvoke(cause, true)
	if (<-got).Err != cause {
		t.Fatal("changed Invoke failure accounting")
	}
}

func TestCallbackObservationExpiry(t *testing.T) {
	for _, canceled := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		_, o := WithObservation(ctx)
		got := make(chan Completion, 2)
		o.OnComplete(func(c Completion) { got <- c })
		timeout := time.Minute
		if !canceled {
			timeout = time.Millisecond
		}
		o.BeginAsync(ctx, timeout)
		o.FinishInvoke(nil, true)
		if canceled {
			cancel()
		}
		select {
		case c := <-got:
			if c.Known || c.Err != nil {
				t.Fatal("expiry fabricated a business error")
			}
		case <-time.After(time.Second):
			t.Fatal("observation did not expire")
		}
		cancel()
		o.Complete(errors.New("late response"))
		select {
		case <-got:
			t.Fatal("late result changed frozen observation")
		default:
		}
	}
}

func TestCallbackCompletionRace(t *testing.T) {
	for range 100 {
		_, o := WithObservation(context.Background())
		var count atomic.Int32
		o.OnComplete(func(Completion) { count.Add(1) })
		o.BeginAsync(context.Background(), time.Minute)
		var wg sync.WaitGroup
		wg.Go(func() { o.Complete(nil) })
		wg.Go(func() { o.FinishInvoke(nil, true) })
		wg.Go(func() { o.complete(Completion{At: time.Now()}) })
		wg.Wait()
		if count.Load() != 1 {
			t.Fatalf("completed %d times", count.Load())
		}
	}
}

func TestBeginAsyncDefaultsAndRepeatedHandoff(t *testing.T) {
	var absent *Observation
	absent.BeginAsync(context.Background(), time.Minute)
	absent.Complete(nil)
	_, o := WithObservation(context.Background())
	defer o.Freeze()
	//nolint:staticcheck // Verify the explicitly supported nil-context fallback.
	o.BeginAsync(nil, 0)
	timer := o.timer
	if timer == nil || o.stopContext == nil || !o.async {
		t.Fatal("default async lifetime was not installed")
	}
	o.BeginAsync(context.Background(), time.Minute)
	if o.timer != timer {
		t.Fatal("repeated handoff replaced the observation lifetime")
	}
	o.Complete(nil)
	o.FinishInvoke(nil, true) // No observer is also a supported terminal path.
	if !o.frozen || o.timer != nil || o.stopContext != nil {
		t.Fatal("completion retained observation resources")
	}
	o.BeginAsync(context.Background(), time.Minute)
	if o.timer != nil {
		t.Fatal("finished observation was restarted")
	}
}
