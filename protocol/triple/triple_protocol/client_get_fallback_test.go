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
package triple_protocol

import (
	"context"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/assert"
	pingv1 "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/gen/proto/connect/ping/v1"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientUnaryGetFallback(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.Handle("/triple.ping.v1.PingService/Ping", NewUnaryHandler(
		"/triple.ping.v1.PingService/Ping", func() interface{} {
			return &pingv1.PingRequest{}
		},
		func(ctx context.Context, r *Request) (*Response, error) {
			req := r.Msg.(*pingv1.PingRequest)
			return NewResponse(&pingv1.PingResponse{
				Number: req.Number,
				Text:   req.Text,
			}), nil
		},
		WithIdempotency(IdempotencyNoSideEffects),
	))
	server := httptest.NewUnstartedServer(mux)
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.Close)

	client := NewClient(
		server.Client(),
		server.URL+"/triple.ping.v1.PingService/Ping",
		WithHTTPGet(),
		withHTTPGetMaxURLSize(1, true),
		WithSendGzip(),
	)
	ctx := context.Background()

	err := client.CallUnary(ctx, NewRequest(nil), NewResponse(&pingv1.PingResponse{}))
	assert.Nil(t, err)

	text := strings.Repeat(".", 256)
	msg := &pingv1.PingResponse{}
	err1 := client.CallUnary(ctx, NewRequest(&pingv1.PingRequest{Text: text}), NewResponse(msg))
	assert.Nil(t, err1)
	assert.Equal(t, msg.Text, text)
}
