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
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"
	"unsafe"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"google.golang.org/grpc"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
	tri "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
)

func Test_generateAttachments(t *testing.T) {
	tests := []struct {
		desc   string
		input  func() http.Header
		expect func(t *testing.T, res map[string]any)
	}{
		{
			desc: "empty header",
			input: func() http.Header {
				return http.Header{}
			},
			expect: func(t *testing.T, res map[string]any) {
				assert.Empty(t, res)
			},
		},
		{
			desc: "normal header with lowercase keys",
			input: func() http.Header {
				header := make(http.Header)
				header.Set("key1", "val1")
				header.Set("key2", "val2_1")
				header.Add("key2", "val2_2")
				return header
			},
			expect: func(t *testing.T, res map[string]any) {
				assert.Len(t, res, 2)
				assert.Equal(t, []string{"val1"}, res["key1"])
				assert.Equal(t, []string{"val2_1", "val2_2"}, res["key2"])
			},
		},
		{
			desc: "normal header with uppercase keys",
			input: func() http.Header {
				header := make(http.Header)
				header.Set("Key1", "val1")
				header.Set("Key2", "val2_1")
				header.Add("Key2", "val2_2")
				return header
			},
			expect: func(t *testing.T, res map[string]any) {
				assert.Len(t, res, 2)
				assert.Equal(t, []string{"val1"}, res["key1"])
				assert.Equal(t, []string{"val2_1", "val2_2"}, res["key2"])
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			atta := generateAttachments(test.input())
			test.expect(t, atta)
		})
	}
}

func TestServer_StartWithHttp2AndHttp3(t *testing.T) {
	// Test configuration for enabling both HTTP/2 and HTTP/3
	tripleConfig := &global.TripleConfig{
		Http3: &global.Http3Config{
			Enable: true, // Enable HTTP/3 (which now means both HTTP/2 and HTTP/3)
		},
	}

	server := NewServer(tripleConfig)

	// This test verifies that the server correctly determines the protocol
	// when Enable is set to true
	// Note: We can't actually start the server in a unit test due to port binding
	// but we can verify the configuration logic
	assert.NotNil(t, server)
	assert.Equal(t, tripleConfig.Http3.Enable, server.cfg.Http3.Enable)
}

func TestServer_ProtocolSelection(t *testing.T) {
	tests := []struct {
		name             string
		http3Config      *global.Http3Config
		expectedProtocol string
	}{
		{
			name: "HTTP/2 only (default)",
			http3Config: &global.Http3Config{
				Enable: false,
			},
			expectedProtocol: constant.CallHTTP2,
		},
		{
			name: "HTTP/2 and HTTP/3 both",
			http3Config: &global.Http3Config{
				Enable: true,
			},
			expectedProtocol: constant.CallHTTP2AndHTTP3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tripleConfig := &global.TripleConfig{
				Http3: tt.http3Config,
			}

			// Extract the protocol selection logic for testing
			var callProtocol string
			if tripleConfig.Http3 != nil && tripleConfig.Http3.Enable {
				callProtocol = constant.CallHTTP2AndHTTP3
			} else {
				callProtocol = constant.CallHTTP2
			}

			assert.Equal(t, tt.expectedProtocol, callProtocol)
		})
	}
}

func TestNewServer(t *testing.T) {
	tests := []struct {
		desc string
		cfg  *global.TripleConfig
	}{
		{
			desc: "nil config",
			cfg:  nil,
		},
		{
			desc: "empty config",
			cfg:  &global.TripleConfig{},
		},
		{
			desc: "config with http3",
			cfg: &global.TripleConfig{
				Http3: &global.Http3Config{
					Enable: true,
				},
			},
		},
		{
			desc: "config with keepalive",
			cfg: &global.TripleConfig{
				KeepAliveInterval: "30s",
				KeepAliveTimeout:  "10s",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			server := NewServer(test.cfg)
			assert.NotNil(t, server)
			assert.Equal(t, test.cfg, server.cfg)
			assert.NotNil(t, server.services)
			assert.Empty(t, server.services)
		})
	}
}

func TestServer_GetServiceInfo(t *testing.T) {
	t.Run("empty services", func(t *testing.T) {
		server := NewServer(nil)
		info := server.GetServiceInfo()
		assert.NotNil(t, info)
		assert.Empty(t, info)
	})

	t.Run("with services", func(t *testing.T) {
		server := NewServer(nil)
		// manually add service info for testing
		server.mu.Lock()
		server.services["test.Service"] = grpc.ServiceInfo{
			Methods: []grpc.MethodInfo{
				{Name: "Method1", IsClientStream: false, IsServerStream: false},
			},
		}
		server.mu.Unlock()

		info := server.GetServiceInfo()
		assert.NotNil(t, info)
		assert.Len(t, info, 1)
		assert.Contains(t, info, "test.Service")
	})

	t.Run("returns copy not reference", func(t *testing.T) {
		server := NewServer(nil)
		server.mu.Lock()
		server.services["test.Service"] = grpc.ServiceInfo{}
		server.mu.Unlock()

		info1 := server.GetServiceInfo()
		info2 := server.GetServiceInfo()

		// modify info1 should not affect info2
		delete(info1, "test.Service")
		assert.Contains(t, info2, "test.Service")
	})
}

func TestServer_SaveServiceInfo(t *testing.T) {
	tests := []struct {
		desc          string
		interfaceName string
		info          *common.ServiceInfo
		expect        func(t *testing.T, server *Server)
	}{
		{
			desc:          "unary method",
			interfaceName: "test.UnaryService",
			info: &common.ServiceInfo{
				Methods: []common.MethodInfo{
					{Name: "UnaryMethod", Type: constant.CallUnary},
				},
			},
			expect: func(t *testing.T, server *Server) {
				info := server.GetServiceInfo()
				assert.Contains(t, info, "test.UnaryService")
				assert.Len(t, info["test.UnaryService"].Methods, 1)
				assert.Equal(t, "UnaryMethod", info["test.UnaryService"].Methods[0].Name)
				assert.False(t, info["test.UnaryService"].Methods[0].IsClientStream)
				assert.False(t, info["test.UnaryService"].Methods[0].IsServerStream)
			},
		},
		{
			desc:          "client stream method",
			interfaceName: "test.ClientStreamService",
			info: &common.ServiceInfo{
				Methods: []common.MethodInfo{
					{Name: "ClientStreamMethod", Type: constant.CallClientStream},
				},
			},
			expect: func(t *testing.T, server *Server) {
				info := server.GetServiceInfo()
				assert.Contains(t, info, "test.ClientStreamService")
				assert.True(t, info["test.ClientStreamService"].Methods[0].IsClientStream)
				assert.False(t, info["test.ClientStreamService"].Methods[0].IsServerStream)
			},
		},
		{
			desc:          "server stream method",
			interfaceName: "test.ServerStreamService",
			info: &common.ServiceInfo{
				Methods: []common.MethodInfo{
					{Name: "ServerStreamMethod", Type: constant.CallServerStream},
				},
			},
			expect: func(t *testing.T, server *Server) {
				info := server.GetServiceInfo()
				assert.Contains(t, info, "test.ServerStreamService")
				assert.False(t, info["test.ServerStreamService"].Methods[0].IsClientStream)
				assert.True(t, info["test.ServerStreamService"].Methods[0].IsServerStream)
			},
		},
		{
			desc:          "bidi stream method",
			interfaceName: "test.BidiStreamService",
			info: &common.ServiceInfo{
				Methods: []common.MethodInfo{
					{Name: "BidiStreamMethod", Type: constant.CallBidiStream},
				},
			},
			expect: func(t *testing.T, server *Server) {
				info := server.GetServiceInfo()
				assert.Contains(t, info, "test.BidiStreamService")
				assert.True(t, info["test.BidiStreamService"].Methods[0].IsClientStream)
				assert.True(t, info["test.BidiStreamService"].Methods[0].IsServerStream)
			},
		},
		{
			desc:          "multiple methods",
			interfaceName: "test.MultiMethodService",
			info: &common.ServiceInfo{
				Methods: []common.MethodInfo{
					{Name: "Method1", Type: constant.CallUnary},
					{Name: "Method2", Type: constant.CallClientStream},
					{Name: "Method3", Type: constant.CallServerStream},
					{Name: "Method4", Type: constant.CallBidiStream},
				},
			},
			expect: func(t *testing.T, server *Server) {
				info := server.GetServiceInfo()
				assert.Contains(t, info, "test.MultiMethodService")
				assert.Len(t, info["test.MultiMethodService"].Methods, 4)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			server := NewServer(nil)
			server.saveServiceInfo(test.interfaceName, test.info, "", "", "")
			test.expect(t, server)
		})
	}
}

func TestServer_SaveServiceInfo_Concurrent(t *testing.T) {
	server := NewServer(nil)
	var wg sync.WaitGroup
	concurrency := 10

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			info := &common.ServiceInfo{
				Methods: []common.MethodInfo{
					{Name: "Method", Type: constant.CallUnary},
				},
			}
			server.saveServiceInfo(fmt.Sprintf("test.Service%d", idx), info, "", "", "")
		}(i)
	}

	wg.Wait()
	assert.Len(t, server.GetServiceInfo(), concurrency)
}

func Test_getHanOpts(t *testing.T) {
	tests := []struct {
		desc       string
		url        *common.URL
		tripleConf *global.TripleConfig
		expectLen  int
	}{
		{
			desc:       "basic url without triple config",
			url:        common.NewURLWithOptions(),
			tripleConf: nil,
			expectLen:  4, // group, version, readMaxBytes, sendMaxBytes
		},
		{
			desc: "url with group and version",
			url: common.NewURLWithOptions(
				common.WithParamsValue(constant.GroupKey, "testGroup"),
				common.WithParamsValue(constant.VersionKey, "1.0.0"),
			),
			tripleConf: nil,
			expectLen:  4,
		},
		{
			desc: "url with max msg size",
			url: common.NewURLWithOptions(
				common.WithParamsValue(constant.MaxServerRecvMsgSize, "10MB"),
				common.WithParamsValue(constant.MaxServerSendMsgSize, "10MB"),
			),
			tripleConf: nil,
			expectLen:  4,
		},
		{
			desc: "with triple config max msg size",
			url:  common.NewURLWithOptions(),
			tripleConf: &global.TripleConfig{
				MaxServerRecvMsgSize: "20MB",
				MaxServerSendMsgSize: "20MB",
			},
			expectLen: 6, // base 4 + 2 from tripleConf
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			opts := getHanOpts(test.url, test.tripleConf)
			assert.Len(t, opts, test.expectLen)
		})
	}
}

// mockRPCService is a mock service for testing createServiceInfoWithReflection
type mockRPCService struct{}

func (m *mockRPCService) Reference() string {
	return "mockRPCService"
}

func (m *mockRPCService) TestMethod(ctx context.Context, req string) (string, error) {
	return req, nil
}

func (m *mockRPCService) TestMethodWithMultipleArgs(ctx context.Context, arg1 string, arg2 int) (string, error) {
	return arg1, nil
}

func Test_createServiceInfoWithReflection(t *testing.T) {
	t.Run("basic service", func(t *testing.T) {
		svc := &mockRPCService{}
		info := createServiceInfoWithReflection(svc)

		assert.NotNil(t, info)
		assert.NotEmpty(t, info.Methods)

		// should have TestMethod, TestMethodWithMultipleArgs, and $invoke
		methodNames := make([]string, 0)
		for _, m := range info.Methods {
			methodNames = append(methodNames, m.Name)
		}
		assert.Contains(t, methodNames, "TestMethod")
		assert.Contains(t, methodNames, "TestMethodWithMultipleArgs")
		assert.Contains(t, methodNames, "$invoke") // generic call method
	})

	t.Run("method type is CallUnary", func(t *testing.T) {
		svc := &mockRPCService{}
		info := createServiceInfoWithReflection(svc)

		for _, m := range info.Methods {
			assert.Equal(t, constant.CallUnary, m.Type)
		}
	})

	t.Run("ReqInitFunc returns correct params", func(t *testing.T) {
		svc := &mockRPCService{}
		info := createServiceInfoWithReflection(svc)

		for _, m := range info.Methods {
			if m.Name == "TestMethod" {
				params := m.ReqInitFunc()
				assert.NotNil(t, params)
				paramsSlice, ok := params.([]any)
				assert.True(t, ok)
				assert.Len(t, paramsSlice, 1) // only req param (ctx is excluded)
			}
			if m.Name == "TestMethodWithMultipleArgs" {
				params := m.ReqInitFunc()
				paramsSlice, ok := params.([]any)
				assert.True(t, ok)
				assert.Len(t, paramsSlice, 2) // arg1 and arg2
			}
		}
	})

	t.Run("generic invoke method", func(t *testing.T) {
		svc := &mockRPCService{}
		info := createServiceInfoWithReflection(svc)

		var invokeMethod *common.MethodInfo
		for i := range info.Methods {
			if info.Methods[i].Name == "$invoke" {
				invokeMethod = &info.Methods[i]
				break
			}
		}

		assert.NotNil(t, invokeMethod)
		assert.Equal(t, constant.CallUnary, invokeMethod.Type)

		params := invokeMethod.ReqInitFunc()
		paramsSlice, ok := params.([]any)
		assert.True(t, ok)
		assert.Len(t, paramsSlice, 3) // methodName, argv types, argv
	})
}

// Test isReflectValueNil safely checks if a reflect.Value is nil
func Test_isReflectValueNil(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected bool
	}{
		// nil nillable types
		{"nil chan", (chan int)(nil), true},
		{"nil map", (map[string]int)(nil), true},
		{"nil slice", ([]int)(nil), true},
		{"nil func", (func())(nil), true},
		{"nil pointer", (*int)(nil), true},
		{"nil unsafe.Pointer", unsafe.Pointer(nil), true},

		// non-nil nillable types
		{"non-nil chan", make(chan int), false},
		{"non-nil map", map[string]int{"a": 1}, false},
		{"non-nil slice", []int{1, 2, 3}, false},
		{"non-nil func", func() {}, false},
		{"non-nil pointer", new(int), false},

		// non-nillable types (should return false, not panic)
		{"int", 42, false},
		{"string", "hello", false},
		{"bool", true, false},
		{"float64", 3.14, false},
		{"struct", struct{ Name string }{"test"}, false},
		{"array", [3]int{1, 2, 3}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := reflect.ValueOf(tt.value)
			// should not panic
			result := isReflectValueNil(v)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test isReflectValueNil with UnsafePointer specifically
func Test_isReflectValueNil_UnsafePointer(t *testing.T) {
	t.Run("nil unsafe.Pointer", func(t *testing.T) {
		v := reflect.ValueOf(unsafe.Pointer(nil))
		assert.True(t, isReflectValueNil(v))
	})

	t.Run("non-nil unsafe.Pointer", func(t *testing.T) {
		x := 42
		v := reflect.ValueOf(unsafe.Pointer(&x))
		assert.False(t, isReflectValueNil(v))
	})
}

// TestHandleServiceWithInfoSaveServiceInfoOnlyOriginalMethods verifies that
// saveServiceInfo only records the original method names so registry and
// gRPC-reflection metadata stay clean.
func TestHandleServiceWithInfoSaveServiceInfoOnlyOriginalMethods(t *testing.T) {
	server := NewServer(nil)
	info := &common.ServiceInfo{
		Methods: []common.MethodInfo{
			{Name: "GetUser", Type: constant.CallUnary},
			{Name: "ListUsers", Type: constant.CallServerStream},
		},
	}
	server.saveServiceInfo("com.example.UserService", info, "", "", "")

	svcInfo := server.GetServiceInfo()
	svc, ok := svcInfo["com.example.UserService"]
	assert.True(t, ok)
	// Only the two original methods; no "getUser" / "listUsers" aliases.
	assert.Len(t, svc.Methods, 2)
	names := make([]string, 0, len(svc.Methods))
	for _, m := range svc.Methods {
		names = append(names, m.Name)
	}
	assert.Contains(t, names, "GetUser")
	assert.Contains(t, names, "ListUsers")
	assert.NotContains(t, names, "getUser")
	assert.NotContains(t, names, "listUsers")
}

type tripleServerTestInvoker struct {
	url      *common.URL
	invokeFn func(context.Context, base.Invocation) result.Result
}

func (m *tripleServerTestInvoker) GetURL() *common.URL {
	if m.url != nil {
		return m.url
	}
	return common.NewURLWithOptions()
}

func (m *tripleServerTestInvoker) IsAvailable() bool {
	return true
}

func (m *tripleServerTestInvoker) Destroy() {
	// No-op: this test double does not own lifecycle resources.
}

func (m *tripleServerTestInvoker) Invoke(ctx context.Context, invocation base.Invocation) result.Result {
	if m.invokeFn == nil {
		return &result.RPCResult{}
	}
	return m.invokeFn(ctx, invocation)
}

type tripleServerTestConn struct {
	reqHeader   http.Header
	respHeader  http.Header
	respTrailer http.Header
	receiveFn   func(any)
	sent        []any
}

const (
	tripleServerTestFromClientMessage = "from-client"
	tripleServerTestServerToken       = "server-token"
	tripleServerTestBidiToken         = "bidi-token"
	tripleServerTestBidiValue         = "bidi-v"
)

func newTripleServerTestConn() *tripleServerTestConn {
	return &tripleServerTestConn{
		reqHeader:   make(http.Header),
		respHeader:  make(http.Header),
		respTrailer: make(http.Header),
	}
}

func (c *tripleServerTestConn) Spec() tri.Spec {
	return tri.Spec{}
}

func (c *tripleServerTestConn) Peer() tri.Peer {
	return tri.Peer{}
}

func (c *tripleServerTestConn) Receive(msg any) error {
	if c.receiveFn != nil {
		c.receiveFn(msg)
	}
	return nil
}

func (c *tripleServerTestConn) RequestHeader() http.Header {
	return c.reqHeader
}

func (c *tripleServerTestConn) ExportableHeader() http.Header {
	return c.reqHeader
}

func (c *tripleServerTestConn) Send(msg any) error {
	c.sent = append(c.sent, msg)
	return nil
}

func (c *tripleServerTestConn) ResponseHeader() http.Header {
	return c.respHeader
}

func (c *tripleServerTestConn) ResponseTrailer() http.Header {
	return c.respTrailer
}

func TestServerValidateTransportSettingsAllowsCompatibleSettings(t *testing.T) {
	server := NewServer(nil)
	server.transportSettings = &transportSettings{
		location:     "127.0.0.1:20000",
		callProtocol: constant.CallHTTP2,
		rawTLSConfig: global.DefaultTLSConfig(),
	}

	err := server.validateTransportSettings(&transportSettings{
		location:     "127.0.0.1:20000",
		callProtocol: constant.CallHTTP2,
		rawTLSConfig: global.DefaultTLSConfig(),
	})
	require.NoError(t, err)
}

func TestServerValidateTransportSettingsRejectsConflictingProtocol(t *testing.T) {
	server := NewServer(nil)
	server.transportSettings = &transportSettings{
		location:     "127.0.0.1:20000",
		callProtocol: constant.CallHTTP2,
		rawTLSConfig: global.DefaultTLSConfig(),
	}

	err := server.validateTransportSettings(&transportSettings{
		location:     "127.0.0.1:20000",
		callProtocol: constant.CallHTTP2AndHTTP3,
		rawTLSConfig: global.DefaultTLSConfig(),
	})
	require.ErrorContains(t, err, "already uses protocol")
}

func TestServerValidateTransportSettingsRejectsConflictingTLS(t *testing.T) {
	server := NewServer(nil)
	server.transportSettings = &transportSettings{
		location:     "127.0.0.1:20000",
		callProtocol: constant.CallHTTP2,
		rawTLSConfig: global.DefaultTLSConfig(),
	}

	err := server.validateTransportSettings(&transportSettings{
		location:     "127.0.0.1:20000",
		callProtocol: constant.CallHTTP2,
		rawTLSConfig: &global.TLSConfig{TLSCertFile: "server.crt", TLSKeyFile: "server.key"},
	})
	require.ErrorContains(t, err, "different TLS settings")
}

func TestServerMountHTTPHandlerRejectsDuplicate(t *testing.T) {
	server := NewServer(nil)

	require.NoError(t, server.MountHTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})))

	err := server.MountHTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	require.ErrorContains(t, err, "already been mounted")
}

func TestResolveHandlerOptionsRejectsUnsupportedSerialization(t *testing.T) {
	url := common.NewURLWithOptions(
		common.WithParamsValue(constant.SerializationKey, "yaml"),
	)

	_, err := resolveHandlerOptions(url)
	require.ErrorContains(t, err, "unsupported serialization: yaml")
}

func TestServerRefreshServiceRejectsInvalidHandlerTripleConfig(t *testing.T) {
	server := NewServer(nil)
	invoker := &tripleServerTestInvoker{
		url: common.NewURLWithOptions(
			common.WithAttribute(constant.TripleConfigKey, "invalid"),
		),
	}

	err := server.refreshService(invoker, nil)
	require.ErrorContains(t, err, "invalid triple config type string")
}

func TestServerRegisterUnaryMethodHandler(t *testing.T) {
	server := newServerForMethodHandlerTest()
	invoker := &tripleServerTestInvoker{
		invokeFn: func(ctx context.Context, invocation base.Invocation) result.Result {
			assert.Equal(t, "UnaryMethod", invocation.MethodName())
			assert.Equal(t, []any{"alice", 7}, invocation.Arguments())
			assert.Equal(t, []string{"v1", "v2"}, invocation.Attachments()["x-test"])

			ctxAttachments, ok := ctx.Value(constant.AttachmentKey).(map[string]any)
			require.True(t, ok)
			assert.Equal(t, []string{"v1", "v2"}, ctxAttachments["x-test"])

			res := &result.RPCResult{}
			res.SetResult("unary-ok")
			res.SetAttachments(map[string]any{
				"resp-one":   "val",
				"resp-multi": []string{"a", "b"},
				"resp-omit":  123,
			})
			return res
		},
	}
	method := common.MethodInfo{
		Name: "UnaryMethod",
		Type: constant.CallUnary,
		ReqInitFunc: func() any {
			var name string
			var age int
			return []any{&name, &age}
		},
	}
	procedure := "/svc/UnaryMethod"
	server.registerMethodHandler(procedure, method, invoker)

	conn := newTripleServerTestConn()
	conn.reqHeader.Add("X-Test", "v1")
	conn.reqHeader.Add("X-Test", "v2")
	conn.receiveFn = func(msg any) {
		args, ok := msg.([]any)
		require.True(t, ok)
		*(args[0].(*string)) = "alice"
		*(args[1].(*int)) = 7
	}
	require.NoError(t, invokeRegisteredHandlerImplementation(server.triServer, procedure, conn))
	require.Len(t, conn.sent, 1)
	assert.Equal(t, []any{"unary-ok"}, conn.sent[0])
	assert.Equal(t, []string{"val"}, conn.respTrailer.Values("resp-one"))
	assert.Equal(t, []string{"a", "b"}, conn.respTrailer.Values("resp-multi"))
	assert.Empty(t, conn.respTrailer.Values("resp-omit"))
}

func TestServerRegisterClientStreamMethodHandler(t *testing.T) {
	server := newServerForMethodHandlerTest()
	invoker := &tripleServerTestInvoker{
		invokeFn: func(ctx context.Context, invocation base.Invocation) result.Result {
			assert.Equal(t, "ClientStreamMethod", invocation.MethodName())
			assert.Equal(t, []any{"client-token"}, invocation.Arguments())
			assert.Equal(t, []string{"trace-123"}, invocation.Attachments()["trace-id"])
			_, ok := ctx.Value(constant.AttachmentKey).(map[string]any)
			assert.True(t, ok)

			res := &result.RPCResult{}
			res.SetResult("client-ok")
			return res
		},
	}
	method := common.MethodInfo{
		Name: "ClientStreamMethod",
		Type: constant.CallClientStream,
		StreamInitFunc: func(baseStream any) any {
			_, ok := baseStream.(*tri.ClientStream)
			require.True(t, ok)
			return "client-token"
		},
	}
	procedure := "/svc/ClientStreamMethod"
	server.registerMethodHandler(procedure, method, invoker)

	conn := newTripleServerTestConn()
	conn.reqHeader.Set("Trace-Id", "trace-123")
	require.NoError(t, invokeRegisteredHandlerImplementation(server.triServer, procedure, conn))
	require.Len(t, conn.sent, 1)
	assert.Equal(t, []any{"client-ok"}, conn.sent[0])
}

func TestServerRegisterServerStreamMethodHandler(t *testing.T) {
	type serverStreamReq struct {
		Message string
	}

	server := newServerForMethodHandlerTest()
	invoker := &tripleServerTestInvoker{
		invokeFn: func(ctx context.Context, invocation base.Invocation) result.Result {
			assert.Equal(t, "ServerStreamMethod", invocation.MethodName())
			require.Len(t, invocation.Arguments(), 2)
			req, ok := invocation.Arguments()[0].(*serverStreamReq)
			require.True(t, ok)
			assert.Equal(t, tripleServerTestFromClientMessage, req.Message)
			assert.Equal(t, tripleServerTestServerToken, invocation.Arguments()[1])
			assert.Equal(t, []string{"v"}, invocation.Attachments()["x-stream"])
			_, ok = ctx.Value(constant.AttachmentKey).(map[string]any)
			assert.True(t, ok)
			return &result.RPCResult{}
		},
	}
	method := common.MethodInfo{
		Name: "ServerStreamMethod",
		Type: constant.CallServerStream,
		ReqInitFunc: func() any {
			return &serverStreamReq{}
		},
		StreamInitFunc: func(baseStream any) any {
			_, ok := baseStream.(*tri.ServerStream)
			require.True(t, ok)
			return tripleServerTestServerToken
		},
	}
	procedure := "/svc/ServerStreamMethod"
	server.registerMethodHandler(procedure, method, invoker)

	conn := newTripleServerTestConn()
	conn.reqHeader.Set("X-Stream", "v")
	conn.receiveFn = func(msg any) {
		req, ok := msg.(*serverStreamReq)
		require.True(t, ok)
		req.Message = tripleServerTestFromClientMessage
	}
	require.NoError(t, invokeRegisteredHandlerImplementation(server.triServer, procedure, conn))
	assert.Empty(t, conn.sent)
}

func TestServerRegisterBidiStreamMethodHandler(t *testing.T) {
	server := newServerForMethodHandlerTest()
	invoker := &tripleServerTestInvoker{
		invokeFn: func(ctx context.Context, invocation base.Invocation) result.Result {
			assert.Equal(t, "BidiStreamMethod", invocation.MethodName())
			assert.Equal(t, []any{tripleServerTestBidiToken}, invocation.Arguments())
			assert.Equal(t, []string{tripleServerTestBidiValue}, invocation.Attachments()["x-bidi"])
			_, ok := ctx.Value(constant.AttachmentKey).(map[string]any)
			assert.True(t, ok)
			return &result.RPCResult{}
		},
	}
	method := common.MethodInfo{
		Name: "BidiStreamMethod",
		Type: constant.CallBidiStream,
		StreamInitFunc: func(baseStream any) any {
			_, ok := baseStream.(*tri.BidiStream)
			require.True(t, ok)
			return tripleServerTestBidiToken
		},
	}
	procedure := "/svc/BidiStreamMethod"
	server.registerMethodHandler(procedure, method, invoker)

	conn := newTripleServerTestConn()
	conn.reqHeader.Set("X-Bidi", tripleServerTestBidiValue)
	require.NoError(t, invokeRegisteredHandlerImplementation(server.triServer, procedure, conn))
	assert.Empty(t, conn.sent)
}

func TestServerRegisterMethodHandlerUnknownType(t *testing.T) {
	server := newServerForMethodHandlerTest()
	procedure := "/svc/Unknown"
	server.registerMethodHandler(procedure, common.MethodInfo{
		Name: "UnknownMethod",
		Type: "unknown",
	}, &tripleServerTestInvoker{})

	_, ok := getServerHandler(server.triServer, procedure)
	assert.False(t, ok)
}

func TestServerHandleServiceWithInfoFallbackHitsStreamingHandlers(t *testing.T) {
	type streamReq struct {
		Message string
	}

	server := newServerForMethodHandlerTest()
	calledMethods := make([]string, 0, 2)
	invoker := &tripleServerTestInvoker{
		invokeFn: func(ctx context.Context, invocation base.Invocation) result.Result {
			calledMethods = append(calledMethods, invocation.MethodName())
			switch invocation.MethodName() {
			case "CountUp":
				require.Len(t, invocation.Arguments(), 2)
				req, ok := invocation.Arguments()[0].(*streamReq)
				require.True(t, ok)
				assert.Equal(t, tripleServerTestFromClientMessage, req.Message)
				assert.Equal(t, tripleServerTestServerToken, invocation.Arguments()[1])
				assert.Equal(t, []string{"stream-v"}, invocation.Attachments()["x-stream"])
			case "CumSum":
				require.Len(t, invocation.Arguments(), 1)
				assert.Equal(t, tripleServerTestBidiToken, invocation.Arguments()[0])
				assert.Equal(t, []string{tripleServerTestBidiValue}, invocation.Attachments()["x-bidi"])
			default:
				t.Fatalf("unexpected method: %s", invocation.MethodName())
			}

			ctxAttachments, ok := ctx.Value(constant.AttachmentKey).(map[string]any)
			require.True(t, ok)
			assert.NotEmpty(t, ctxAttachments)
			return &result.RPCResult{}
		},
	}
	info := &common.ServiceInfo{
		Methods: []common.MethodInfo{
			{
				Name: "CountUp",
				Type: constant.CallServerStream,
				ReqInitFunc: func() any {
					return &streamReq{}
				},
				StreamInitFunc: func(baseStream any) any {
					_, ok := baseStream.(*tri.ServerStream)
					require.True(t, ok)
					return tripleServerTestServerToken
				},
			},
			{
				Name: "CumSum",
				Type: constant.CallBidiStream,
				StreamInitFunc: func(baseStream any) any {
					_, ok := baseStream.(*tri.BidiStream)
					require.True(t, ok)
					return tripleServerTestBidiToken
				},
			},
		},
	}
	server.handleServiceWithInfo("svc.Fallback", invoker, info)

	serverStreamConn := newTripleServerTestConn()
	serverStreamConn.reqHeader.Set("X-Stream", "stream-v")
	serverStreamConn.receiveFn = func(msg any) {
		req, ok := msg.(*streamReq)
		require.True(t, ok)
		req.Message = tripleServerTestFromClientMessage
	}
	pattern, err := invokeRegisteredHandlerImplementationByRequestPath(
		server.triServer,
		"/svc.Fallback/countUp",
		serverStreamConn,
	)
	require.NoError(t, err)
	assert.Equal(t, "/svc.Fallback/CountUp", pattern)

	bidiConn := newTripleServerTestConn()
	bidiConn.reqHeader.Set("X-Bidi", tripleServerTestBidiValue)
	pattern, err = invokeRegisteredHandlerImplementationByRequestPath(
		server.triServer,
		"/svc.Fallback/cumSum",
		bidiConn,
	)
	require.NoError(t, err)
	assert.Equal(t, "/svc.Fallback/CumSum", pattern)

	assert.Equal(t, []string{"CountUp", "CumSum"}, calledMethods)
}

func newServerForMethodHandlerTest() *Server {
	return &Server{triServer: tri.NewServer("127.0.0.1:0", nil)}
}

type tripleVariadicReflectionServiceForTest struct{}

func (s *tripleVariadicReflectionServiceForTest) HelloVariadic(ctx context.Context, prefix string, names ...string) (string, error) {
	return prefix + ":" + fmt.Sprint(len(names)), nil
}

func TestExtractUnaryInvocationArgs(t *testing.T) {
	t.Run("from non-idl argument slice", func(t *testing.T) {
		name := "alice"
		age := 18
		args := extractUnaryInvocationArgs([]any{&name, &age})
		assert.Equal(t, []any{"alice", 18}, args)
	})

	t.Run("from single message in idl mode", func(t *testing.T) {
		msg := struct{ Name string }{Name: "idl"}
		args := extractUnaryInvocationArgs(msg)
		assert.Equal(t, []any{msg}, args)
	})
}

func TestBuildMethodInfoWithReflectionVariadic(t *testing.T) {
	svc := &tripleVariadicReflectionServiceForTest{}
	method, ok := reflect.TypeOf(svc).MethodByName("HelloVariadic")
	require.True(t, ok)

	methodInfo := buildMethodInfoWithReflection(method)
	require.NotNil(t, methodInfo)

	t.Run("generic packed variadic tail uses call slice", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), constant.DubboCtxKey(constant.GenericVariadicCallSliceKey), true)
		res, err := methodInfo.MethodFunc(ctx, []any{"hello", []string{"alice", "bob"}}, svc)
		require.NoError(t, err)
		assert.Equal(t, "hello:2", res)
	})

	t.Run("ordinary discrete variadic call remains unchanged", func(t *testing.T) {
		res, err := methodInfo.MethodFunc(context.Background(), []any{"hello", "alice", "bob"}, svc)
		require.NoError(t, err)
		assert.Equal(t, "hello:2", res)
	})
}

func TestWrapTripleResponse(t *testing.T) {
	resp := tri.NewResponse("already-wrapped")
	assert.Same(t, resp, wrapTripleResponse(resp))

	wrapped := wrapTripleResponse("plain-result")
	assert.Equal(t, []any{"plain-result"}, wrapped.Msg)
}

func TestAppendTripleOutgoingAttachments(t *testing.T) {
	ctx := tri.NewOutgoingContext(context.Background(), make(http.Header))
	appendTripleOutgoingAttachments(ctx, map[string]any{
		"one":   "1",
		"multi": []string{"a", "b"},
		"omit":  100,
	})

	outgoing := tri.ExtractFromOutgoingContext(ctx)
	require.NotNil(t, outgoing)
	assert.Equal(t, []string{"1"}, outgoing.Values("one"))
	assert.Equal(t, []string{"a", "b"}, outgoing.Values("multi"))
	assert.Empty(t, outgoing.Values("omit"))
}

const tripleServerDefaultImplementationKey = "/"

// These helpers execute the default registered implementation directly so the
// tests can verify registerMethodHandler's invocation wiring without depending
// on protocol-specific HTTP framing details.
func invokeRegisteredHandlerImplementation(triServer *tri.Server, procedure string, conn tri.StreamingHandlerConn) error {
	handler, ok := getServerHandler(triServer, procedure)
	if !ok {
		return fmt.Errorf("handler for procedure %s not found", procedure)
	}
	implementation, ok := getDefaultHandlerImplementation(handler)
	if !ok {
		return fmt.Errorf("default implementation for procedure %s not found", procedure)
	}
	return implementation(context.Background(), conn)
}

func invokeRegisteredHandlerImplementationByRequestPath(
	triServer *tri.Server,
	requestPath string,
	conn tri.StreamingHandlerConn,
) (string, error) {
	handler, pattern, ok := getServerHandlerByRequestPath(triServer, requestPath)
	if !ok {
		return "", fmt.Errorf("handler for request path %s not found", requestPath)
	}
	implementation, ok := getDefaultHandlerImplementation(handler)
	if !ok {
		return "", fmt.Errorf("default implementation for request path %s not found", requestPath)
	}
	return pattern, implementation(context.Background(), conn)
}

func getServerHandler(triServer *tri.Server, procedure string) (*tri.Handler, bool) {
	if triServer == nil {
		return nil, false
	}
	handlersField := reflect.ValueOf(triServer).Elem().FieldByName("handlers")
	handlersValue, ok := extractUnexportedValue(handlersField)
	if !ok {
		return nil, false
	}
	handlers, ok := handlersValue.Interface().(map[string]*tri.Handler)
	if !ok {
		return nil, false
	}
	handler, ok := handlers[procedure]
	return handler, ok
}

func getServerHandlerByRequestPath(triServer *tri.Server, requestPath string) (*tri.Handler, string, bool) {
	if triServer == nil {
		return nil, "", false
	}
	muxField := reflect.ValueOf(triServer).Elem().FieldByName("mux")
	muxValue, ok := extractUnexportedValue(muxField)
	if !ok || !muxValue.IsValid() || muxValue.IsNil() {
		return nil, "", false
	}

	handlerMethod := muxValue.MethodByName("Handler")
	if !handlerMethod.IsValid() {
		return nil, "", false
	}
	req := httptest.NewRequest(http.MethodPost, requestPath, nil)
	results := handlerMethod.Call([]reflect.Value{reflect.ValueOf(req)})
	if len(results) != 2 {
		return nil, "", false
	}

	handler, ok := results[0].Interface().(http.Handler)
	if !ok || handler == nil {
		return nil, "", false
	}
	pattern, ok := results[1].Interface().(string)
	if !ok || pattern == "" {
		return nil, "", false
	}
	triHandler, ok := handler.(*tri.Handler)
	if !ok {
		return nil, "", false
	}

	return triHandler, pattern, true
}

func getDefaultHandlerImplementation(handler *tri.Handler) (tri.StreamingHandlerFunc, bool) {
	if handler == nil {
		return nil, false
	}
	implField := reflect.ValueOf(handler).Elem().FieldByName("implementations")
	implValue, ok := extractUnexportedValue(implField)
	if !ok {
		return nil, false
	}
	implementations, ok := implValue.Interface().(map[string]tri.StreamingHandlerFunc)
	if !ok {
		return nil, false
	}
	implementation, ok := implementations[tripleServerDefaultImplementationKey]
	return implementation, ok
}

func extractUnexportedValue(field reflect.Value) (reflect.Value, bool) {
	if !field.IsValid() || !field.CanAddr() {
		return reflect.Value{}, false
	}
	return reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem(), true
}
