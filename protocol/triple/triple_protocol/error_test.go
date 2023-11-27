// Copyright 2021-2023 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package triple_protocol

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

import (
	"google.golang.org/protobuf/proto"

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
