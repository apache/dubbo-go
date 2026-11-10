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
	"errors"
	"testing"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/rpcerror"
)

func TestObserveResponseDecodeFailureKeepsBusinessWaitPending(t *testing.T) {
	id := SequenceID()
	cause := rpcerror.Mark(errors.New("response body"), rpcerror.Codec)
	failure := &ResponseDecodeError{RequestID: id, Cause: cause}
	if failure.Error() != cause.Error() || !errors.Is(failure, cause) {
		t.Fatal("decode metadata changed error text or cause")
	}
	session, other := new(int), new(int)
	if ObserveResponseDecodeFailure(id, session, failure) {
		t.Fatal("missing pending request accepted evidence")
	}
	p := NewPendingResponse(id)
	AddPendingResponse(p)
	t.Cleanup(func() { RemovePendingResponse(SequenceType(id)) })
	if ObserveResponseDecodeFailure(id, session, failure) {
		t.Fatal("unmonitored pending request accepted evidence")
	}
	_, o := rpcerror.WithObservation(context.Background())
	p.Observation = o
	o.BindSession(session)
	if ObserveResponseDecodeFailure(id, other, failure) || !ObserveResponseDecodeFailure(id, session, failure) {
		t.Fatal("evidence was not restricted to the sending session")
	}
	if GetPendingResponse(SequenceType(id)) != p {
		t.Fatal("monitoring removed the business pending request")
	}
	select {
	case <-p.Done:
		t.Fatal("monitoring completed the business waiter")
	default:
	}
	if o.Freeze() != rpcerror.Codec || ObserveResponseDecodeFailure(id, session, failure) {
		t.Fatal("late evidence changed the frozen observation")
	}
}
