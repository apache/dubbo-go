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
)

// Kind is the protocol-agnostic failure category declared by the code that
// produced an error. It complements Origin: Origin is local transport evidence
// (network/codec) collected by remoting, while Kind is the semantic category a
// protocol, invoker or filter knows for certain at its production point, such
// as a Dubbo response status or a client-side read timeout.
//
// The categories mirror Java Dubbo's RpcException codes as consumed by
// MetricsSupport#getMetricsKey, so metrics, tracing and logging can share one
// taxonomy without depending on a particular protocol.
type Kind uint8

const (
	// KindUnknown means no producer declared a category.
	KindUnknown Kind = iota
	// KindTimeout is a call that did not finish before its deadline
	// (Java TIMEOUT_EXCEPTION / TIMEOUT_TERMINATE).
	KindTimeout
	// KindLimit is a call rejected by flow control (Java LIMIT_EXCEEDED_EXCEPTION).
	KindLimit
	// KindServiceUnavailable is a call that could not be served by the selected
	// invoker: destroyed invoker, closed client, missing service or method
	// (Java FORBIDDEN_EXCEPTION / NO_INVOKER_AVAILABLE_AFTER_FILTER / METHOD_NOT_FOUND).
	KindServiceUnavailable
	// KindBusiness is an error returned or thrown by the provider application
	// itself (Java BIZ_EXCEPTION).
	KindBusiness
	// KindCodec is a serialization failure reported by the remote peer, for
	// example a Dubbo BAD_REQUEST/BAD_RESPONSE status (Java SERIALIZATION_EXCEPTION).
	// Local codec failures are recorded as Origin Codec instead.
	KindCodec

	kindConflict Kind = 255
)

// String returns the stable, low-cardinality name of the kind.
func (k Kind) String() string {
	switch k {
	case KindTimeout:
		return "timeout"
	case KindLimit:
		return "limit"
	case KindServiceUnavailable:
		return "service_unavailable"
	case KindBusiness:
		return "business"
	case KindCodec:
		return "codec"
	default:
		return "unknown"
	}
}

func (k Kind) valid() bool {
	return k >= KindTimeout && k <= KindCodec
}

type kindError struct {
	kind  Kind
	text  string
	cause error
}

func (e *kindError) Error() string {
	if e.cause != nil {
		return e.cause.Error()
	}
	return e.text
}

func (e *kindError) Unwrap() error { return e.cause }

// NewKindError returns a new error that carries kind. It is intended for
// package-level sentinels, which keep their identity for == and errors.Is
// comparisons while still being classifiable.
func NewKindError(kind Kind, text string) error {
	if !kind.valid() {
		return errors.New(text)
	}
	return &kindError{kind: kind, text: text}
}

// WithKind attaches kind to err, preserving the message and the cause chain
// (errors.Is/As and Unwrap keep working). Call it only where the category is
// certain. An already declared kind is retained, so an outer layer cannot
// relabel a failure that an inner layer classified first.
func WithKind(err error, kind Kind) error {
	if isNil(err) || !kind.valid() || declaredKind(err) != KindUnknown {
		return err
	}
	return &kindError{kind: kind, cause: err}
}

// KindOf returns the kind declared anywhere in err's chain. Conflicting
// declarations (for example from errors.Join) yield KindUnknown. An expired
// context deadline is a timeout by definition and is reported as KindTimeout
// when nothing else was declared; a canceled context is not.
func KindOf(err error) Kind {
	if isNil(err) {
		return KindUnknown
	}
	found := declaredKind(err)
	if found.valid() {
		return found
	}
	if found == KindUnknown && errors.Is(err, context.DeadlineExceeded) {
		return KindTimeout
	}
	return KindUnknown
}

// declaredKind returns the kind explicitly declared in err's chain, which may
// be kindConflict; it does not apply the context deadline fallback.
func declaredKind(err error) Kind {
	var found Kind
	collectKinds(err, &found)
	return found
}

func collectKinds(err error, found *Kind) {
	if isNil(err) {
		return
	}
	if e, ok := err.(*kindError); ok && e != nil {
		if *found != KindUnknown && *found != e.kind {
			*found = kindConflict
		} else {
			*found = e.kind
		}
	}
	switch e := err.(type) {
	case interface{ Unwrap() []error }:
		for _, cause := range e.Unwrap() {
			collectKinds(cause, found)
		}
	case interface{ Unwrap() error }:
		collectKinds(e.Unwrap(), found)
	}
}
