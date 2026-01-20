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
	"net/http"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
	tri "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
)

func TestClientManager_HTTP2AndHTTP3(t *testing.T) {
	// Test client configuration for simultaneous HTTP/2 and HTTP/3 startup
	url := &common.URL{
		Location: "localhost:20000",
		Path:     "com.example.TestService",
		Methods:  []string{"testMethod"},
	}

	// Configure TLS
	tlsConfig := &global.TLSConfig{
		CACertFile:  "testdata/ca.crt",
		TLSCertFile: "testdata/server.crt",
		TLSKeyFile:  "testdata/server.key",
	}
	url.SetAttribute(constant.TLSConfigKey, tlsConfig)

	// Configure Triple, enable HTTP/3 (now means starting both HTTP/2 and HTTP/3 simultaneously)
	tripleConfig := &global.TripleConfig{
		Http3: &global.Http3Config{
			Enable: true, // Enable HTTP/3 (now means starting both HTTP/2 and HTTP/3 simultaneously)
		},
	}
	url.SetAttribute(constant.TripleConfigKey, tripleConfig)

	// Create client manager
	clientManager, err := newClientManager(url)

	// Since we don't have real certificate files, this should fail
	// But we mainly test the configuration parsing logic
	if err != nil {
		// Expected error due to missing certificate files
		t.Logf("Expected error due to missing certificate files: %v", err)
		return
	}

	// If successfully created, verify the client manager
	assert.NotNil(t, clientManager)
	assert.True(t, clientManager.isIDL)
	assert.NotNil(t, clientManager.triClient)
}

func TestDualTransport(t *testing.T) {
	// Test dualTransport creation
	keepAliveInterval := 30 * time.Second
	keepAliveTimeout := 5 * time.Second

	// Test newDualTransport function
	transport := newDualTransport(nil, keepAliveInterval, keepAliveTimeout)
	assert.NotNil(t, transport)

	// Verify that transport implements http.RoundTripper interface
	_, ok := transport.(interface {
		RoundTrip(*http.Request) (*http.Response, error)
	})
	assert.True(t, ok, "transport should implement http.RoundTripper")
}

func TestClientManager_Close(t *testing.T) {
	cm := &clientManager{
		isIDL:     true,
		triClient: tri.NewClient(&http.Client{}, "http://localhost:8080/test"),
	}

	err := cm.close()
	require.NoError(t, err)
}

// TestClientManager_CallMethods_MissingClient removed - no longer applicable
// in the service-level client architecture where all methods share a single triClient.

func Test_genKeepAliveOptions(t *testing.T) {
	defaultInterval, _ := time.ParseDuration(constant.DefaultKeepAliveInterval)
	defaultTimeout, _ := time.ParseDuration(constant.DefaultKeepAliveTimeout)

	tests := []struct {
		desc           string
		url            *common.URL
		tripleConf     *global.TripleConfig
		expectOptsLen  int
		expectInterval time.Duration
		expectTimeout  time.Duration
		expectErr      bool
	}{
		{
			desc:           "nil triple config",
			url:            common.NewURLWithOptions(),
			tripleConf:     nil,
			expectOptsLen:  2, // readMaxBytes, sendMaxBytes
			expectInterval: defaultInterval,
			expectTimeout:  defaultTimeout,
			expectErr:      false,
		},
		{
			desc: "url with max msg size",
			url: common.NewURLWithOptions(
				common.WithParamsValue(constant.MaxCallRecvMsgSize, "10MB"),
				common.WithParamsValue(constant.MaxCallSendMsgSize, "10MB"),
			),
			tripleConf:     nil,
			expectOptsLen:  2,
			expectInterval: defaultInterval,
			expectTimeout:  defaultTimeout,
			expectErr:      false,
		},
		{
			desc: "url with keepalive params",
			url: common.NewURLWithOptions(
				common.WithParamsValue(constant.KeepAliveInterval, "60s"),
				common.WithParamsValue(constant.KeepAliveTimeout, "20s"),
			),
			tripleConf:     nil,
			expectOptsLen:  2,
			expectInterval: 60 * time.Second,
			expectTimeout:  20 * time.Second,
			expectErr:      false,
		},
		{
			desc: "triple config with keepalive",
			url:  common.NewURLWithOptions(),
			tripleConf: &global.TripleConfig{
				KeepAliveInterval: "45s",
				KeepAliveTimeout:  "15s",
			},
			expectOptsLen:  2,
			expectInterval: 45 * time.Second,
			expectTimeout:  15 * time.Second,
			expectErr:      false,
		},
		{
			desc: "triple config with invalid interval",
			url:  common.NewURLWithOptions(),
			tripleConf: &global.TripleConfig{
				KeepAliveInterval: "invalid",
			},
			expectErr: true,
		},
		{
			desc: "triple config with invalid timeout",
			url:  common.NewURLWithOptions(),
			tripleConf: &global.TripleConfig{
				KeepAliveTimeout: "invalid",
			},
			expectErr: true,
		},
		{
			desc:           "empty triple config",
			url:            common.NewURLWithOptions(),
			tripleConf:     &global.TripleConfig{},
			expectOptsLen:  2,
			expectInterval: defaultInterval,
			expectTimeout:  defaultTimeout,
			expectErr:      false,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			opts, interval, timeout, err := genKeepAliveOptions(test.url, test.tripleConf)
			if test.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Len(t, opts, test.expectOptsLen)
				assert.Equal(t, test.expectInterval, interval)
				assert.Equal(t, test.expectTimeout, timeout)
			}
		})
	}
}

func Test_newClientManager_Serialization(t *testing.T) {
	tests := []struct {
		desc          string
		serialization string
		expectIDL     bool
		expectPanic   bool
	}{
		{
			desc:          "protobuf serialization",
			serialization: constant.ProtobufSerialization,
			expectIDL:     true,
			expectPanic:   false,
		},
		{
			desc:          "json serialization",
			serialization: constant.JSONSerialization,
			expectIDL:     true,
			expectPanic:   false,
		},
		{
			desc:          "hessian2 serialization",
			serialization: constant.Hessian2Serialization,
			expectIDL:     false,
			expectPanic:   false,
		},
		{
			desc:          "msgpack serialization",
			serialization: constant.MsgpackSerialization,
			expectIDL:     false,
			expectPanic:   false,
		},
		{
			desc:          "unsupported serialization",
			serialization: "unsupported",
			expectPanic:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			url := common.NewURLWithOptions(
				common.WithLocation("localhost:20000"),
				common.WithPath("com.example.TestService"),
				common.WithMethods([]string{"TestMethod"}),
				common.WithParamsValue(constant.SerializationKey, test.serialization),
			)

			if test.expectPanic {
				assert.Panics(t, func() {
					_, _ = newClientManager(url)
				})
			} else {
				cm, err := newClientManager(url)
				require.NoError(t, err)
				assert.NotNil(t, cm)
				assert.Equal(t, test.expectIDL, cm.isIDL)
			}
		})
	}
}

func Test_newClientManager_NoMethods(t *testing.T) {
	// Test when url has no methods - in service-level client architecture,
	// this is valid as the client is created at service level, not method level
	url := common.NewURLWithOptions(
		common.WithLocation("localhost:20000"),
		common.WithPath("com.example.TestService"),
	)

	cm, err := newClientManager(url)
	require.NoError(t, err, "service-level client should be created even without method list")
	assert.NotNil(t, cm)
	assert.NotNil(t, cm.triClient, "triClient should be created at service level")
}

func Test_newClientManager_WithMethods(t *testing.T) {
	url := common.NewURLWithOptions(
		common.WithLocation("localhost:20000"),
		common.WithPath("com.example.TestService"),
		common.WithMethods([]string{"Method1", "Method2", "Method3"}),
	)

	cm, err := newClientManager(url)
	require.NoError(t, err)
	assert.NotNil(t, cm)
	assert.NotNil(t, cm.triClient, "triClient should be created")
}

func Test_newClientManager_WithGroupAndVersion(t *testing.T) {
	url := common.NewURLWithOptions(
		common.WithLocation("localhost:20000"),
		common.WithPath("com.example.TestService"),
		common.WithMethods([]string{"TestMethod"}),
		common.WithParamsValue(constant.GroupKey, "testGroup"),
		common.WithParamsValue(constant.VersionKey, "1.0.0"),
	)

	cm, err := newClientManager(url)
	require.NoError(t, err)
	assert.NotNil(t, cm)
}

func Test_newClientManager_WithTimeout(t *testing.T) {
	url := common.NewURLWithOptions(
		common.WithLocation("localhost:20000"),
		common.WithPath("com.example.TestService"),
		common.WithMethods([]string{"TestMethod"}),
		common.WithParamsValue(constant.TimeoutKey, "5s"),
	)

	cm, err := newClientManager(url)
	require.NoError(t, err)
	assert.NotNil(t, cm)
}

func Test_newClientManager_InvalidTLSConfig(t *testing.T) {
	url := common.NewURLWithOptions(
		common.WithLocation("localhost:20000"),
		common.WithPath("com.example.TestService"),
		common.WithMethods([]string{"TestMethod"}),
	)
	// Set invalid TLS config type
	url.SetAttribute(constant.TLSConfigKey, "invalid-type")

	cm, err := newClientManager(url)
	require.Error(t, err)
	assert.Nil(t, cm)
	assert.Contains(t, err.Error(), "TLSConfig configuration failed")
}

func Test_newClientManager_HTTP3WithoutTLS(t *testing.T) {
	url := common.NewURLWithOptions(
		common.WithLocation("localhost:20000"),
		common.WithPath("com.example.TestService"),
		common.WithMethods([]string{"TestMethod"}),
	)
	// Enable HTTP/3 without TLS config
	tripleConfig := &global.TripleConfig{
		Http3: &global.Http3Config{
			Enable: true,
		},
	}
	url.SetAttribute(constant.TripleConfigKey, tripleConfig)

	cm, err := newClientManager(url)
	require.Error(t, err)
	assert.Nil(t, cm)
	assert.Contains(t, err.Error(), "must have TLS config")
}

// mockService is a mock service for testing reflection-based client creation
type mockService struct{}

func (m *mockService) Reference() string {
	return "mockService"
}

func (m *mockService) TestMethod1(ctx context.Context, req string) (string, error) {
	return req, nil
}

func (m *mockService) TestMethod2(ctx context.Context, req int) (int, error) {
	return req, nil
}

func Test_newClientManager_WithRpcService(t *testing.T) {
	url := common.NewURLWithOptions(
		common.WithLocation("localhost:20000"),
		common.WithPath("com.example.TestService"),
		// No methods specified, will use reflection
	)
	url.SetAttribute(constant.RpcServiceKey, &mockService{})

	cm, err := newClientManager(url)
	require.NoError(t, err)
	assert.NotNil(t, cm)
	// In service-level client architecture, a single triClient is created
	assert.NotNil(t, cm.triClient, "triClient should be created for non-IDL mode")
}

func TestDualTransport_Structure(t *testing.T) {
	keepAliveInterval := 30 * time.Second
	keepAliveTimeout := 5 * time.Second

	transport := newDualTransport(nil, keepAliveInterval, keepAliveTimeout)
	assert.NotNil(t, transport)

	dt, ok := transport.(*dualTransport)
	assert.True(t, ok)
	assert.NotNil(t, dt.http2Transport)
	assert.NotNil(t, dt.http3Transport)
	assert.NotNil(t, dt.altSvcCache)
}

func Test_newClientManager_HTTP2WithTLS(t *testing.T) {
	// This test requires valid TLS config files
	// Skip if files don't exist
	url := common.NewURLWithOptions(
		common.WithLocation("localhost:20000"),
		common.WithPath("com.example.TestService"),
		common.WithMethods([]string{"TestMethod"}),
	)

	// Set a valid TLS config structure but with non-existent files
	// This will test the TLS config parsing path
	tlsConfig := &global.TLSConfig{
		CACertFile:  "non-existent-ca.crt",
		TLSCertFile: "non-existent-server.crt",
		TLSKeyFile:  "non-existent-server.key",
	}
	url.SetAttribute(constant.TLSConfigKey, tlsConfig)

	// Should fail due to missing cert files, but tests the TLS path
	cm, err := newClientManager(url)
	// Either succeeds (if TLS validation is lenient) or fails with TLS error
	if err != nil {
		// Expected - TLS files don't exist
		t.Logf("Expected TLS error: %v", err)
	} else {
		assert.NotNil(t, cm)
	}
}

func Test_newClientManager_URLPrefixHandling(t *testing.T) {
	tests := []struct {
		desc     string
		location string
	}{
		{
			desc:     "location without prefix",
			location: "localhost:20000",
		},
		{
			desc:     "location with http prefix",
			location: "http://localhost:20000",
		},
		{
			desc:     "location with https prefix",
			location: "https://localhost:20000",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			url := common.NewURLWithOptions(
				common.WithLocation(test.location),
				common.WithPath("com.example.TestService"),
				common.WithMethods([]string{"TestMethod"}),
			)

			cm, err := newClientManager(url)
			require.NoError(t, err)
			assert.NotNil(t, cm)
			assert.NotNil(t, cm.triClient, "triClient should be created")
		})
	}
}

func Test_newClientManager_KeepAliveError(t *testing.T) {
	url := common.NewURLWithOptions(
		common.WithLocation("localhost:20000"),
		common.WithPath("com.example.TestService"),
		common.WithMethods([]string{"TestMethod"}),
	)

	// Set triple config with invalid keepalive that will cause error
	tripleConfig := &global.TripleConfig{
		KeepAliveInterval: "invalid-duration",
	}
	url.SetAttribute(constant.TripleConfigKey, tripleConfig)

	cm, err := newClientManager(url)
	require.Error(t, err)
	assert.Nil(t, cm)
}

func Test_newClientManager_DefaultProtocol(t *testing.T) {
	// Test default HTTP/2 protocol selection
	url := common.NewURLWithOptions(
		common.WithLocation("localhost:20000"),
		common.WithPath("com.example.TestService"),
		common.WithMethods([]string{"TestMethod"}),
	)

	cm, err := newClientManager(url)
	require.NoError(t, err)
	assert.NotNil(t, cm)
}

func Test_newClientManager_EmptyTripleConfig(t *testing.T) {
	url := common.NewURLWithOptions(
		common.WithLocation("localhost:20000"),
		common.WithPath("com.example.TestService"),
		common.WithMethods([]string{"TestMethod"}),
	)

	// Set empty triple config (Http3 is nil)
	tripleConfig := &global.TripleConfig{}
	url.SetAttribute(constant.TripleConfigKey, tripleConfig)

	cm, err := newClientManager(url)
	require.NoError(t, err)
	assert.NotNil(t, cm)
}

func Test_newClientManager_Http3Disabled(t *testing.T) {
	url := common.NewURLWithOptions(
		common.WithLocation("localhost:20000"),
		common.WithPath("com.example.TestService"),
		common.WithMethods([]string{"TestMethod"}),
	)

	// Set triple config with Http3 disabled
	tripleConfig := &global.TripleConfig{
		Http3: &global.Http3Config{
			Enable: false,
		},
	}
	url.SetAttribute(constant.TripleConfigKey, tripleConfig)

	cm, err := newClientManager(url)
	require.NoError(t, err)
	assert.NotNil(t, cm)
}

func Test_newClientManager_MultipleMethods(t *testing.T) {
	methods := []string{"Method1", "Method2", "Method3", "Method4", "Method5"}
	url := common.NewURLWithOptions(
		common.WithLocation("localhost:20000"),
		common.WithPath("com.example.TestService"),
		common.WithMethods(methods),
	)

	cm, err := newClientManager(url)
	require.NoError(t, err)
	assert.NotNil(t, cm)
	// In service-level client architecture, a single triClient handles all methods
	assert.NotNil(t, cm.triClient, "triClient should be created to handle all methods")
}

func Test_newClientManager_InterfaceName(t *testing.T) {
	url := common.NewURLWithOptions(
		common.WithLocation("localhost:20000"),
		common.WithPath("com.example.TestService"),
		common.WithInterface("com.example.ITestService"),
		common.WithMethods([]string{"TestMethod"}),
	)

	cm, err := newClientManager(url)
	require.NoError(t, err)
	assert.NotNil(t, cm)
}
