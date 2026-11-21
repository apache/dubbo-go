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
	"fmt"
	"testing"
)

type statusError struct {
	code uint32
	err  error
}

func (s statusError) Error() string         { return s.err.Error() }
func (s statusError) Unwrap() error         { return s.err }
func (s statusError) RPCTripleCode() uint32 { return s.code }

func TestClassificationContract(t *testing.T) {
	plain := errors.New("network codec timeout No provider available")
	tests := []struct {
		name string
		err  error
		want Category
	}{
		{"no text guessing", plain, CategoryUnknown},
		{"deadline", context.DeadlineExceeded, CategoryTimeout},
		{"cancel", context.Canceled, CategoryUnknown},
		{"status timeout wins", statusError{4, Mark(plain, Codec)}, CategoryTimeout},
		{"status limit wins", statusError{8, Mark(plain, Codec)}, CategoryLimit},
		{"permission", statusError{7, WithKind(plain, KindBusiness)}, CategoryServiceUnavailable},
		{"business", statusError{17, Mark(plain, Network)}, CategoryBusiness},
		{"origin wins", Mark(WithKind(plain, KindTimeout), Codec), CategoryCodec},
		{"origin over unavailable", statusError{14, Mark(plain, Network)}, CategoryNetwork},
		{"kind over unavailable", statusError{14, WithKind(plain, KindBusiness)}, CategoryBusiness},
		{"unavailable", statusError{14, plain}, CategoryServiceUnavailable},
		{"internal", statusError{13, plain}, CategoryUnknown},
		{"origin conflict sticky", errors.Join(Mark(plain, Network), Mark(plain, Codec), Mark(plain, Network)), CategoryUnknown},
		{"kind conflict sticky", errors.Join(WithKind(plain, KindLimit), WithKind(plain, KindBusiness), context.DeadlineExceeded), CategoryUnknown},
		{"status conflict sticky", errors.Join(statusError{4, plain}, statusError{8, plain}, statusError{4, plain}), CategoryUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Classify(fmt.Errorf("wrapped: %w", tt.err)); got.Category != tt.want {
				t.Fatalf("got %+v want %v", got, tt.want)
			}
		})
	}
}
