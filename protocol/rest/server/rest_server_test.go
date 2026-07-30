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
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

import (
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/rest/config"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

type RestContextService struct{}

func (s *RestContextService) Handle() (string, error) {
	return "ok", nil
}

type contextCaptureInvoker struct {
	*base.BaseInvoker
	ctx            context.Context
	lastInvocation base.Invocation
}

func (i *contextCaptureInvoker) Invoke(ctx context.Context, inv base.Invocation) result.Result {
	i.ctx = ctx
	i.lastInvocation = inv
	return &result.RPCResult{Rest: "ok"}
}

type testRestRequest struct {
	request *http.Request
}

func (r *testRestRequest) RawRequest() *http.Request { return r.request }

func (r *testRestRequest) PathParameter(string) string { return "" }

func (r *testRestRequest) PathParameters() map[string]string { return nil }

func (r *testRestRequest) QueryParameter(string) string { return "" }

func (r *testRestRequest) QueryParameters(string) []string { return nil }

func (r *testRestRequest) BodyParameter(string) (string, error) { return "", nil }

func (r *testRestRequest) HeaderParameter(string) string { return "" }

func (r *testRestRequest) ReadEntity(any) error { return nil }

type testRestResponse struct {
	*httptest.ResponseRecorder
}

func (r *testRestResponse) WriteError(status int, err error) error {
	r.WriteHeader(status)
	if err == nil {
		return nil
	}
	_, writeErr := fmt.Fprint(r, err)
	return writeErr
}

func (r *testRestResponse) WriteEntity(value any) error {
	_, err := fmt.Fprint(r, value)
	return err
}

func TestGetRouteFuncPropagatesRequestContext(t *testing.T) {
	const interfaceName = "RestContextService"
	const protocol = "rest"
	const version = "context-test"

	_, err := common.ServiceMap.Register(interfaceName, protocol, "", version, &RestContextService{})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, common.ServiceMap.UnRegister(interfaceName, protocol, common.ServiceKey(interfaceName, "", version)))
	})

	url := common.NewURLWithOptions(
		common.WithProtocol(protocol),
		common.WithPath(interfaceName),
		common.WithParamsValue(constant.VersionKey, version),
	)
	invoker := &contextCaptureInvoker{BaseInvoker: base.NewBaseInvoker(url)}
	methodConfig := &config.RestMethodConfig{
		MethodName:     "Handle",
		PathParamsMap:  map[int]string{},
		QueryParamsMap: map[int]string{},
		HeadersMap:     map[int]string{},
		Body:           -1,
	}

	type contextKey struct{}
	requestCtx := context.WithValue(context.Background(), contextKey{}, "request-value")
	request := &testRestRequest{request: httptest.NewRequestWithContext(requestCtx, http.MethodGet, "/", nil)}
	response := &testRestResponse{ResponseRecorder: httptest.NewRecorder()}

	GetRouteFunc(invoker, methodConfig)(request, response)

	require.Equal(t, "request-value", invoker.ctx.Value(contextKey{}))
	assertedInvocation, ok := invoker.lastInvocation.(*invocation.RPCInvocation)
	require.True(t, ok)
	require.Equal(t, "request-value", assertedInvocation.Context().Value(contextKey{}))
}
