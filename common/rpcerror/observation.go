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
	"reflect"
	"sync"
	"time"
)

// Observation belongs to one local RPC attempt. It never completes a business
// request, closes a response channel, or invokes a user's callback.
type Observation struct {
	mu          sync.Mutex
	invocation  any
	session     any
	origin      Origin
	kind        Kind
	frozen      bool
	claimed     bool
	async       bool
	returned    bool
	pending     *Completion
	notify      []func(Completion)
	completed   *Completion
	identity    any
	traceID     string
	timer       *time.Timer
	stopContext func() bool
}

type observationKey struct{}

func WithObservation(ctx context.Context) (context.Context, *Observation) {
	if ctx == nil {
		ctx = context.Background()
	}
	o := &Observation{}
	return context.WithValue(ctx, observationKey{}, o), o
}

// WithInvocationObservation binds a consumer observation to its invocation.
// Inherited contexts alone do not authorize another RPC to use the observation.
// Invocation identity must be a non-nil pointer, not an equal value copy.
func WithInvocationObservation(ctx context.Context, invocation any) (context.Context, *Observation) {
	ctx, o := WithObservation(ctx)
	if validInvocationIdentity(invocation) {
		o.invocation = invocation
	}
	return ctx, o
}

func ObservationFromContext(ctx context.Context) *Observation {
	if ctx == nil {
		return nil
	}
	o, _ := ctx.Value(observationKey{}).(*Observation)
	return o
}

// ClaimObservation transfers monitoring to the matching remoting request once.
// The invocation reference is released on claim, so callback observations do not
// retain the mutable invocation. Unbound/provider observations cannot be claimed.
func ClaimObservation(ctx context.Context, invocation any) *Observation {
	o := ObservationFromContext(ctx)
	if o == nil || !validInvocationIdentity(invocation) {
		return nil
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.frozen || o.invocation == nil || o.invocation != invocation {
		return nil
	}
	o.claimed = true
	o.invocation = nil
	o.identity = nil
	return o
}

func validInvocationIdentity(invocation any) bool {
	v := reflect.ValueOf(invocation)
	return v.IsValid() && v.Kind() == reflect.Pointer && !v.IsNil()
}

// BindSession runs before sending, including before a very fast response.
func (o *Observation) BindSession(session any) {
	if o == nil || session == nil || !reflect.TypeOf(session).Comparable() {
		return
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if !o.frozen {
		o.session = session
	}
}

// Record requires both a confirmed source and the exact session used to send.
func (o *Observation) Record(session any, err error) bool {
	if o == nil || session == nil || !reflect.TypeOf(session).Comparable() {
		return false
	}
	origin := Of(err)
	if origin == Unknown {
		return false
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.frozen || o.pending != nil || o.session == nil || o.session != session {
		return false
	}
	if o.origin == Unknown {
		o.origin = origin
	} else if o.origin != origin {
		o.origin = Origin(255)
	}
	return true
}

// Freeze is the linearization point for the monitoring snapshot. It drops the
// session reference; late I/O cannot change an event already sent to metrics.
func (o *Observation) Freeze() Origin {
	if o == nil {
		return Unknown
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	o.finishLocked()
	if o.origin == Codec || o.origin == Network {
		return o.origin
	}
	return Unknown
}

// RecordKind records a failure category confirmed by the synchronous invoker
// that owns this observation's context, without changing the error returned to
// the caller. Use it where wrapping is unsafe, e.g. a provider application's
// error that protocols re-encode by its concrete type. The first recorded kind
// wins and a frozen observation ignores late evidence. A kind declared on the
// error itself (see WithKind) still takes precedence during classification.
func (o *Observation) RecordKind(kind Kind) {
	if o == nil || !kind.valid() {
		return
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if !o.frozen && o.pending == nil && o.kind == KindUnknown {
		o.kind = kind
	}
}

// RecordKind records kind on the observation carried by ctx, if any.
func RecordKind(ctx context.Context, kind Kind) {
	ObservationFromContext(ctx).RecordKind(kind)
}

// RecordKindFor is RecordKind for consumer-side code such as filters: it
// records only on an observation bound to this exact, not yet claimed
// invocation, so a context inherited from an enclosing request (for example a
// provider handler issuing a downstream call) is never relabeled.
func RecordKindFor(ctx context.Context, invocation any, kind Kind) {
	o := ObservationFromContext(ctx)
	if o == nil || !kind.valid() || !validInvocationIdentity(invocation) {
		return
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if !o.frozen && o.pending == nil && o.invocation != nil && o.invocation == invocation && o.kind == KindUnknown {
		o.kind = kind
	}
}

// EnsureObservation shares evidence between filters on the same invocation.
// The creator alone calls FinishInvoke after all inner filters return.
func EnsureObservation(ctx context.Context, invocation any, consumer bool) (context.Context, *Observation, bool) {
	if ctx == nil {
		ctx = context.Background()
	}
	if o := ObservationFromContext(ctx); o != nil && validInvocationIdentity(invocation) {
		o.mu.Lock()
		reuse := !o.frozen && o.identity == invocation
		if reuse && consumer && !o.claimed && o.invocation == nil {
			o.invocation = invocation
		}
		o.mu.Unlock()
		if reuse {
			return ctx, o, false
		}
	}
	var o *Observation
	if consumer {
		ctx, o = WithInvocationObservation(ctx, invocation)
	} else {
		ctx, o = WithObservation(ctx)
	}
	o.identity = invocation
	return ctx, o, true
}

type spanOwnerKey struct{}

// WithSpanOwner gives RPC instrumentation exclusive error-event ownership of
// this span. Context loggers may still record on unrelated child spans.
func WithSpanOwner(ctx context.Context, spanID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, spanOwnerKey{}, spanID)
}
func OwnsSpan(ctx context.Context, spanID string) bool {
	if ctx == nil {
		return false
	}
	id, _ := ctx.Value(spanOwnerKey{}).(string)
	return id != "" && id == spanID
}

// RecordTraceID shares a valid trace ID with observers outside the trace filter.
func (o *Observation) RecordTraceID(id string) {
	if o == nil {
		return
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if !o.frozen && o.pending == nil {
		o.traceID = id
	}
}

// RebindObservation transfers the identity when a generic filter translates an
// invocation. Unrelated, claimed and frozen observations cannot be rebound.
func RebindObservation(ctx context.Context, oldInvocation, newInvocation any) {
	o := ObservationFromContext(ctx)
	if o == nil || !validInvocationIdentity(oldInvocation) || !validInvocationIdentity(newInvocation) {
		return
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.frozen || o.claimed || o.identity != oldInvocation {
		return
	}
	o.identity = newInvocation
	if o.invocation == oldInvocation {
		o.invocation = newInvocation
	}
}
