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

package base

import (
	"context"
	"reflect"
	"sync/atomic"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

// A streaming call returns as soon as its stream is open, so filters that
// observe a call from start to end cannot finish when Invoke returns. They
// register a watch for the stream's end instead, and the protocol that opens
// the stream claims it. Like a retry mark, a watch names its invocation: another
// RPC started with an inherited context does not claim it.
type streamWatchKey struct{}

type streamWatch struct {
	invocation Invocation
	done       func(err error)
	parent     *streamWatch
	claimed    atomic.Bool
}

// IsStreamingCall reports whether invocation is a client, server or bidi
// streaming call.
func IsStreamingCall(invocation Invocation) bool {
	callType, _ := invocation.GetAttribute(constant.CallTypeKey)
	switch callType {
	case constant.CallClientStream, constant.CallServerStream, constant.CallBidiStream:
		return true
	}
	return false
}

// WatchStream registers done for the end of the stream that invocation opens
// with the returned context. After Invoke returns, claimed reports whether the
// protocol opened the stream and took over the watch. If it did, done is called
// once when the stream ends, with nil for a normal end. If it did not, done is
// never called and the caller finishes its observation as for a unary call.
func WatchStream(ctx context.Context, invocation Invocation, done func(err error)) (_ context.Context, claimed func() bool) {
	w := &streamWatch{invocation: invocation, done: done}
	w.parent, _ = ctx.Value(streamWatchKey{}).(*streamWatch)
	return context.WithValue(ctx, streamWatchKey{}, w), w.claimed.Load
}

// ClaimStreamWatches is called by a protocol once it has opened the stream of
// invocation. It claims the watches registered for invocation and returns the
// function that the protocol calls exactly once when the stream ends, or nil if
// there is no watch. Identity is compared only for pointers.
func ClaimStreamWatches(ctx context.Context, invocation Invocation) func(err error) {
	if reflect.ValueOf(invocation).Kind() != reflect.Pointer {
		return nil
	}
	var done []func(error)
	for w, _ := ctx.Value(streamWatchKey{}).(*streamWatch); w != nil; w = w.parent {
		if w.invocation == invocation && w.claimed.CompareAndSwap(false, true) {
			done = append(done, w.done)
		}
	}
	if len(done) == 0 {
		return nil
	}
	return func(err error) {
		for _, d := range done {
			d(err)
		}
	}
}
