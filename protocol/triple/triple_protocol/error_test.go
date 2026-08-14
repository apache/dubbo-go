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

package triple_protocol

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

import (
	"google.golang.org/protobuf/proto"

	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
)

import (
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/assert"
)

func TestErrorNilUnderlying(t *testing.T) {
	t.Parallel()
	err := NewError(CodeUnknown, nil)
	assert.NotNil(t, err)
	assert.Equal(t, err.Error(), CodeUnknown.String())
	assert.Equal(t, err.Code(), CodeUnknown)
	assert.Zero(t, err.Details())
	detail, detailErr := NewErrorDetail(&emptypb.Empty{})
	assert.Nil(t, detailErr)
	err.AddDetail(detail)
	assert.Equal(t, len(err.Details()), 1)
	assert.Equal(t, err.Details()[0].Type(), "google.protobuf.Empty")
	err.Meta().Set("foo", "bar")
	assert.Equal(t, err.Meta().Get("foo"), "bar")
	assert.Equal(t, CodeOf(err), CodeUnknown)
}

func TestErrorFormatting(t *testing.T) {
	t.Parallel()
	assert.Equal(
		t,
		NewError(CodeUnavailable, errors.New("")).Error(),
		CodeUnavailable.String(),
	)
	got := NewError(CodeUnavailable, errors.New("foo")).Error()
	assert.True(t, strings.Contains(got, CodeUnavailable.String()))
	assert.True(t, strings.Contains(got, "foo"))
}

func TestErrorCode(t *testing.T) {
	t.Parallel()
	err := fmt.Errorf(
		"another: %w",
		NewError(CodeUnavailable, errors.New("foo")),
	)
	tripleErr, ok := asError(err)
	assert.True(t, ok)
	assert.Equal(t, tripleErr.Code(), CodeUnavailable)
}

func TestCodeOf(t *testing.T) {
	t.Parallel()
	assert.Equal(
		t,
		CodeOf(NewError(CodeUnavailable, errors.New("foo"))),
		CodeUnavailable,
	)
	assert.Equal(t, CodeOf(errors.New("foo")), CodeUnknown)
}

func TestErrorDetails(t *testing.T) {
	t.Parallel()
	second := durationpb.New(time.Second)
	detail, err := NewErrorDetail(second)
	assert.Nil(t, err)
	tripleErr := NewError(CodeUnknown, errors.New("error with details"))
	assert.Zero(t, tripleErr.Details())
	tripleErr.AddDetail(detail)
	assert.Equal(t, len(tripleErr.Details()), 1)
	unmarshaled, err := tripleErr.Details()[0].Value()
	assert.Nil(t, err)
	assert.Equal(t, unmarshaled, proto.Message(second))
	secondBin, err := proto.Marshal(second)
	assert.Nil(t, err)
	assert.Equal(t, detail.Bytes(), secondBin)
}

func TestErrorIs(t *testing.T) {
	t.Parallel()
	// errors.New and fmt.Errorf return *errors.errorString. errors.Is
	// considers two *errors.errorStrings equal iff they have the same address.
	err := errors.New("oh no")
	assert.False(t, errors.Is(err, errors.New("oh no")))
	assert.True(t, errors.Is(err, err))
	// Our errors should have the same semantics. Note that we'd need to extend
	// the ErrorDetail interface to support value equality.
	tripleErr := NewError(CodeUnavailable, err)
	assert.False(t, errors.Is(tripleErr, NewError(CodeUnavailable, err)))
	assert.True(t, errors.Is(tripleErr, tripleErr))
}

// TestNewWireError verifies that IsWireError returns true for wire errors
// created by NewWireError (including when wrapped) and false for regular
// Triple errors and non-Triple errors.
func TestNewWireError(t *testing.T) {
	t.Parallel()
	// Wire error is detected by IsWireError
	wireErr := NewWireError(CodeUnavailable, errors.New("server down"))
	assert.True(t, IsWireError(wireErr))
	// Regular error is not a wire error
	plainErr := NewError(CodeUnavailable, errors.New("client issue"))
	assert.False(t, IsWireError(plainErr))
	// Non-triple error is not a wire error
	assert.False(t, IsWireError(errors.New("not triple")))
	// Wrapped wire error is still detected
	wrapped := fmt.Errorf("wrapped: %w", wireErr)
	assert.True(t, IsWireError(wrapped))
}

// TestNewErrorDetailWithAny verifies that NewErrorDetail uses an *anypb.Any
// directly without wrapping it into another Any.
func TestNewErrorDetailWithAny(t *testing.T) {
	t.Parallel()
	// When msg is already an *anypb.Any, it should be used directly
	// without wrapping into another Any.
	anyMsg := &anypb.Any{
		TypeUrl: "type.googleapis.com/google.protobuf.Empty",
		Value:   []byte{},
	}
	detail, err := NewErrorDetail(anyMsg)
	assert.Nil(t, err)
	assert.Equal(t, detail.Type(), "google.protobuf.Empty")
}

// TestErrorDetailBytesReturnsCopy verifies that ErrorDetail.Bytes returns a
// copy of the serialized detail and that mutating it does not affect the
// internal state.
func TestErrorDetailBytesReturnsCopy(t *testing.T) {
	t.Parallel()
	detail, err := NewErrorDetail(durationpb.New(time.Second))
	assert.Nil(t, err)
	first := detail.Bytes()
	assert.NotZero(t, first)
	// Mutate the returned slice; internal state should be unaffected
	first[0] = ^first[0]
	second := detail.Bytes()
	assert.NotEqual(t, first, second)
}

// TestWrapIfContextError verifies that wrapIfContextError wraps
// context.Canceled and context.DeadlineExceeded with the corresponding
// Triple codes, leaves already-coded and plain errors unchanged, and
// returns nil for nil input.
func TestWrapIfContextError(t *testing.T) {
	t.Parallel()
	// context.Canceled -> CodeCanceled
	err := wrapIfContextError(context.Canceled)
	tripleErr, ok := asError(err)
	assert.True(t, ok)
	assert.Equal(t, tripleErr.Code(), CodeCanceled)
	assert.ErrorIs(t, err, context.Canceled)
	// context.DeadlineExceeded -> CodeDeadlineExceeded
	err = wrapIfContextError(context.DeadlineExceeded)
	tripleErr, ok = asError(err)
	assert.True(t, ok)
	assert.Equal(t, tripleErr.Code(), CodeDeadlineExceeded)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	// Already coded error is returned unchanged (same instance)
	coded := NewError(CodeNotFound, errors.New("not found"))
	assert.True(t, wrapIfContextError(coded) == coded)
	// Plain error is returned as-is (not coded)
	plainErr := errors.New("plain")
	assert.True(t, wrapIfContextError(plainErr) == plainErr)
	// nil
	assert.Nil(t, wrapIfContextError(nil))
}

// TestWrapIfUncoded verifies that wrapIfUncoded returns nil for nil,
// preserves already-coded errors, wraps context.Canceled via
// wrapIfContextError, and wraps plain errors with CodeUnknown.
func TestWrapIfUncoded(t *testing.T) {
	t.Parallel()
	// nil returns nil
	assert.Nil(t, wrapIfUncoded(nil))
	// Already coded error is returned unchanged (same instance)
	coded := NewError(CodeNotFound, errors.New("not found"))
	err := wrapIfUncoded(coded)
	assert.True(t, err == coded)
	// context.Canceled gets wrapped with CodeCanceled
	err = wrapIfUncoded(context.Canceled)
	result, ok := asError(err)
	assert.True(t, ok)
	assert.Equal(t, result.Code(), CodeCanceled)
	// Plain error gets wrapped with CodeUnknown
	err = wrapIfUncoded(errors.New("plain"))
	result, ok = asError(err)
	assert.True(t, ok)
	assert.Equal(t, result.Code(), CodeUnknown)
}
