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
	"dubbo.apache.org/dubbo-go/v3/common/rpcerror"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
)

// ErrorType represents the classification of RPC errors
//
// It buckets RPC failures into coarse-grained categories so that the metrics
// module can aggregate them independently (e.g. count business failures vs.
// network failures separately). Each category maps to a failure layer —
// network, codec, or protocol — except business failures, which are
// application-level. Classification combines compatible Triple/gRPC statuses,
// producer-declared kinds (rpcerror.Kind) and confirmed local failure origins
// (see [classifyError]). The same taxonomy is shared by provider and consumer.
type ErrorType uint8

const (
	// ErrorTypeUnknown is for unknown or unclassified errors
	// Fallback for failures that cannot be attributed to a specific layer: nil
	// errors, unmarked plain Go errors, and any
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
	// ErrorTypeNetworkFailure requires a confirmed local transport origin.
	// A generic CodeInternal or CodeUnavailable is not sufficient evidence.
	ErrorTypeNetworkFailure ErrorType = 5
	// ErrorTypeCodec requires a confirmed local serialization origin, rather
	// than guessing from Internal/InvalidArgument or scanning error messages.
	ErrorTypeCodec ErrorType = 6
)

// String returns the stable, low-cardinality name of the error type. The
// names match rpcerror.Kind names and the failure metric suffixes.
func (t ErrorType) String() string {
	switch t {
	case ErrorTypeTimeout:
		return "timeout"
	case ErrorTypeLimit:
		return "limit"
	case ErrorTypeServiceUnavailable:
		return "service_unavailable"
	case ErrorTypeBusinessFailed:
		return "business"
	case ErrorTypeNetworkFailure:
		return "network"
	case ErrorTypeCodec:
		return "codec"
	default:
		return "unknown"
	}
}

// classifyEvent combines the returned error with request-scoped evidence that
// was collected without altering the error seen by the caller: the local
// transport origin and the kind recorded on the observation.
func classifyEvent(e *metricsEvent) ErrorType {
	return classifyError(rpcerror.WithKind(rpcerror.Mark(e.err, e.origin), e.kind))
}

// classifyError maps an RPC error to one failure category, protocol-agnostically
// and in this order:
//
//  1. Triple/gRPC statuses Timeout, Limit, PermissionDenied and Business keep
//     their priority, so existing Triple classification is unchanged.
//  2. A confirmed local Network/Codec origin (rpcerror.Of). It is evidence of
//     the root cause, so it wins over a symptom such as the read timeout that
//     follows an undecodable response.
//  3. A kind declared by the producer (rpcerror.KindOf), e.g. a Dubbo response
//     status, a client read timeout, an expired context deadline, a closed
//     client or a provider application error.
//  4. An unmarked Triple Unavailable status is ServiceUnavailable.
//
// Unmarked or conflicting evidence falls back to Unknown. Error messages are
// never parsed.
func classifyError(err error) ErrorType {
	if err == nil {
		return ErrorTypeUnknown
	}
	code := triple_protocol.CodeOf(err)

	switch code {
	case triple_protocol.CodeDeadlineExceeded:
		return ErrorTypeTimeout
	case triple_protocol.CodeResourceExhausted:
		return ErrorTypeLimit
	case triple_protocol.CodePermissionDenied:
		return ErrorTypeServiceUnavailable
	case triple_protocol.CodeBizError:
		return ErrorTypeBusinessFailed
	}
	switch rpcerror.Of(err) {
	case rpcerror.Network:
		return ErrorTypeNetworkFailure
	case rpcerror.Codec:
		return ErrorTypeCodec
	}
	switch rpcerror.KindOf(err) {
	case rpcerror.KindTimeout:
		return ErrorTypeTimeout
	case rpcerror.KindLimit:
		return ErrorTypeLimit
	case rpcerror.KindServiceUnavailable:
		return ErrorTypeServiceUnavailable
	case rpcerror.KindBusiness:
		return ErrorTypeBusinessFailed
	case rpcerror.KindCodec:
		return ErrorTypeCodec
	}
	if code == triple_protocol.CodeUnavailable {
		return ErrorTypeServiceUnavailable
	}
	return ErrorTypeUnknown
}
