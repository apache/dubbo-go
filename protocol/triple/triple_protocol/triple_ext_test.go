// Copyright 2021-2023 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package triple_protocol_test

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

import (
	"google.golang.org/protobuf/proto"

	"google.golang.org/protobuf/reflect/protoregistry"
)

import (
	triple "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/assert"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/gen/proto/connect/import/v1/importv1connect"
	pingv1 "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/gen/proto/connect/ping/v1"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/gen/proto/connect/ping/v1/pingv1connect"
)

const errorMessage = "oh no"

// The ping server implementation used in the tests returns errors if the
// client doesn't set a header, and the server sets headers and trailers on the
// response.
const (
	headerValue  = "some header value"
	trailerValue = "some trailer value"
	clientHeader = "Triple-Client-Header"
	// use this header to tell server to mock timeout scenario
	clientTimeoutHeader         = "Triple-Client-Timeout-Header"
	handlerHeader               = "Triple-Handler-Header"
	handlerTrailer              = "Triple-Handler-Trailer"
	clientMiddlewareErrorHeader = "Triple-Trigger-HTTP-Error"

	// since there is no math.MaxInt for go1.16, we need to define it for compatibility
	intSize = 32 << (^uint(0) >> 63) // 32 or 64
	maxInt  = 1<<(intSize-1) - 1
)

func TestServer(t *testing.T) {
	t.Parallel()
	testPing := func(t *testing.T, client pingv1connect.PingServiceClient) { //nolint:thelper
		t.Run("ping", func(t *testing.T) {
			num := int64(42)
			request := triple.NewRequest(&pingv1.PingRequest{Number: num})
			request.Header().Set(clientHeader, headerValue)
			expect := &pingv1.PingResponse{Number: num}
			msg := &pingv1.PingResponse{}
			response := triple.NewResponse(msg)
			err := client.Ping(context.Background(), request, response)
			assert.Nil(t, err)
			assert.Equal(t, response.Msg.(*pingv1.PingResponse), expect)
			assert.Equal(t, response.Header().Values(handlerHeader), []string{headerValue})
			assert.Equal(t, response.Trailer().Values(handlerTrailer), []string{trailerValue})
		})
		t.Run("zero_ping", func(t *testing.T) {
			request := triple.NewRequest(&pingv1.PingRequest{})
			request.Header().Set(clientHeader, headerValue)
			msg := &pingv1.PingResponse{}
			response := triple.NewResponse(msg)
			err := client.Ping(context.Background(), request, response)
			assert.Nil(t, err)
			var expect pingv1.PingResponse
			assert.Equal(t, msg, &expect)
			assert.Equal(t, response.Header().Values(handlerHeader), []string{headerValue})
			assert.Equal(t, response.Trailer().Values(handlerTrailer), []string{trailerValue})
		})
		t.Run("large_ping", func(t *testing.T) {
			// Using a large payload splits the request and response over multiple
			// packets, ensuring that we're managing HTTP readers and writers
			// correctly.
			if testing.Short() {
				t.Skipf("skipping %s test in short mode", t.Name())
			}
			hellos := strings.Repeat("hello", 1024*1024) // ~5mb
			request := triple.NewRequest(&pingv1.PingRequest{Text: hellos})
			request.Header().Set(clientHeader, headerValue)
			msg := &pingv1.PingResponse{}
			response := triple.NewResponse(msg)
			err := client.Ping(context.Background(), request, response)
			assert.Nil(t, err)
			assert.Equal(t, msg.Text, hellos)
			assert.Equal(t, response.Header().Values(handlerHeader), []string{headerValue})
			assert.Equal(t, response.Trailer().Values(handlerTrailer), []string{trailerValue})
		})
		t.Run("ping_error", func(t *testing.T) {
			// please see pingServer.Ping().
			// if we do not send clientHeader: headerValue to pingServer.Ping(), it would return error
			err := client.Ping(
				context.Background(),
				triple.NewRequest(&pingv1.PingRequest{}),
				triple.NewResponse(&pingv1.PingResponse{}),
			)
			assert.Equal(t, triple.CodeOf(err), triple.CodeInvalidArgument)
		})
		t.Run("ping_invalid_timeout", func(t *testing.T) {
			// invalid Deadline
			ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
			defer cancel()
			request := triple.NewRequest(&pingv1.PingRequest{})
			request.Header().Set(clientHeader, headerValue)
			// since we would inspect ctx error before sending request, this invocation would return DeadlineExceeded directly
			err := client.Ping(ctx, request, triple.NewResponse(&pingv1.PingResponse{}))
			assert.Equal(t, triple.CodeOf(err), triple.CodeDeadlineExceeded)
		})
		t.Run("ping_timeout", func(t *testing.T) {
			ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second))
			defer cancel()
			request := triple.NewRequest(&pingv1.PingRequest{})
			request.Header().Set(clientHeader, headerValue)
			// tell server to mock timeout
			request.Header().Set(clientTimeoutHeader, (2 * time.Second).String())
			err := client.Ping(ctx, request, triple.NewResponse(&pingv1.PingResponse{}))
			assert.Equal(t, triple.CodeOf(err), triple.CodeDeadlineExceeded)
		})
	}
	testSum := func(t *testing.T, client pingv1connect.PingServiceClient) { //nolint:thelper
		t.Run("sum", func(t *testing.T) {
			const (
				upTo   = 10
				expect = 55 // 1+10 + 2+9 + ... + 5+6 = 55
			)
			stream, err := client.Sum(context.Background())
			assert.Nil(t, err)
			stream.RequestHeader().Set(clientHeader, headerValue)
			for i := int64(1); i <= upTo; i++ {
				sendErr := stream.Send(&pingv1.SumRequest{Number: i})
				assert.Nil(t, sendErr, assert.Sprintf("send %d", i))
			}
			msg := &pingv1.SumResponse{}
			response := triple.NewResponse(msg)
			err = stream.CloseAndReceive(response)
			assert.Nil(t, err)
			assert.Equal(t, msg.Sum, int64(expect))
			assert.Equal(t, response.Header().Values(handlerHeader), []string{headerValue})
			assert.Equal(t, response.Trailer().Values(handlerTrailer), []string{trailerValue})
		})
		t.Run("sum_error", func(t *testing.T) {
			stream, err := client.Sum(context.Background())
			assert.Nil(t, err)
			if sendErr := stream.Send(&pingv1.SumRequest{Number: 1}); sendErr != nil {
				assert.ErrorIs(t, sendErr, io.EOF)
				assert.Equal(t, triple.CodeOf(sendErr), triple.CodeUnknown)
			}
			err = stream.CloseAndReceive(triple.NewResponse(&pingv1.SumResponse{}))
			assert.Equal(t, triple.CodeOf(err), triple.CodeInvalidArgument)
		})
		t.Run("sum_close_and_receive_without_send", func(t *testing.T) {
			stream, err := client.Sum(context.Background())
			assert.Nil(t, err)
			stream.RequestHeader().Set(clientHeader, headerValue)
			msg := &pingv1.SumResponse{}
			got := triple.NewResponse(msg)
			err = stream.CloseAndReceive(got)
			assert.Nil(t, err)
			assert.Equal(t, msg, &pingv1.SumResponse{}) // receive header only stream
			assert.Equal(t, got.Header().Values(handlerHeader), []string{headerValue})
		})
		t.Run("sum_invalid_timeout", func(t *testing.T) {
			ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
			defer cancel()
			stream, err := client.Sum(ctx)
			assert.Nil(t, err)
			stream.RequestHeader().Set(clientHeader, headerValue)
			msg := &pingv1.SumResponse{}
			got := triple.NewResponse(msg)
			err = stream.CloseAndReceive(got)
			// todo(DMwangnima): for now, invalid timeout would be encoded as "Grpc-Timeout: 0n".
			// it would not inspect err like unary call. We should refer to grpc-go.
			assert.Equal(t, triple.CodeOf(err), triple.CodeDeadlineExceeded)
		})
		t.Run("sum_timeout", func(t *testing.T) {
			ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second))
			defer cancel()
			stream, err := client.Sum(ctx)
			assert.Nil(t, err)
			stream.RequestHeader().Set(clientHeader, headerValue)
			stream.RequestHeader().Set(clientTimeoutHeader, (2 * time.Second).String())
			msg := &pingv1.SumResponse{}
			got := triple.NewResponse(msg)
			err = stream.CloseAndReceive(got)
			assert.Equal(t, triple.CodeOf(err), triple.CodeDeadlineExceeded)
		})
	}
	testCountUp := func(t *testing.T, client pingv1connect.PingServiceClient) { //nolint:thelper
		t.Run("count_up", func(t *testing.T) {
			const upTo = 5
			got := make([]int64, 0, upTo)
			expect := make([]int64, 0, upTo)
			for i := 1; i <= upTo; i++ {
				expect = append(expect, int64(i))
			}
			request := triple.NewRequest(&pingv1.CountUpRequest{Number: upTo})
			request.Header().Set(clientHeader, headerValue)
			stream, err := client.CountUp(context.Background(), request)
			assert.Nil(t, err)
			for stream.Receive(&pingv1.CountUpResponse{}) {
				msg := stream.Msg().(*pingv1.CountUpResponse)
				got = append(got, msg.Number)
			}
			assert.Nil(t, stream.Err())
			assert.Nil(t, stream.Close())
			assert.Equal(t, got, expect)
		})
		t.Run("count_up_error", func(t *testing.T) {
			stream, err := client.CountUp(
				context.Background(),
				triple.NewRequest(&pingv1.CountUpRequest{Number: 1}),
			)
			assert.Nil(t, err)
			for stream.Receive(&pingv1.CountUpResponse{}) {
				t.Fatalf("expected error, shouldn't receive any messages")
			}
			assert.Equal(t, triple.CodeOf(stream.Err()), triple.CodeInvalidArgument)
		})
		t.Run("count_up_invalid_argument", func(t *testing.T) {
			request := triple.NewRequest(&pingv1.CountUpRequest{Number: -1})
			request.Header().Set(clientHeader, headerValue)
			stream, err := client.CountUp(context.Background(), request)
			assert.Nil(t, err)
			for stream.Receive(&pingv1.CountUpResponse{}) {
				t.Fatalf("expected error, shouldn't receive any messages")
			}
			assert.Equal(t, triple.CodeOf(stream.Err()), triple.CodeInvalidArgument)
		})
		t.Run("count_up_invalid_timeout", func(t *testing.T) {
			ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
			defer cancel()
			_, err := client.CountUp(ctx, triple.NewRequest(&pingv1.CountUpRequest{Number: 1}))
			assert.NotNil(t, err)
			assert.Equal(t, triple.CodeOf(err), triple.CodeDeadlineExceeded)
		})
		t.Run("count_up_timeout", func(t *testing.T) {
			ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second))
			defer cancel()
			request := triple.NewRequest(&pingv1.CountUpRequest{Number: 1})
			request.Header().Set(clientHeader, headerValue)
			request.Header().Set(clientTimeoutHeader, (2 * time.Second).String())
			stream, err := client.CountUp(ctx, request)
			assert.Nil(t, err)
			for stream.Receive(&pingv1.CountUpResponse{}) {
				t.Fatalf("expected error, shouldn't receive any messages")
			}
			assert.Equal(t, triple.CodeOf(stream.Err()), triple.CodeDeadlineExceeded)
		})
	}
	testCumSum := func(t *testing.T, client pingv1connect.PingServiceClient, expectSuccess bool) { //nolint:thelper
		t.Run("cumsum", func(t *testing.T) {
			send := []int64{3, 5, 1}
			expect := []int64{3, 8, 9}
			var got []int64
			stream, err := client.CumSum(context.Background())
			assert.Nil(t, err)
			stream.RequestHeader().Set(clientHeader, headerValue)
			if !expectSuccess { // server doesn't support HTTP/2
				failNoHTTP2(t, stream)
				return
			}
			var wg sync.WaitGroup
			wg.Add(2)
			go func() {
				defer wg.Done()
				for i, n := range send {
					err := stream.Send(&pingv1.CumSumRequest{Number: n})
					assert.Nil(t, err, assert.Sprintf("send error #%d", i))
				}
				assert.Nil(t, stream.CloseRequest())
			}()
			go func() {
				defer wg.Done()
				for {
					msg := &pingv1.CumSumResponse{}
					err := stream.Receive(msg)
					if errors.Is(err, io.EOF) {
						break
					}
					assert.Nil(t, err)
					got = append(got, msg.Sum)
				}
				assert.Nil(t, stream.CloseResponse())
			}()
			wg.Wait()
			assert.Equal(t, got, expect)
			assert.Equal(t, stream.ResponseHeader().Values(handlerHeader), []string{headerValue})
			assert.Equal(t, stream.ResponseTrailer().Values(handlerTrailer), []string{trailerValue})
		})
		t.Run("cumsum_error", func(t *testing.T) {
			stream, err := client.CumSum(context.Background())
			assert.Nil(t, err)
			if !expectSuccess { // server doesn't support HTTP/2
				failNoHTTP2(t, stream)
				return
			}
			if sendErr := stream.Send(&pingv1.CumSumRequest{Number: 42}); sendErr != nil {
				assert.ErrorIs(t, sendErr, io.EOF)
				assert.Equal(t, triple.CodeOf(err), triple.CodeUnknown)
			}
			// We didn't send the headers the server expects, so we should now get an
			// error.
			err = stream.Receive(&pingv1.CumSumResponse{})
			assert.Equal(t, triple.CodeOf(err), triple.CodeInvalidArgument)
			assert.True(t, triple.IsWireError(err))
		})
		t.Run("cumsum_empty_stream", func(t *testing.T) {
			stream, err := client.CumSum(context.Background())
			assert.Nil(t, err)
			stream.RequestHeader().Set(clientHeader, headerValue)
			if !expectSuccess { // server doesn't support HTTP/2
				failNoHTTP2(t, stream)
				return
			}
			// Deliberately closing with calling Send to test the behavior of Receive.
			// This test case is based on the grpc interop tests.
			assert.Nil(t, stream.CloseRequest())
			response := &pingv1.CumSumResponse{}
			err = stream.Receive(response)
			assert.True(t, errors.Is(err, io.EOF))
			assert.False(t, triple.IsWireError(err))
			assert.Nil(t, stream.CloseResponse()) // clean-up the stream
		})
		t.Run("cumsum_cancel_after_first_response", func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			stream, err := client.CumSum(ctx)
			assert.Nil(t, err)
			stream.RequestHeader().Set(clientHeader, headerValue)
			if !expectSuccess { // server doesn't support HTTP/2
				failNoHTTP2(t, stream)
				cancel()
				return
			}
			var got []int64
			expect := []int64{42}
			if sendErr := stream.Send(&pingv1.CumSumRequest{Number: 42}); sendErr != nil {
				assert.ErrorIs(t, err, io.EOF)
				assert.Equal(t, triple.CodeOf(err), triple.CodeUnknown)
			}
			msg := &pingv1.CumSumResponse{}
			err = stream.Receive(msg)
			assert.Nil(t, err)
			got = append(got, msg.Sum)
			cancel()
			err = stream.Receive(&pingv1.CumSumResponse{})
			assert.Equal(t, triple.CodeOf(err), triple.CodeCanceled)
			assert.Equal(t, got, expect)
			assert.False(t, triple.IsWireError(err))
		})
		t.Run("cumsum_cancel_before_send", func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			stream, err := client.CumSum(ctx)
			assert.Nil(t, err)
			stream.RequestHeader().Set(clientHeader, headerValue)
			assert.Nil(t, stream.Send(&pingv1.CumSumRequest{Number: 8}))
			cancel()
			// On a subsequent send, ensure that we are still catching context
			// cancellations.
			err = stream.Send(&pingv1.CumSumRequest{Number: 19})
			assert.Equal(t, triple.CodeOf(err), triple.CodeCanceled, assert.Sprintf("%v", err))
			assert.False(t, triple.IsWireError(err))
		})
	}
	testErrors := func(t *testing.T, client pingv1connect.PingServiceClient) { //nolint:thelper
		assertIsHTTPMiddlewareError := func(tb testing.TB, err error) {
			tb.Helper()
			assert.NotNil(tb, err)
			var tripleErr *triple.Error
			assert.True(tb, errors.As(err, &tripleErr))
			expect := newHTTPMiddlewareError()
			assert.Equal(tb, tripleErr.Code(), expect.Code())
			assert.Equal(tb, tripleErr.Message(), expect.Message())
			for k, v := range expect.Meta() {
				assert.Equal(tb, tripleErr.Meta().Values(k), v)
			}
			assert.Equal(tb, len(tripleErr.Details()), len(expect.Details()))
		}
		t.Run("errors", func(t *testing.T) {
			request := triple.NewRequest(&pingv1.FailRequest{
				Code: int32(triple.CodeResourceExhausted),
			})
			request.Header().Set(clientHeader, headerValue)
			response := triple.NewResponse(&pingv1.FailResponse{})
			err := client.Fail(context.Background(), request, response)
			assert.NotNil(t, err)
			var tripleErr *triple.Error
			ok := errors.As(err, &tripleErr)
			assert.True(t, ok, assert.Sprintf("conversion to *triple.Error"))
			assert.True(t, triple.IsWireError(err))
			assert.Equal(t, tripleErr.Code(), triple.CodeResourceExhausted)
			assert.Equal(t, tripleErr.Error(), "resource_exhausted: "+errorMessage)
			assert.Zero(t, tripleErr.Details())
			assert.Equal(t, tripleErr.Meta().Values(handlerHeader), []string{headerValue})
			assert.Equal(t, tripleErr.Meta().Values(handlerTrailer), []string{trailerValue})
		})
		t.Run("middleware_errors_unary", func(t *testing.T) {
			request := triple.NewRequest(&pingv1.PingRequest{})
			request.Header().Set(clientMiddlewareErrorHeader, headerValue)
			res := triple.NewResponse(&pingv1.PingResponse{})
			err := client.Ping(context.Background(), request, res)
			assertIsHTTPMiddlewareError(t, err)
		})
		//t.Run("middleware_errors_streaming", func(t *testing.T) {
		//	request := triple.NewRequest(&pingv1.CountUpRequest{Number: 10})
		//	request.Header().Set(clientMiddlewareErrorHeader, headerValue)
		//	stream, err := client.CountUp(context.Background(), request)
		//	assert.Nil(t, err)
		//	assert.False(t, stream.Receive(&pingv1.CountUpResponse{}))
		//	assertIsHTTPMiddlewareError(t, stream.Err())
		//})
	}
	testMatrix := func(t *testing.T, server *httptest.Server, bidi bool) { //nolint:thelper
		run := func(t *testing.T, stream bool, opts ...triple.ClientOption) {
			t.Helper()
			client := pingv1connect.NewPingServiceClient(server.Client(), server.URL, opts...)
			testPing(t, client)
			if !stream {
				return
			}
			testSum(t, client)
			testCountUp(t, client)
			testCumSum(t, client, bidi)
			testErrors(t, client)
		}
		t.Run("triple", func(t *testing.T) {
			t.Run("proto", func(t *testing.T) {
				run(t, false, triple.WithTriple())
			})
			t.Run("proto_gzip", func(t *testing.T) {
				run(t, false, triple.WithTriple(), triple.WithSendGzip())
			})
			t.Run("json_gzip", func(t *testing.T) {
				run(
					t,
					false,
					triple.WithTriple(),
					triple.WithProtoJSON(),
					triple.WithSendGzip(),
				)
			})
		})
		t.Run("grpc", func(t *testing.T) {
			t.Run("proto", func(t *testing.T) {
				run(t, true)
			})
			t.Run("proto_gzip", func(t *testing.T) {
				run(t, true, triple.WithSendGzip())
			})
			t.Run("json_gzip", func(t *testing.T) {
				run(
					t,
					true,
					triple.WithProtoJSON(),
					triple.WithSendGzip(),
				)
			})
		})
	}

	mux := http.NewServeMux()
	pingRoute, pingHandler := pingv1connect.NewPingServiceHandler(
		pingServer{checkMetadata: true},
	)
	errorWriter := triple.NewErrorWriter()
	// Add some net/http middleware to the ping service so we can also exercise ErrorWriter.
	mux.Handle(pingRoute, http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Header.Get(clientMiddlewareErrorHeader) != "" {
			defer request.Body.Close()
			if _, err := io.Copy(io.Discard, request.Body); err != nil {
				t.Errorf("drain request body: %v", err)
			}
			if !errorWriter.IsSupported(request) {
				t.Errorf("ErrorWriter doesn't support Content-Type %q", request.Header.Get("Content-Type"))
			}
			if err := errorWriter.Write(response, request, newHTTPMiddlewareError()); err != nil {
				t.Errorf("send RPC error from HTTP middleware: %v", err)
			}
			return
		}
		pingHandler.ServeHTTP(response, request)
	}))

	t.Run("http1", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(mux)
		defer server.Close()
		testMatrix(t, server, false /* bidi */)
	})
	t.Run("http2", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewUnstartedServer(mux)
		server.EnableHTTP2 = true
		server.StartTLS()
		defer server.Close()
		testMatrix(t, server, true /* bidi */)
	})
}

func TestConcurrentStreams(t *testing.T) {
	if testing.Short() {
		t.Skipf("skipping %s test in short mode", t.Name())
	}
	t.Parallel()
	mux := http.NewServeMux()
	mux.Handle(pingv1connect.NewPingServiceHandler(pingServer{}))
	server := httptest.NewUnstartedServer(mux)
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.Close)
	var done, start sync.WaitGroup
	start.Add(1)
	for i := 0; i < 100; i++ {
		done.Add(1)
		go func() {
			defer done.Done()
			client := pingv1connect.NewPingServiceClient(server.Client(), server.URL)
			var total int64
			sum, err := client.CumSum(context.Background())
			assert.Nil(t, err)
			start.Wait()
			for i := 0; i < 100; i++ {
				num := rand.Int63n(1000) //NOSONAR
				total += num
				if err := sum.Send(&pingv1.CumSumRequest{Number: num}); err != nil {
					t.Errorf("failed to send request: %v", err)
					break
				}
				resp := &pingv1.CumSumResponse{}
				err := sum.Receive(resp)
				if err != nil {
					t.Errorf("failed to receive from stream: %v", err)
					break
				}
				if total != resp.Sum {
					t.Errorf("expected %d == %d", total, resp.Sum)
					break
				}
			}
			if err := sum.CloseRequest(); err != nil {
				t.Errorf("failed to close request: %v", err)
			}
			if err := sum.CloseResponse(); err != nil {
				t.Errorf("failed to close response: %v", err)
			}
		}()
	}
	start.Done()
	done.Wait()
}

func TestHeaderBasic(t *testing.T) {
	t.Parallel()
	const (
		key  = "Test-Key"
		cval = "client value"
		hval = "client value"
	)

	pingServer := &pluggablePingServer{
		ping: func(ctx context.Context, request *triple.Request) (*triple.Response, error) {
			assert.Equal(t, request.Header().Get(key), cval)
			response := triple.NewResponse(&pingv1.PingResponse{})
			response.Header().Set(key, hval)
			return response, nil
		},
	}
	mux := http.NewServeMux()
	mux.Handle(pingv1connect.NewPingServiceHandler(pingServer))
	server := httptest.NewServer(mux)
	defer server.Close()

	client := pingv1connect.NewPingServiceClient(server.Client(), server.URL)
	request := triple.NewRequest(&pingv1.PingRequest{})
	request.Header().Set(key, cval)
	response := triple.NewResponse(&pingv1.PingResponse{})
	err := client.Ping(context.Background(), request, response)
	assert.Nil(t, err)
	assert.Equal(t, response.Header().Get(key), hval)
}

func TestTimeoutParsing(t *testing.T) {
	t.Parallel()
	const timeout = 10 * time.Minute
	pingServer := &pluggablePingServer{
		ping: func(ctx context.Context, request *triple.Request) (*triple.Response, error) {
			deadline, ok := ctx.Deadline()
			assert.True(t, ok)
			remaining := time.Until(deadline)
			assert.True(t, remaining > 0)
			assert.True(t, remaining <= timeout)
			return triple.NewResponse(&pingv1.PingResponse{}), nil
		},
	}
	mux := http.NewServeMux()
	mux.Handle(pingv1connect.NewPingServiceHandler(pingServer))
	server := httptest.NewServer(mux)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	client := pingv1connect.NewPingServiceClient(server.Client(), server.URL)
	response := triple.NewResponse(&pingv1.PingResponse{})
	err := client.Ping(ctx, triple.NewRequest(&pingv1.PingRequest{}), response)
	assert.Nil(t, err)
}

func TestFailCodec(t *testing.T) {
	t.Parallel()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	server := httptest.NewServer(handler)
	defer server.Close()
	client := pingv1connect.NewPingServiceClient(
		server.Client(),
		server.URL,
		triple.WithCodec(failCodec{}),
	)
	stream, _ := client.CumSum(context.Background())
	err := stream.Send(&pingv1.CumSumRequest{})
	var tripleErr *triple.Error
	assert.NotNil(t, err)
	assert.True(t, errors.As(err, &tripleErr))
	assert.Equal(t, tripleErr.Code(), triple.CodeInternal)
}

func TestContextError(t *testing.T) {
	t.Parallel()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	server := httptest.NewServer(handler)
	defer server.Close()
	client := pingv1connect.NewPingServiceClient(
		server.Client(),
		server.URL,
	)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	stream, err := client.CumSum(ctx)
	assert.Nil(t, err)
	err = stream.Send(nil)
	var tripleErr *triple.Error
	assert.NotNil(t, err)
	assert.True(t, errors.As(err, &tripleErr))
	assert.Equal(t, tripleErr.Code(), triple.CodeCanceled)
	assert.False(t, triple.IsWireError(err))
}

func TestGRPCMarshalStatusError(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.Handle(pingv1connect.NewPingServiceHandler(
		pingServer{},
		triple.WithCodec(failCodec{}),
	))
	server := httptest.NewUnstartedServer(mux)
	server.EnableHTTP2 = true
	server.StartTLS()
	defer server.Close()

	assertInternalError := func(tb testing.TB, opts ...triple.ClientOption) {
		tb.Helper()
		client := pingv1connect.NewPingServiceClient(server.Client(), server.URL, opts...)
		request := triple.NewRequest(&pingv1.FailRequest{Code: int32(triple.CodeResourceExhausted)})
		res := triple.NewResponse(&pingv1.FailResponse{})
		err := client.Fail(context.Background(), request, res)
		tb.Log(err)
		assert.NotNil(t, err)
		var tripleErr *triple.Error
		ok := errors.As(err, &tripleErr)
		assert.True(t, ok)
		assert.Equal(t, tripleErr.Code(), triple.CodeInternal)
		assert.True(
			t,
			strings.HasSuffix(tripleErr.Message(), ": boom"),
		)
	}

	// Only applies to gRPC protocols, where we're marshaling the Status protobuf
	// message to binary.
	assertInternalError(t)
}

func TestGRPCMissingTrailersError(t *testing.T) {
	t.Parallel()

	trimTrailers := func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Header.Del("Te")
			handler.ServeHTTP(&trimTrailerWriter{w: w}, r)
		})
	}

	mux := http.NewServeMux()
	mux.Handle(pingv1connect.NewPingServiceHandler(
		pingServer{checkMetadata: true},
	))
	server := httptest.NewUnstartedServer(trimTrailers(mux))
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.Close)
	client := pingv1connect.NewPingServiceClient(server.Client(), server.URL)

	assertErrorNoTrailers := func(t *testing.T, err error) {
		t.Helper()
		assert.NotNil(t, err)
		var tripleErr *triple.Error
		ok := errors.As(err, &tripleErr)
		assert.True(t, ok)
		assert.Equal(t, tripleErr.Code(), triple.CodeInternal)
		assert.True(
			t,
			strings.HasSuffix(tripleErr.Message(), "gRPC protocol error: no Grpc-Status trailer"),
		)
	}

	assertNilOrEOF := func(t *testing.T, err error) {
		t.Helper()
		if err != nil {
			assert.ErrorIs(t, err, io.EOF)
		}
	}

	t.Run("ping", func(t *testing.T) {
		t.Parallel()
		request := triple.NewRequest(&pingv1.PingRequest{Number: 1, Text: "foobar"})
		res := triple.NewResponse(&pingv1.PingResponse{})
		err := client.Ping(context.Background(), request, res)
		assertErrorNoTrailers(t, err)
	})
	t.Run("sum", func(t *testing.T) {
		t.Parallel()
		stream, err := client.Sum(context.Background())
		assert.Nil(t, err)
		err = stream.Send(&pingv1.SumRequest{Number: 1})
		assertNilOrEOF(t, err)
		res := triple.NewResponse(&pingv1.SumResponse{})
		err = stream.CloseAndReceive(res)
		assertErrorNoTrailers(t, err)
	})
	t.Run("count_up", func(t *testing.T) {
		t.Parallel()
		stream, err := client.CountUp(context.Background(), triple.NewRequest(&pingv1.CountUpRequest{Number: 10}))
		assert.Nil(t, err)
		assert.False(t, stream.Receive(&pingv1.CountUpResponse{}))
		assertErrorNoTrailers(t, stream.Err())
	})
	t.Run("cumsum", func(t *testing.T) {
		t.Parallel()
		stream, err := client.CumSum(context.Background())
		assert.Nil(t, err)
		assertNilOrEOF(t, stream.Send(&pingv1.CumSumRequest{Number: 10}))
		err = stream.Receive(&pingv1.CumSumResponse{})
		assertErrorNoTrailers(t, err)
		assert.Nil(t, stream.CloseResponse())
	})
	t.Run("cumsum_empty_stream", func(t *testing.T) {
		t.Parallel()
		stream, err := client.CumSum(context.Background())
		assert.Nil(t, err)
		assert.Nil(t, stream.CloseRequest())
		err = stream.Receive(&pingv1.CumSumResponse{})
		assertErrorNoTrailers(t, err)
		assert.Nil(t, stream.CloseResponse())
	})
}

func TestUnavailableIfHostInvalid(t *testing.T) {
	t.Parallel()
	client := pingv1connect.NewPingServiceClient(
		http.DefaultClient,
		"https://api.invalid/",
	)
	err := client.Ping(
		context.Background(),
		triple.NewRequest(&pingv1.PingRequest{}),
		triple.NewResponse(&pingv1.PingResponse{}),
	)
	assert.NotNil(t, err)
	assert.Equal(t, triple.CodeOf(err), triple.CodeUnavailable)
}

func TestBidiRequiresHTTP2(t *testing.T) {
	t.Parallel()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := io.WriteString(w, "hello world")
		assert.Nil(t, err)
	})
	server := httptest.NewServer(handler)
	defer server.Close()
	client := pingv1connect.NewPingServiceClient(
		server.Client(),
		server.URL,
	)
	stream, err := client.CumSum(context.Background())
	assert.Nil(t, err)
	assert.Nil(t, stream.Send(&pingv1.CumSumRequest{}))
	assert.Nil(t, stream.CloseRequest())
	err = stream.Receive(&pingv1.CumSumResponse{})
	assert.NotNil(t, err)
	var tripleErr *triple.Error
	assert.True(t, errors.As(err, &tripleErr))
	assert.Equal(t, tripleErr.Code(), triple.CodeUnimplemented)
	assert.True(
		t,
		strings.HasSuffix(tripleErr.Message(), ": bidi streams require at least HTTP/2"),
	)
}

func TestCompressMinBytesClient(t *testing.T) {
	t.Parallel()
	assertContentType := func(tb testing.TB, text, expect string) {
		tb.Helper()
		mux := http.NewServeMux()
		mux.Handle("/", http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			assert.Equal(tb, request.Header.Get("Content-Encoding"), expect)
		}))
		server := httptest.NewServer(mux)
		tb.Cleanup(server.Close)
		err := pingv1connect.NewPingServiceClient(
			server.Client(),
			server.URL,
			triple.WithTriple(),
			triple.WithSendGzip(),
			triple.WithCompressMinBytes(8),
		).Ping(context.Background(), triple.NewRequest(&pingv1.PingRequest{Text: text}), triple.NewResponse(&pingv1.PingResponse{}))
		assert.Nil(tb, err)
	}
	t.Run("request_uncompressed", func(t *testing.T) {
		t.Parallel()
		assertContentType(t, "ping", "")
	})
	t.Run("request_compressed", func(t *testing.T) {
		t.Parallel()
		assertContentType(t, "pingping", "gzip")
	})

	t.Run("request_uncompressed", func(t *testing.T) {
		t.Parallel()
		assertContentType(t, "ping", "")
	})
	t.Run("request_compressed", func(t *testing.T) {
		t.Parallel()
		assertContentType(t, strings.Repeat("ping", 2), "gzip")
	})
}

func TestCompressMinBytes(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.Handle(pingv1connect.NewPingServiceHandler(
		pingServer{},
		triple.WithCompressMinBytes(8),
	))
	server := httptest.NewServer(mux)
	t.Cleanup(func() {
		server.Close()
	})
	client := server.Client()

	getPingResponse := func(t *testing.T, pingText string) *http.Response {
		t.Helper()
		request := &pingv1.PingRequest{Text: pingText}
		requestBytes, err := proto.Marshal(request)
		assert.Nil(t, err)
		req, err := http.NewRequestWithContext(
			context.Background(),
			http.MethodPost,
			server.URL+"/"+pingv1connect.PingServiceName+"/Ping",
			bytes.NewReader(requestBytes),
		)
		assert.Nil(t, err)
		req.Header.Set("Content-Type", "application/proto")
		response, err := client.Do(req)
		assert.Nil(t, err)
		t.Cleanup(func() {
			assert.Nil(t, response.Body.Close())
		})
		return response
	}

	t.Run("response_uncompressed", func(t *testing.T) {
		t.Parallel()
		assert.False(t, getPingResponse(t, "ping").Uncompressed) //nolint:bodyclose
	})

	t.Run("response_compressed", func(t *testing.T) {
		t.Parallel()
		assert.True(t, getPingResponse(t, strings.Repeat("ping", 2)).Uncompressed) //nolint:bodyclose
	})
}

func TestCustomCompression(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	compressionName := "deflate"
	decompressor := func() triple.Decompressor {
		// Need to instantiate with a reader - before decompressing Reset(io.Reader) is called
		return newDeflateReader(strings.NewReader(""))
	}
	compressor := func() triple.Compressor {
		w, err := flate.NewWriter(&strings.Builder{}, flate.DefaultCompression)
		if err != nil {
			t.Fatalf("failed to create flate writer: %v", err)
		}
		return w
	}
	mux.Handle(pingv1connect.NewPingServiceHandler(
		pingServer{},
		triple.WithCompression(compressionName, decompressor, compressor),
	))
	server := httptest.NewServer(mux)
	defer server.Close()

	client := pingv1connect.NewPingServiceClient(server.Client(),
		server.URL,
		triple.WithAcceptCompression(compressionName, decompressor, compressor),
		triple.WithSendCompression(compressionName),
	)
	request := &pingv1.PingRequest{Text: "testing 1..2..3.."}
	msg := &pingv1.PingResponse{}
	response := triple.NewResponse(msg)
	err := client.Ping(context.Background(), triple.NewRequest(request), response)
	assert.Nil(t, err)
	assert.Equal(t, msg, &pingv1.PingResponse{Text: request.Text})
}

func TestClientWithoutGzipSupport(t *testing.T) {
	// See https://github.com/bufbuild/connect-go/pull/349 for why we want to
	// support this. TL;DR is that Microsoft's dapr sidecar can't handle
	// asymmetric compression.
	t.Parallel()
	mux := http.NewServeMux()
	mux.Handle(pingv1connect.NewPingServiceHandler(pingServer{}))
	server := httptest.NewServer(mux)
	defer server.Close()

	client := pingv1connect.NewPingServiceClient(server.Client(),
		server.URL,
		triple.WithAcceptCompression("gzip", nil, nil),
		triple.WithSendGzip(),
	)
	request := &pingv1.PingRequest{Text: "gzip me!"}
	err := client.Ping(context.Background(), triple.NewRequest(request), triple.NewResponse(&pingv1.PingResponse{}))
	assert.NotNil(t, err)
	assert.Equal(t, triple.CodeOf(err), triple.CodeUnknown)
	assert.True(t, strings.Contains(err.Error(), "unknown compression"))
}

func TestInvalidHeaderTimeout(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.Handle(pingv1connect.NewPingServiceHandler(pingServer{}))
	server := httptest.NewServer(mux)
	t.Cleanup(func() {
		server.Close()
	})
	getPingResponseWithTimeout := func(t *testing.T, timeout string) *http.Response {
		t.Helper()
		request, err := http.NewRequestWithContext(
			context.Background(),
			http.MethodPost,
			server.URL+"/"+pingv1connect.PingServiceName+"/Ping",
			strings.NewReader("{}"),
		)
		assert.Nil(t, err)
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Triple-Timeout-Ms", timeout)
		response, err := server.Client().Do(request)
		assert.Nil(t, err)
		t.Cleanup(func() {
			assert.Nil(t, response.Body.Close())
		})
		return response
	}
	t.Run("timeout_non_numeric", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, getPingResponseWithTimeout(t, "10s").StatusCode, http.StatusBadRequest) //nolint:bodyclose
	})
	t.Run("timeout_out_of_range", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, getPingResponseWithTimeout(t, "12345678901").StatusCode, http.StatusBadRequest) //nolint:bodyclose
	})
}

// protocol does not know the concrete type without reflection

//func TestInterceptorReturnsWrongType(t *testing.T) {
//	t.Parallel()
//	mux := http.NewServeMux()
//	mux.Handle(pingv1connect.NewPingServiceHandler(pingServer{}))
//	server := httptest.NewServer(mux)
//	defer server.Close()
//	client := pingv1connect.NewPingServiceClient(server.Client(), server.URL, triple.WithTriple(), triple.WithInterceptors(triple.UnaryInterceptorFunc(func(next triple.UnaryFunc) triple.UnaryFunc {
//		return func(ctx context.Context, request triple.AnyRequest, response triple.AnyResponse) error {
//			if err := next(ctx, request, response); err != nil {
//				return err
//			}
//			return nil
//		}
//	})))
//	err := client.Ping(context.Background(), triple.NewRequest(&pingv1.PingRequest{Text: "hello!"}), triple.NewResponse(&pingv1.PingResponse{}))
//	assert.NotNil(t, err)
//	var tripleErr *triple.Error
//	assert.True(t, errors.As(err, &tripleErr))
//	assert.Equal(t, tripleErr.Code(), triple.CodeInternal)
//	assert.True(t, strings.Contains(tripleErr.Message(), "unexpected client response type"))
//}

func TestHandlerWithReadMaxBytes(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	readMaxBytes := 1024
	mux.Handle(pingv1connect.NewPingServiceHandler(
		pingServer{},
		triple.WithReadMaxBytes(readMaxBytes),
	))
	readMaxBytesMatrix := func(t *testing.T, client pingv1connect.PingServiceClient, compressed bool) {
		t.Helper()
		t.Run("equal_read_max", func(t *testing.T) {
			t.Parallel()
			// Serializes to exactly readMaxBytes (1024) - no errors expected
			pingRequest := &pingv1.PingRequest{Text: strings.Repeat("a", 1021)}
			assert.Equal(t, proto.Size(pingRequest), readMaxBytes)
			err := client.Ping(context.Background(), triple.NewRequest(pingRequest), triple.NewResponse(&pingv1.PingResponse{}))
			assert.Nil(t, err)
		})
		t.Run("read_max_plus_one", func(t *testing.T) {
			t.Parallel()
			// Serializes to readMaxBytes+1 (1025) - expect invalid argument.
			// This will be over the limit after decompression but under with compression.
			pingRequest := &pingv1.PingRequest{Text: strings.Repeat("a", 1022)}
			if compressed {
				compressedSize := gzipCompressedSize(t, pingRequest)
				assert.True(t, compressedSize < readMaxBytes, assert.Sprintf("expected compressed size %d < %d", compressedSize, readMaxBytes))
			}
			err := client.Ping(context.Background(), triple.NewRequest(pingRequest), triple.NewResponse(&pingv1.PingResponse{}))
			assert.NotNil(t, err, assert.Sprintf("expected non-nil error for large message"))
			assert.Equal(t, triple.CodeOf(err), triple.CodeResourceExhausted)
			assert.True(t, strings.HasSuffix(err.Error(), fmt.Sprintf("message size %d is larger than configured max %d", proto.Size(pingRequest), readMaxBytes)))
		})
		t.Run("read_max_large", func(t *testing.T) {
			t.Parallel()
			if testing.Short() {
				t.Skipf("skipping %s test in short mode", t.Name())
			}
			// Serializes to much larger than readMaxBytes (5 MiB)
			pingRequest := &pingv1.PingRequest{Text: strings.Repeat("abcde", 1024*1024)}
			expectedSize := proto.Size(pingRequest)
			// With gzip request compression, the error should indicate the envelope size (before decompression) is too large.
			if compressed {
				expectedSize = gzipCompressedSize(t, pingRequest)
				assert.True(t, expectedSize > readMaxBytes, assert.Sprintf("expected compressed size %d > %d", expectedSize, readMaxBytes))
			}
			err := client.Ping(context.Background(), triple.NewRequest(pingRequest), triple.NewResponse(&pingv1.PingResponse{}))
			assert.NotNil(t, err, assert.Sprintf("expected non-nil error for large message"))
			assert.Equal(t, triple.CodeOf(err), triple.CodeResourceExhausted)
			assert.Equal(t, err.Error(), fmt.Sprintf("resource_exhausted: message size %d is larger than configured max %d", expectedSize, readMaxBytes))
		})
	}
	newHTTP2Server := func(t *testing.T) *httptest.Server {
		t.Helper()
		server := httptest.NewUnstartedServer(mux)
		server.EnableHTTP2 = true
		server.StartTLS()
		t.Cleanup(server.Close)
		return server
	}
	t.Run("triple", func(t *testing.T) {
		t.Parallel()
		server := newHTTP2Server(t)
		client := pingv1connect.NewPingServiceClient(server.Client(), server.URL)
		readMaxBytesMatrix(t, client, false)
	})
	t.Run("connect_gzip", func(t *testing.T) {
		t.Parallel()
		server := newHTTP2Server(t)
		client := pingv1connect.NewPingServiceClient(server.Client(), server.URL, triple.WithSendGzip())
		readMaxBytesMatrix(t, client, true)
	})
	t.Run("grpc", func(t *testing.T) {
		t.Parallel()
		server := newHTTP2Server(t)
		client := pingv1connect.NewPingServiceClient(server.Client(), server.URL)
		readMaxBytesMatrix(t, client, false)
	})
	t.Run("grpc_gzip", func(t *testing.T) {
		t.Parallel()
		server := newHTTP2Server(t)
		client := pingv1connect.NewPingServiceClient(server.Client(), server.URL, triple.WithSendGzip())
		readMaxBytesMatrix(t, client, true)
	})
}

func TestHandlerWithHTTPMaxBytes(t *testing.T) {
	// This is similar to Connect's own ReadMaxBytes option, but applied to the
	// whole stream using the stdlib's http.MaxBytesHandler.
	t.Parallel()
	const readMaxBytes = 128
	mux := http.NewServeMux()
	pingRoute, pingHandler := pingv1connect.NewPingServiceHandler(pingServer{})
	mux.Handle(pingRoute, triple.MaxBytesHandler(pingHandler, readMaxBytes))
	run := func(t *testing.T, client pingv1connect.PingServiceClient, compressed bool) {
		t.Helper()
		t.Run("below_read_max", func(t *testing.T) {
			t.Parallel()
			err := client.Ping(context.Background(), triple.NewRequest(&pingv1.PingRequest{}), triple.NewResponse(&pingv1.PingResponse{}))
			assert.Nil(t, err)
		})
		t.Run("just_above_max", func(t *testing.T) {
			t.Parallel()
			pingRequest := &pingv1.PingRequest{Text: strings.Repeat("a", readMaxBytes*10)}
			err := client.Ping(context.Background(), triple.NewRequest(pingRequest), triple.NewResponse(&pingv1.PingResponse{}))
			if compressed {
				compressedSize := gzipCompressedSize(t, pingRequest)
				assert.True(t, compressedSize < readMaxBytes, assert.Sprintf("expected compressed size %d < %d", compressedSize, readMaxBytes))
				assert.Nil(t, err)
				return
			}
			assert.NotNil(t, err, assert.Sprintf("expected non-nil error for large message"))
			assert.Equal(t, triple.CodeOf(err), triple.CodeResourceExhausted)
		})
		t.Run("read_max_large", func(t *testing.T) {
			t.Parallel()
			if testing.Short() {
				t.Skipf("skipping %s test in short mode", t.Name())
			}
			pingRequest := &pingv1.PingRequest{Text: strings.Repeat("abcde", 1024*1024)}
			if compressed {
				expectedSize := gzipCompressedSize(t, pingRequest)
				assert.True(t, expectedSize > readMaxBytes, assert.Sprintf("expected compressed size %d > %d", expectedSize, readMaxBytes))
			}
			err := client.Ping(context.Background(), triple.NewRequest(pingRequest), triple.NewResponse(&pingv1.PingResponse{}))
			assert.NotNil(t, err, assert.Sprintf("expected non-nil error for large message"))
			assert.Equal(t, triple.CodeOf(err), triple.CodeResourceExhausted)
		})
	}
	newHTTP2Server := func(t *testing.T) *httptest.Server {
		t.Helper()
		server := httptest.NewUnstartedServer(mux)
		server.EnableHTTP2 = true
		server.StartTLS()
		t.Cleanup(server.Close)
		return server
	}
	t.Run("triple", func(t *testing.T) {
		t.Parallel()
		server := newHTTP2Server(t)
		client := pingv1connect.NewPingServiceClient(server.Client(), server.URL)
		run(t, client, false)
	})
	t.Run("connect_gzip", func(t *testing.T) {
		t.Parallel()
		server := newHTTP2Server(t)
		client := pingv1connect.NewPingServiceClient(server.Client(), server.URL, triple.WithSendGzip())
		run(t, client, true)
	})
	t.Run("grpc", func(t *testing.T) {
		t.Parallel()
		server := newHTTP2Server(t)
		client := pingv1connect.NewPingServiceClient(server.Client(), server.URL)
		run(t, client, false)
	})
	t.Run("grpc_gzip", func(t *testing.T) {
		t.Parallel()
		server := newHTTP2Server(t)
		client := pingv1connect.NewPingServiceClient(server.Client(), server.URL, triple.WithSendGzip())
		run(t, client, true)
	})
}

func TestClientWithReadMaxBytes(t *testing.T) {
	t.Parallel()
	createServer := func(tb testing.TB, enableCompression bool) *httptest.Server {
		tb.Helper()
		mux := http.NewServeMux()
		var compressionOption triple.HandlerOption
		if enableCompression {
			compressionOption = triple.WithCompressMinBytes(1)
		} else {
			compressionOption = triple.WithCompressMinBytes(maxInt)
		}
		mux.Handle(pingv1connect.NewPingServiceHandler(pingServer{}, compressionOption))
		server := httptest.NewUnstartedServer(mux)
		server.EnableHTTP2 = true
		server.StartTLS()
		tb.Cleanup(server.Close)
		return server
	}
	serverUncompressed := createServer(t, false)
	serverCompressed := createServer(t, true)
	readMaxBytes := 1024
	readMaxBytesMatrix := func(t *testing.T, client pingv1connect.PingServiceClient, compressed bool) {
		t.Helper()
		t.Run("equal_read_max", func(t *testing.T) {
			t.Parallel()
			// Serializes to exactly readMaxBytes (1024) - no errors expected
			pingRequest := &pingv1.PingRequest{Text: strings.Repeat("a", 1021)}
			assert.Equal(t, proto.Size(pingRequest), readMaxBytes)
			err := client.Ping(context.Background(), triple.NewRequest(pingRequest), triple.NewResponse(&pingv1.PingResponse{}))
			assert.Nil(t, err)
		})
		t.Run("read_max_plus_one", func(t *testing.T) {
			t.Parallel()
			// Serializes to readMaxBytes+1 (1025) - expect resource exhausted.
			// This will be over the limit after decompression but under with compression.
			pingRequest := &pingv1.PingRequest{Text: strings.Repeat("a", 1022)}
			err := client.Ping(context.Background(), triple.NewRequest(pingRequest), triple.NewResponse(&pingv1.PingResponse{}))
			assert.NotNil(t, err, assert.Sprintf("expected non-nil error for large message"))
			assert.Equal(t, triple.CodeOf(err), triple.CodeResourceExhausted)
			assert.True(t, strings.HasSuffix(err.Error(), fmt.Sprintf("message size %d is larger than configured max %d", proto.Size(pingRequest), readMaxBytes)))
		})
		t.Run("read_max_large", func(t *testing.T) {
			t.Parallel()
			if testing.Short() {
				t.Skipf("skipping %s test in short mode", t.Name())
			}
			// Serializes to much larger than readMaxBytes (5 MiB)
			pingRequest := &pingv1.PingRequest{Text: strings.Repeat("abcde", 1024*1024)}
			expectedSize := proto.Size(pingRequest)
			// With gzip response compression, the error should indicate the envelope size (before decompression) is too large.
			if compressed {
				expectedSize = gzipCompressedSize(t, pingRequest)
				assert.True(t, expectedSize > readMaxBytes, assert.Sprintf("expected compressed size %d > %d", expectedSize, readMaxBytes))
			}
			assert.True(t, expectedSize > readMaxBytes, assert.Sprintf("expected compressed size %d > %d", expectedSize, readMaxBytes))
			err := client.Ping(context.Background(), triple.NewRequest(pingRequest), triple.NewResponse(&pingv1.PingResponse{}))
			assert.NotNil(t, err, assert.Sprintf("expected non-nil error for large message"))
			assert.Equal(t, triple.CodeOf(err), triple.CodeResourceExhausted)
			assert.Equal(t, err.Error(), fmt.Sprintf("resource_exhausted: message size %d is larger than configured max %d", expectedSize, readMaxBytes))
		})
	}
	t.Run("triple", func(t *testing.T) {
		t.Parallel()
		client := pingv1connect.NewPingServiceClient(serverUncompressed.Client(), serverUncompressed.URL, triple.WithReadMaxBytes(readMaxBytes))
		readMaxBytesMatrix(t, client, false)
	})
	t.Run("connect_gzip", func(t *testing.T) {
		t.Parallel()
		client := pingv1connect.NewPingServiceClient(serverCompressed.Client(), serverCompressed.URL, triple.WithReadMaxBytes(readMaxBytes))
		readMaxBytesMatrix(t, client, true)
	})
	t.Run("grpc", func(t *testing.T) {
		t.Parallel()
		client := pingv1connect.NewPingServiceClient(serverUncompressed.Client(), serverUncompressed.URL, triple.WithReadMaxBytes(readMaxBytes))
		readMaxBytesMatrix(t, client, false)
	})
	t.Run("grpc_gzip", func(t *testing.T) {
		t.Parallel()
		client := pingv1connect.NewPingServiceClient(serverCompressed.Client(), serverCompressed.URL, triple.WithReadMaxBytes(readMaxBytes))
		readMaxBytesMatrix(t, client, true)
	})
}

func TestHandlerWithSendMaxBytes(t *testing.T) {
	t.Parallel()
	sendMaxBytes := 1024
	sendMaxBytesMatrix := func(t *testing.T, client pingv1connect.PingServiceClient, compressed bool) {
		t.Helper()
		t.Run("equal_send_max", func(t *testing.T) {
			t.Parallel()
			// Serializes to exactly sendMaxBytes (1024) - no errors expected
			pingRequest := &pingv1.PingRequest{Text: strings.Repeat("a", 1021)}
			assert.Equal(t, proto.Size(pingRequest), sendMaxBytes)
			err := client.Ping(context.Background(), triple.NewRequest(pingRequest), triple.NewResponse(&pingv1.PingResponse{}))
			assert.Nil(t, err)
		})
		t.Run("send_max_plus_one", func(t *testing.T) {
			t.Parallel()
			// Serializes to sendMaxBytes+1 (1025) - expect invalid argument.
			// This will be over the limit after decompression but under with compression.
			pingRequest := &pingv1.PingRequest{Text: strings.Repeat("a", 1022)}
			if compressed {
				compressedSize := gzipCompressedSize(t, pingRequest)
				assert.True(t, compressedSize < sendMaxBytes, assert.Sprintf("expected compressed size %d < %d", compressedSize, sendMaxBytes))
			}
			err := client.Ping(context.Background(), triple.NewRequest(pingRequest), triple.NewResponse(&pingv1.PingResponse{}))
			if compressed {
				assert.Nil(t, err)
			} else {
				assert.NotNil(t, err, assert.Sprintf("expected non-nil error for large message"))
				assert.Equal(t, triple.CodeOf(err), triple.CodeResourceExhausted)
				assert.True(t, strings.HasSuffix(err.Error(), fmt.Sprintf("message size %d exceeds sendMaxBytes %d", proto.Size(pingRequest), sendMaxBytes)))
			}
		})
		t.Run("send_max_large", func(t *testing.T) {
			t.Parallel()
			if testing.Short() {
				t.Skipf("skipping %s test in short mode", t.Name())
			}
			// Serializes to much larger than sendMaxBytes (5 MiB)
			pingRequest := &pingv1.PingRequest{Text: strings.Repeat("abcde", 1024*1024)}
			expectedSize := proto.Size(pingRequest)
			// With gzip request compression, the error should indicate the envelope size (before decompression) is too large.
			if compressed {
				expectedSize = gzipCompressedSize(t, pingRequest)
				assert.True(t, expectedSize > sendMaxBytes, assert.Sprintf("expected compressed size %d > %d", expectedSize, sendMaxBytes))
			}
			err := client.Ping(context.Background(), triple.NewRequest(pingRequest), triple.NewResponse(&pingv1.PingResponse{}))
			assert.NotNil(t, err, assert.Sprintf("expected non-nil error for large message"))
			assert.Equal(t, triple.CodeOf(err), triple.CodeResourceExhausted)
			if compressed {
				assert.Equal(t, err.Error(), fmt.Sprintf("resource_exhausted: compressed message size %d exceeds sendMaxBytes %d", expectedSize, sendMaxBytes))
			} else {
				assert.Equal(t, err.Error(), fmt.Sprintf("resource_exhausted: message size %d exceeds sendMaxBytes %d", expectedSize, sendMaxBytes))
			}
		})
	}
	newHTTP2Server := func(t *testing.T, compressed bool, sendMaxBytes int) *httptest.Server {
		t.Helper()
		mux := http.NewServeMux()
		options := []triple.HandlerOption{triple.WithSendMaxBytes(sendMaxBytes)}
		if compressed {
			options = append(options, triple.WithCompressMinBytes(1))
		} else {
			options = append(options, triple.WithCompressMinBytes(maxInt))
		}
		mux.Handle(pingv1connect.NewPingServiceHandler(
			pingServer{},
			options...,
		))
		server := httptest.NewUnstartedServer(mux)
		server.EnableHTTP2 = true
		server.StartTLS()
		t.Cleanup(server.Close)
		return server
	}
	t.Run("triple", func(t *testing.T) {
		t.Parallel()
		server := newHTTP2Server(t, false, sendMaxBytes)
		client := pingv1connect.NewPingServiceClient(server.Client(), server.URL)
		sendMaxBytesMatrix(t, client, false)
	})
	t.Run("connect_gzip", func(t *testing.T) {
		t.Parallel()
		server := newHTTP2Server(t, true, sendMaxBytes)
		client := pingv1connect.NewPingServiceClient(server.Client(), server.URL)
		sendMaxBytesMatrix(t, client, true)
	})
	t.Run("grpc", func(t *testing.T) {
		t.Parallel()
		server := newHTTP2Server(t, false, sendMaxBytes)
		client := pingv1connect.NewPingServiceClient(server.Client(), server.URL)
		sendMaxBytesMatrix(t, client, false)
	})
	t.Run("grpc_gzip", func(t *testing.T) {
		t.Parallel()
		server := newHTTP2Server(t, true, sendMaxBytes)
		client := pingv1connect.NewPingServiceClient(server.Client(), server.URL)
		sendMaxBytesMatrix(t, client, true)
	})
}

func TestClientWithSendMaxBytes(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.Handle(pingv1connect.NewPingServiceHandler(pingServer{}))
	server := httptest.NewUnstartedServer(mux)
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.Close)
	sendMaxBytesMatrix := func(t *testing.T, client pingv1connect.PingServiceClient, sendMaxBytes int, compressed bool) {
		t.Helper()
		t.Run("equal_send_max", func(t *testing.T) {
			t.Parallel()
			// Serializes to exactly sendMaxBytes (1024) - no errors expected
			pingRequest := &pingv1.PingRequest{Text: strings.Repeat("a", 1021)}
			assert.Equal(t, proto.Size(pingRequest), sendMaxBytes)
			err := client.Ping(context.Background(), triple.NewRequest(pingRequest), triple.NewResponse(&pingv1.PingResponse{}))
			assert.Nil(t, err)
		})
		t.Run("send_max_plus_one", func(t *testing.T) {
			t.Parallel()
			// Serializes to sendMaxBytes+1 (1025) - expect resource exhausted.
			pingRequest := &pingv1.PingRequest{Text: strings.Repeat("a", 1022)}
			assert.Equal(t, proto.Size(pingRequest), sendMaxBytes+1)
			err := client.Ping(context.Background(), triple.NewRequest(pingRequest), triple.NewResponse(&pingv1.PingResponse{}))
			if compressed {
				assert.True(t, gzipCompressedSize(t, pingRequest) < sendMaxBytes)
				assert.Nil(t, err, assert.Sprintf("expected nil error for compressed message < sendMaxBytes"))
			} else {
				assert.NotNil(t, err, assert.Sprintf("expected non-nil error for large message"))
				assert.Equal(t, triple.CodeOf(err), triple.CodeResourceExhausted)
				assert.True(t, strings.HasSuffix(err.Error(), fmt.Sprintf("message size %d exceeds sendMaxBytes %d", proto.Size(pingRequest), sendMaxBytes)))
			}
		})
		t.Run("send_max_large", func(t *testing.T) {
			t.Parallel()
			if testing.Short() {
				t.Skipf("skipping %s test in short mode", t.Name())
			}
			// Serializes to much larger than sendMaxBytes (5 MiB)
			pingRequest := &pingv1.PingRequest{Text: strings.Repeat("abcde", 1024*1024)}
			expectedSize := proto.Size(pingRequest)
			// With gzip response compression, the error should indicate the envelope size (before decompression) is too large.
			if compressed {
				expectedSize = gzipCompressedSize(t, pingRequest)
			}
			assert.True(t, expectedSize > sendMaxBytes)
			err := client.Ping(context.Background(), triple.NewRequest(pingRequest), triple.NewResponse(&pingv1.PingResponse{}))
			assert.NotNil(t, err, assert.Sprintf("expected non-nil error for large message"))
			assert.Equal(t, triple.CodeOf(err), triple.CodeResourceExhausted)
			if compressed {
				assert.Equal(t, err.Error(), fmt.Sprintf("resource_exhausted: compressed message size %d exceeds sendMaxBytes %d", expectedSize, sendMaxBytes))
			} else {
				assert.Equal(t, err.Error(), fmt.Sprintf("resource_exhausted: message size %d exceeds sendMaxBytes %d", expectedSize, sendMaxBytes))
			}
		})
	}
	t.Run("triple", func(t *testing.T) {
		t.Parallel()
		sendMaxBytes := 1024
		client := pingv1connect.NewPingServiceClient(server.Client(), server.URL, triple.WithSendMaxBytes(sendMaxBytes))
		sendMaxBytesMatrix(t, client, sendMaxBytes, false)
	})
	t.Run("connect_gzip", func(t *testing.T) {
		t.Parallel()
		sendMaxBytes := 1024
		client := pingv1connect.NewPingServiceClient(server.Client(), server.URL, triple.WithSendMaxBytes(sendMaxBytes), triple.WithSendGzip())
		sendMaxBytesMatrix(t, client, sendMaxBytes, true)
	})
	t.Run("grpc", func(t *testing.T) {
		t.Parallel()
		sendMaxBytes := 1024
		client := pingv1connect.NewPingServiceClient(server.Client(), server.URL, triple.WithSendMaxBytes(sendMaxBytes))
		sendMaxBytesMatrix(t, client, sendMaxBytes, false)
	})
	t.Run("grpc_gzip", func(t *testing.T) {
		t.Parallel()
		sendMaxBytes := 1024
		client := pingv1connect.NewPingServiceClient(server.Client(), server.URL, triple.WithSendMaxBytes(sendMaxBytes), triple.WithSendGzip())
		sendMaxBytesMatrix(t, client, sendMaxBytes, true)
	})
}

func TestBidiStreamServerSendsFirstMessage(t *testing.T) {
	t.Parallel()
	run := func(t *testing.T, opts ...triple.ClientOption) {
		t.Helper()
		headersSent := make(chan struct{})
		pingServer := &pluggablePingServer{
			cumSum: func(ctx context.Context, stream *triple.BidiStream) error {
				close(headersSent)
				return nil
			},
		}
		mux := http.NewServeMux()
		mux.Handle(pingv1connect.NewPingServiceHandler(pingServer))
		server := httptest.NewUnstartedServer(mux)
		server.EnableHTTP2 = true
		server.StartTLS()
		t.Cleanup(server.Close)

		client := pingv1connect.NewPingServiceClient(
			server.Client(),
			server.URL,
			triple.WithClientOptions(opts...),
			triple.WithInterceptors(&assertPeerInterceptor{t}),
		)
		stream, err := client.CumSum(context.Background())
		assert.Nil(t, err)
		t.Cleanup(func() {
			assert.Nil(t, stream.CloseRequest())
			assert.Nil(t, stream.CloseResponse())
		})
		// tolerate EOF when server closes stream concurrently
		if err = stream.Send(nil); err != nil {
			assert.ErrorIs(t, err, io.EOF)
		}
		select {
		case <-time.After(time.Second):
			t.Error("timed out to get request headers")
		case <-headersSent:
		}
	}
	t.Run("grpc", func(t *testing.T) {
		t.Parallel()
		run(t)
	})
}

func TestStreamForServer(t *testing.T) {
	t.Parallel()
	newPingServer := func(pingServer pingv1connect.PingServiceHandler) (pingv1connect.PingServiceClient, *httptest.Server) {
		mux := http.NewServeMux()
		mux.Handle(pingv1connect.NewPingServiceHandler(pingServer))
		server := httptest.NewUnstartedServer(mux)
		server.EnableHTTP2 = true
		server.StartTLS()
		client := pingv1connect.NewPingServiceClient(
			server.Client(),
			server.URL,
		)
		return client, server
	}
	t.Run("not-proto-message", func(t *testing.T) {
		t.Parallel()
		client, server := newPingServer(&pluggablePingServer{
			cumSum: func(ctx context.Context, stream *triple.BidiStream) error {
				return stream.Conn().Send("foobar")
			},
		})
		t.Cleanup(server.Close)
		stream, err := client.CumSum(context.Background())
		assert.Nil(t, err)
		// tolerate EOF when server closes stream concurrently
		if err = stream.Send(nil); err != nil {
			assert.ErrorIs(t, err, io.EOF)
		}
		err = stream.Receive(&pingv1.CumSumResponse{})
		assert.NotNil(t, err)
		assert.Equal(t, triple.CodeOf(err), triple.CodeInternal)
		assert.Nil(t, stream.CloseRequest())
	})
	t.Run("nil-message", func(t *testing.T) {
		t.Parallel()
		client, server := newPingServer(&pluggablePingServer{
			cumSum: func(ctx context.Context, stream *triple.BidiStream) error {
				return stream.Send(nil)
			},
		})
		t.Cleanup(server.Close)
		stream, err := client.CumSum(context.Background())
		assert.Nil(t, err)
		// tolerate EOF when server closes stream concurrently
		if err = stream.Send(nil); err != nil {
			assert.ErrorIs(t, err, io.EOF)
		}
		err = stream.Receive(&pingv1.CumSumResponse{})
		assert.NotNil(t, err)
		assert.Equal(t, triple.CodeOf(err), triple.CodeUnknown)
		assert.Nil(t, stream.CloseRequest())
	})
	t.Run("get-spec", func(t *testing.T) {
		t.Parallel()
		client, server := newPingServer(&pluggablePingServer{
			cumSum: func(ctx context.Context, stream *triple.BidiStream) error {
				assert.Equal(t, stream.Spec().StreamType, triple.StreamTypeBidi)
				assert.Equal(t, stream.Spec().Procedure, pingv1connect.PingServiceCumSumProcedure)
				assert.False(t, stream.Spec().IsClient)
				return nil
			},
		})
		t.Cleanup(server.Close)
		stream, err := client.CumSum(context.Background())
		assert.Nil(t, err)
		// tolerate EOF when server closes stream concurrently
		if err = stream.Send(nil); err != nil {
			assert.ErrorIs(t, err, io.EOF)
		}
		assert.Nil(t, stream.CloseRequest())
	})
	t.Run("server-stream", func(t *testing.T) {
		t.Parallel()
		client, server := newPingServer(&pluggablePingServer{
			countUp: func(ctx context.Context, req *triple.Request, stream *triple.ServerStream) error {
				assert.Equal(t, stream.Conn().Spec().StreamType, triple.StreamTypeServer)
				assert.Equal(t, stream.Conn().Spec().Procedure, pingv1connect.PingServiceCountUpProcedure)
				assert.False(t, stream.Conn().Spec().IsClient)
				assert.Nil(t, stream.Send(&pingv1.CountUpResponse{Number: 1}))
				return nil
			},
		})
		t.Cleanup(server.Close)
		stream, err := client.CountUp(context.Background(), triple.NewRequest(&pingv1.CountUpRequest{}))
		assert.Nil(t, err)
		assert.NotNil(t, stream)
		assert.Nil(t, stream.Close())
	})
	t.Run("server-stream-send", func(t *testing.T) {
		t.Parallel()
		client, server := newPingServer(&pluggablePingServer{
			countUp: func(ctx context.Context, req *triple.Request, stream *triple.ServerStream) error {
				assert.Nil(t, stream.Send(&pingv1.CountUpResponse{Number: 1}))
				return nil
			},
		})
		t.Cleanup(server.Close)
		stream, err := client.CountUp(context.Background(), triple.NewRequest(&pingv1.CountUpRequest{}))
		assert.Nil(t, err)
		assert.True(t, stream.Receive(&pingv1.CountUpResponse{}))
		msg := stream.Msg().(*pingv1.CountUpResponse)
		assert.NotNil(t, msg)
		assert.Equal(t, msg.Number, int64(1))
		assert.Nil(t, stream.Close())
	})
	t.Run("server-stream-send-nil", func(t *testing.T) {
		t.Parallel()
		client, server := newPingServer(&pluggablePingServer{
			countUp: func(ctx context.Context, req *triple.Request, stream *triple.ServerStream) error {
				stream.ResponseHeader().Set("foo", "bar")
				stream.ResponseTrailer().Set("bas", "blah")
				assert.Nil(t, stream.Send(nil))
				return nil
			},
		})
		t.Cleanup(server.Close)
		stream, err := client.CountUp(context.Background(), triple.NewRequest(&pingv1.CountUpRequest{}))
		assert.Nil(t, err)
		assert.False(t, stream.Receive(&pingv1.CountUpResponse{}))
		headers := stream.ResponseHeader()
		assert.NotNil(t, headers)
		assert.Equal(t, headers.Get("foo"), "bar")
		trailers := stream.ResponseTrailer()
		assert.NotNil(t, trailers)
		assert.Equal(t, trailers.Get("bas"), "blah")
		assert.Nil(t, stream.Close())
	})
	t.Run("client-stream", func(t *testing.T) {
		t.Parallel()
		client, server := newPingServer(&pluggablePingServer{
			sum: func(ctx context.Context, stream *triple.ClientStream) (*triple.Response, error) {
				assert.Equal(t, stream.Spec().StreamType, triple.StreamTypeClient)
				assert.Equal(t, stream.Spec().Procedure, pingv1connect.PingServiceSumProcedure)
				assert.False(t, stream.Spec().IsClient)
				assert.True(t, stream.Receive(&pingv1.SumRequest{}))
				msg := stream.Msg().(*pingv1.SumRequest)
				assert.NotNil(t, msg)
				assert.Equal(t, msg.Number, int64(1))
				return triple.NewResponse(&pingv1.SumResponse{Sum: 1}), nil
			},
		})
		t.Cleanup(server.Close)
		stream, err := client.Sum(context.Background())
		assert.Nil(t, err)
		assert.Nil(t, stream.Send(&pingv1.SumRequest{Number: 1}))
		msg := &pingv1.SumResponse{}
		res := triple.NewResponse(msg)
		err = stream.CloseAndReceive(res)
		assert.Nil(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, msg.Sum, int64(1))
	})
	t.Run("client-stream-conn", func(t *testing.T) {
		t.Parallel()
		client, server := newPingServer(&pluggablePingServer{
			sum: func(ctx context.Context, stream *triple.ClientStream) (*triple.Response, error) {
				assert.NotNil(t, stream.Conn().Send("not-proto"))
				return triple.NewResponse(&pingv1.SumResponse{}), nil
			},
		})
		t.Cleanup(server.Close)
		stream, err := client.Sum(context.Background())
		assert.Nil(t, err)
		assert.Nil(t, stream.Send(&pingv1.SumRequest{Number: 1}))
		res := triple.NewResponse(&pingv1.SumResponse{})
		err = stream.CloseAndReceive(res)
		assert.Nil(t, err)
	})
	t.Run("client-stream-send-msg", func(t *testing.T) {
		t.Parallel()
		client, server := newPingServer(&pluggablePingServer{
			sum: func(ctx context.Context, stream *triple.ClientStream) (*triple.Response, error) {
				assert.Nil(t, stream.Conn().Send(&pingv1.SumResponse{Sum: 2}))
				return triple.NewResponse(&pingv1.SumResponse{}), nil
			},
		})
		t.Cleanup(server.Close)
		stream, err := client.Sum(context.Background())
		assert.Nil(t, err)
		assert.Nil(t, stream.Send(&pingv1.SumRequest{Number: 1}))
		res := triple.NewResponse(&pingv1.SumResponse{})
		err = stream.CloseAndReceive(res)
		assert.NotNil(t, err)
		assert.Equal(t, triple.CodeOf(err), triple.CodeUnknown)
	})
}

func TestTripleHTTPErrorCodes(t *testing.T) {
	t.Parallel()
	checkHTTPStatus := func(t *testing.T, tripleCode triple.Code, wantHttpStatus int) {
		t.Helper()
		mux := http.NewServeMux()
		pluggableServer := &pluggablePingServer{
			ping: func(_ context.Context, _ *triple.Request) (*triple.Response, error) {
				return nil, triple.NewError(tripleCode, errors.New("error"))
			},
		}
		mux.Handle(pingv1connect.NewPingServiceHandler(pluggableServer))
		server := httptest.NewServer(mux)
		t.Cleanup(server.Close)
		req, err := http.NewRequestWithContext(
			context.Background(),
			http.MethodPost,
			server.URL+"/"+pingv1connect.PingServiceName+"/Ping",
			strings.NewReader("{}"),
		)
		assert.Nil(t, err)
		req.Header.Set("Content-Type", "application/json")
		resp, err := server.Client().Do(req)
		assert.Nil(t, err)
		defer resp.Body.Close()
		assert.Equal(t, wantHttpStatus, resp.StatusCode)
		connectClient := pingv1connect.NewPingServiceClient(server.Client(), server.URL)
		err = connectClient.Ping(context.Background(), triple.NewRequest(&pingv1.PingRequest{}), triple.NewResponse(&pingv1.PingResponse{}))
		assert.NotNil(t, err)
	}
	t.Run("CodeCanceled-408", func(t *testing.T) {
		t.Parallel()
		checkHTTPStatus(t, triple.CodeCanceled, 408)
	})
	t.Run("CodeUnknown-500", func(t *testing.T) {
		t.Parallel()
		checkHTTPStatus(t, triple.CodeUnknown, 500)
	})
	t.Run("CodeInvalidArgument-400", func(t *testing.T) {
		t.Parallel()
		checkHTTPStatus(t, triple.CodeInvalidArgument, 400)
	})
	t.Run("CodeDeadlineExceeded-408", func(t *testing.T) {
		t.Parallel()
		checkHTTPStatus(t, triple.CodeDeadlineExceeded, 408)
	})
	t.Run("CodeNotFound-404", func(t *testing.T) {
		t.Parallel()
		checkHTTPStatus(t, triple.CodeNotFound, 404)
	})
	t.Run("CodeAlreadyExists-409", func(t *testing.T) {
		t.Parallel()
		checkHTTPStatus(t, triple.CodeAlreadyExists, 409)
	})
	t.Run("CodePermissionDenied-403", func(t *testing.T) {
		t.Parallel()
		checkHTTPStatus(t, triple.CodePermissionDenied, 403)
	})
	t.Run("CodeResourceExhausted-429", func(t *testing.T) {
		t.Parallel()
		checkHTTPStatus(t, triple.CodeResourceExhausted, 429)
	})
	t.Run("CodeFailedPrecondition-412", func(t *testing.T) {
		t.Parallel()
		checkHTTPStatus(t, triple.CodeFailedPrecondition, 412)
	})
	t.Run("CodeAborted-409", func(t *testing.T) {
		t.Parallel()
		checkHTTPStatus(t, triple.CodeAborted, 409)
	})
	t.Run("CodeOutOfRange-400", func(t *testing.T) {
		t.Parallel()
		checkHTTPStatus(t, triple.CodeOutOfRange, 400)
	})
	t.Run("CodeUnimplemented-404", func(t *testing.T) {
		t.Parallel()
		checkHTTPStatus(t, triple.CodeUnimplemented, 404)
	})
	t.Run("CodeInternal-500", func(t *testing.T) {
		t.Parallel()
		checkHTTPStatus(t, triple.CodeInternal, 500)
	})
	t.Run("CodeUnavailable-503", func(t *testing.T) {
		t.Parallel()
		checkHTTPStatus(t, triple.CodeUnavailable, 503)
	})
	t.Run("CodeDataLoss-500", func(t *testing.T) {
		t.Parallel()
		checkHTTPStatus(t, triple.CodeDataLoss, 500)
	})
	t.Run("CodeUnauthenticated-401", func(t *testing.T) {
		t.Parallel()
		checkHTTPStatus(t, triple.CodeUnauthenticated, 401)
	})
	t.Run("100-500", func(t *testing.T) {
		t.Parallel()
		checkHTTPStatus(t, 100, 500)
	})
	t.Run("0-500", func(t *testing.T) {
		t.Parallel()
		checkHTTPStatus(t, 0, 500)
	})
}

func TestFailCompression(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	compressorName := "fail"
	compressor := func() triple.Compressor { return failCompressor{} }
	decompressor := func() triple.Decompressor { return failDecompressor{} }
	mux.Handle(
		pingv1connect.NewPingServiceHandler(
			pingServer{},
			triple.WithCompression(compressorName, decompressor, compressor),
		),
	)
	server := httptest.NewUnstartedServer(mux)
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.Close)
	pingclient := pingv1connect.NewPingServiceClient(
		server.Client(),
		server.URL,
		triple.WithAcceptCompression(compressorName, decompressor, compressor),
		triple.WithSendCompression(compressorName),
	)
	err := pingclient.Ping(
		context.Background(),
		triple.NewRequest(&pingv1.PingRequest{
			Text: "ping",
		}),
		triple.NewResponse(&pingv1.PingResponse{}),
	)
	assert.NotNil(t, err)
	assert.Equal(t, triple.CodeOf(err), triple.CodeInternal)
}

func TestUnflushableResponseWriter(t *testing.T) {
	t.Parallel()
	assertIsFlusherErr := func(t *testing.T, err error) {
		t.Helper()
		assert.NotNil(t, err)
		assert.Equal(t, triple.CodeOf(err), triple.CodeInternal, assert.Sprintf("got %v", err))
		assert.True(
			t,
			// please see checkServerStreamsCanFlush() for detail
			strings.HasSuffix(err.Error(), "unflushableWriter does not implement http.Flusher"),
			assert.Sprintf("error doesn't reference http.Flusher: %s", err.Error()),
		)
	}
	mux := http.NewServeMux()
	path, handler := pingv1connect.NewPingServiceHandler(pingServer{})
	wrapped := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.ServeHTTP(&unflushableWriter{w}, r)
	})
	mux.Handle(path, wrapped)
	server := httptest.NewUnstartedServer(mux)
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.Close)

	tests := []struct {
		name    string
		options []triple.ClientOption
	}{
		{"grpc", nil},
	}
	for _, test := range tests {
		tt := test
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pingclient := pingv1connect.NewPingServiceClient(server.Client(), server.URL, tt.options...)
			stream, err := pingclient.CountUp(
				context.Background(),
				triple.NewRequest(&pingv1.CountUpRequest{Number: 5}),
			)
			if err != nil {
				assertIsFlusherErr(t, err)
				return
			}
			assert.False(t, stream.Receive(&pingv1.CountUpResponse{}))
			assertIsFlusherErr(t, stream.Err())
		})
	}
}

func TestGRPCErrorMetadataIsTrailersOnly(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.Handle(pingv1connect.NewPingServiceHandler(pingServer{}))
	server := httptest.NewUnstartedServer(mux)
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.Close)

	protoBytes, err := proto.Marshal(&pingv1.FailRequest{Code: int32(triple.CodeInternal)})
	assert.Nil(t, err)
	// Manually construct a gRPC prefix. Data is uncompressed, so the first byte
	// is 0. Set the last 4 bytes to the message length.
	var prefix [5]byte
	binary.BigEndian.PutUint32(prefix[1:5], uint32(len(protoBytes)))
	body := append(prefix[:], protoBytes...)
	// Manually send off a gRPC request.
	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		server.URL+pingv1connect.PingServiceFailProcedure,
		bytes.NewReader(body),
	)
	assert.Nil(t, err)
	req.Header.Set("Content-Type", "application/grpc")
	res, err := server.Client().Do(req)
	assert.Nil(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)
	assert.Equal(t, res.Header.Get("Content-Type"), "application/grpc")
	// pingServer.Fail adds handlerHeader and handlerTrailer to the error
	// metadata. The gRPC protocol should send all error metadata as trailers.
	assert.Zero(t, res.Header.Get(handlerHeader))
	assert.Zero(t, res.Header.Get(handlerTrailer))
	_, err = io.Copy(io.Discard, res.Body)
	assert.Nil(t, err)
	assert.Nil(t, res.Body.Close())
	assert.Equal(t, res.Trailer.Get(handlerHeader), headerValue)
	assert.Equal(t, res.Trailer.Get(handlerTrailer), trailerValue)
}

func TestTripleProtocolHeaderSentByDefault(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.Handle(pingv1connect.NewPingServiceHandler(pingServer{}, triple.WithRequireTripleProtocolHeader()))
	server := httptest.NewUnstartedServer(mux)
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.Close)

	client := pingv1connect.NewPingServiceClient(server.Client(), server.URL)
	err := client.Ping(context.Background(), triple.NewRequest(&pingv1.PingRequest{}), triple.NewResponse(&pingv1.PingResponse{}))
	assert.Nil(t, err)

	stream, err := client.CumSum(context.Background())
	assert.Nil(t, err)
	assert.Nil(t, stream.Send(&pingv1.CumSumRequest{}))
	err = stream.Receive(&pingv1.CumSumResponse{})
	assert.Nil(t, err)
	assert.Nil(t, stream.CloseRequest())
	assert.Nil(t, stream.CloseResponse())
}

// todo(DMwangnima): we need to expose this functionality as a configuration to dubbo-go
func TestTripleProtocolHeaderRequired(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.Handle(pingv1connect.NewPingServiceHandler(
		pingServer{},
		triple.WithRequireTripleProtocolHeader(),
	))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	tests := []struct {
		desc    string
		headers http.Header
	}{
		{"empty header", http.Header{}},
		{"invalid version", http.Header{"Triple-Protocol-Version": []string{"0"}}},
	}
	for _, test := range tests {
		req, err := http.NewRequestWithContext(
			context.Background(),
			http.MethodPost,
			server.URL+"/"+pingv1connect.PingServiceName+"/Ping",
			strings.NewReader("{}"),
		)
		assert.Nil(t, err)
		req.Header.Set("Content-Type", "application/json")
		for k, v := range test.headers {
			req.Header[k] = v
		}
		response, err := server.Client().Do(req)
		assert.Nil(t, err)
		assert.Nil(t, response.Body.Close())
		assert.Equal(t, response.StatusCode, http.StatusBadRequest)
	}
}

func TestAllowCustomUserAgent(t *testing.T) {
	t.Parallel()

	const customAgent = "custom"
	mux := http.NewServeMux()
	mux.Handle(pingv1connect.NewPingServiceHandler(&pluggablePingServer{
		ping: func(_ context.Context, req *triple.Request) (*triple.Response, error) {
			agent := req.Header().Get("User-Agent")
			assert.Equal(t, agent, customAgent)
			msg := req.Msg.(*pingv1.PingRequest)
			return triple.NewResponse(&pingv1.PingResponse{Number: msg.Number}), nil
		},
	}))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	// If the user has set a User-Agent, we shouldn't clobber it.
	tests := []struct {
		protocol string
		opts     []triple.ClientOption
	}{
		{"triple", []triple.ClientOption{triple.WithTriple()}},
		{"grpc", nil},
	}
	for _, test := range tests {
		client := pingv1connect.NewPingServiceClient(server.Client(), server.URL, test.opts...)
		req := triple.NewRequest(&pingv1.PingRequest{Number: 42})
		req.Header().Set("User-Agent", customAgent)
		err := client.Ping(context.Background(), req, triple.NewResponse(&pingv1.PingResponse{}))
		assert.Nil(t, err)
	}
}

func TestBidiOverHTTP1(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.Handle(pingv1connect.NewPingServiceHandler(pingServer{}))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	// Clients expecting a full-duplex connection that end up with a simplex
	// HTTP/1.1 connection shouldn't hang. Instead, the server should close the
	// TCP connection.
	client := pingv1connect.NewPingServiceClient(server.Client(), server.URL)
	stream, err := client.CumSum(context.Background())
	assert.Nil(t, err)
	if sendErr := stream.Send(&pingv1.CumSumRequest{Number: 2}); sendErr != nil {
		assert.ErrorIs(t, sendErr, io.EOF)
	}
	err = stream.Receive(&pingv1.CumSumResponse{})
	assert.NotNil(t, err)
	assert.Equal(t, triple.CodeOf(err), triple.CodeUnknown)
	assert.Equal(t, err.Error(), "unknown: HTTP status 505 HTTP Version Not Supported")
	assert.Nil(t, stream.CloseRequest())
	assert.Nil(t, stream.CloseResponse())
}

func TestHandlerReturnsNilResponse(t *testing.T) {
	// When user-written handlers return nil responses _and_ nil errors, ensure
	// that the resulting panic includes at least the name of the procedure.
	t.Parallel()

	var panics int
	recoverPanic := func(_ context.Context, spec triple.Spec, _ http.Header, p any) error {
		panics++
		assert.NotNil(t, p)
		str := fmt.Sprint(p)
		assert.True(
			t,
			strings.Contains(str, spec.Procedure),
			assert.Sprintf("%q does not contain procedure %q", str, spec.Procedure),
		)
		return triple.NewError(triple.CodeInternal, errors.New(str))
	}

	mux := http.NewServeMux()
	mux.Handle(pingv1connect.NewPingServiceHandler(&pluggablePingServer{
		ping: func(ctx context.Context, req *triple.Request) (*triple.Response, error) {
			return nil, nil //nolint: nilnil
		},
		sum: func(ctx context.Context, req *triple.ClientStream) (*triple.Response, error) {
			return nil, nil //nolint: nilnil
		},
	}, triple.WithRecover(recoverPanic)))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := pingv1connect.NewPingServiceClient(server.Client(), server.URL)

	err := client.Ping(context.Background(), triple.NewRequest(&pingv1.PingRequest{}), triple.NewResponse(&pingv1.PingResponse{}))
	assert.NotNil(t, err)
	assert.Equal(t, triple.CodeOf(err), triple.CodeInternal)

	stream, err := client.Sum(context.Background())
	assert.Nil(t, err)
	err = stream.CloseAndReceive(triple.NewResponse(&pingv1.SumResponse{}))
	assert.NotNil(t, err)
	assert.Equal(t, triple.CodeOf(err), triple.CodeInternal)

	assert.Equal(t, panics, 2)
}

// TestBlankImportCodeGeneration tests that services.triple.go is generated with
// blank import statements to services.pb.go so that the service's Descriptor is
// available in the global proto registry.
func TestBlankImportCodeGeneration(t *testing.T) {
	t.Parallel()
	desc, err := protoregistry.GlobalFiles.FindDescriptorByName(importv1connect.ImportServiceName)
	assert.Nil(t, err)
	assert.NotNil(t, desc)
}

func TestDefaultTimeout(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.Handle(pingv1connect.NewPingServiceHandler(pingServer{}))
	server := httptest.NewUnstartedServer(mux)
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.Close)

	defaultTimeout := 3 * time.Second
	serverTimeout := 2 * time.Second
	tests := []struct {
		desc    string
		cliOpts []triple.ClientOption
	}{
		{
			desc: "Triple protocol",
			cliOpts: []triple.ClientOption{
				triple.WithTriple(),
				triple.WithTimeout(defaultTimeout),
			},
		},
		{
			desc: "gRPC protocol",
			cliOpts: []triple.ClientOption{
				triple.WithTimeout(defaultTimeout),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			client := pingv1connect.NewPingServiceClient(server.Client(), server.URL, test.cliOpts...)
			request := triple.NewRequest(&pingv1.PingRequest{})
			request.Header().Set(clientHeader, headerValue)
			// tell server to mock timeout
			request.Header().Set(clientTimeoutHeader, (serverTimeout).String())
			err := client.Ping(context.Background(), request, triple.NewResponse(&pingv1.PingResponse{}))
			assert.Nil(t, err)

			// specify timeout to override default timeout
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()
			newRequest := triple.NewRequest(&pingv1.PingRequest{})
			newRequest.Header().Set(clientHeader, headerValue)
			// tell server to mock timeout
			newRequest.Header().Set(clientTimeoutHeader, (serverTimeout).String())
			newErr := client.Ping(ctx, request, triple.NewResponse(&pingv1.PingResponse{}))
			assert.Equal(t, triple.CodeOf(newErr), triple.CodeDeadlineExceeded)
		})
	}
}

type unflushableWriter struct {
	w http.ResponseWriter
}

func (w *unflushableWriter) Header() http.Header         { return w.w.Header() }
func (w *unflushableWriter) Write(b []byte) (int, error) { return w.w.Write(b) }
func (w *unflushableWriter) WriteHeader(code int)        { w.w.WriteHeader(code) }

func gzipCompressedSize(tb testing.TB, message proto.Message) int {
	tb.Helper()
	uncompressed, err := proto.Marshal(message)
	assert.Nil(tb, err)
	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)
	_, err = gzipWriter.Write(uncompressed)
	assert.Nil(tb, err)
	assert.Nil(tb, gzipWriter.Close())
	return buf.Len()
}

type failCodec struct{}

func (c failCodec) Name() string {
	return "proto"
}

func (c failCodec) Marshal(message any) ([]byte, error) {
	return nil, errors.New("boom")
}

func (c failCodec) Unmarshal(data []byte, message any) error {
	protoMessage, ok := message.(proto.Message)
	if !ok {
		return fmt.Errorf("not protobuf: %T", message)
	}
	return proto.Unmarshal(data, protoMessage)
}

type pluggablePingServer struct {
	pingv1connect.UnimplementedPingServiceHandler

	ping    func(context.Context, *triple.Request) (*triple.Response, error)
	sum     func(context.Context, *triple.ClientStream) (*triple.Response, error)
	countUp func(context.Context, *triple.Request, *triple.ServerStream) error
	cumSum  func(context.Context, *triple.BidiStream) error
}

func (p *pluggablePingServer) Ping(
	ctx context.Context,
	request *triple.Request,
) (*triple.Response, error) {
	return p.ping(ctx, request)
}

func (p *pluggablePingServer) Sum(
	ctx context.Context,
	stream *triple.ClientStream,
) (*triple.Response, error) {
	return p.sum(ctx, stream)
}

func (p *pluggablePingServer) CountUp(
	ctx context.Context,
	req *triple.Request,
	stream *triple.ServerStream,
) error {
	return p.countUp(ctx, req, stream)
}

func (p *pluggablePingServer) CumSum(
	ctx context.Context,
	stream *triple.BidiStream,
) error {
	return p.cumSum(ctx, stream)
}

func failNoHTTP2(tb testing.TB, stream *triple.BidiStreamForClient) {
	tb.Helper()
	if err := stream.Send(&pingv1.CumSumRequest{}); err != nil {
		assert.ErrorIs(tb, err, io.EOF)
		assert.Equal(tb, triple.CodeOf(err), triple.CodeUnknown)
	}
	assert.Nil(tb, stream.CloseRequest())
	err := stream.Receive(&pingv1.CumSumResponse{})
	assert.NotNil(tb, err) // should be 505
	assert.True(
		tb,
		strings.Contains(err.Error(), "HTTP status 505"),
		assert.Sprintf("expected 505, got %v", err),
	)
	assert.Nil(tb, stream.CloseResponse())
}

func expectClientHeader(check bool, req triple.AnyRequest) error {
	if !check {
		return nil
	}
	if err := expectMetadata(req.Header(), "header", clientHeader, headerValue); err != nil {
		return err
	}
	return nil
}

func expectMetadata(meta http.Header, metaType, key, value string) error {
	if got := meta.Get(key); got != value {
		return triple.NewError(triple.CodeInvalidArgument, fmt.Errorf(
			"%s %q: got %q, expected %q",
			metaType,
			key,
			got,
			value,
		))
	}
	return nil
}

type pingServer struct {
	pingv1connect.UnimplementedPingServiceHandler

	checkMetadata bool
}

func (p pingServer) Ping(ctx context.Context, request *triple.Request) (*triple.Response, error) {
	if err := expectClientHeader(p.checkMetadata, request); err != nil {
		return nil, err
	}
	if timeoutStr := request.Header().Get(clientTimeoutHeader); timeoutStr != "" {
		// got timeout instruction
		timeout, _ := time.ParseDuration(timeoutStr)
		time.Sleep(timeout)
	}
	if request.Peer().Addr == "" {
		return nil, triple.NewError(triple.CodeInternal, errors.New("no peer address"))
	}
	if request.Peer().Protocol == "" {
		return nil, triple.NewError(triple.CodeInternal, errors.New("no peer protocol"))
	}
	msg := request.Msg.(*pingv1.PingRequest)
	response := triple.NewResponse(
		&pingv1.PingResponse{
			Number: msg.Number,
			Text:   msg.Text,
		},
	)
	response.Header().Set(handlerHeader, headerValue)
	response.Trailer().Set(handlerTrailer, trailerValue)
	return response, nil
}

func (p pingServer) Fail(ctx context.Context, request *triple.Request) (*triple.Response, error) {
	if err := expectClientHeader(p.checkMetadata, request); err != nil {
		return nil, err
	}
	if request.Peer().Addr == "" {
		return nil, triple.NewError(triple.CodeInternal, errors.New("no peer address"))
	}
	if request.Peer().Protocol == "" {
		return nil, triple.NewError(triple.CodeInternal, errors.New("no peer protocol"))
	}
	msg := request.Msg.(*pingv1.FailRequest)
	err := triple.NewError(triple.Code(msg.Code), errors.New(errorMessage))
	err.Meta().Set(handlerHeader, headerValue)
	err.Meta().Set(handlerTrailer, trailerValue)
	return nil, err
}

func (p pingServer) Sum(
	ctx context.Context,
	stream *triple.ClientStream,
) (*triple.Response, error) {
	if p.checkMetadata {
		if err := expectMetadata(stream.RequestHeader(), "header", clientHeader, headerValue); err != nil {
			return nil, err
		}
	}
	if timeoutStr := stream.RequestHeader().Get(clientTimeoutHeader); timeoutStr != "" {
		// got timeout instruction
		timeout, _ := time.ParseDuration(timeoutStr)
		time.Sleep(timeout)
	}
	if stream.Peer().Addr == "" {
		return nil, triple.NewError(triple.CodeInternal, errors.New("no peer address"))
	}
	if stream.Peer().Protocol == "" {
		return nil, triple.NewError(triple.CodeInternal, errors.New("no peer protocol"))
	}
	var sum int64

	for stream.Receive(&pingv1.SumRequest{}) {
		msg := stream.Msg().(*pingv1.SumRequest)
		sum += msg.Number
	}
	if stream.Err() != nil {
		return nil, stream.Err()
	}
	response := triple.NewResponse(&pingv1.SumResponse{Sum: sum})
	response.Header().Set(handlerHeader, headerValue)
	response.Trailer().Set(handlerTrailer, trailerValue)
	return response, nil
}

func (p pingServer) CountUp(
	ctx context.Context,
	request *triple.Request,
	stream *triple.ServerStream,
) error {
	if err := expectClientHeader(p.checkMetadata, request); err != nil {
		return err
	}
	if timeoutStr := request.Header().Get(clientTimeoutHeader); timeoutStr != "" {
		// got timeout instruction
		timeout, _ := time.ParseDuration(timeoutStr)
		time.Sleep(timeout)
	}
	if request.Peer().Addr == "" {
		return triple.NewError(triple.CodeInternal, errors.New("no peer address"))
	}
	if request.Peer().Protocol == "" {
		return triple.NewError(triple.CodeInternal, errors.New("no peer protocol"))
	}
	msg := request.Msg.(*pingv1.CountUpRequest)
	if msg.Number <= 0 {
		return triple.NewError(triple.CodeInvalidArgument, fmt.Errorf(
			"number must be positive: got %v",
			msg.Number,
		))
	}
	stream.ResponseHeader().Set(handlerHeader, headerValue)
	stream.ResponseTrailer().Set(handlerTrailer, trailerValue)
	for i := int64(1); i <= msg.Number; i++ {
		if err := stream.Send(&pingv1.CountUpResponse{Number: i}); err != nil {
			return err
		}
	}
	return nil
}

func (p pingServer) CumSum(
	ctx context.Context,
	stream *triple.BidiStream,
) error {
	var sum int64
	if p.checkMetadata {
		if err := expectMetadata(stream.RequestHeader(), "header", clientHeader, headerValue); err != nil {
			return err
		}
	}
	if stream.Peer().Addr == "" {
		return triple.NewError(triple.CodeInternal, errors.New("no peer address"))
	}
	if stream.Peer().Protocol == "" {
		return triple.NewError(triple.CodeInternal, errors.New("no peer address"))
	}
	stream.ResponseHeader().Set(handlerHeader, headerValue)
	stream.ResponseTrailer().Set(handlerTrailer, trailerValue)
	for {
		msg := &pingv1.CumSumRequest{}
		err := stream.Receive(msg)
		if errors.Is(err, io.EOF) {
			return nil
		} else if err != nil {
			return err
		}
		sum += msg.Number
		if err := stream.Send(&pingv1.CumSumResponse{Sum: sum}); err != nil {
			return err
		}
	}
}

type deflateReader struct {
	r io.ReadCloser
}

func newDeflateReader(r io.Reader) *deflateReader {
	return &deflateReader{r: flate.NewReader(r)}
}

func (d *deflateReader) Read(p []byte) (int, error) {
	return d.r.Read(p)
}

func (d *deflateReader) Close() error {
	return d.r.Close()
}

func (d *deflateReader) Reset(reader io.Reader) error {
	if resetter, ok := d.r.(flate.Resetter); ok {
		return resetter.Reset(reader, nil)
	}
	return fmt.Errorf("flate reader should implement flate.Resetter")
}

var _ triple.Decompressor = (*deflateReader)(nil)

type trimTrailerWriter struct {
	w http.ResponseWriter
}

func (l *trimTrailerWriter) Header() http.Header {
	return l.w.Header()
}

// Write writes b to underlying writer and counts written size.
func (l *trimTrailerWriter) Write(b []byte) (int, error) {
	l.removeTrailers()
	return l.w.Write(b)
}

// WriteHeader writes s to underlying writer and retains the status.
func (l *trimTrailerWriter) WriteHeader(s int) {
	l.removeTrailers()
	l.w.WriteHeader(s)
}

// Flush implements http.Flusher.
func (l *trimTrailerWriter) Flush() {
	l.removeTrailers()
	if f, ok := l.w.(http.Flusher); ok {
		f.Flush()
	}
}

func (l *trimTrailerWriter) removeTrailers() {
	for _, v := range l.w.Header().Values("Trailer") {
		l.w.Header().Del(v)
	}
	l.w.Header().Del("Trailer")
	for k := range l.w.Header() {
		if strings.HasPrefix(k, http.TrailerPrefix) {
			l.w.Header().Del(k)
		}
	}
}

func newHTTPMiddlewareError() *triple.Error {
	err := triple.NewError(triple.CodeResourceExhausted, errors.New("error from HTTP middleware"))
	err.Meta().Set("Middleware-Foo", "bar")
	return err
}

type failDecompressor struct {
	triple.Decompressor
}

type failCompressor struct{}

func (failCompressor) Write([]byte) (int, error) {
	return 0, errors.New("failCompressor")
}

func (failCompressor) Close() error {
	return errors.New("failCompressor")
}

func (failCompressor) Reset(io.Writer) {}
