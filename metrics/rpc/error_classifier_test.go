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

package rpc

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
)

func TestClassifyError_Timeout(t *testing.T) {
	err := triple_protocol.NewError(triple_protocol.CodeDeadlineExceeded, errors.New("timeout"))
	errType := classifyError(err)
	assert.Equal(t, ErrorTypeTimeout, errType)
}

func TestClassifyError_Limit(t *testing.T) {
	err := triple_protocol.NewError(triple_protocol.CodeResourceExhausted, errors.New("limit exceeded"))
	errType := classifyError(err)
	assert.Equal(t, ErrorTypeLimit, errType)
}

func TestClassifyError_ServiceUnavailable(t *testing.T) {
	tests := []struct {
		name string
		code triple_protocol.Code
	}{
		{
			name: "CodeUnavailable",
			code: triple_protocol.CodeUnavailable,
		},
		{
			name: "CodePermissionDenied",
			code: triple_protocol.CodePermissionDenied,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := triple_protocol.NewError(tt.code, errors.New("service unavailable"))
			errType := classifyError(err)
			assert.Equal(t, ErrorTypeServiceUnavailable, errType)
		})
	}
}

func TestClassifyError_BusinessFailed(t *testing.T) {
	err := triple_protocol.NewError(triple_protocol.CodeBizError, errors.New("business error"))
	errType := classifyError(err)
	assert.Equal(t, ErrorTypeBusinessFailed, errType)
}

func TestClassifyError_Unknown(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{
			name: "standard error",
			err:  errors.New("some error"),
		},
		{
			name: "CodeUnknown",
			err:  triple_protocol.NewError(triple_protocol.CodeUnknown, errors.New("unknown error")),
		},
		{
			name: "CodeCanceled",
			err:  triple_protocol.NewError(triple_protocol.CodeCanceled, errors.New("canceled")),
		},
		{
			name: "CodeInvalidArgument",
			err:  triple_protocol.NewError(triple_protocol.CodeInvalidArgument, errors.New("invalid argument")),
		},
		{
			name: "CodeNotFound",
			err:  triple_protocol.NewError(triple_protocol.CodeNotFound, errors.New("not found")),
		},
		{
			name: "CodeAlreadyExists",
			err:  triple_protocol.NewError(triple_protocol.CodeAlreadyExists, errors.New("already exists")),
		},
		{
			name: "CodeAborted",
			err:  triple_protocol.NewError(triple_protocol.CodeAborted, errors.New("aborted")),
		},
		{
			name: "CodeOutOfRange",
			err:  triple_protocol.NewError(triple_protocol.CodeOutOfRange, errors.New("out of range")),
		},
		{
			name: "CodeUnimplemented",
			err:  triple_protocol.NewError(triple_protocol.CodeUnimplemented, errors.New("unimplemented")),
		},
		{
			name: "CodeInternal",
			err:  triple_protocol.NewError(triple_protocol.CodeInternal, errors.New("internal error")),
		},
		{
			name: "CodeDataLoss",
			err:  triple_protocol.NewError(triple_protocol.CodeDataLoss, errors.New("data loss")),
		},
		{
			name: "CodeUnauthenticated",
			err:  triple_protocol.NewError(triple_protocol.CodeUnauthenticated, errors.New("unauthenticated")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errType := classifyError(tt.err)
			assert.Equal(t, ErrorTypeUnknown, errType)
		})
	}
}

func TestErrorType_Values(t *testing.T) {
	// Verify ErrorType constants have expected values
	assert.Equal(t, ErrorTypeUnknown, ErrorType(0))
	assert.Equal(t, ErrorTypeTimeout, ErrorType(1))
	assert.Equal(t, ErrorTypeLimit, ErrorType(2))
	assert.Equal(t, ErrorTypeServiceUnavailable, ErrorType(3))
	assert.Equal(t, ErrorTypeBusinessFailed, ErrorType(4))
	assert.Equal(t, ErrorTypeNetworkFailure, ErrorType(5))
	assert.Equal(t, ErrorTypeCodec, ErrorType(6))
}

func TestClassifyError_AllErrorTypesClassification(t *testing.T) {
	// Test that we can classify into all defined error types
	// (except ErrorTypeNetworkFailure and ErrorTypeCodec which are not yet used in classifyError)
	tests := []struct {
		name     string
		err      error
		expected ErrorType
	}{
		{
			name:     "timeout",
			err:      triple_protocol.NewError(triple_protocol.CodeDeadlineExceeded, errors.New("timeout")),
			expected: ErrorTypeTimeout,
		},
		{
			name:     "limit",
			err:      triple_protocol.NewError(triple_protocol.CodeResourceExhausted, errors.New("limit")),
			expected: ErrorTypeLimit,
		},
		{
			name:     "unavailable",
			err:      triple_protocol.NewError(triple_protocol.CodeUnavailable, errors.New("unavailable")),
			expected: ErrorTypeServiceUnavailable,
		},
		{
			name:     "permission denied",
			err:      triple_protocol.NewError(triple_protocol.CodePermissionDenied, errors.New("denied")),
			expected: ErrorTypeServiceUnavailable,
		},
		{
			name:     "business failed",
			err:      triple_protocol.NewError(triple_protocol.CodeBizError, errors.New("biz error")),
			expected: ErrorTypeBusinessFailed,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: ErrorTypeUnknown,
		},
		{
			name:     "unknown error",
			err:      errors.New("unknown"),
			expected: ErrorTypeUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errType := classifyError(tt.err)
			assert.Equal(t, tt.expected, errType)
		})
	}
}

// noUnwrapError is an error type that hides the wrapped error behind its
// Error() method and does not implement Unwrap, so errors.As cannot see
// through it.
type noUnwrapError struct {
	cause error
}

func (e *noUnwrapError) Error() string {
	return fmt.Sprintf("no unwrap: %v", e.cause)
}

func TestClassifyError_Nil(t *testing.T) {
	var typedNil *triple_protocol.Error
	tests := []struct {
		name string
		err  error
	}{
		{
			name: "untyped nil",
			err:  nil,
		},
		{
			name: "typed nil triple error",
			err:  typedNil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, ErrorTypeUnknown, classifyError(tt.err))
		})
	}
}

func TestClassifyError_Boundary(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want ErrorType
	}{
		{
			name: "empty standard error",
			err:  errors.New(""),
			want: ErrorTypeUnknown,
		},
		{
			name: "context deadline exceeded",
			err:  context.DeadlineExceeded,
			want: ErrorTypeUnknown,
		},
		{
			name: "context canceled",
			err:  context.Canceled,
			want: ErrorTypeUnknown,
		},
		{
			name: "plain error wrapping a triple error without Unwrap",
			err:  &noUnwrapError{cause: triple_protocol.NewError(triple_protocol.CodeDeadlineExceeded, errors.New("timeout"))},
			want: ErrorTypeUnknown,
		},
		{
			name: "CodeInternal not mapped to a dedicated type yet",
			err:  triple_protocol.NewError(triple_protocol.CodeInternal, errors.New("internal")),
			want: ErrorTypeUnknown,
		},
		{
			name: "CodeDataLoss",
			err:  triple_protocol.NewError(triple_protocol.CodeDataLoss, errors.New("data loss")),
			want: ErrorTypeUnknown,
		},
		{
			name: "CodeUnauthenticated",
			err:  triple_protocol.NewError(triple_protocol.CodeUnauthenticated, errors.New("unauthenticated")),
			want: ErrorTypeUnknown,
		},
		{
			name: "empty message triple error is classified by its code",
			err:  triple_protocol.NewError(triple_protocol.CodeUnavailable, nil),
			want: ErrorTypeServiceUnavailable,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, classifyError(tt.err))
		})
	}
}

func TestClassifyError_Wrapped(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want ErrorType
	}{
		{
			name: "single level wrap of timeout",
			err:  fmt.Errorf("rpc failed: %w", triple_protocol.NewError(triple_protocol.CodeDeadlineExceeded, errors.New("timeout"))),
			want: ErrorTypeTimeout,
		},
		{
			name: "single level wrap of limit",
			err:  fmt.Errorf("rpc failed: %w", triple_protocol.NewError(triple_protocol.CodeResourceExhausted, errors.New("limit"))),
			want: ErrorTypeLimit,
		},
		{
			name: "single level wrap of service unavailable",
			err:  fmt.Errorf("rpc failed: %w", triple_protocol.NewError(triple_protocol.CodeUnavailable, errors.New("unavailable"))),
			want: ErrorTypeServiceUnavailable,
		},
		{
			name: "single level wrap of business failed",
			err:  fmt.Errorf("rpc failed: %w", triple_protocol.NewError(triple_protocol.CodeBizError, errors.New("biz error"))),
			want: ErrorTypeBusinessFailed,
		},
		{
			name: "multi level wrap",
			err:  fmt.Errorf("outer: %w", fmt.Errorf("inner: %w", triple_protocol.NewError(triple_protocol.CodeDeadlineExceeded, errors.New("timeout")))),
			want: ErrorTypeTimeout,
		},
		{
			name: "wrap of plain error stays unknown",
			err:  fmt.Errorf("rpc failed: %w", errors.New("plain error")),
			want: ErrorTypeUnknown,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, classifyError(tt.err))
		})
	}
}
