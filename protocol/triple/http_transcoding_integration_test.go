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
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"golang.org/x/net/http2"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
	tri "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
)

func TestHTTPTranscodingEndToEndHTTP11H2CAndTriple(t *testing.T) {
	_, requestDescriptor, responseDescriptor := registerHTTPTranscodingTestDescriptor(t)
	requestFields := requestDescriptor.Fields()
	responseFields := responseDescriptor.Fields()
	bookDescriptor := requestFields.ByName("book").Message()

	var invocations atomic.Int32
	invoker := &tripleServerTestInvoker{
		url: common.NewURLWithOptions(
			common.WithInterface("triple.http.test.Library"),
			common.WithIp("127.0.0.1"),
			common.WithPort("0"),
			common.WithParamsValue(constant.GroupKey, "g"),
			common.WithParamsValue(constant.VersionKey, "v1"),
		),
		invokeFn: func(_ context.Context, inv base.Invocation) result.Result {
			invocations.Add(1)
			request := inv.Arguments()[0].(proto.Message).ProtoReflect()
			name := request.Get(requestFields.ByName("name")).String()
			count := request.Get(requestFields.ByName("count")).String()
			book := request.Get(requestFields.ByName("book")).Message()
			response := dynamicpb.NewMessage(responseDescriptor)
			response.Set(responseFields.ByName("message"), protoreflect.ValueOfString(
				fmt.Sprintf("%s:%s:%s", name, book.Get(bookDescriptor.Fields().ByName("title")).String(), count),
			))
			res := &result.RPCResult{}
			// Returning a Triple response exercises the same adapter shape used by
			// the generated service path while keeping the dynamic test messages
			// on the protobuf codec path.
			res.SetResult(tri.NewResponse(response))
			return res
		},
	}

	addr := reserveHTTPTranscodingAddr(t)
	cfg := &global.TripleConfig{
		HTTPTranscoding: &global.HTTPTranscodingConfig{Enabled: true},
		Cors:            &global.CorsConfig{AllowOrigins: []string{"https://example.com"}},
	}
	triServer := tri.NewServer(addr, cfg)
	server := &Server{triServer: triServer, cfg: cfg}
	info := &common.ServiceInfo{
		InterfaceName: "triple.http.test.Library",
		Methods: []common.MethodInfo{{
			Name:        "GetBook",
			Type:        constant.CallUnary,
			ReqInitFunc: func() any { return dynamicpb.NewMessage(requestDescriptor) },
		}},
	}
	require.NoError(t, server.handleServiceWithInfo(
		"triple.http.test.Library", invoker, info, tri.WithGroup("g"), tri.WithVersion("v1"),
	))

	errCh := make(chan error, 1)
	go func() { errCh <- triServer.Run(constant.CallHTTP2, nil) }()
	t.Cleanup(func() {
		require.NoError(t, triServer.Stop())
		require.NoError(t, <-errCh)
	})
	waitForHTTPTranscodingListener(t, addr)

	http11Client := &http.Client{Timeout: 3 * time.Second}
	patch := newIntegrationHTTPTranscodingRequest(t, http.MethodPatch, "http://"+addr+"/v1/authors/alice?count=7", `{"title":"book"}`)
	patch.Header.Set("Content-Type", "application/json")
	patch.Header.Set("Origin", "https://example.com")
	patch.Header.Set(tri.TripleServiceGroup, "g")
	patch.Header.Set(tri.TripleServiceVersion, "v1")
	patchResponse, err := http11Client.Do(patch)
	require.NoError(t, err)
	assert.Equal(t, 1, patchResponse.ProtoMajor)
	assert.Equal(t, http.StatusOK, patchResponse.StatusCode)
	assert.Equal(t, `"alice:book:7"`, string(readIntegrationHTTPTranscodingBody(t, patchResponse)))
	assert.Equal(t, "https://example.com", patchResponse.Header.Get("Access-Control-Allow-Origin"))
	assert.Equal(t, int32(1), invocations.Load())

	h2Transport := &http2.Transport{
		AllowHTTP: true,
		DialTLSContext: func(ctx context.Context, network, address string, _ *tls.Config) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, network, address)
		},
	}
	t.Cleanup(h2Transport.CloseIdleConnections)
	h2Client := &http.Client{Transport: h2Transport, Timeout: 3 * time.Second}
	h2Request := newIntegrationHTTPTranscodingRequest(t, http.MethodGet, "http://"+addr+"/v1/books/alice?count=9", "")
	h2Request.Header.Set(tri.TripleServiceGroup, "g")
	h2Request.Header.Set(tri.TripleServiceVersion, "v1")
	h2Response, err := h2Client.Do(h2Request)
	require.NoError(t, err)
	assert.Equal(t, 2, h2Response.ProtoMajor)
	assert.Equal(t, http.StatusOK, h2Response.StatusCode)
	assert.Equal(t, `{"message":"alice::9"}`, string(readIntegrationHTTPTranscodingBody(t, h2Response)))
	assert.Equal(t, int32(2), invocations.Load())

	rpcClient := tri.NewClient(http11Client, "http://"+addr+"/triple.http.test.Library", tri.WithTriple(), tri.WithGroup("g"), tri.WithVersion("v1"))
	rpcRequest := dynamicpb.NewMessage(requestDescriptor)
	rpcRequest.Set(requestFields.ByName("name"), protoreflect.ValueOfString("alice"))
	rpcResponse := dynamicpb.NewMessage(responseDescriptor)
	require.NoError(t, rpcClient.CallUnary(context.Background(), tri.NewRequest(rpcRequest), "GetBook", tri.NewResponse(rpcResponse)))
	assert.Equal(t, "alice::0", rpcMessage(responseFields, rpcResponse))
	assert.Equal(t, int32(3), invocations.Load())

	options := newIntegrationHTTPTranscodingRequest(t, http.MethodOptions, "http://"+addr+"/v1/books/alice", "")
	options.Header.Set("Origin", "https://example.com")
	options.Header.Set("Access-Control-Request-Method", http.MethodGet)
	optionsResponse, err := http11Client.Do(options)
	require.NoError(t, err)
	defer optionsResponse.Body.Close()
	assert.Equal(t, http.StatusNoContent, optionsResponse.StatusCode)
	assert.Contains(t, optionsResponse.Header.Get("Access-Control-Allow-Methods"), http.MethodGet)
}

func rpcMessage(fields protoreflect.FieldDescriptors, response proto.Message) string {
	return response.ProtoReflect().Get(fields.ByName("message")).String()
}

func newIntegrationHTTPTranscodingRequest(t *testing.T, method, target, body string) *http.Request {
	t.Helper()
	request, err := http.NewRequest(method, target, strings.NewReader(body))
	require.NoError(t, err)
	return request
}

func readIntegrationHTTPTranscodingBody(t *testing.T, response *http.Response) []byte {
	t.Helper()
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	return body
}

func reserveHTTPTranscodingAddr(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := listener.Addr().String()
	require.NoError(t, listener.Close())
	return addr
}

func waitForHTTPTranscodingListener(t *testing.T, addr string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("Triple listener %s did not start: %v", addr, err)
		}
		time.Sleep(20 * time.Millisecond)
	}
}
