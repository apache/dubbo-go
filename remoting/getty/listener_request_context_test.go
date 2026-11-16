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
 * See the License for the specific language regarding permissions and
 * limitations under the License.
 */

package getty

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

import (
	dubboGetty "github.com/apache/dubbo-getty"

	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

type contextTestSession struct {
	dubboGetty.Session
	mu         sync.Mutex
	closeCount int
	remote     string
}

func (s *contextTestSession) WritePkg(pkg any, _ time.Duration) (int, int, error) {
	return 0, 0, nil
}

func (s *contextTestSession) IsClosed() bool { return false }

func (s *contextTestSession) Stat() string { return "mock-session" }

func (s *contextTestSession) LocalAddr() string { return "127.0.0.1:1" }

func (s *contextTestSession) RemoteAddr() string {
	if s.remote != "" {
		return s.remote
	}
	return "127.0.0.1:2"
}

func (s *contextTestSession) Close() {
	s.mu.Lock()
	s.closeCount++
	s.mu.Unlock()
}

func (s *contextTestSession) closes() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closeCount
}

func newContextTestHandler(handler func(*invocation.RPCInvocation) result.RPCResult) *RpcServerHandler {
	return NewRpcServerHandler(8, time.Minute, &Server{requestHandler: handler})
}

func deliverOneway(t *testing.T, h *RpcServerHandler, session dubboGetty.Session, attachments map[string]any) {
	t.Helper()
	if attachments == nil {
		attachments = map[string]any{}
	}
	inv := invocation.NewRPCInvocation("Echo", nil, attachments)
	req := remoting.NewRequest("2.0.2")
	req.TwoWay = false
	req.Data = inv
	h.OnMessage(session, &remoting.DecodeResult{IsRequest: true, Result: req})
}

const testWait = time.Second

func waitCtx(ctx context.Context, timeout time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(timeout):
		return errors.New("timed out waiting for context cancellation")
	}
}

func waitErr(t *testing.T, ch <-chan error, msg string) error {
	t.Helper()
	select {
	case err := <-ch:
		return err
	case <-time.After(testWait):
		t.Fatal(msg)
	}
	return nil
}

func waitSignal(t *testing.T, ch <-chan struct{}, msg string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(testWait):
		t.Fatal(msg)
	}
}

func waitStarted(t *testing.T, ch <-chan context.Context, msg string) context.Context {
	t.Helper()
	select {
	case ctx := <-ch:
		return ctx
	case <-time.After(testWait):
		t.Fatal(msg)
	}
	return nil
}

// TestServerRequestContextFollowsSessionAndTimeout checks the provider
// request context: it is canceled when the session closes, a valid timeout
// attachment becomes a deadline, finishing one request does not close the
// shared session, and closing one session does not cancel another.
func TestServerRequestContextFollowsSessionAndTimeout(t *testing.T) {
	t.Run("disconnect cancels in-flight request", func(t *testing.T) {
		started := make(chan context.Context, 1)
		done := make(chan error, 1)
		session := &contextTestSession{}
		h := newContextTestHandler(func(inv *invocation.RPCInvocation) result.RPCResult {
			ctx := inv.Context()
			started <- ctx
			done <- waitCtx(ctx, testWait)
			return result.RPCResult{}
		})
		require.NoError(t, h.OnOpen(session))
		t.Cleanup(func() { h.OnClose(session) })

		go deliverOneway(t, h, session, nil)

		ctx := waitStarted(t, started, "handler was not started")
		require.NoError(t, ctx.Err())
		h.OnClose(session)
		require.ErrorIs(t, waitErr(t, done, "request context was not canceled after session close"), context.Canceled)
	})

	t.Run("timeout attachment sets deadline without closing session", func(t *testing.T) {
		done := make(chan error, 1)
		session := &contextTestSession{}
		h := newContextTestHandler(func(inv *invocation.RPCInvocation) result.RPCResult {
			ctx := inv.Context()
			if _, hasDeadline := ctx.Deadline(); !hasDeadline {
				done <- errors.New("missing deadline")
				return result.RPCResult{}
			}
			done <- waitCtx(ctx, testWait)
			return result.RPCResult{}
		})
		require.NoError(t, h.OnOpen(session))
		t.Cleanup(func() { h.OnClose(session) })

		go deliverOneway(t, h, session, map[string]any{constant.TimeoutKey: "30ms"})

		require.ErrorIs(t, waitErr(t, done, "request context was not canceled by timeout attachment"), context.DeadlineExceeded)
		require.Equal(t, 0, session.closes(), "timeout must not close the shared session")

		// Observe the next request while the handler is still running;
		// returning from the handler is supposed to cancel that request ctx.
		type secondResult struct {
			err         error
			hasDeadline bool
		}
		second := make(chan secondResult, 1)
		h.server.requestHandler = func(inv *invocation.RPCInvocation) result.RPCResult {
			ctx := inv.Context()
			_, hasDeadline := ctx.Deadline()
			second <- secondResult{err: ctx.Err(), hasDeadline: hasDeadline}
			return result.RPCResult{}
		}
		deliverOneway(t, h, session, nil)
		select {
		case got := <-second:
			require.NoError(t, got.err)
			require.False(t, got.hasDeadline)
		case <-time.After(testWait):
			t.Fatal("session was not reusable after the timed-out request")
		}
	})

	t.Run("request return cancels context and leaves session open", func(t *testing.T) {
		var ctx context.Context
		session := &contextTestSession{}
		h := newContextTestHandler(func(inv *invocation.RPCInvocation) result.RPCResult {
			ctx = inv.Context()
			return result.RPCResult{}
		})
		require.NoError(t, h.OnOpen(session))
		deliverOneway(t, h, session, nil)
		require.NotNil(t, ctx)
		require.ErrorIs(t, waitCtx(ctx, testWait), context.Canceled, "request context was not canceled after the handler returned")
		require.Equal(t, 0, session.closes())
	})

	t.Run("closing one session does not cancel another", func(t *testing.T) {
		startedA := make(chan struct{}, 1)
		startedB := make(chan struct{}, 1)
		doneA := make(chan error, 1)
		doneB := make(chan error, 1)
		sessionA := &contextTestSession{remote: "session-a"}
		sessionB := &contextTestSession{remote: "session-b"}
		h := newContextTestHandler(func(inv *invocation.RPCInvocation) result.RPCResult {
			ctx := inv.Context()
			addr, _ := inv.GetAttachment(constant.RemoteAddr)
			if addr == "session-a" {
				startedA <- struct{}{}
				doneA <- waitCtx(ctx, testWait)
				return result.RPCResult{}
			}
			startedB <- struct{}{}
			doneB <- waitCtx(ctx, testWait)
			return result.RPCResult{}
		})
		require.NoError(t, h.OnOpen(sessionA))
		require.NoError(t, h.OnOpen(sessionB))
		t.Cleanup(func() {
			h.OnClose(sessionA)
			h.OnClose(sessionB)
		})

		go deliverOneway(t, h, sessionA, nil)
		go deliverOneway(t, h, sessionB, nil)
		waitSignal(t, startedA, "session A handler was not started")
		waitSignal(t, startedB, "session B handler was not started")

		h.OnClose(sessionA)
		require.ErrorIs(t, waitErr(t, doneA, "session A was not canceled"), context.Canceled)
		select {
		case err := <-doneB:
			t.Fatalf("session B was canceled when session A closed: %v", err)
		case <-time.After(30 * time.Millisecond):
		}
		h.OnClose(sessionB)
		require.ErrorIs(t, waitErr(t, doneB, "session B was not canceled after its own close"), context.Canceled)
		require.Equal(t, 0, sessionA.closes()+sessionB.closes(), "OnClose must not itself close the session")
	})
}

func TestServerStopCancelsInFlightRequest(t *testing.T) {
	started := make(chan struct{}, 1)
	done := make(chan error, 1)
	session := &contextTestSession{}
	server := &Server{}
	h := NewRpcServerHandler(8, time.Minute, server)
	server.rpcHandler = h
	server.requestHandler = func(inv *invocation.RPCInvocation) result.RPCResult {
		started <- struct{}{}
		done <- waitCtx(inv.Context(), testWait)
		return result.RPCResult{}
	}
	require.NoError(t, h.OnOpen(session))
	t.Cleanup(server.Stop)
	go deliverOneway(t, h, session, nil)
	waitSignal(t, started, "handler was not started")
	server.Stop()
	require.ErrorIs(t, waitErr(t, done, "request context was not canceled after Server.Stop"), context.Canceled)
	require.Equal(t, 0, session.closes())
}

// TestServerRequestContextCanceledIfSessionAlreadyReleased is the regression
// for a queued OnMessage that runs after OnClose/cancelAll have already
// removed the session. Getty's task pool can do this: addTask only checks
// IsClosed at the start of the task, and Server.Stop cancelAll happens
// before tcpServer.Close. The handler must not see a live Background ctx.
func TestServerRequestContextCanceledIfSessionAlreadyReleased(t *testing.T) {
	assertAlreadyCanceled := func(t *testing.T, release func(*RpcServerHandler, *contextTestSession)) {
		t.Helper()
		var errInHandler error
		session := &contextTestSession{}
		h := newContextTestHandler(func(inv *invocation.RPCInvocation) result.RPCResult {
			errInHandler = inv.Context().Err()
			return result.RPCResult{}
		})
		require.NoError(t, h.OnOpen(session))
		release(h, session)
		deliverOneway(t, h, session, nil)
		require.ErrorIs(t, errInHandler, context.Canceled,
			"handler ran with a live context after the session was already released")
	}

	t.Run("on close before bind", func(t *testing.T) {
		assertAlreadyCanceled(t, func(h *RpcServerHandler, session *contextTestSession) {
			h.OnClose(session)
		})
	})
	t.Run("cancelAll before bind", func(t *testing.T) {
		assertAlreadyCanceled(t, func(h *RpcServerHandler, session *contextTestSession) {
			h.cancelAll()
		})
	})
}

func TestTimeoutFromAttachment(t *testing.T) {
	parse := func(raw any) (time.Duration, bool) {
		inv := invocation.NewRPCInvocation("Echo", nil, map[string]any{})
		if raw != nil {
			inv.SetAttachment(constant.TimeoutKey, raw)
		}
		return timeoutFromAttachment(inv)
	}

	d, ok := parse("30ms")
	require.True(t, ok)
	require.Equal(t, 30*time.Millisecond, d)

	d, ok = parse("5000")
	require.True(t, ok)
	require.Equal(t, 5*time.Second, d)

	_, ok = parse("")
	require.False(t, ok)
	_, ok = parse("0")
	require.False(t, ok)
	_, ok = parse("-1")
	require.False(t, ok)
	_, ok = parse("bogus")
	require.False(t, ok)
	_, ok = parse(nil)
	require.False(t, ok)
}

func TestBindRequestContextTimeoutKind(t *testing.T) {
	session := &contextTestSession{}
	h := newContextTestHandler(func(inv *invocation.RPCInvocation) result.RPCResult { return result.RPCResult{} })
	require.NoError(t, h.OnOpen(session))
	t.Cleanup(func() { h.OnClose(session) })
	inv := invocation.NewRPCInvocation("Echo", nil, map[string]any{constant.TimeoutKey: "20ms"})
	cancel := h.bindRequestContext(session, inv)
	defer cancel()
	ctx := inv.Context()
	_, ok := ctx.Deadline()
	require.True(t, ok)
	require.ErrorIs(t, waitCtx(ctx, testWait), context.DeadlineExceeded)
}

func TestServerOnOpenAfterCancelAllStartsLive(t *testing.T) {
	var errInHandler error
	h := newContextTestHandler(func(inv *invocation.RPCInvocation) result.RPCResult {
		errInHandler = inv.Context().Err()
		return result.RPCResult{}
	})
	first := &contextTestSession{}
	require.NoError(t, h.OnOpen(first))
	h.cancelAll()
	next := &contextTestSession{}
	require.NoError(t, h.OnOpen(next))
	t.Cleanup(func() { h.OnClose(next) })
	deliverOneway(t, h, next, nil)
	require.NoError(t, errInHandler)
}
