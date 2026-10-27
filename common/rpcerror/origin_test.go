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
)

func TestOrigins(t *testing.T) {
	var typedNil *net.OpError
	if Of(typedNil) != Unknown || FromNetworkOperation(typedNil) != typedNil || Mark(typedNil, Codec) != typedNil {
		t.Fatal("typed nil must be safe")
	}
	cause := &net.OpError{Op: "write", Net: "tcp", Err: errors.New("fixture")}
	marked := Mark(cause, Codec)
	wrapped := fmt.Errorf("outer: %w", marked)
	var got *net.OpError
	if !errors.Is(wrapped, cause) || !errors.As(wrapped, &got) || got != cause || marked.Error() != cause.Error() || Of(wrapped) != Codec {
		t.Fatal("origin wrapping changed cause or message")
	}
	if FromNetworkOperation(marked) != marked || Mark(nil, Codec) != nil {
		t.Fatal("must preserve codec markers and nil")
	}
	if Of(FromNetworkOperation(cause)) != Network || Of(cause) != Unknown {
		t.Fatal("network detection must be explicit at a transport boundary")
	}
	for _, err := range []error{context.Canceled, context.DeadlineExceeded, errors.New("internal"), &net.OpError{Err: context.DeadlineExceeded}} {
		if FromNetworkOperation(err) != err {
			t.Fatal("changed unconfirmed/context error")
		}
	}
	if Of(errors.Join(Mark(errors.New("a"), Codec), Mark(errors.New("b"), Network))) != Unknown {
		t.Fatal("conflicting origins must remain unknown")
	}
}
