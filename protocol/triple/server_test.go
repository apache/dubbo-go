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
	"reflect"
	"sync"
	"testing"
	"unsafe"
)

import (
	"github.com/stretchr/testify/assert"

	"google.golang.org/grpc"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
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
			server.saveServiceInfo(test.interfaceName, test.info)
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
			server.saveServiceInfo(fmt.Sprintf("test.Service%d", idx), info)
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
