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
	"time"
)

// Completion is a monitoring terminal state, not an RPC result or user callback.
// Known=false means observation expired without proving business success/failure.
type Completion struct {
	TraceID string
	Err     error
	Known   bool
	Origin  Origin
	// Kind is request-scoped category evidence recorded via RecordKind.
	Kind Kind
	At   time.Time
}

// OnComplete subscribes once, including after completion. Callbacks run outside
// the mutex and a failed monitoring subscriber cannot affect business callbacks.
func (o *Observation) OnComplete(notify func(Completion)) {
	if o == nil || notify == nil {
		return
	}
	o.mu.Lock()
	if o.completed != nil {
		c := *o.completed
		o.mu.Unlock()
		callObserver(notify, c)
		return
	}
	if !o.frozen {
		o.notify = append(o.notify, notify)
	}
	o.mu.Unlock()
}

func callObserver(notify func(Completion), c Completion) {
	defer func() { _ = recover() }()
	notify(c)
}

func (o *Observation) publishLocked(c Completion) func(Completion) {
	o.completed = &c
	return o.finishLocked()
}

// BeginAsync hands final observation to a callback-based request. Expiry only
// releases monitoring resources: it neither deletes PendingResponse nor calls
// the user callback. It is installed before the request is sent.
func (o *Observation) BeginAsync(ctx context.Context, timeout time.Duration) {
	if o == nil {
		return
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.frozen || o.async {
		return
	}
	o.async = true
	if ctx == nil {
		ctx = context.Background()
	}
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	o.timer = time.AfterFunc(timeout, func() { o.complete(Completion{At: time.Now()}) })
	o.stopContext = context.AfterFunc(ctx, func() { o.complete(Completion{At: time.Now()}) })
}

// Complete observes a real response, without modifying the response itself.
func (o *Observation) Complete(err error) {
	if isNil(err) {
		err = nil
	}
	if o != nil {
		o.complete(Completion{Err: err, Known: true, At: time.Now()})
	}
}

func (o *Observation) complete(c Completion) {
	o.mu.Lock()
	if o.frozen || o.pending != nil {
		o.mu.Unlock()
		return
	}
	c.Origin = o.origin
	c.Kind = o.kind
	c.TraceID = o.traceID
	if c.Origin != Codec && c.Origin != Network {
		c.Origin = Unknown
	}
	if !o.returned {
		o.pending = &c
		o.mu.Unlock()
		return
	}
	notify := o.publishLocked(c)
	o.mu.Unlock()
	if notify != nil {
		notify(c)
	}
}

// FinishInvoke resolves the send/result versus fast-callback race. A synchronous
// send error wins over a buffered response for metrics, matching Invoke's return.
// A successful async submission is not itself a successful completed RPC.
func (o *Observation) FinishInvoke(err error, hasResult bool) {
	if isNil(err) {
		err = nil
	}
	o.mu.Lock()
	if o.frozen {
		o.mu.Unlock()
		return
	}
	o.returned = true
	if o.async && hasResult && err == nil {
		if o.pending == nil {
			o.mu.Unlock()
			return
		}
		c := *o.pending
		notify := o.publishLocked(c)
		o.mu.Unlock()
		if notify != nil {
			notify(c)
		}
		return
	}
	c := Completion{Err: err, Known: hasResult, Origin: o.origin, Kind: o.kind, TraceID: o.traceID, At: time.Now()}
	if c.Origin != Codec && c.Origin != Network {
		c.Origin = Unknown
	}
	notify := o.publishLocked(c)
	o.mu.Unlock()
	if notify != nil {
		notify(c)
	}
}

func (o *Observation) finishLocked() func(Completion) {
	o.frozen = true
	o.invocation = nil
	o.identity = nil
	o.session = nil
	o.pending = nil
	if o.timer != nil {
		o.timer.Stop()
		o.timer = nil
	}
	if o.stopContext != nil {
		o.stopContext()
		o.stopContext = nil
	}
	observers := o.notify
	o.notify = nil
	return func(c Completion) {
		for _, notify := range observers {
			callObserver(notify, c)
		}
	}
}
