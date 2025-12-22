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
	"strconv"
	"strings"
	"testing"
)

import (
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/assert"
)

func TestCode(t *testing.T) {
	t.Parallel()
	var valid []Code
	for code := minCode; code <= maxCode; code++ {
		valid = append(valid, code)
	}
	// Ensures that we don't forget to update the mapping in the Stringer
	// implementation.
	for _, code := range valid {
		assert.False(
			t,
			strings.HasPrefix(code.String(), "code_"),
			assert.Sprintf("update Code.String() method for new code %v", code),
		)
		assertCodeRoundTrips(t, code)
	}
	assertCodeRoundTrips(t, Code(999))
}

func assertCodeRoundTrips(tb testing.TB, code Code) {
	tb.Helper()
	encoded, err := code.MarshalText()
	assert.Nil(tb, err)
	var decoded Code
	assert.Nil(tb, decoded.UnmarshalText(encoded))
	assert.Equal(tb, decoded, code)
	if code >= minCode && code <= maxCode {
		var invalid Code
		// For the known codes, we only accept the canonical string representation: "canceled", not "code_1".
		assert.NotNil(tb, invalid.UnmarshalText([]byte("code_"+strconv.Itoa(int(code)))))
	}
}

func TestCodeString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		code Code
		want string
	}{
		{CodeCanceled, "canceled"},
		{CodeUnknown, "unknown"},
		{CodeInvalidArgument, "invalid_argument"},
		{CodeDeadlineExceeded, "deadline_exceeded"},
		{CodeNotFound, "not_found"},
		{CodeAlreadyExists, "already_exists"},
		{CodePermissionDenied, "permission_denied"},
		{CodeResourceExhausted, "resource_exhausted"},
		{CodeFailedPrecondition, "failed_precondition"},
		{CodeAborted, "aborted"},
		{CodeOutOfRange, "out_of_range"},
		{CodeUnimplemented, "unimplemented"},
		{CodeInternal, "internal"},
		{CodeUnavailable, "unavailable"},
		{CodeDataLoss, "data_loss"},
		{CodeUnauthenticated, "unauthenticated"},
		// Unknown codes should return "code_N" format
		{Code(0), "code_0"},
		{Code(100), "code_100"},
		{Code(999), "code_999"},
		{CodeBizError, "code_17"}, // CodeBizError is not in the switch
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.code.String(), tt.want)
		})
	}
}

func TestCodeMarshalText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		code Code
		want string
	}{
		{CodeCanceled, "canceled"},
		{CodeUnknown, "unknown"},
		{CodeInternal, "internal"},
		{Code(999), "code_999"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			data, err := tt.code.MarshalText()
			assert.Nil(t, err)
			assert.Equal(t, string(data), tt.want)
		})
	}
}

func TestCodeUnmarshalText(t *testing.T) {
	t.Parallel()

	t.Run("valid codes", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			input string
			want  Code
		}{
			{"canceled", CodeCanceled},
			{"unknown", CodeUnknown},
			{"invalid_argument", CodeInvalidArgument},
			{"deadline_exceeded", CodeDeadlineExceeded},
			{"not_found", CodeNotFound},
			{"already_exists", CodeAlreadyExists},
			{"permission_denied", CodePermissionDenied},
			{"resource_exhausted", CodeResourceExhausted},
			{"failed_precondition", CodeFailedPrecondition},
			{"aborted", CodeAborted},
			{"out_of_range", CodeOutOfRange},
			{"unimplemented", CodeUnimplemented},
			{"internal", CodeInternal},
			{"unavailable", CodeUnavailable},
			{"data_loss", CodeDataLoss},
			{"unauthenticated", CodeUnauthenticated},
		}

		for _, tt := range tests {
			t.Run(tt.input, func(t *testing.T) {
				t.Parallel()
				var code Code
				err := code.UnmarshalText([]byte(tt.input))
				assert.Nil(t, err)
				assert.Equal(t, code, tt.want)
			})
		}
	})

	t.Run("non-canonical codes", func(t *testing.T) {
		t.Parallel()
		// Non-canonical codes outside the valid range should work
		var code Code
		err := code.UnmarshalText([]byte("code_999"))
		assert.Nil(t, err)
		assert.Equal(t, code, Code(999))

		// code_0 is outside minCode-maxCode range
		var code0 Code
		err = code0.UnmarshalText([]byte("code_0"))
		assert.Nil(t, err)
		assert.Equal(t, code0, Code(0))
	})

	t.Run("invalid codes", func(t *testing.T) {
		t.Parallel()
		invalidInputs := []string{
			"invalid",
			"CANCELED", // case sensitive
			"code_",    // missing number
			"code_abc", // not a number
			"code_1",   // canonical codes can't use code_N format
			"code_16",  // canonical codes can't use code_N format
			"",         // empty string
			"code_-1",  // negative number
		}

		for _, input := range invalidInputs {
			t.Run(fmt.Sprintf("input_%s", input), func(t *testing.T) {
				t.Parallel()
				var code Code
				err := code.UnmarshalText([]byte(input))
				assert.NotNil(t, err)
			})
		}
	})
}

func TestCodeOf_Extended(t *testing.T) {
	t.Parallel()

	t.Run("with triple error", func(t *testing.T) {
		t.Parallel()
		tripleErr := NewError(CodeNotFound, errors.New("not found"))
		assert.Equal(t, CodeOf(tripleErr), CodeNotFound)
	})

	t.Run("with wrapped triple error", func(t *testing.T) {
		t.Parallel()
		tripleErr := NewError(CodePermissionDenied, errors.New("permission denied"))
		wrappedErr := fmt.Errorf("wrapped: %w", tripleErr)
		assert.Equal(t, CodeOf(wrappedErr), CodePermissionDenied)
	})

	t.Run("with non-triple error", func(t *testing.T) {
		t.Parallel()
		err := errors.New("regular error")
		assert.Equal(t, CodeOf(err), CodeUnknown)
	})

	t.Run("with nil error", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, CodeOf(nil), CodeUnknown)
	})
}

func TestCodeConstants(t *testing.T) {
	t.Parallel()

	// Verify code values match gRPC specification
	assert.Equal(t, uint32(CodeCanceled), uint32(1))
	assert.Equal(t, uint32(CodeUnknown), uint32(2))
	assert.Equal(t, uint32(CodeInvalidArgument), uint32(3))
	assert.Equal(t, uint32(CodeDeadlineExceeded), uint32(4))
	assert.Equal(t, uint32(CodeNotFound), uint32(5))
	assert.Equal(t, uint32(CodeAlreadyExists), uint32(6))
	assert.Equal(t, uint32(CodePermissionDenied), uint32(7))
	assert.Equal(t, uint32(CodeResourceExhausted), uint32(8))
	assert.Equal(t, uint32(CodeFailedPrecondition), uint32(9))
	assert.Equal(t, uint32(CodeAborted), uint32(10))
	assert.Equal(t, uint32(CodeOutOfRange), uint32(11))
	assert.Equal(t, uint32(CodeUnimplemented), uint32(12))
	assert.Equal(t, uint32(CodeInternal), uint32(13))
	assert.Equal(t, uint32(CodeUnavailable), uint32(14))
	assert.Equal(t, uint32(CodeDataLoss), uint32(15))
	assert.Equal(t, uint32(CodeUnauthenticated), uint32(16))
	assert.Equal(t, uint32(CodeBizError), uint32(17))

	// Verify min/max bounds
	assert.Equal(t, minCode, CodeCanceled)
	assert.Equal(t, maxCode, CodeUnauthenticated)
}
