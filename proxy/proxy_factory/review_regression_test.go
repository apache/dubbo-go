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

package proxy_factory

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

import (
	"google.golang.org/protobuf/types/known/wrapperspb"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/rpcerror"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	tri "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
)

func TestInfoProxyPreservesTripleStatusAndMetadata(t *testing.T) {
	for _, code := range []tri.Code{tri.CodeUnavailable, tri.CodeDeadlineExceeded, tri.CodeBizError} {
		for _, wrapped := range []bool{false, true} {
			t.Run(fmt.Sprintf("code-%d/wrapped-%v", code, wrapped), func(t *testing.T) {
				source := tri.NewError(code, context.DeadlineExceeded)
				source.Meta().Set("X-Attribution", "preserve")
				detail, err := tri.NewErrorDetail(wrapperspb.String("detail"))
				if err != nil {
					t.Fatal(err)
				}
				source.AddDetail(detail)
				var returned error = source
				if wrapped {
					returned = fmt.Errorf("provider wrapper: %w", source)
				}
				info := &common.ServiceInfo{Methods: []common.MethodInfo{{Name: "Call", MethodFunc: func(context.Context, []any, any) (any, error) { return nil, returned }}}}
				invoker := newInfoInvoker(newURL("tri", "ReviewTripleErrors"), info, nil)
				res := invoker.Invoke(context.Background(), invocation.NewRPCInvocation("Call", nil, nil))
				if res.Error() != returned {
					t.Fatalf("existing Triple error replaced: %T", res.Error())
				}
				got := rpcerror.Classify(res.Error())
				if got.ProtocolCode != uint32(code) || got.Category == rpcerror.CategoryUnknown {
					t.Fatalf("double status/conflict: %+v", got)
				}
				if source.Meta().Get("X-Attribution") != "preserve" || len(source.Details()) != 1 || source.Details()[0] != detail {
					t.Fatal("Triple metadata/details lost")
				}
			})
		}
	}
}

func TestInfoProxySanitizesUpstreamWireTripleError(t *testing.T) {
	for _, wrapped := range []bool{false, true} {
		t.Run(fmt.Sprintf("wrapped-%v", wrapped), func(t *testing.T) {
			source := tri.NewWireError(tri.CodeUnavailable, errors.New("upstream temporarily unavailable"))
			source.Meta().Set("Content-Encoding", "gzip")
			source.Meta().Set("Content-Type", "application/json")
			source.Meta().Set("Content-Length", "99")
			source.Meta().Set("Grpc-Encoding", "gzip")
			source.Meta().Set("Triple-Protocol-Version", "0.1.0")
			source.Meta().Set("Date", "Thu, 26 Nov 2026 00:00:00 GMT")
			source.Meta().Set("X-Attribution", "preserve")
			detail, err := tri.NewErrorDetail(wrapperspb.String("wire-detail"))
			if err != nil {
				t.Fatal(err)
			}
			source.AddDetail(detail)

			var returned error = source
			if wrapped {
				returned = fmt.Errorf("upstream wrapper: %w", source)
			}
			info := &common.ServiceInfo{Methods: []common.MethodInfo{{Name: "Call", MethodFunc: func(context.Context, []any, any) (any, error) { return nil, returned }}}}
			invoker := newInfoInvoker(newURL("tri", "ReviewWireTripleErrors"), info, nil)
			res := invoker.Invoke(context.Background(), invocation.NewRPCInvocation("Call", nil, nil))
			if tri.IsWireError(res.Error()) {
				t.Fatal("upstream wire error was not sanitized")
			}
			var gotErr *tri.Error
			if !errors.As(res.Error(), &gotErr) {
				t.Fatalf("expected *triple_protocol.Error, got %T", res.Error())
			}
			if gotErr.Code() != tri.CodeUnavailable || gotErr.Message() != "upstream temporarily unavailable" {
				t.Fatalf("unexpected code/message: code=%v msg=%q", gotErr.Code(), gotErr.Message())
			}
			if gotErr.Meta().Get("X-Attribution") != "preserve" {
				t.Fatalf("business metadata lost: %v", gotErr.Meta())
			}
			for _, forbidden := range []string{"Content-Encoding", "Content-Type", "Content-Length", "Grpc-Encoding", "Triple-Protocol-Version", "Date"} {
				if v := gotErr.Meta().Get(forbidden); v != "" {
					t.Fatalf("transport header %s leaked: %q", forbidden, v)
				}
			}
			if len(gotErr.Details()) != 1 || gotErr.Details()[0] != detail {
				t.Fatalf("error details lost: %v", gotErr.Details())
			}
		})
	}
}

func TestInfoProxyForwardsGzipCompressedUpstreamTripleErrorOverHTTP(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		if _, err := gz.Write([]byte(`{"code":"unavailable","message":"upstream temporarily unavailable"}`)); err != nil {
			t.Errorf("gzip write: %v", err)
			return
		}
		if err := gz.Close(); err != nil {
			t.Errorf("gzip close: %v", err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("X-Upstream-Reason", "overloaded")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write(buf.Bytes())
	}))
	defer upstream.Close()

	upstreamClient := tri.NewClient(upstream.Client(), upstream.URL+"/test.Upstream", tri.WithTriple())
	info := &common.ServiceInfo{
		Methods: []common.MethodInfo{{
			Name: "Call",
			MethodFunc: func(ctx context.Context, _ []any, _ any) (any, error) {
				req := tri.NewRequest(wrapperspb.String("ping"))
				resp := tri.NewResponse(&wrapperspb.StringValue{})
				err := upstreamClient.CallUnary(ctx, req, "Call", resp)
				return nil, err
			},
		}},
	}
	invoker := newInfoInvoker(newURL("tri", "test.Downstream"), info, nil)

	downstreamHandler := tri.NewUnaryHandler(
		"/test.Downstream/Call",
		func() any { return &wrapperspb.StringValue{} },
		func(ctx context.Context, req *tri.Request) (*tri.Response, error) {
			res := invoker.Invoke(ctx, invocation.NewRPCInvocation("Call", []any{req.Msg}, nil))
			if res.Error() != nil {
				return nil, res.Error()
			}
			return tri.NewResponse(res.Result()), nil
		},
	)
	downstream := httptest.NewServer(downstreamHandler)
	defer downstream.Close()

	client := tri.NewClient(downstream.Client(), downstream.URL+"/test.Downstream", tri.WithTriple())
	err := client.CallUnary(context.Background(), tri.NewRequest(wrapperspb.String("ping")), "Call", tri.NewResponse(&wrapperspb.StringValue{}))
	if err == nil {
		t.Fatal("expected error from downstream server")
	}
	if tri.CodeOf(err) != tri.CodeUnavailable {
		t.Fatalf("expected CodeUnavailable, got %v (%v)", tri.CodeOf(err), err)
	}
	if !strings.Contains(err.Error(), "upstream temporarily unavailable") {
		t.Fatalf("upstream error message degraded: %v", err)
	}
	var triErr *tri.Error
	if !errors.As(err, &triErr) {
		t.Fatalf("expected *triple_protocol.Error, got %T", err)
	}
	if triErr.Meta().Get("X-Upstream-Reason") != "overloaded" {
		t.Fatalf("expected business metadata to be preserved, got %v", triErr.Meta())
	}
	if gotEnc := triErr.Meta().Get("Content-Encoding"); gotEnc != "" {
		t.Fatalf("expected uncompressed downstream error response, got Content-Encoding=%q", gotEnc)
	}
}
