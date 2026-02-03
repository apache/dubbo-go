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
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
)

// ErrorType represents the classification of RPC errors
type ErrorType uint8

const (
	// ErrorTypeUnknown is for unknown or unclassified errors
	ErrorTypeUnknown ErrorType = iota
	// ErrorTypeTimeout is for timeout exceptions (CodeDeadlineExceeded)
	ErrorTypeTimeout
	// ErrorTypeLimit is for rate limit exceeded exceptions (CodeResourceExhausted)
	ErrorTypeLimit
	// ErrorTypeServiceUnavailable is for service unavailable exceptions (CodeUnavailable, CodePermissionDenied)
	ErrorTypeServiceUnavailable
	// ErrorTypeBusinessFailed is for business logic exceptions (CodeBizError)
	ErrorTypeBusinessFailed
	// ErrorTypeNetworkFailure is for network failure exceptions (CodeInternal)
	// TODO: Map appropriate internal/network error codes to this type when available.
	ErrorTypeNetworkFailure
	// ErrorTypeCodec is for codec errors (CodeInternal)
	// TODO: Map appropriate internal/codec error codes to this type when available.
	ErrorTypeCodec
)

// classifyError classifies an error based on triple protocol error codes.
// This function supports triple and gRPC protocol errors.
func classifyError(err error) ErrorType {
	if err == nil {
		return ErrorTypeUnknown
	}
	// TODO: Support dubbo protocol error classification
	// Get the error code from triple protocol error
	code := triple_protocol.CodeOf(err)

	switch code {
	case triple_protocol.CodeDeadlineExceeded:
		return ErrorTypeTimeout
	case triple_protocol.CodeResourceExhausted:
		return ErrorTypeLimit
	case triple_protocol.CodeUnavailable, triple_protocol.CodePermissionDenied:
		return ErrorTypeServiceUnavailable
	case triple_protocol.CodeBizError:
		return ErrorTypeBusinessFailed
	default:
		return ErrorTypeUnknown
	}
}
