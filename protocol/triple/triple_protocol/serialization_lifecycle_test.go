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

package triple_protocol

import (
	"context"
	"dubbo.apache.org/dubbo-go/v3/serialization"
	"errors"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type codecConnProbe struct {
	StreamingClientConn
	send          func(any) error
	receive       func(any) error
	closeResponse func() error
}

func (c *codecConnProbe) Send(v any) error {
	if c.send != nil {
		return c.send(v)
	}
	return nil
}
func (c *codecConnProbe) Receive(v any) error {
	if c.receive != nil {
		return c.receive(v)
	}
	return nil
}
func (c *codecConnProbe) CloseRequest() error { return nil }
func (c *codecConnProbe) CloseResponse() error {
	if c.closeResponse != nil {
		return c.closeResponse()
	}
	return nil
}

type codecCloseProbe struct{ calls atomic.Int64 }

func (*codecCloseProbe) Name() string                { return "probe" }
func (*codecCloseProbe) Marshal(any) ([]byte, error) { return nil, nil }
func (*codecCloseProbe) Unmarshal([]byte, any) error { return nil }
func (c *codecCloseProbe) Close() error              { c.calls.Add(1); return nil }
func codecWait(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(3 * time.Second):
		t.Fatal("barrier timed out")
	}
}
func TestCodecStreamLeaseTermination(t *testing.T) {
	terminal := errors.New("remote failure")
	for _, end := range []error{io.EOF, terminal} {
		t.Run(end.Error(), func(t *testing.T) {
			p := &codecCloseProbe{}
			r := &serialization.Resources{}
			r.Add(p)
			require.True(t, r.Begin())
			n := 0
			raw := &codecConnProbe{send: func(any) error { return io.EOF }, receive: func(any) error {
				n++
				if n == 1 {
					return nil
				}
				return end
			}}
			c := &serializationClientConn{StreamingClientConn: raw, resources: r}
			require.NoError(t, c.Receive(nil))
			require.NoError(t, c.CloseRequest())
			require.ErrorIs(t, c.Send(nil), io.EOF)
			require.Zero(t, p.calls.Load())
			require.False(t, c.closed)
			require.ErrorIs(t, c.Receive(nil), end)
			require.True(t, c.closed)
			require.NoError(t, c.CloseResponse())
			require.NoError(t, c.CloseResponse())
			require.NoError(t, r.Close())
			require.EqualValues(t, 1, p.calls.Load())
			require.Equal(t, CodeUnavailable, CodeOf(c.Send(nil)))
		})
	}
}
func TestCodecStreamCloseWaitsForOperations(t *testing.T) {
	for _, operation := range []string{"send", "receive"} {
		t.Run(operation, func(t *testing.T) {
			entered, gate, finished := make(chan struct{}), make(chan struct{}), make(chan struct{})
			var once sync.Once
			unblock := func() { once.Do(func() { close(gate) }) }
			defer unblock()
			p := &codecCloseProbe{}
			r := &serialization.Resources{}
			r.Add(p)
			require.True(t, r.Begin())
			blocked := func(any) error { close(entered); <-gate; return nil }
			closing := make(chan struct{})
			raw := &codecConnProbe{closeResponse: func() error { close(closing); return nil }}
			if operation == "send" {
				raw.send = blocked
			} else {
				raw.receive = blocked
			}
			c := &serializationClientConn{StreamingClientConn: raw, resources: r}
			go func() {
				defer close(finished)
				if operation == "send" {
					_ = c.Send(nil)
				} else {
					_ = c.Receive(nil)
				}
			}()
			codecWait(t, entered)
			closed := make(chan struct{})
			go func() { _ = c.CloseResponse(); close(closed) }()
			codecWait(t, closing)
			select {
			case <-closed:
				t.Fatal("released active operation")
			default:
			}
			require.Zero(t, p.calls.Load())
			unblock()
			codecWait(t, finished)
			codecWait(t, closed)
			require.NoError(t, r.Close())
			require.EqualValues(t, 1, p.calls.Load())
		})
	}
}

type codecCallProbe struct {
	Codec
	phase           string
	entered, gate   chan struct{}
	once            sync.Once
	active          atomic.Bool
	closedDuringUse atomic.Bool
	closes          atomic.Int64
}

func (c *codecCallProbe) pause(phase string) {
	if phase == c.phase {
		c.once.Do(func() { c.active.Store(true); close(c.entered); <-c.gate; c.active.Store(false) })
	}
}
func (c *codecCallProbe) Marshal(v any) ([]byte, error) { c.pause("encode"); return c.Codec.Marshal(v) }
func (c *codecCallProbe) Unmarshal(b []byte, v any) error {
	c.pause("decode")
	return c.Codec.Unmarshal(b, v)
}
func (c *codecCallProbe) Close() error {
	if c.active.Load() {
		c.closedDuringUse.Store(true)
	}
	c.closes.Add(1)
	return nil
}
func TestCodecCompleteCallResourceBoundaries(t *testing.T) {
	for _, phase := range []string{"client-decode", "handler-decode", "handler-encode"} {
		t.Run(phase, func(t *testing.T) {
			c := &codecCallProbe{Codec: &protoBinaryCodec{}, entered: make(chan struct{}), gate: make(chan struct{})}
			if phase == "handler-encode" {
				c.phase = "encode"
			} else {
				c.phase = "decode"
			}
			var unblock sync.Once
			defer unblock.Do(func() { close(c.gate) })
			owner := &serialization.Resources{}
			owner.Add(c)
			var clientOpts []ClientOption
			var handlerOpts []HandlerOption
			if phase == "client-decode" {
				clientOpts = []ClientOption{WithCodec(c), WithSerializationResources(owner)}
			} else {
				_, handlerOpts = ConfiguredSerializationOptions(c, map[string]Codec{}, map[string]Codec{"proto": c}, false, false)
				handlerOpts = append(handlerOpts, WithSerializationResources(owner))
			}
			handler := NewUnaryHandler("/codec.Service/Call", func() any { return &wrapperspb.StringValue{} }, func(context.Context, *Request) (*Response, error) {
				return NewResponse(wrapperspb.String("reply")), nil
			}, handlerOpts...)
			server := httptest.NewUnstartedServer(handler)
			server.EnableHTTP2 = true
			server.StartTLS()
			defer func() { unblock.Do(func() { close(c.gate) }); server.Close() }()
			client := NewClient(server.Client(), server.URL+"/codec.Service", clientOpts...)
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			done := make(chan error, 1)
			go func() {
				done <- client.CallUnary(ctx, NewRequest(wrapperspb.String("request")), "Call", NewResponse(new(wrapperspb.StringValue)))
			}()
			codecWait(t, c.entered)
			stopped := make(chan struct{})
			go func() { _ = owner.Close(); close(stopped) }()
			// Close must remain blocked while the codec callback is running.
			select {
			case <-stopped:
				t.Fatal("codec owner released an active callback")
			case <-time.After(20 * time.Millisecond):
			}
			unblock.Do(func() { close(c.gate) })
			select {
			case err := <-done:
				require.NoError(t, err)
			case <-time.After(3 * time.Second):
				t.Fatal("call did not finish")
			}
			codecWait(t, stopped)
			require.False(t, c.closedDuringUse.Load())
			require.EqualValues(t, 1, c.closes.Load())
			if phase == "client-decode" {
				require.Equal(t, CodeUnavailable, CodeOf(client.CallUnary(ctx, NewRequest(&emptypb.Empty{}), "Call", NewResponse(&emptypb.Empty{}))))
			} else {
				require.Error(t, client.CallUnary(ctx, NewRequest(&emptypb.Empty{}), "Call", NewResponse(&emptypb.Empty{})))
			}
		})
	}
	// Failure before a connection is created must return the lease as well.
	owner := &serialization.Resources{}
	probe := &codecCloseProbe{}
	owner.Add(probe)
	client := NewClient(&http.Client{Transport: codecFailedRoundTripper{}}, "http://codec.local/codec.Service", WithSerializationResources(owner))
	require.Error(t, client.CallUnary(context.Background(), NewRequest(&emptypb.Empty{}), "Call", NewResponse(&emptypb.Empty{})))
	closed := make(chan struct{})
	go func() { _ = owner.Close(); close(closed) }()
	codecWait(t, closed)
	require.EqualValues(t, 1, probe.calls.Load())
}

type codecFailedRoundTripper struct{}

func (codecFailedRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("transport failure")
}

func TestCodecHandlerReplacementKeepsInflightContext(t *testing.T) {
	server := NewServer("", nil)
	oldProbe, newProbe := &codecCloseProbe{}, &codecCloseProbe{}
	oldOwner, newOwner := &serialization.Resources{}, &serialization.Resources{}
	oldOwner.Add(oldProbe)
	newOwner.Add(newProbe)
	entered, gate := make(chan struct{}), make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(gate) }) }
	defer unblock()
	options := func(owner *serialization.Resources) []HandlerOption {
		return []HandlerOption{WithGroup("group"), WithVersion("version"), WithSerializationResources(owner)}
	}
	require.NoError(t, server.RegisterUnaryHandler("/codec.Replace/Call", func() any { return &wrapperspb.StringValue{} }, func(context.Context, *Request) (*Response, error) {
		close(entered)
		<-gate
		return NewResponse(wrapperspb.String("old")), nil
	}, options(oldOwner)...))
	local := httptest.NewUnstartedServer(server.mux)
	local.EnableHTTP2 = true
	local.StartTLS()
	defer func() { unblock(); local.Close() }()
	client := NewClient(local.Client(), local.URL+"/codec.Replace", WithGroup("group"), WithVersion("version"))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	oldDone := make(chan error, 1)
	oldResponse := new(wrapperspb.StringValue)
	go func() {
		oldDone <- client.CallUnary(ctx, NewRequest(wrapperspb.String("old")), "Call", NewResponse(oldResponse))
	}()
	codecWait(t, entered)
	oldClosed := make(chan struct{})
	go func() { _ = oldOwner.Close(); close(oldClosed) }()
	registered := make(chan error, 1)
	go func() {
		registered <- server.RegisterUnaryHandler("/codec.Replace/Call", func() any { return &wrapperspb.StringValue{} }, func(context.Context, *Request) (*Response, error) { return NewResponse(wrapperspb.String("new")), nil }, options(newOwner)...)
	}()
	select {
	case err := <-registered:
		require.NoError(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("registration blocked on an active request")
	}
	newResponse := new(wrapperspb.StringValue)
	require.NoError(t, client.CallUnary(ctx, NewRequest(wrapperspb.String("new")), "Call", NewResponse(newResponse)))
	require.Equal(t, "new", newResponse.Value)
	require.Zero(t, oldProbe.calls.Load())
	require.Zero(t, newProbe.calls.Load())
	unblock()
	select {
	case err := <-oldDone:
		require.NoError(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("old call did not finish")
	}
	require.Equal(t, "old", oldResponse.Value)
	codecWait(t, oldClosed)
	require.EqualValues(t, 1, oldProbe.calls.Load())
	missing := NewClient(local.Client(), local.URL+"/codec.Replace", WithGroup("missing"))
	require.Equal(t, CodeUnimplemented, CodeOf(missing.CallUnary(ctx, NewRequest(&wrapperspb.StringValue{}), "Call", NewResponse(new(wrapperspb.StringValue)))))
	require.NoError(t, newOwner.Close())
	require.EqualValues(t, 1, newProbe.calls.Load())
	require.Equal(t, CodeUnavailable, CodeOf(client.CallUnary(ctx, NewRequest(&wrapperspb.StringValue{}), "Call", NewResponse(new(wrapperspb.StringValue)))))
}
