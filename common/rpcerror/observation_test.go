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
	"testing"
)

func TestObservationIsolationAndFreeze(t *testing.T) {
	ctx, a := WithObservation(context.Background())
	_, b := WithObservation(ctx)
	s1, s2 := new(int), new(int)
	a.BindSession(s1)
	b.BindSession(s1)
	err := Mark(errors.New("codec"), Codec)
	if ObservationFromContext(ctx) != a || a.Record(s2, err) || !a.Record(s1, err) {
		t.Fatal("session identity not enforced")
	}
	if a.Freeze() != Codec || b.Freeze() != Unknown || a.Record(s1, err) {
		t.Fatal("isolation or freeze failed")
	}
}
func TestObservationRace(t *testing.T) {
	for range 100 {
		_, o := WithObservation(context.Background())
		session := new(int)
		o.BindSession(session)
		var wg sync.WaitGroup
		wg.Go(func() { o.Record(session, Mark(errors.New("codec"), Codec)) })
		got := o.Freeze()
		wg.Wait()
		if got != Unknown && got != Codec {
			t.Fatal(got)
		}
		if o.Freeze() != got {
			t.Fatal("snapshot changed")
		}
	}
}

func TestObservationRejectsUnconfirmedEvidence(t *testing.T) {
	_, o := WithObservation(context.Background())
	defer o.Freeze()
	session := new(int)
	cause := Mark(errors.New("codec"), Codec)
	var absent *Observation
	absent.BindSession(session)
	if absent.Record(session, cause) || absent.Freeze() != Unknown {
		t.Fatal("nil observation accepted evidence")
	}
	for _, invalid := range []any{nil, []int{1}, map[string]int{"id": 1}} {
		o.BindSession(invalid)
		if o.session != nil || o.Record(invalid, cause) {
			t.Fatal("invalid session accepted")
		}
	}
	o.BindSession(session)
	if o.Record(session, errors.New("unmarked")) {
		t.Fatal("unconfirmed origin accepted")
	}
	if !o.Record(session, cause) || !o.Record(session, Mark(errors.New("network"), Network)) {
		t.Fatal("confirmed evidence not recorded")
	}
	if o.Freeze() != Unknown {
		t.Fatal("conflicting evidence must not choose an origin")
	}
	o.BindSession(session)
	if o.session != nil || o.Record(session, cause) {
		t.Fatal("frozen observation accepted a session or evidence")
	}
}
