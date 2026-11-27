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
	"fmt"
	"net"
	"testing"
	"time"
)

func TestKindErrorsPreserveIdentityAndCause(t *testing.T) {
	sentinel := NewKindError(KindServiceUnavailable, "closed")
	if sentinel.Error() != "closed" || KindOf(sentinel) != KindServiceUnavailable {
		t.Fatal("sentinel must keep its message and kind")
	}
	wrapped := fmt.Errorf("outer: %w", sentinel)
	if !errors.Is(wrapped, sentinel) || KindOf(wrapped) != KindServiceUnavailable {
		t.Fatal("kind must survive wrapping")
	}
	if NewKindError(KindUnknown, "plain").Error() != "plain" || KindOf(NewKindError(KindUnknown, "plain")) != KindUnknown {
		t.Fatal("invalid kind must produce a plain error")
	}

	cause := &net.OpError{Op: "read", Net: "tcp", Err: errors.New("fixture")}
	marked := WithKind(cause, KindBusiness)
	var got *net.OpError
	if marked.Error() != cause.Error() || !errors.Is(marked, cause) || !errors.As(marked, &got) || got != cause {
		t.Fatal("WithKind changed the message or cause chain")
	}
	if errors.Unwrap(marked) != cause {
		t.Fatal("root cause must be reachable by Unwrap")
	}
}

func TestWithKindKeepsFirstDeclaration(t *testing.T) {
	inner := WithKind(errors.New("timeout"), KindTimeout)
	if outer := WithKind(fmt.Errorf("ctx: %w", inner), KindBusiness); KindOf(outer) != KindTimeout {
		t.Fatal("an outer layer must not relabel an inner classification")
	}
	var typedNil *net.OpError
	if WithKind(nil, KindTimeout) != nil || WithKind(typedNil, KindTimeout) != typedNil || KindOf(typedNil) != KindUnknown {
		t.Fatal("nil and typed nil must be preserved")
	}
	plain := errors.New("plain")
	if WithKind(plain, KindUnknown) != plain || WithKind(plain, kindConflict) != plain {
		t.Fatal("invalid kinds must not wrap")
	}
}

func TestKindOfConflictsAndContext(t *testing.T) {
	joined := errors.Join(WithKind(errors.New("a"), KindTimeout), WithKind(errors.New("b"), KindLimit))
	if KindOf(joined) != KindUnknown {
		t.Fatal("conflicting kinds must not be guessed")
	}
	same := errors.Join(WithKind(errors.New("a"), KindLimit), WithKind(errors.New("b"), KindLimit))
	if KindOf(same) != KindLimit {
		t.Fatal("agreeing kinds are not a conflict")
	}
	sticky := errors.Join(WithKind(errors.New("a"), KindTimeout), WithKind(errors.New("b"), KindLimit), WithKind(errors.New("c"), KindTimeout))
	if KindOf(sticky) != KindUnknown {
		t.Fatal("a conflict must stay sticky across later siblings")
	}
	if KindOf(fmt.Errorf("call: %w", context.DeadlineExceeded)) != KindTimeout {
		t.Fatal("an expired deadline is a timeout")
	}
	if KindOf(context.Canceled) != KindUnknown || KindOf(errors.New("plain")) != KindUnknown {
		t.Fatal("cancellation and plain errors are not classified")
	}
	if KindOf(WithKind(context.DeadlineExceeded, KindLimit)) != KindLimit {
		t.Fatal("an explicit kind wins over the deadline fallback")
	}
	if KindOf(errors.Join(context.DeadlineExceeded, WithKind(errors.New("a"), KindTimeout), WithKind(errors.New("b"), KindLimit))) != KindUnknown {
		t.Fatal("conflicts must not fall back to the deadline rule")
	}
}

func TestKindNames(t *testing.T) {
	for kind, want := range map[Kind]string{
		KindUnknown: "unknown", KindTimeout: "timeout", KindLimit: "limit",
		KindServiceUnavailable: "service_unavailable", KindBusiness: "business", KindCodec: "codec", kindConflict: "unknown",
	} {
		if kind.String() != want {
			t.Fatalf("%d: got %q want %q", kind, kind.String(), want)
		}
	}
}

func TestObservationRecordKind(t *testing.T) {
	ctx, o := WithObservation(context.Background())
	RecordKind(ctx, KindUnknown)
	RecordKind(ctx, KindBusiness)
	RecordKind(ctx, KindLimit)
	got := make(chan Completion, 1)
	o.OnComplete(func(c Completion) { got <- c })
	o.FinishInvoke(errors.New("biz"), true)
	if c := <-got; c.Kind != KindBusiness {
		t.Fatalf("first recorded kind must win, got %v", c.Kind)
	}
	o.RecordKind(KindTimeout)
	if o.kind != KindBusiness {
		t.Fatal("a frozen observation must ignore late evidence")
	}
	RecordKind(context.Background(), KindBusiness) // no observation: no-op
	var absent *Observation
	absent.RecordKind(KindBusiness)
}

func TestObservationRecordKindAsyncCompletion(t *testing.T) {
	ctx, o := WithObservation(context.Background())
	got := make(chan Completion, 1)
	o.OnComplete(func(c Completion) { got <- c })
	o.BeginAsync(context.Background(), time.Minute)
	RecordKind(ctx, KindTimeout)
	o.Complete(errors.New("late"))
	o.FinishInvoke(nil, true)
	if c := <-got; c.Kind != KindTimeout || !c.Known {
		t.Fatalf("callback completion lost the kind: %+v", c)
	}
}

func TestRecordKindForRequiresOwnInvocation(t *testing.T) {
	type invocation struct{ name string }
	own, other := &invocation{"own"}, &invocation{"other"}

	// A provider (unbound) observation inherited by a downstream call is not relabeled.
	provider, p := WithObservation(context.Background())
	RecordKindFor(provider, own, KindLimit)
	if p.kind != KindUnknown {
		t.Fatal("unbound observation must not accept consumer evidence")
	}

	consumer, c := WithInvocationObservation(context.Background(), own)
	RecordKindFor(consumer, other, KindLimit)
	RecordKindFor(consumer, *own, KindLimit)
	RecordKindFor(consumer, nil, KindLimit)
	if c.kind != KindUnknown {
		t.Fatal("only the bound invocation may record a kind")
	}
	RecordKindFor(consumer, own, KindLimit)
	if c.kind != KindLimit {
		t.Fatal("the bound invocation must record its kind")
	}

	claimed, cl := WithInvocationObservation(context.Background(), own)
	ClaimObservation(claimed, own)
	RecordKindFor(claimed, own, KindLimit)
	if cl.kind != KindUnknown {
		t.Fatal("a claimed observation is owned by remoting")
	}
	RecordKindFor(context.Background(), own, KindLimit)
}
