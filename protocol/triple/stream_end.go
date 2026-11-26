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

package triple

import (
	"context"
	"errors"
	"io"
	"sync"
)

import (
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	tri "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
)

// streamEnd detects the end of one client stream: a receive that fails or
// reaches the end of the stream, a send that fails, closing the receive side,
// or the end of the call's context, whichever comes first. A stream that is
// never closed and whose context never ends does not end either, just as its
// HTTP/2 stream is never released.
type streamEnd struct {
	mu     sync.Mutex
	ended  bool
	err    error
	notify func(error)
	stop   func() bool
}

type streamEndKey struct{}

// openStream opens a client stream with open and, once it is open, hands its
// end to the observers that watch the stream of invocation.
func openStream(ctx context.Context, invocation base.Invocation, open func(context.Context) (any, error)) (any, error) {
	end := &streamEnd{}
	ctx = context.WithValue(ctx, streamEndKey{}, end)
	stream, err := open(ctx)
	if err == nil {
		if notify := base.ClaimStreamWatches(ctx, invocation); notify != nil {
			end.watch(ctx, notify)
		}
	}
	return stream, err
}

// watch installs notify. A stream that already ended notifies at once.
func (e *streamEnd) watch(ctx context.Context, notify func(error)) {
	e.mu.Lock()
	if e.ended {
		err := e.err
		e.mu.Unlock()
		notify(err)
		return
	}
	e.notify = notify
	e.stop = context.AfterFunc(ctx, func() { e.finish(contextError(ctx)) })
	e.mu.Unlock()
}

func (e *streamEnd) finish(err error) {
	e.mu.Lock()
	if e.ended {
		e.mu.Unlock()
		return
	}
	e.ended = true
	e.err = err
	notify, stop := e.notify, e.stop
	e.notify, e.stop = nil, nil
	e.mu.Unlock()
	if stop != nil {
		stop()
	}
	if notify != nil {
		notify(err)
	}
}

// contextError reports the end of ctx with the code that the stream itself
// returns for it.
func contextError(ctx context.Context) error {
	err := ctx.Err()
	if errors.Is(err, context.DeadlineExceeded) {
		return tri.NewError(tri.CodeDeadlineExceeded, err)
	}
	return tri.NewError(tri.CodeCanceled, err)
}

// streamEndInterceptor wraps the connection of each client stream opened by
// openStream so that its end is observed. Unary calls pass through unchanged.
type streamEndInterceptor struct{}

func (streamEndInterceptor) WrapUnary(next tri.UnaryFunc) tri.UnaryFunc { return next }

func (streamEndInterceptor) WrapUnaryHandler(next tri.UnaryHandlerFunc) tri.UnaryHandlerFunc {
	return next
}

func (streamEndInterceptor) WrapStreamingClient(next tri.StreamingClientFunc) tri.StreamingClientFunc {
	return func(ctx context.Context, spec tri.Spec) tri.StreamingClientConn {
		conn := next(ctx, spec)
		if end, ok := ctx.Value(streamEndKey{}).(*streamEnd); ok {
			return &endObservedConn{StreamingClientConn: conn, ctx: ctx, end: end}
		}
		return conn
	}
}

func (streamEndInterceptor) WrapStreamingHandler(next tri.StreamingHandlerFunc) tri.StreamingHandlerFunc {
	return next
}

type endObservedConn struct {
	tri.StreamingClientConn
	ctx context.Context
	end *streamEnd
}

// Send and CloseRequest return io.EOF when the server has ended the stream; the
// status then comes from Receive.
func (c *endObservedConn) Send(msg any) error {
	err := c.StreamingClientConn.Send(msg)
	if err != nil && !errors.Is(err, io.EOF) {
		c.end.finish(err)
	}
	return err
}

func (c *endObservedConn) CloseRequest() error {
	err := c.StreamingClientConn.CloseRequest()
	if err != nil && !errors.Is(err, io.EOF) {
		c.end.finish(err)
	}
	return err
}

func (c *endObservedConn) Receive(msg any) error {
	err := c.StreamingClientConn.Receive(msg)
	if errors.Is(err, io.EOF) {
		c.end.finish(nil)
	} else if err != nil {
		c.end.finish(err)
	}
	return err
}

// CloseResponse after the context ended ends the stream with the end of the
// context, which may not have been observed yet. The caller still gets the
// result of closing.
func (c *endObservedConn) CloseResponse() error {
	err := c.StreamingClientConn.CloseResponse()
	endErr := err
	if endErr == nil && c.ctx.Err() != nil {
		endErr = contextError(c.ctx)
	}
	c.end.finish(endErr)
	return err
}
