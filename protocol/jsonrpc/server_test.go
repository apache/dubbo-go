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

package jsonrpc

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

// sendHTTPRequest writes an HTTP request to conn and returns the parsed response.
// A read deadline is set to avoid hanging when the server may not respond
// (e.g. valid content type but no registered service).
func sendHTTPRequest(t *testing.T, conn net.Conn, contentType string) (*http.Response, error) {
	t.Helper()

	err := conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	require.NoError(t, err)

	req := "POST /test HTTP/1.1\r\n" +
		"Host: localhost\r\n" +
		"Content-Type: " + contentType + "\r\n" +
		"Content-Length: 0\r\n" +
		"\r\n"
	_, err = conn.Write([]byte(req))
	require.NoError(t, err)

	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	return resp, err
}

func TestServeRequest_ServiceNotFound(t *testing.T) {
	GetProtocol()

	serverConn, clientConn := net.Pipe()
	defer require.NoError(t, clientConn.Close())
	defer require.NoError(t, serverConn.Close())

	header := map[string]string{
		"Path":         "com.example.UnregisteredService",
		"HttpMethod":   "POST",
		"Content-Type": "application/json",
	}
	body := []byte(`{"jsonrpc":"2.0","method":"com.example.UnregisteredService.SayHello","id":1}`)

	err := serveRequest(context.Background(), header, body, serverConn)
	require.Error(t, err)
	require.Contains(t, err.Error(), "service not found")
}

func TestContextFromRequestPreservesRequestContext(t *testing.T) {
	type contextKey struct{}
	requestCtx, cancel := context.WithCancel(context.WithValue(context.Background(), contextKey{}, "request-value"))
	defer cancel()
	request := httptest.NewRequestWithContext(requestCtx, http.MethodPost, "/test", nil)

	ctx := contextFromRequest(request)
	require.Equal(t, "request-value", ctx.Value(contextKey{}))
	cancel()
	require.Error(t, ctx.Err())
}

func TestHandlePkg_ContentType(t *testing.T) {
	tests := []struct {
		name            string
		contentType     string
		wantUnsupported bool
	}{
		{
			name:            "unsupported content type",
			contentType:     "text/html",
			wantUnsupported: true,
		},
		{
			name:            "malformed content type",
			contentType:     ";;;invalid",
			wantUnsupported: true,
		},
		{
			name:            "json with charset",
			contentType:     "application/json; charset=utf-8",
			wantUnsupported: false,
		},
		{
			name:            "json-rpc with charset",
			contentType:     "application/json-rpc; charset=utf-8",
			wantUnsupported: false,
		},
		{
			name:            "plain json",
			contentType:     "application/json",
			wantUnsupported: false,
		},
		{
			name:            "plain json-rpc",
			contentType:     "application/json-rpc",
			wantUnsupported: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serverConn, clientConn := net.Pipe()
			defer func() {
				require.NoError(t, clientConn.Close())
			}()
			defer func() {
				require.NoError(t, serverConn.Close())
			}()

			s := NewServer()
			go s.handlePkg(serverConn)

			resp, err := sendHTTPRequest(t, clientConn, tt.contentType)
			if err != nil {
				t.Fatalf("failed to read response: %v", err)
			}
			defer func() {
				require.NoError(t, resp.Body.Close())
			}()

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			if tt.wantUnsupported {
				require.Equal(t, 500, resp.StatusCode)
				require.Contains(t, string(body), "unsupported content type",
					"response body should contain 'unsupported content type'")
			} else {
				require.NotContains(t, string(body), "unsupported content type",
					"%s should be accepted as valid content type", tt.contentType)
			}
		})
	}
}

type blockingInvoker struct {
	base.BaseInvoker
	started  chan struct{}
	canceled chan struct{}
}

func (i *blockingInvoker) Invoke(ctx context.Context, _ base.Invocation) result.Result {
	close(i.started)
	<-ctx.Done()
	close(i.canceled)
	return &result.RPCResult{Err: ctx.Err()}
}

func TestHandlePkgCancelsInvocationWhenClientDisconnects(t *testing.T) {
	const servicePath = "context-cancel-test"
	protocol := GetProtocol().(*JsonrpcProtocol)
	invoker := &blockingInvoker{
		BaseInvoker: *base.NewBaseInvoker(common.NewURLWithOptions(
			common.WithProtocol(JSONRPC),
			common.WithPath("/"+servicePath),
		)),
		started:  make(chan struct{}),
		canceled: make(chan struct{}),
	}
	protocol.SetExporterMap(servicePath, NewJsonrpcExporter(servicePath, invoker, protocol.ExporterMap()))
	t.Cleanup(func() { protocol.ExporterMap().Delete(servicePath) })

	server := NewServer()
	serverConn, clientConn := net.Pipe()
	handleDone := make(chan struct{})
	go func() {
		server.handlePkg(serverConn)
		close(handleDone)
	}()

	body := `{"jsonrpc":"2.0","method":"Blocked","params":[],"id":1}`
	request := "POST /" + servicePath + " HTTP/1.1\r\n" +
		"Host: localhost\r\n" +
		"Content-Type: application/json\r\n" +
		"Content-Length: " + fmt.Sprint(len(body)) + "\r\n\r\n" + body
	_, err := clientConn.Write([]byte(request))
	require.NoError(t, err)

	select {
	case <-invoker.started:
	case <-time.After(time.Second):
		t.Fatal("invoker was not started")
	}
	require.NoError(t, clientConn.Close())

	select {
	case <-invoker.canceled:
	case <-time.After(time.Second):
		t.Fatal("invocation context was not canceled after client disconnect")
	}
	select {
	case <-handleDone:
	case <-time.After(time.Second):
		t.Fatal("connection handler did not exit")
	}
	require.NoError(t, serverConn.Close())
}

type orderedResponseInvoker struct {
	base.BaseInvoker
	firstStarted chan struct{}
	secondReady  chan struct{}
	releaseFirst chan struct{}
}

type signalingResult struct {
	ready chan struct{}
}

func (r signalingResult) MarshalJSON() ([]byte, error) {
	close(r.ready)
	return json.Marshal("fast")
}

func (i *orderedResponseInvoker) Invoke(_ context.Context, invocation base.Invocation) result.Result {
	switch invocation.MethodName() {
	case "Slow":
		close(i.firstStarted)
		<-i.releaseFirst
		return &result.RPCResult{Rest: "slow"}
	case "Fast":
		return &result.RPCResult{Rest: signalingResult{ready: i.secondReady}}
	default:
		return &result.RPCResult{Err: fmt.Errorf("unexpected method %s", invocation.MethodName())}
	}
}

func TestHandlePkgPreservesPipelinedResponseOrder(t *testing.T) {
	const servicePath = "response-order-test"
	protocol := GetProtocol().(*JsonrpcProtocol)
	invoker := &orderedResponseInvoker{
		BaseInvoker:  *base.NewBaseInvoker(common.NewURLWithOptions(common.WithProtocol(JSONRPC))),
		firstStarted: make(chan struct{}),
		secondReady:  make(chan struct{}),
		releaseFirst: make(chan struct{}),
	}
	protocol.SetExporterMap(servicePath, NewJsonrpcExporter(servicePath, invoker, protocol.ExporterMap()))
	t.Cleanup(func() { protocol.ExporterMap().Delete(servicePath) })

	serverConn, clientConn := net.Pipe()
	handleDone := make(chan struct{})
	go func() {
		NewServer().handlePkg(serverConn)
		close(handleDone)
	}()

	var releaseOnce sync.Once
	releaseFirst := func() { releaseOnce.Do(func() { close(invoker.releaseFirst) }) }
	t.Cleanup(func() {
		releaseFirst()
		_ = clientConn.Close()
		select {
		case <-handleDone:
		case <-time.After(time.Second):
			t.Error("connection handler did not exit")
		}
	})
	require.NoError(t, clientConn.SetReadDeadline(time.Now().Add(3*time.Second)))

	type response struct {
		id     int
		result string
		err    error
	}
	responses := make(chan response, 2)
	go func() {
		reader := bufio.NewReader(clientConn)
		for range 2 {
			httpResponse, err := http.ReadResponse(reader, nil)
			if err != nil {
				responses <- response{err: err}
				return
			}
			var payload struct {
				ID     int    `json:"id"`
				Result string `json:"result"`
			}
			err = json.NewDecoder(httpResponse.Body).Decode(&payload)
			httpResponse.Body.Close()
			responses <- response{id: payload.ID, result: payload.Result, err: err}
		}
	}()

	makeRequest := func(method string, id int) string {
		body := fmt.Sprintf(`{"jsonrpc":"2.0","method":%q,"params":[],"id":%d}`, method, id)
		return "POST /" + servicePath + " HTTP/1.1\r\n" +
			"Host: localhost\r\n" +
			"Content-Type: application/json\r\n" +
			"Content-Length: " + fmt.Sprint(len(body)) + "\r\n\r\n" + body
	}

	writeDone := make(chan error, 1)
	go func() {
		_, err := clientConn.Write([]byte(makeRequest("Slow", 1) + makeRequest("Fast", 2)))
		writeDone <- err
	}()
	select {
	case <-invoker.firstStarted:
	case <-time.After(time.Second):
		t.Fatal("first invocation was not started")
	}
	select {
	case <-invoker.secondReady:
	case <-time.After(time.Second):
		t.Fatal("second response was not encoded")
	}
	require.NoError(t, <-writeDone)

	select {
	case got := <-responses:
		t.Fatalf("received response %d before the first request completed", got.id)
	case <-time.After(100 * time.Millisecond):
	}

	releaseFirst()
	readResponse := func() response {
		select {
		case got := <-responses:
			return got
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for response")
			return response{}
		}
	}
	first := readResponse()
	require.NoError(t, first.err)
	require.Equal(t, 1, first.id)
	require.Equal(t, "slow", first.result)
	second := readResponse()
	require.NoError(t, second.err)
	require.Equal(t, 2, second.id)
	require.Equal(t, "fast", second.result)
}

type boundedWindowInvoker struct {
	base.BaseInvoker
	started      chan struct{}
	encoded      chan struct{}
	releaseFirst chan struct{}
}

type boundedWindowResult struct {
	encoded chan struct{}
}

func (r boundedWindowResult) MarshalJSON() ([]byte, error) {
	r.encoded <- struct{}{}
	return json.Marshal("done")
}

func (i *boundedWindowInvoker) Invoke(_ context.Context, invocation base.Invocation) result.Result {
	i.started <- struct{}{}
	if invocation.MethodName() == "Blocked" {
		<-i.releaseFirst
	}
	return &result.RPCResult{Rest: boundedWindowResult{encoded: i.encoded}}
}

func TestHandlePkgBoundsPipelinedRequestWindow(t *testing.T) {
	const (
		servicePath  = "bounded-request-window-test"
		requestCount = maxRequestWindowPerConnection + 1
	)
	protocol := GetProtocol().(*JsonrpcProtocol)
	invoker := &boundedWindowInvoker{
		BaseInvoker:  *base.NewBaseInvoker(common.NewURLWithOptions(common.WithProtocol(JSONRPC))),
		started:      make(chan struct{}, requestCount),
		encoded:      make(chan struct{}, requestCount),
		releaseFirst: make(chan struct{}),
	}
	protocol.SetExporterMap(servicePath, NewJsonrpcExporter(servicePath, invoker, protocol.ExporterMap()))
	t.Cleanup(func() { protocol.ExporterMap().Delete(servicePath) })

	serverConn, clientConn := net.Pipe()
	handleDone := make(chan struct{})
	go func() {
		NewServer().handlePkg(serverConn)
		close(handleDone)
	}()

	var releaseOnce sync.Once
	releaseFirst := func() { releaseOnce.Do(func() { close(invoker.releaseFirst) }) }
	t.Cleanup(func() {
		releaseFirst()
		_ = clientConn.Close()
		select {
		case <-handleDone:
		case <-time.After(time.Second):
			t.Error("connection handler did not exit")
		}
	})
	require.NoError(t, clientConn.SetDeadline(time.Now().Add(5*time.Second)))

	makeRequest := func(method string, id int) string {
		body := fmt.Sprintf(`{"jsonrpc":"2.0","method":%q,"params":[],"id":%d}`, method, id)
		return "POST /" + servicePath + " HTTP/1.1\r\n" +
			"Host: localhost\r\n" +
			"Content-Type: application/json\r\n" +
			"Content-Length: " + fmt.Sprint(len(body)) + "\r\n\r\n" + body
	}

	requests := makeRequest("Blocked", 1)
	for id := 2; id <= requestCount; id++ {
		requests += makeRequest("Fast", id)
	}
	writeDone := make(chan error, 1)
	go func() {
		_, err := clientConn.Write([]byte(requests))
		writeDone <- err
	}()

	waitForSignals := func(signals <-chan struct{}, count int, description string) {
		for range count {
			select {
			case <-signals:
			case <-time.After(2 * time.Second):
				t.Fatalf("timed out waiting for %s %d", description, count)
			}
		}
	}
	waitForSignals(invoker.started, maxRequestWindowPerConnection, "started invocations")
	waitForSignals(invoker.encoded, maxRequestWindowPerConnection-1, "encoded responses")
	select {
	case <-invoker.started:
		t.Fatalf("more than %d invocations started before the response window advanced", maxRequestWindowPerConnection)
	case <-time.After(100 * time.Millisecond):
	}

	readDone := make(chan error, 1)
	go func() {
		reader := bufio.NewReader(clientConn)
		for range requestCount {
			response, err := http.ReadResponse(reader, nil)
			if err != nil {
				readDone <- err
				return
			}
			_, err = io.Copy(io.Discard, response.Body)
			closeErr := response.Body.Close()
			if err != nil {
				readDone <- err
				return
			}
			if closeErr != nil {
				readDone <- closeErr
				return
			}
		}
		readDone <- nil
	}()

	releaseFirst()
	waitForSignals(invoker.started, 1, "invocations after advancing the response window")
	require.NoError(t, <-writeDone)
	require.NoError(t, <-readDone)
}

type requestTimeoutInvoker struct {
	base.BaseInvoker
	longStarted   chan struct{}
	shortTimedOut chan struct{}
	longCanceled  chan struct{}
	releaseLong   chan struct{}
}

func (i *requestTimeoutInvoker) Invoke(ctx context.Context, invocation base.Invocation) result.Result {
	switch invocation.MethodName() {
	case "Long":
		close(i.longStarted)
		select {
		case <-ctx.Done():
			close(i.longCanceled)
			return &result.RPCResult{Err: ctx.Err()}
		case <-i.releaseLong:
			return &result.RPCResult{Rest: "long"}
		}
	case "Short":
		<-ctx.Done()
		close(i.shortTimedOut)
		return &result.RPCResult{Err: ctx.Err()}
	default:
		return &result.RPCResult{Err: fmt.Errorf("unexpected method %s", invocation.MethodName())}
	}
}

func TestHandlePkgIsolatesPipelinedRequestTimeouts(t *testing.T) {
	const servicePath = "request-timeout-test"
	protocol := GetProtocol().(*JsonrpcProtocol)
	invoker := &requestTimeoutInvoker{
		BaseInvoker:   *base.NewBaseInvoker(common.NewURLWithOptions(common.WithProtocol(JSONRPC))),
		longStarted:   make(chan struct{}),
		shortTimedOut: make(chan struct{}),
		longCanceled:  make(chan struct{}),
		releaseLong:   make(chan struct{}),
	}
	protocol.SetExporterMap(servicePath, NewJsonrpcExporter(servicePath, invoker, protocol.ExporterMap()))
	t.Cleanup(func() { protocol.ExporterMap().Delete(servicePath) })

	serverConn, clientConn := net.Pipe()
	handleDone := make(chan struct{})
	go func() {
		NewServer().handlePkg(serverConn)
		close(handleDone)
	}()

	var releaseOnce sync.Once
	releaseLong := func() { releaseOnce.Do(func() { close(invoker.releaseLong) }) }
	t.Cleanup(func() {
		releaseLong()
		_ = clientConn.Close()
		select {
		case <-handleDone:
		case <-time.After(time.Second):
			t.Error("connection handler did not exit")
		}
	})
	require.NoError(t, clientConn.SetReadDeadline(time.Now().Add(3*time.Second)))

	makeRequest := func(method string, id int, timeout time.Duration) string {
		body := fmt.Sprintf(`{"jsonrpc":"2.0","method":%q,"params":[],"id":%d}`, method, id)
		return "POST /" + servicePath + " HTTP/1.1\r\n" +
			"Host: localhost\r\n" +
			"Content-Type: application/json\r\n" +
			"Timeout: " + timeout.String() + "\r\n" +
			"Content-Length: " + fmt.Sprint(len(body)) + "\r\n\r\n" + body
	}

	writeDone := make(chan error, 1)
	go func() {
		_, err := clientConn.Write([]byte(
			makeRequest("Long", 1, 5*time.Second) + makeRequest("Short", 2, 50*time.Millisecond),
		))
		writeDone <- err
	}()

	select {
	case <-invoker.longStarted:
	case <-time.After(time.Second):
		t.Fatal("long invocation was not started")
	}
	select {
	case <-invoker.shortTimedOut:
	case <-time.After(time.Second):
		t.Fatal("short invocation did not reach its context deadline")
	}
	require.NoError(t, <-writeDone)
	select {
	case <-invoker.longCanceled:
		t.Fatal("short request timeout canceled the long request")
	case <-time.After(250 * time.Millisecond):
	}

	releaseLong()
	reader := bufio.NewReader(clientConn)
	for id := 1; id <= 2; id++ {
		httpResponse, err := http.ReadResponse(reader, nil)
		require.NoError(t, err)
		var payload struct {
			ID int `json:"id"`
		}
		require.NoError(t, json.NewDecoder(httpResponse.Body).Decode(&payload))
		require.NoError(t, httpResponse.Body.Close())
		require.Equal(t, id, payload.ID)
	}
}
