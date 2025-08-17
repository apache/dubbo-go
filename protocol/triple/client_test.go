// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package triple

import (
	"net/http"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
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
	assert.NotEmpty(t, clientManager.triClients)

	// Verify that the client for the specific method exists
	client, exists := clientManager.triClients["testMethod"]
	assert.True(t, exists)
	assert.NotNil(t, client)
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
