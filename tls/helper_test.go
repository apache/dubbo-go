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

package tls

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/global"
)

// testCertFiles holds paths to generated test certificate files
type testCertFiles struct {
	caCertFile     string
	serverCertFile string
	serverKeyFile  string
	clientCertFile string
	clientKeyFile  string
	tempDir        string
}

// generateTestCerts generates self-signed CA, server, and client certificates for testing
func generateTestCerts(t *testing.T) *testCertFiles {
	tempDir, err := os.MkdirTemp("", "tls_test")
	assert.NoError(t, err)

	// Generate CA key pair
	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.NoError(t, err)

	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test CA"},
			CommonName:   "Test CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	caCertDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	assert.NoError(t, err)

	caCertFile := filepath.Join(tempDir, "ca.pem")
	err = writePEMFile(caCertFile, "CERTIFICATE", caCertDER)
	assert.NoError(t, err)

	caCert, err := x509.ParseCertificate(caCertDER)
	assert.NoError(t, err)

	// Generate server key pair
	serverKey, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.NoError(t, err)

	serverTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			Organization: []string{"Test Server"},
			CommonName:   "localhost",
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(24 * time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:    []string{"localhost"},
	}

	serverCertDER, err := x509.CreateCertificate(rand.Reader, serverTemplate, caCert, &serverKey.PublicKey, caKey)
	assert.NoError(t, err)

	serverCertFile := filepath.Join(tempDir, "server.pem")
	err = writePEMFile(serverCertFile, "CERTIFICATE", serverCertDER)
	assert.NoError(t, err)

	serverKeyFile := filepath.Join(tempDir, "server-key.pem")
	err = writePEMFile(serverKeyFile, "RSA PRIVATE KEY", x509.MarshalPKCS1PrivateKey(serverKey))
	assert.NoError(t, err)

	// Generate client key pair
	clientKey, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.NoError(t, err)

	clientTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(3),
		Subject: pkix.Name{
			Organization: []string{"Test Client"},
			CommonName:   "client",
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(24 * time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	clientCertDER, err := x509.CreateCertificate(rand.Reader, clientTemplate, caCert, &clientKey.PublicKey, caKey)
	assert.NoError(t, err)

	clientCertFile := filepath.Join(tempDir, "client.pem")
	err = writePEMFile(clientCertFile, "CERTIFICATE", clientCertDER)
	assert.NoError(t, err)

	clientKeyFile := filepath.Join(tempDir, "client-key.pem")
	err = writePEMFile(clientKeyFile, "RSA PRIVATE KEY", x509.MarshalPKCS1PrivateKey(clientKey))
	assert.NoError(t, err)

	return &testCertFiles{
		caCertFile:     caCertFile,
		serverCertFile: serverCertFile,
		serverKeyFile:  serverKeyFile,
		clientCertFile: clientCertFile,
		clientKeyFile:  clientKeyFile,
		tempDir:        tempDir,
	}
}

// writePEMFile writes data to a PEM file
func writePEMFile(filename, blockType string, data []byte) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	return pem.Encode(file, &pem.Block{
		Type:  blockType,
		Bytes: data,
	})
}

// cleanup removes the temporary directory and all test certificates
func (tc *testCertFiles) cleanup() {
	os.RemoveAll(tc.tempDir)
}

func TestIsServerTLSValid(t *testing.T) {
	tests := []struct {
		name     string
		tlsConf  *global.TLSConfig
		expected bool
	}{
		{
			name:     "Nil TLSConfig",
			tlsConf:  nil,
			expected: false,
		},
		{
			name: "Valid TLSConfig with cert and key",
			tlsConf: &global.TLSConfig{
				TLSCertFile: "cert.pem",
				TLSKeyFile:  "key.pem",
			},
			expected: true,
		},
		{
			name: "Invalid TLSConfig with empty cert and key",
			tlsConf: &global.TLSConfig{
				TLSCertFile: "",
				TLSKeyFile:  "",
			},
			expected: false,
		},
		{
			name: "Invalid TLSConfig with only cert",
			tlsConf: &global.TLSConfig{
				TLSCertFile: "cert.pem",
				TLSKeyFile:  "",
			},
			expected: false,
		},
		{
			name: "Invalid TLSConfig with only key",
			tlsConf: &global.TLSConfig{
				TLSCertFile: "",
				TLSKeyFile:  "key.pem",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsServerTLSValid(tt.tlsConf)
			assert.Equal(t, tt.expected, result, "Test case %s failed", tt.name)
		})
	}
}

func TestIsClientTLSValid(t *testing.T) {
	tests := []struct {
		name     string
		tlsConf  *global.TLSConfig
		expected bool
	}{
		{
			name:     "Nil TLSConfig",
			tlsConf:  nil,
			expected: false,
		},
		{
			name: "Valid Client TLSConfig with CA cert",
			tlsConf: &global.TLSConfig{
				CACertFile: "ca.pem",
			},
			expected: true,
		},
		{
			name: "Invalid Client TLSConfig with empty CA cert",
			tlsConf: &global.TLSConfig{
				CACertFile: "",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsClientTLSValid(tt.tlsConf)
			assert.Equal(t, tt.expected, result, "Test case %s failed", tt.name)
		})
	}
}

func TestGetServerTlSConfig(t *testing.T) {
	certs := generateTestCerts(t)
	defer certs.cleanup()

	tests := []struct {
		name        string
		tlsConf     *global.TLSConfig
		expectNil   bool
		expectError bool
		validate    func(t *testing.T, cfg *tls.Config)
	}{
		{
			name: "Empty cert and key returns nil config",
			tlsConf: &global.TLSConfig{
				TLSCertFile: "",
				TLSKeyFile:  "",
			},
			expectNil:   true,
			expectError: false,
		},
		{
			name: "Valid server TLS without mTLS",
			tlsConf: &global.TLSConfig{
				TLSCertFile:   certs.serverCertFile,
				TLSKeyFile:    certs.serverKeyFile,
				TLSServerName: "localhost",
			},
			expectNil:   false,
			expectError: false,
			validate: func(t *testing.T, cfg *tls.Config) {
				assert.Len(t, cfg.Certificates, 1)
				assert.Equal(t, "localhost", cfg.ServerName)
				assert.Nil(t, cfg.ClientCAs)
				assert.Equal(t, tls.NoClientCert, cfg.ClientAuth)
			},
		},
		{
			name: "Valid server TLS with mTLS",
			tlsConf: &global.TLSConfig{
				CACertFile:    certs.caCertFile,
				TLSCertFile:   certs.serverCertFile,
				TLSKeyFile:    certs.serverKeyFile,
				TLSServerName: "localhost",
			},
			expectNil:   false,
			expectError: false,
			validate: func(t *testing.T, cfg *tls.Config) {
				assert.Len(t, cfg.Certificates, 1)
				assert.Equal(t, "localhost", cfg.ServerName)
				assert.NotNil(t, cfg.ClientCAs)
				assert.Equal(t, tls.RequireAndVerifyClientCert, cfg.ClientAuth)
			},
		},
		{
			name: "Invalid cert file path",
			tlsConf: &global.TLSConfig{
				TLSCertFile: "/nonexistent/cert.pem",
				TLSKeyFile:  certs.serverKeyFile,
			},
			expectNil:   true,
			expectError: true,
		},
		{
			name: "Invalid key file path",
			tlsConf: &global.TLSConfig{
				TLSCertFile: certs.serverCertFile,
				TLSKeyFile:  "/nonexistent/key.pem",
			},
			expectNil:   true,
			expectError: true,
		},
		{
			name: "Invalid CA cert file path",
			tlsConf: &global.TLSConfig{
				CACertFile:  "/nonexistent/ca.pem",
				TLSCertFile: certs.serverCertFile,
				TLSKeyFile:  certs.serverKeyFile,
			},
			expectNil:   true,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := GetServerTlSConfig(tt.tlsConf)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			if tt.expectNil {
				assert.Nil(t, cfg)
			} else {
				assert.NotNil(t, cfg)
				if tt.validate != nil {
					tt.validate(t, cfg)
				}
			}
		})
	}
}

func TestGetServerTlSConfig_InvalidCACert(t *testing.T) {
	certs := generateTestCerts(t)
	defer certs.cleanup()

	// Create an invalid CA cert file
	invalidCACertFile := filepath.Join(certs.tempDir, "invalid_ca.pem")
	err := os.WriteFile(invalidCACertFile, []byte("invalid cert content"), 0644)
	assert.NoError(t, err)

	tlsConf := &global.TLSConfig{
		CACertFile:  invalidCACertFile,
		TLSCertFile: certs.serverCertFile,
		TLSKeyFile:  certs.serverKeyFile,
	}

	cfg, err := GetServerTlSConfig(tlsConf)
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "failed to parse root certificate")
}

func TestGetClientTlSConfig(t *testing.T) {
	certs := generateTestCerts(t)
	defer certs.cleanup()

	tests := []struct {
		name        string
		tlsConf     *global.TLSConfig
		expectNil   bool
		expectError bool
		validate    func(t *testing.T, cfg *tls.Config)
	}{
		{
			name: "Empty CA cert returns nil config",
			tlsConf: &global.TLSConfig{
				CACertFile: "",
			},
			expectNil:   true,
			expectError: false,
		},
		{
			name: "Valid client TLS without mTLS",
			tlsConf: &global.TLSConfig{
				CACertFile:    certs.caCertFile,
				TLSServerName: "localhost",
			},
			expectNil:   false,
			expectError: false,
			validate: func(t *testing.T, cfg *tls.Config) {
				assert.NotNil(t, cfg.RootCAs)
				assert.Equal(t, "localhost", cfg.ServerName)
				assert.Empty(t, cfg.Certificates)
			},
		},
		{
			name: "Valid client TLS with mTLS",
			tlsConf: &global.TLSConfig{
				CACertFile:    certs.caCertFile,
				TLSCertFile:   certs.clientCertFile,
				TLSKeyFile:    certs.clientKeyFile,
				TLSServerName: "localhost",
			},
			expectNil:   false,
			expectError: false,
			validate: func(t *testing.T, cfg *tls.Config) {
				assert.NotNil(t, cfg.RootCAs)
				assert.Equal(t, "localhost", cfg.ServerName)
				assert.Len(t, cfg.Certificates, 1)
			},
		},
		{
			name: "Invalid CA cert file path",
			tlsConf: &global.TLSConfig{
				CACertFile: "/nonexistent/ca.pem",
			},
			expectNil:   true,
			expectError: true,
		},
		{
			name: "Invalid client cert file path",
			tlsConf: &global.TLSConfig{
				CACertFile:  certs.caCertFile,
				TLSCertFile: "/nonexistent/client.pem",
				TLSKeyFile:  certs.clientKeyFile,
			},
			expectNil:   true,
			expectError: true,
		},
		{
			name: "Invalid client key file path",
			tlsConf: &global.TLSConfig{
				CACertFile:  certs.caCertFile,
				TLSCertFile: certs.clientCertFile,
				TLSKeyFile:  "/nonexistent/key.pem",
			},
			expectNil:   true,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := GetClientTlSConfig(tt.tlsConf)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			if tt.expectNil {
				assert.Nil(t, cfg)
			} else {
				assert.NotNil(t, cfg)
				if tt.validate != nil {
					tt.validate(t, cfg)
				}
			}
		})
	}
}

func TestGetClientTlSConfig_InvalidCACert(t *testing.T) {
	certs := generateTestCerts(t)
	defer certs.cleanup()

	// Create an invalid CA cert file
	invalidCACertFile := filepath.Join(certs.tempDir, "invalid_ca.pem")
	err := os.WriteFile(invalidCACertFile, []byte("invalid cert content"), 0644)
	assert.NoError(t, err)

	tlsConf := &global.TLSConfig{
		CACertFile: invalidCACertFile,
	}

	cfg, err := GetClientTlSConfig(tlsConf)
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "failed to parse root certificate")
}
