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

// Package rpcerror describes local RPC failure origins without depending on
// metrics, a particular protocol, or a transport implementation.
package rpcerror

import (
	"context"
	"errors"
	"net"
	"reflect"
)

// Origin identifies a confirmed local failure source, not a wire status code.
type Origin uint8

const (
	Unknown Origin = iota
	Network
	Codec
)

type sourceError struct {
	origin Origin
	cause  error
}

func (e *sourceError) Error() string { return e.cause.Error() }
func (e *sourceError) Unwrap() error { return e.cause }

// Mark preserves the message and cause chain. Call only at a confirmed source;
// neither a generic Internal status nor an arbitrary user error proves origin.
// An existing marker is retained, so a codec failure crossing an I/O boundary
// cannot be relabeled as a network failure.
func Mark(err error, origin Origin) error {
	if isNil(err) || (origin != Network && origin != Codec) || Of(err) != Unknown {
		return err
	}
	return &sourceError{origin: origin, cause: err}
}

// Of follows wrapping and rejects conflicting markers (including errors.Join).
func Of(err error) Origin {
	var found Origin
	collectOrigins(err, &found)
	if found == Network || found == Codec {
		return found
	}
	return Unknown
}

// Keep conflicts sticky across the whole error tree, including later siblings.
func collectOrigins(err error, found *Origin) {
	if isNil(err) {
		return
	}
	if e, ok := err.(*sourceError); ok && e != nil {
		if *found != Unknown && *found != e.origin {
			*found = Origin(255)
		} else {
			*found = e.origin
		}
	}
	switch e := err.(type) {
	case interface{ Unwrap() []error }:
		for _, cause := range e.Unwrap() {
			collectOrigins(cause, found)
		}
	case interface{ Unwrap() error }:
		collectOrigins(e.Unwrap(), found)
	}
}

// FromNetworkOperation is for trusted transport call sites only. Context
// cancellation, an existing codec marker, and unstructured errors are preserved.
func FromNetworkOperation(err error) error {
	if isNil(err) || Of(err) != Unknown || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	var op *net.OpError
	if errors.As(err, &op) && op != nil {
		return Mark(err, Network)
	}
	return err
}

func isNil(err error) bool {
	if err == nil {
		return true
	}
	v := reflect.ValueOf(err)
	switch v.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan, reflect.Interface:
		return v.IsNil()
	default:
		return false
	}
}
