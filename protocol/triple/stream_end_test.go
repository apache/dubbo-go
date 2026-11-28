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
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"google.golang.org/protobuf/types/known/wrapperspb"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	tri "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
)

// streamWatchRecorder records how a watched stream ended.
type streamWatchRecorder struct {
	mu    sync.Mutex
	calls int
	err   error
	ended chan struct{}
}

func newStreamWatchRecorder() *streamWatchRecorder {
	return &streamWatchRecorder{ended: make(chan struct{})}
}

func (r *streamWatchRecorder) done(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls++
	if r.calls == 1 {
		r.err = err
		close(r.ended)
	}
}

func (r *streamWatchRecorder) result() (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls, r.err
}

func (r *streamWatchRecorder) wait(t *testing.T) error {
	t.Helper()
	select {
	case <-r.ended:
	case <-time.After(5 * time.Second):
		t.Fatal("stream end was not observed")
	}
	calls, err := r.result()
	require.Equal(t, 1, calls)
	return err
}

func newStreamEndTestServer(t *testing.T) *tri.Client {
	t.Helper()
	server := httptest.NewUnstartedServer(newStreamEndTestMux())
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.Close)
	return tri.NewClient(server.Client(), server.URL+"/svc", tri.WithInterceptors(streamEndInterceptor{}))
}

func newStreamEndTestMux() *http.ServeMux {
	newMsg := func() any { return &wrapperspb.StringValue{} }
	mux := http.NewServeMux()
	mux.Handle("/svc/Bidi", tri.NewBidiStreamHandler("/svc/Bidi", func(ctx context.Context, stream *tri.BidiStream) error {
		for {
			msg := &wrapperspb.StringValue{}
			if err := stream.Receive(msg); err != nil {
				if errors.Is(err, io.EOF) {
					return nil
				}
				return err
			}
			switch msg.GetValue() {
			case "fail":
				return tri.NewError(tri.CodeUnavailable, errors.New("provider failed"))
			case "hang":
				<-ctx.Done()
				return ctx.Err()
			}
			if err := stream.Send(msg); err != nil {
				return err
			}
		}
	}))
	mux.Handle("/svc/Server", tri.NewServerStreamHandler("/svc/Server", newMsg, func(_ context.Context, req *tri.Request, stream *tri.ServerStream) error {
		if err := stream.Send(&wrapperspb.StringValue{Value: "first"}); err != nil {
			return err
		}
		if req.Msg.(*wrapperspb.StringValue).GetValue() == "fail" {
			return tri.NewError(tri.CodeUnavailable, errors.New("provider failed"))
		}
		return nil
	}))
	mux.Handle("/svc/Client", tri.NewClientStreamHandler("/svc/Client", func(_ context.Context, stream *tri.ClientStream) (*tri.Response, error) {
		for stream.Receive(&wrapperspb.StringValue{}) {
			if stream.Msg().(*wrapperspb.StringValue).GetValue() == "fail" {
				return nil, tri.NewError(tri.CodeUnavailable, errors.New("provider failed"))
			}
		}
		if err := stream.Err(); err != nil {
			return nil, err
		}
		return tri.NewResponse(&wrapperspb.StringValue{Value: "ok"}), nil
	}))
	return mux
}

// watchStream registers a watch as a consumer filter does.
func watchStream(ctx context.Context) (context.Context, base.Invocation, func() bool, *streamWatchRecorder) {
	inv := invocation.NewRPCInvocation("Call", nil, nil)
	recorder := newStreamWatchRecorder()
	ctx, claimed := base.WatchStream(ctx, inv, recorder.done)
	return ctx, inv, claimed, recorder
}

func requireNotEnded(t *testing.T, recorder *streamWatchRecorder) {
	t.Helper()
	calls, _ := recorder.result()
	require.Zero(t, calls, "the watch ended before the stream did")
}

func TestOpenStreamObservesBidiStream(t *testing.T) {
	client := newStreamEndTestServer(t)
	tests := []struct {
		name     string
		messages []string
		wantCode tri.Code
	}{
		{name: "normal end", messages: []string{"a", "b"}},
		{name: "provider error", messages: []string{"a", "fail"}, wantCode: tri.CodeUnavailable},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, inv, claimed, recorder := watchStream(context.Background())
			raw, err := openStream(ctx, inv, func(ctx context.Context) (any, error) {
				return client.CallBidiStream(ctx, "Bidi")
			})
			require.NoError(t, err)
			require.True(t, claimed())
			stream := raw.(*tri.BidiStreamForClient)
			for _, m := range test.messages {
				require.NoError(t, stream.Send(&wrapperspb.StringValue{Value: m}))
			}
			require.NoError(t, stream.CloseRequest())
			requireNotEnded(t, recorder)
			var recvErr error
			for recvErr == nil {
				recvErr = stream.Receive(&wrapperspb.StringValue{})
			}
			got := recorder.wait(t)
			if test.wantCode == 0 {
				require.ErrorIs(t, recvErr, io.EOF)
				require.NoError(t, got)
			} else {
				require.Equal(t, test.wantCode, tri.CodeOf(got))
				require.Equal(t, recvErr, got)
			}
			require.NoError(t, stream.CloseResponse())
			calls, _ := recorder.result()
			require.Equal(t, 1, calls)
		})
	}
}

func TestOpenStreamObservesServerStream(t *testing.T) {
	client := newStreamEndTestServer(t)
	for _, test := range []struct {
		request  string
		wantCode tri.Code
	}{{request: "ok"}, {request: "fail", wantCode: tri.CodeUnavailable}} {
		t.Run(test.request, func(t *testing.T) {
			ctx, inv, claimed, recorder := watchStream(context.Background())
			raw, err := openStream(ctx, inv, func(ctx context.Context) (any, error) {
				return client.CallServerStream(ctx, tri.NewRequest(&wrapperspb.StringValue{Value: test.request}), "Server")
			})
			require.NoError(t, err)
			require.True(t, claimed())
			stream := raw.(*tri.ServerStreamForClient)
			requireNotEnded(t, recorder)
			for stream.Receive(&wrapperspb.StringValue{}) {
			}
			got := recorder.wait(t)
			if test.wantCode == 0 {
				require.NoError(t, got)
			} else {
				require.Equal(t, test.wantCode, tri.CodeOf(got))
			}
			require.NoError(t, stream.Close())
		})
	}
}

func TestOpenStreamObservesClientStream(t *testing.T) {
	client := newStreamEndTestServer(t)
	for _, test := range []struct {
		message  string
		wantCode tri.Code
	}{{message: "ok"}, {message: "fail", wantCode: tri.CodeUnavailable}} {
		t.Run(test.message, func(t *testing.T) {
			ctx, inv, claimed, recorder := watchStream(context.Background())
			raw, err := openStream(ctx, inv, func(ctx context.Context) (any, error) {
				return client.CallClientStream(ctx, "Client")
			})
			require.NoError(t, err)
			require.True(t, claimed())
			stream := raw.(*tri.ClientStreamForClient)
			require.NoError(t, stream.Send(&wrapperspb.StringValue{Value: test.message}))
			requireNotEnded(t, recorder)
			closeErr := stream.CloseAndReceive(tri.NewResponse(&wrapperspb.StringValue{}))
			got := recorder.wait(t)
			if test.wantCode == 0 {
				require.NoError(t, closeErr)
				require.NoError(t, got)
			} else {
				require.Equal(t, test.wantCode, tri.CodeOf(closeErr))
				require.Equal(t, test.wantCode, tri.CodeOf(got))
			}
		})
	}
}

func TestOpenStreamObservesContextEnd(t *testing.T) {
	client := newStreamEndTestServer(t)
	tests := []struct {
		name     string
		ctx      func() (context.Context, context.CancelFunc)
		wantCode tri.Code
	}{
		{name: "canceled", ctx: func() (context.Context, context.CancelFunc) {
			return context.WithCancel(context.Background())
		}, wantCode: tri.CodeCanceled},
		{name: "deadline", ctx: func() (context.Context, context.CancelFunc) {
			return context.WithTimeout(context.Background(), 200*time.Millisecond)
		}, wantCode: tri.CodeDeadlineExceeded},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parent, cancel := test.ctx()
			defer cancel()
			ctx, inv, claimed, recorder := watchStream(parent)
			raw, err := openStream(ctx, inv, func(ctx context.Context) (any, error) {
				return client.CallBidiStream(ctx, "Bidi")
			})
			require.NoError(t, err)
			require.True(t, claimed())
			stream := raw.(*tri.BidiStreamForClient)
			require.NoError(t, stream.Send(&wrapperspb.StringValue{Value: "hang"}))
			// The consumer neither receives nor closes the stream; the end of
			// its context alone ends the observation.
			if test.wantCode == tri.CodeCanceled {
				requireNotEnded(t, recorder)
				cancel()
			}
			require.Equal(t, test.wantCode, tri.CodeOf(recorder.wait(t)))
			_ = stream.CloseResponse()
		})
	}
}

// TestTripleInvokerEndsClientSpanWithStream goes through the production client
// manager, TripleInvoker and client trace filter.
func TestTripleInvokerEndsClientSpanWithStream(t *testing.T) {
	server := httptest.NewServer(h2c.NewHandler(newStreamEndTestMux(), &http2.Server{}))
	t.Cleanup(server.Close)
	url, err := common.NewURL("tri://" + server.Listener.Addr().String() + "/svc?interface=svc")
	require.NoError(t, err)
	cm, err := newClientManager(url)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cm.close() })

	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	t.Cleanup(func() { otel.SetTracerProvider(previous) })
	filter, ok := extension.GetFilter(constant.OTELClientTraceKey)
	require.True(t, ok)

	inv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("Bidi"))
	inv.SetAttribute(constant.CallTypeKey, constant.CallBidiStream)
	res := filter.Invoke(context.Background(), newTestTripleInvoker(url, cm), inv)
	require.NoError(t, res.Error())
	stream := res.Result().(*tri.BidiStreamForClient)

	require.NoError(t, stream.Send(&wrapperspb.StringValue{Value: "a"}))
	require.NoError(t, stream.Receive(&wrapperspb.StringValue{}))
	require.Empty(t, recorder.Ended(), "the client span ended before the stream did")
	require.NoError(t, stream.Send(&wrapperspb.StringValue{Value: "fail"}))
	recvErr := stream.Receive(&wrapperspb.StringValue{})
	require.Equal(t, tri.CodeUnavailable, tri.CodeOf(recvErr))

	ended := recorder.Ended()
	require.Len(t, ended, 1)
	require.Equal(t, codes.Error, ended[0].Status().Code)
	require.Contains(t, ended[0].Status().Description, "provider failed")
	require.NoError(t, stream.CloseResponse())
	require.Len(t, recorder.Ended(), 1)
}

func TestOpenStreamFailureLeavesWatchToCaller(t *testing.T) {
	ctx, inv, claimed, recorder := watchStream(context.Background())
	openErr := errors.New("open failed")
	_, err := openStream(ctx, inv, func(context.Context) (any, error) { return nil, openErr })
	require.ErrorIs(t, err, openErr)
	assert.False(t, claimed())
	requireNotEnded(t, recorder)
}

func TestOpenStreamClaimsOnlyItsInvocation(t *testing.T) {
	ctx, _, claimed, recorder := watchStream(context.Background())
	other := invocation.NewRPCInvocation("Other", nil, nil)
	end := &streamEnd{}
	_, err := openStream(ctx, other, func(ctx context.Context) (any, error) {
		end = ctx.Value(streamEndKey{}).(*streamEnd)
		return struct{}{}, nil
	})
	require.NoError(t, err)
	assert.False(t, claimed())
	end.finish(nil)
	requireNotEnded(t, recorder)
}

type fakeStreamingConn struct {
	tri.StreamingClientConn
	sendErr  error
	received []error // results of successive Receive calls
}

func (c *fakeStreamingConn) Send(any) error      { return c.sendErr }
func (c *fakeStreamingConn) CloseRequest() error { return c.sendErr }
func (c *fakeStreamingConn) Receive(any) error {
	if len(c.received) == 0 {
		return io.EOF
	}
	err := c.received[0]
	c.received = c.received[1:]
	return err
}
func (c *fakeStreamingConn) CloseResponse() error { return nil }

func TestEndObservedConnTerminalEvents(t *testing.T) {
	sendFailure := errors.New("send failed")
	receive := func(times int) func(*endObservedConn) {
		return func(c *endObservedConn) {
			for range times {
				_ = c.Receive(nil)
			}
		}
	}
	tests := []struct {
		name     string
		conn     *fakeStreamingConn
		ctx      context.Context
		act      func(*endObservedConn)
		ended    bool
		wantErr  error
		wantCode tri.Code
	}{
		{name: "send reaching the end of the stream", conn: &fakeStreamingConn{sendErr: io.EOF},
			act: func(c *endObservedConn) { _ = c.Send(nil); _ = c.CloseRequest() }},
		{name: "send failure", conn: &fakeStreamingConn{sendErr: sendFailure},
			act: func(c *endObservedConn) { _ = c.Send(nil) }, ended: true, wantErr: sendFailure},
		{name: "close request failure", conn: &fakeStreamingConn{sendErr: sendFailure},
			act: func(c *endObservedConn) { _ = c.CloseRequest() }, ended: true, wantErr: sendFailure},
		{name: "received message", conn: &fakeStreamingConn{received: []error{nil}}, act: receive(1)},
		{name: "end of stream", conn: &fakeStreamingConn{received: []error{nil, io.EOF}}, act: receive(2), ended: true},
		{name: "close response", conn: &fakeStreamingConn{},
			act: func(c *endObservedConn) { _ = c.CloseResponse() }, ended: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			end := &streamEnd{}
			recorder := newStreamWatchRecorder()
			end.watch(context.Background(), recorder.done)
			test.act(&endObservedConn{StreamingClientConn: test.conn, ctx: context.Background(), end: end})
			calls, err := recorder.result()
			if !test.ended {
				require.Zero(t, calls)
				return
			}
			require.Equal(t, 1, calls)
			if test.wantCode != 0 {
				require.Equal(t, test.wantCode, tri.CodeOf(err))
			} else {
				require.Equal(t, test.wantErr, err)
			}
		})
	}
}

func TestCloseResponseAfterContextEnd(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	end := &streamEnd{}
	recorder := newStreamWatchRecorder()
	end.watch(context.Background(), recorder.done)
	conn := &endObservedConn{StreamingClientConn: &fakeStreamingConn{}, ctx: ctx, end: end}

	require.NoError(t, conn.CloseResponse(), "the caller gets the result of closing")
	require.Equal(t, tri.CodeCanceled, tri.CodeOf(recorder.wait(t)), "the stream ended with its context")
}

func TestStreamEndNotifiesOnce(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	end := &streamEnd{}
	recorder := newStreamWatchRecorder()
	end.watch(ctx, recorder.done)
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			end.finish(nil)
		})
	}
	cancel()
	wg.Wait()
	recorder.wait(t)
	time.Sleep(10 * time.Millisecond) // let a late context callback run, if any
	calls, _ := recorder.result()
	require.Equal(t, 1, calls)
}

func TestStreamEndBeforeWatchNotifiesAtOnce(t *testing.T) {
	end := &streamEnd{}
	failure := errors.New("ended early")
	end.finish(failure)
	recorder := newStreamWatchRecorder()
	end.watch(context.Background(), recorder.done)
	require.Equal(t, failure, recorder.wait(t))
}
