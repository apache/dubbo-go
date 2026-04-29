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

package server

import (
	"context"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	_ "dubbo.apache.org/dubbo-go/v3/filter/accesslog"         // Register default provider filters for exportServices in this integration test.
	_ "dubbo.apache.org/dubbo-go/v3/filter/echo"              // Register default provider filters for exportServices in this integration test.
	_ "dubbo.apache.org/dubbo-go/v3/filter/exec_limit"        // Register default provider filters for exportServices in this integration test.
	_ "dubbo.apache.org/dubbo-go/v3/filter/generic"           // Register default provider filters for exportServices in this integration test.
	_ "dubbo.apache.org/dubbo-go/v3/filter/graceful_shutdown" // Register default provider filters for exportServices in this integration test.
	_ "dubbo.apache.org/dubbo-go/v3/filter/token"             // Register default provider filters for exportServices in this integration test.
	_ "dubbo.apache.org/dubbo-go/v3/filter/tps"               // Register default provider filters for exportServices in this integration test.
	"dubbo.apache.org/dubbo-go/v3/protocol"
	_ "dubbo.apache.org/dubbo-go/v3/protocol/protocolwrapper" // Register protocol wrappers used by exportServices in this integration test.
	_ "dubbo.apache.org/dubbo-go/v3/protocol/triple"          // Register the Triple protocol used by exportServices in this integration test.
	tri "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
	_ "dubbo.apache.org/dubbo-go/v3/proxy/proxy_factory" // Register the default proxy factory used during export in this integration test.
)

const (
	tripleCaseRouteHelloBody    = "\"Hello test\""
	tripleCaseRouteNotFoundBody = "404 page not found\n"
)

type TripleCaseRouteService struct{}

func (s *TripleCaseRouteService) Reference() string {
	return "com.example.GreetService"
}

func (s *TripleCaseRouteService) SayHello(context.Context, *emptypb.Empty) (*wrapperspb.StringValue, error) {
	return wrapperspb.String("Hello test"), nil
}

func TestTripleCaseInsensitiveRoute(t *testing.T) {
	t.Run("pascal case registration accepts lowercase fallback", func(t *testing.T) {
		runTripleCaseRouteIntegration(
			t,
			"com.example.GreetService",
			"SayHello",
			"/com.example.GreetService/SayHello",
			[]tripleRouteExpectation{
				{
					name:       "canonical method name",
					path:       "/com.example.GreetService/SayHello",
					wantStatus: http.StatusOK,
					wantBody:   tripleCaseRouteHelloBody,
				},
				{
					name:       "lowercase method name fallback",
					path:       "/com.example.GreetService/sayHello",
					wantStatus: http.StatusOK,
					wantBody:   tripleCaseRouteHelloBody,
				},
				{
					name:       "unknown method remains not found",
					path:       "/com.example.GreetService/DeleteUser",
					wantStatus: http.StatusNotFound,
					wantBody:   tripleCaseRouteNotFoundBody,
				},
			},
		)
	})

	t.Run("camel case registration accepts uppercase fallback", func(t *testing.T) {
		runTripleCaseRouteIntegration(
			t,
			"com.example.JavaStyleGreetService",
			"sayHello",
			"/com.example.JavaStyleGreetService/sayHello",
			[]tripleRouteExpectation{
				{
					name:       "registered camel case method name",
					path:       "/com.example.JavaStyleGreetService/sayHello",
					wantStatus: http.StatusOK,
					wantBody:   tripleCaseRouteHelloBody,
				},
				{
					name:       "uppercase method name fallback",
					path:       "/com.example.JavaStyleGreetService/SayHello",
					wantStatus: http.StatusOK,
					wantBody:   tripleCaseRouteHelloBody,
				},
				{
					name:       "unknown method remains not found",
					path:       "/com.example.JavaStyleGreetService/DeleteUser",
					wantStatus: http.StatusNotFound,
					wantBody:   tripleCaseRouteNotFoundBody,
				},
			},
		)
	})
}

type tripleRouteExpectation struct {
	name       string
	path       string
	wantStatus int
	wantBody   string
}

func runTripleCaseRouteIntegration(
	t *testing.T,
	interfaceName string,
	methodName string,
	readyPath string,
	expectations []tripleRouteExpectation,
) {
	t.Helper()

	port := testFreePort(t)
	srv, err := NewServer(
		WithServerProtocol(
			protocol.WithTriple(),
			protocol.WithIp("127.0.0.1"),
			protocol.WithPort(port),
		),
	)
	require.NoError(t, err)

	service := &TripleCaseRouteService{}
	info := &common.ServiceInfo{
		InterfaceName: interfaceName,
		ServiceType:   service,
		Methods: []common.MethodInfo{
			{
				Name: methodName,
				Type: constant.CallUnary,
				ReqInitFunc: func() any {
					return &emptypb.Empty{}
				},
				MethodFunc: func(ctx context.Context, args []any, handler any) (any, error) {
					req := args[0].(*emptypb.Empty)
					res, callErr := handler.(*TripleCaseRouteService).SayHello(ctx, req)
					if callErr != nil {
						return nil, callErr
					}
					return tri.NewResponse(res), nil
				},
			},
		},
	}

	err = srv.Register(service, info, WithInterface(info.InterfaceName), WithNotRegister())
	require.NoError(t, err)
	ctx := context.Background()
	err = srv.exportServices(ctx)
	require.NoError(t, err)

	t.Cleanup(func() {
		extension.GetProtocol(constant.TriProtocol).Destroy()
	})

	client := &http.Client{Timeout: 2 * time.Second}
	baseURL := "http://127.0.0.1:" + strconv.Itoa(port)
	waitTripleRouteReady(t, client, baseURL+readyPath)

	for _, tt := range expectations {
		t.Run(tt.name, func(t *testing.T) {
			status, body, reqErr := tripleRouteRequest(client, baseURL+tt.path)
			require.NoError(t, reqErr)
			assert.Equal(t, tt.wantStatus, status)
			assert.Equal(t, tt.wantBody, body)
		})
	}
}

func testFreePort(t *testing.T) int {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() {
		_ = listener.Close()
	}()

	return listener.Addr().(*net.TCPAddr).Port
}

func tripleRouteRequest(client *http.Client, url string) (int, string, error) {
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader("{}"))
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, "", err
	}
	return resp.StatusCode, string(body), nil
}

func waitTripleRouteReady(t *testing.T, client *http.Client, url string) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	var lastStatus int
	var lastBody string
	var lastErr error

	for time.Now().Before(deadline) {
		lastStatus, lastBody, lastErr = tripleRouteRequest(client, url)
		if lastErr == nil && lastStatus == http.StatusOK && lastBody == tripleCaseRouteHelloBody {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Fatalf("triple route not ready: status=%d body=%q err=%v", lastStatus, lastBody, lastErr)
}
