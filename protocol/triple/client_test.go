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

func TestParseAltSvcHeader(t *testing.T) {
	dt := &dualTransport{}

	tests := []struct {
		name     string
		header   string
		expected []*altSvcInfo
	}{
		{
			name:   "HTTP/3 only",
			header: `h3=":443"; ma=86400`,
			expected: []*altSvcInfo{
				{
					protocol: "h3",
					host:     "",
					port:     "443",
					expires:  time.Now().Add(24 * time.Hour),
				},
			},
		},
		{
			name:   "HTTP/3 with host and port",
			header: `h3="example.com:443"; ma=3600`,
			expected: []*altSvcInfo{
				{
					protocol: "h3",
					host:     "example.com",
					port:     "443",
					expires:  time.Now().Add(1 * time.Hour),
				},
			},
		},
		{
			name:   "Multiple protocols",
			header: `h3=":443"; ma=86400, h2=":443"; ma=86400`,
			expected: []*altSvcInfo{
				{
					protocol: "h3",
					host:     "",
					port:     "443",
					expires:  time.Now().Add(24 * time.Hour),
				},
				{
					protocol: "h2",
					host:     "",
					port:     "443",
					expires:  time.Now().Add(24 * time.Hour),
				},
			},
		},
		{
			name:   "HTTP/3 with different port",
			header: `h3=":8443"; ma=7200`,
			expected: []*altSvcInfo{
				{
					protocol: "h3",
					host:     "",
					port:     "8443",
					expires:  time.Now().Add(2 * time.Hour),
				},
			},
		},
		{
			name:     "Empty header",
			header:   "",
			expected: []*altSvcInfo{},
		},
		{
			name:     "Invalid format",
			header:   `invalid-format`,
			expected: []*altSvcInfo{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dt.parseAltSvcHeader(tt.header)
			assert.Len(t, result, len(tt.expected))

			for i, expected := range tt.expected {
				if i < len(result) {
					assert.Equal(t, expected.protocol, result[i].protocol)
					assert.Equal(t, expected.host, result[i].host)
					assert.Equal(t, expected.port, result[i].port)
					// Check that expires time is within a reasonable range
					assert.WithinDuration(t, expected.expires, result[i].expires, 5*time.Second)
				}
			}
		})
	}
}

func TestParseAuthority(t *testing.T) {
	dt := &dualTransport{}

	tests := []struct {
		name         string
		authority    string
		expectedHost string
		expectedPort string
	}{
		{
			name:         "Port only",
			authority:    ":443",
			expectedHost: "",
			expectedPort: "443",
		},
		{
			name:         "Host and port",
			authority:    "example.com:443",
			expectedHost: "example.com",
			expectedPort: "443",
		},
		{
			name:         "Host only",
			authority:    "example.com",
			expectedHost: "example.com",
			expectedPort: "",
		},
		{
			name:         "Different port",
			authority:    ":8443",
			expectedHost: "",
			expectedPort: "8443",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, port := dt.parseAuthority(tt.authority)
			assert.Equal(t, tt.expectedHost, host)
			assert.Equal(t, tt.expectedPort, port)
		})
	}
}

func TestUpdateAltSvcCache(t *testing.T) {
	dt := &dualTransport{
		altSvcCache: make(map[string]*altSvcInfo),
	}

	// Test HTTP/3 preference
	headers := http.Header{}
	headers.Set("Alt-Svc", `h3=":443"; ma=86400, h2=":443"; ma=86400`)

	dt.updateAltSvcCache("example.com", headers)

	dt.altSvcMu.RLock()
	cached, exists := dt.altSvcCache["example.com"]
	dt.altSvcMu.RUnlock()

	assert.True(t, exists)
	assert.Equal(t, "h3", cached.protocol)
	assert.Equal(t, "443", cached.port)
}

func TestGetAltSvcForHost(t *testing.T) {
	dt := &dualTransport{
		altSvcCache: make(map[string]*altSvcInfo),
	}

	// Test expired cache
	expiredAltSvc := &altSvcInfo{
		protocol: "h3",
		host:     "example.com",
		port:     "443",
		expires:  time.Now().Add(-1 * time.Hour), // Expired
	}

	dt.altSvcMu.Lock()
	dt.altSvcCache["example.com"] = expiredAltSvc
	dt.altSvcMu.Unlock()

	result := dt.getAltSvcForHost("example.com")
	assert.Nil(t, result, "Should return nil for expired cache")

	// Test valid cache
	validAltSvc := &altSvcInfo{
		protocol: "h3",
		host:     "example.com",
		port:     "443",
		expires:  time.Now().Add(1 * time.Hour), // Valid
	}

	dt.altSvcMu.Lock()
	dt.altSvcCache["example.com"] = validAltSvc
	dt.altSvcMu.Unlock()

	result = dt.getAltSvcForHost("example.com")
	assert.NotNil(t, result)
	assert.Equal(t, "h3", result.protocol)
}

func TestDualTransportWithAltSvc(t *testing.T) {
	// Test that dualTransport properly handles Alt-Svc headers
	dt := newDualTransport(nil, 30*time.Second, 5*time.Second)

	// Verify that the transport is created with alt-svc cache
	dualTransport, ok := dt.(*dualTransport)
	assert.True(t, ok)
	assert.NotNil(t, dualTransport.altSvcCache)
	assert.NotNil(t, dualTransport.http2Transport)
	assert.NotNil(t, dualTransport.http3Transport)
}
