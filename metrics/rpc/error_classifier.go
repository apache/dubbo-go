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
//
// It buckets RPC failures into coarse-grained categories so that the metrics
// module can aggregate them independently (e.g. count business failures vs.
// network failures separately). Each category maps to a failure layer —
// network, codec, or protocol — except business failures, which are
// application-level. The categories are derived from the Triple / gRPC status
// codes returned by the protocol layer (see [classifyError]).
type ErrorType uint8

const (
	// ErrorTypeUnknown is for unknown or unclassified errors
	// Fallback for failures that cannot be attributed to a specific layer: nil
	// errors, plain Go errors that do not carry a Triple status code, and any
	// network / codec / protocol code the classifier does not handle explicitly
	// (see the default branch of [classifyError]).
	ErrorTypeUnknown ErrorType = 0
	// ErrorTypeTimeout is for timeout exceptions (CodeDeadlineExceeded)
	// Protocol layer: the call did not finish before the deadline configured by
	// the caller, surfaced as CodeDeadlineExceeded. This is a protocol-level
	// timeout, distinct from a network-layer socket read timeout and from any
	// codec-layer delay.
	ErrorTypeTimeout ErrorType = 1
	// ErrorTypeLimit is for rate limit exceeded exceptions (CodeResourceExhausted)
	// Protocol layer: the protocol layer returns CodeResourceExhausted to signal
	// back-pressure, e.g. a per-user quota or the upstream TPS limiter rejecting
	// the request. It is not a network-layer or codec-layer fault.
	ErrorTypeLimit ErrorType = 2
	// ErrorTypeServiceUnavailable is for service unavailable exceptions (CodeUnavailable, CodePermissionDenied)
	// Protocol layer: the protocol layer returns CodeUnavailable (the service is
	// down or shedding load) or CodePermissionDenied (the caller is not allowed
	// to invoke the service). These map to the same bucket because, from a
	// consumer perspective, both mean "this call cannot be served and may
	// succeed after remediation (retry/back-off, fix credentials)".
	ErrorTypeServiceUnavailable ErrorType = 3
	// ErrorTypeBusinessFailed is for business logic exceptions (CodeBizError)
	// Application layer (outside the network / codec / protocol layers): the
	// provider executed the RPC successfully at the network, codec, and protocol
	// layers, but the application code returned an error, surfaced by the
	// protocol layer as CodeBizError. Distinguishing it from network and codec
	// failures lets metrics separate "the call reached the server" from "the
	// call failed to reach the server".
	ErrorTypeBusinessFailed ErrorType = 4
	// ErrorTypeNetworkFailure is for network failure exceptions (CodeInternal)
	// Network layer: covers transport faults such as refused connections,
	// broken pipes, DNS resolution failures, and socket read/write timeouts.
	//
	// Some transport-layer faults are surfaced by the Triple protocol as
	// CodeInternal — for example HTTP/2 stream errors such as PROTOCOL_ERROR,
	// INTERNAL_ERROR, FLOW_CONTROL_ERROR, FRAME_SIZE_ERROR, and CONNECT_ERROR
	// are mapped to CodeInternal. Because CodeInternal is also used for codec
	// and server-side invariant errors (see [ErrorTypeCodec]), these transport
	// faults cannot be told apart from them at the status-code level, so they
	// fall through to [ErrorTypeUnknown] today.
	// TODO: At present, this error type has not been produced. If available, please map the appropriate internal/network error code to this type.
	ErrorTypeNetworkFailure ErrorType = 5
	// ErrorTypeCodec is for codec errors (CodeInternal)
	// Codec layer: covers serialization failures — malformed payloads,
	// unsupported serialization types, and errors while (de)serializing
	// request or response bodies.
	//
	// Codec errors are not surfaced through a single status code: marshal and
	// compression failures map to CodeInternal, while unmarshal and decompress
	// failures map to CodeInvalidArgument. Because they are spread across
	// multiple codes (neither of which is codec-specific), they cannot be
	// distinguished from network and other errors by status code alone, and
	// fall through to [ErrorTypeUnknown] today.
	// TODO: At present, this error type has not been produced. If available, please map the appropriate internal/codec error code to this type.
	ErrorTypeCodec ErrorType = 6
)

// classifyError classifies an error based on triple protocol error codes.
// This function supports triple and gRPC protocol errors.
//
// It extracts the status code with [triple_protocol.CodeOf] and maps it to an
// [ErrorType] by failure layer. Protocol-layer codes map as follows:
// CodeDeadlineExceeded -> [ErrorTypeTimeout], CodeResourceExhausted ->
// [ErrorTypeLimit], CodeUnavailable and CodePermissionDenied ->
// [ErrorTypeServiceUnavailable]. The application-layer CodeBizError maps to
// [ErrorTypeBusinessFailed]. Plain Go errors (those that do not carry a Triple
// status code) and every unhandled code fall back to [ErrorTypeUnknown].
//
// [ErrorTypeNetworkFailure] (network layer) and [ErrorTypeCodec] (codec layer)
// are reserved but not yet produced: some transport-layer faults and codec
// failures are surfaced as CodeInternal (and codec failures also as
// CodeInvalidArgument), which cannot be split reliably between the two, so it
// is left to the default branch.
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
