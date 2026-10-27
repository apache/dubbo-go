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
)

// Observation belongs to one local RPC attempt. It never completes a business
// request, closes a response channel, or invokes a user's callback.
type Observation struct {
	mu      sync.Mutex
	session any
	origin  Origin
	frozen  bool
}

type observationKey struct{}

func WithObservation(ctx context.Context) (context.Context, *Observation) {
	o := &Observation{}
	return context.WithValue(ctx, observationKey{}, o), o
}
func ObservationFromContext(ctx context.Context) *Observation {
	if ctx == nil {
		return nil
	}
	o, _ := ctx.Value(observationKey{}).(*Observation)
	return o
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
	if o.frozen || o.session == nil || o.session != session {
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
	o.frozen = true
	o.session = nil
	if o.origin == Codec || o.origin == Network {
		return o.origin
	}
	return Unknown
}
