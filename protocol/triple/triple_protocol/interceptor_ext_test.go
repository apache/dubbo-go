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

package triple_protocol_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

import (
	triple "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/assert"
	pingv1 "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/gen/proto/connect/ping/v1"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/gen/proto/connect/ping/v1/pingv1connect"
)

func TestOnionOrderingEndToEnd(t *testing.T) {
	t.Parallel()
	// Helper function: returns a function that asserts that there's some value
	// set for header "expect", and adds a value for header "add".
	newInspector := func(expect, add string) func(triple.Spec, http.Header) {
		return func(spec triple.Spec, header http.Header) {
			if expect != "" {
				assert.NotZero(
					t,
					header.Get(expect),
					assert.Sprintf(
						"%s (IsClient %v): header %q missing: %v",
						spec.Procedure,
						spec.IsClient,
						expect,
						header,
					),
				)
			}
			header.Set(add, "v")
		}
	}
	// Helper function: asserts that there's a value present for header keys
	// "one", "two", "three", and "four".
	assertAllPresent := func(spec triple.Spec, header http.Header) {
		for _, key := range []string{"one", "two", "three", "four"} {
			assert.NotZero(
				t,
				header.Get(key),
				assert.Sprintf(
					"%s (IsClient %v): checking all headers, %q missing: %v",
					spec.Procedure,
					spec.IsClient,
					key,
					header,
				),
			)
		}
	}

	// The client and handler interceptor onions are the meat of the test. The
	// order of interceptor execution must be the same for unary and streaming
	// procedures.
	//
	// Requests should fall through the client onion from top to bottom, traverse
	// the network, and then fall through the handler onion from top to bottom.
	// Responses should climb up the handler onion, traverse the network, and
	// then climb up the client onion.
	//
	// The request and response sides of this onion are numbered to make the
	// intended order clear.
	clientOnion := triple.WithInterceptors(
		newHeaderInterceptor(
			// 1 (start). request: should see protocol-related headers
			func(_ triple.Spec, h http.Header) {
				assert.NotZero(t, h.Get("Content-Type"))
			},
			// 12 (end). response: check "one"-"four"
			assertAllPresent,
		),
		newHeaderInterceptor(
			newInspector("", "one"),       // 2. request: add header "one"
			newInspector("three", "four"), // 11. response: check "three", add "four"
		),
		newHeaderInterceptor(
			newInspector("one", "two"),   // 3. request: check "one", add "two"
			newInspector("two", "three"), // 10. response: check "two", add "three"
		),
	)
	handlerOnion := triple.WithInterceptors(
		newHeaderInterceptor(
			newInspector("two", "three"), // 4. request: check "two", add "three"
			newInspector("one", "two"),   // 9. response: check "one", add "two"
		),
		newHeaderInterceptor(
			newInspector("three", "four"), // 5. request: check "three", add "four"
			newInspector("", "one"),       // 8. response: add "one"
		),
		newHeaderInterceptor(
			assertAllPresent, // 6. request: check "one"-"four"
			nil,              // 7. response: no-op
		),
	)

	mux := http.NewServeMux()
	mux.Handle(
		pingv1connect.NewPingServiceHandler(
			pingServer{},
			handlerOnion,
		),
	)
	server := httptest.NewServer(mux)
	defer server.Close()

	client := pingv1connect.NewPingServiceClient(
		server.Client(),
		server.URL,
		clientOnion,
	)

	err := client.Ping(context.Background(), triple.NewRequest(&pingv1.PingRequest{Number: 10}), triple.NewResponse(&pingv1.PingResponse{}))
	assert.Nil(t, err)

	//responses, err := client.CountUp(context.Background(), triple.NewRequest(&pingv1.CountUpRequest{Number: 10}))
	//assert.Nil(t, err)
	//var sum int64
	//for responses.Receive(&pingv1.CountUpResponse{}) {
	//	msg := responses.Msg().(pingv1.CountUpResponse)
	//	sum += msg.Number
	//}
	//assert.Equal(t, sum, 55)
	//assert.Nil(t, responses.Close())
}

//func TestEmptyUnaryInterceptorFunc(t *testing.T) {
//	t.Parallel()
//	mux := http.NewServeMux()
//	interceptor := triple.UnaryInterceptorFunc(func(next triple.UnaryFunc) triple.UnaryFunc {
//		return func(ctx context.Context, request triple.AnyRequest, response triple.AnyResponse) error {
//			return next(ctx, request, response)
//		}
//	})
//	mux.Handle(pingv1connect.NewPingServiceHandler(pingServer{}, triple.WithInterceptors(interceptor)))
//	server := httptest.NewServer(mux)
//	t.Cleanup(server.Close)
//	connectClient := pingv1connect.NewPingServiceClient(server.Client(), server.URL, triple.WithInterceptors(interceptor))
//	err := connectClient.Ping(context.Background(), triple.NewRequest(&pingv1.PingRequest{}), triple.NewResponse(&pingv1.PingResponse{}))
//	assert.Nil(t, err)
//	sumStream, err := connectClient.Sum(context.Background())
//	assert.Nil(t, err)
//	assert.Nil(t, sumStream.Send(&pingv1.SumRequest{Number: 1}))
//	err = sumStream.CloseAndReceive(triple.NewResponse(&pingv1.SumResponse{}))
//	assert.Nil(t, err)
//	countUpStream, err := connectClient.CountUp(context.Background(), triple.NewRequest(&pingv1.CountUpRequest{}))
//	assert.Nil(t, err)
//	for countUpStream.Receive(&pingv1.CountUpResponse{}) {
//		assert.NotNil(t, countUpStream.Msg())
//	}
//	assert.Nil(t, countUpStream.Close())
//}

// headerInterceptor makes it easier to write interceptors that inspect or
// mutate HTTP headers. It applies the same logic to unary and streaming
// procedures, wrapping the send or receive side of the stream as appropriate.
//
// It's useful as a testing harness to make sure that we're chaining
// interceptors in the correct order.
type headerInterceptor struct {
	inspectRequestHeader  func(triple.Spec, http.Header)
	inspectResponseHeader func(triple.Spec, http.Header)
}

// newHeaderInterceptor constructs a headerInterceptor. Nil function pointers
// are treated as no-ops.
func newHeaderInterceptor(
	inspectRequestHeader func(triple.Spec, http.Header),
	inspectResponseHeader func(triple.Spec, http.Header),
) *headerInterceptor {
	interceptor := headerInterceptor{
		inspectRequestHeader:  inspectRequestHeader,
		inspectResponseHeader: inspectResponseHeader,
	}
	if interceptor.inspectRequestHeader == nil {
		interceptor.inspectRequestHeader = func(_ triple.Spec, _ http.Header) {}
	}
	if interceptor.inspectResponseHeader == nil {
		interceptor.inspectResponseHeader = func(_ triple.Spec, _ http.Header) {}
	}
	return &interceptor
}

func (h *headerInterceptor) WrapUnaryHandler(next triple.UnaryHandlerFunc) triple.UnaryHandlerFunc {
	return func(ctx context.Context, request triple.AnyRequest) (triple.AnyResponse, error) {
		h.inspectRequestHeader(request.Spec(), request.Header())
		response, err := next(ctx, request)
		if err != nil {
			return nil, err
		}
		h.inspectResponseHeader(request.Spec(), response.Header())
		return response, nil
	}
}

func (h *headerInterceptor) WrapUnary(next triple.UnaryFunc) triple.UnaryFunc {
	return func(ctx context.Context, req triple.AnyRequest, res triple.AnyResponse) error {
		h.inspectRequestHeader(req.Spec(), req.Header())
		err := next(ctx, req, res)
		if err != nil {
			return err
		}
		h.inspectResponseHeader(req.Spec(), res.Header())
		return nil
	}
}

func (h *headerInterceptor) WrapStreamingClient(next triple.StreamingClientFunc) triple.StreamingClientFunc {
	return func(ctx context.Context, spec triple.Spec) triple.StreamingClientConn {
		return &headerInspectingClientConn{
			StreamingClientConn:   next(ctx, spec),
			inspectRequestHeader:  h.inspectRequestHeader,
			inspectResponseHeader: h.inspectResponseHeader,
		}
	}
}

func (h *headerInterceptor) WrapStreamingHandler(next triple.StreamingHandlerFunc) triple.StreamingHandlerFunc {
	return func(ctx context.Context, conn triple.StreamingHandlerConn) error {
		h.inspectRequestHeader(conn.Spec(), conn.RequestHeader())
		return next(ctx, &headerInspectingHandlerConn{
			StreamingHandlerConn:  conn,
			inspectResponseHeader: h.inspectResponseHeader,
		})
	}
}

type headerInspectingHandlerConn struct {
	triple.StreamingHandlerConn

	inspectedResponse     bool
	inspectResponseHeader func(triple.Spec, http.Header)
}

func (hc *headerInspectingHandlerConn) Send(msg interface{}) error {
	if !hc.inspectedResponse {
		hc.inspectResponseHeader(hc.Spec(), hc.ResponseHeader())
		hc.inspectedResponse = true
	}
	return hc.StreamingHandlerConn.Send(msg)
}

type headerInspectingClientConn struct {
	triple.StreamingClientConn

	inspectedRequest      bool
	inspectRequestHeader  func(triple.Spec, http.Header)
	inspectedResponse     bool
	inspectResponseHeader func(triple.Spec, http.Header)
}

func (cc *headerInspectingClientConn) Send(msg interface{}) error {
	if !cc.inspectedRequest {
		cc.inspectRequestHeader(cc.Spec(), cc.RequestHeader())
		cc.inspectedRequest = true
	}
	return cc.StreamingClientConn.Send(msg)
}

func (cc *headerInspectingClientConn) Receive(msg interface{}) error {
	err := cc.StreamingClientConn.Receive(msg)
	if !cc.inspectedResponse {
		cc.inspectResponseHeader(cc.Spec(), cc.ResponseHeader())
		cc.inspectedResponse = true
	}
	return err
}
