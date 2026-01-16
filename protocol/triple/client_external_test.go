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

package triple_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
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
	"github.com/pkg/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple"
)

// blackHoleAddress is an address that is reserved for documentation and should not be routable.
// See RFC 5737. Connections to it should time out.
const blackHoleAddress = "192.0.2.1:8080"

// generateTestCerts generates self-signed CA and server certificates for testing.
// Returns paths to the generated certificate files and a cleanup function.
func generateTestCerts(t *testing.T) (caCertFile, serverCertFile, serverKeyFile string, cleanup func()) {
	tempDir, err := os.MkdirTemp("", "triple_test")
	require.NoError(t, err)

	cleanup = func() {
		os.RemoveAll(tempDir)
	}

	// Generate CA key pair
	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

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
	require.NoError(t, err)

	caCertFile = filepath.Join(tempDir, "ca.crt")
	err = writePEMFile(caCertFile, "CERTIFICATE", caCertDER)
	require.NoError(t, err)

	caCert, err := x509.ParseCertificate(caCertDER)
	require.NoError(t, err)

	// Generate server key pair
	serverKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

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
	require.NoError(t, err)

	serverCertFile = filepath.Join(tempDir, "server.crt")
	err = writePEMFile(serverCertFile, "CERTIFICATE", serverCertDER)
	require.NoError(t, err)

	serverKeyFile = filepath.Join(tempDir, "server.key")
	err = writePEMFile(serverKeyFile, "RSA PRIVATE KEY", x509.MarshalPKCS1PrivateKey(serverKey))
	require.NoError(t, err)

	return caCertFile, serverCertFile, serverKeyFile, cleanup
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

// testClientInvokeWithTimeout tests that a Triple client invocation correctly times out.
func testClientInvokeWithTimeout(t *testing.T, tlsConfig *global.TLSConfig) {

	// 1. Get an instance of TripleProtocol
	proto := triple.GetProtocol()

	// 2. Create a URL that will time out
	url, err := common.NewURL(
		"tri://"+blackHoleAddress,
		common.WithMethods([]string{"test"}),
		common.WithProtocol(triple.TRIPLE),
		common.WithParamsValue(constant.IDLMode, constant.NONIDL),
		common.WithParamsValue(constant.SerializationKey, constant.MsgpackSerialization),
	)
	require.NoError(t, err)

	if tlsConfig != nil {
		url.SetAttribute(constant.TLSConfigKey, tlsConfig)
		url.SetParam(constant.SslEnabledKey, "true")
	}

	// 3. Get an invoker, which will be a *TripleClient
	invoker := proto.Refer(url)
	require.NotNil(t, invoker)

	// 4. Create a dummy invocation with raw parameters
	inv := invocation.NewRPCInvocationWithOptions(
		invocation.WithMethodName("test"),
		invocation.WithParameterRawValues([]any{"param1", "param2"}),
	)
	inv.SetAttribute(constant.CallTypeKey, constant.CallUnary)

	// 5. Create a context with a short timeout
	timeout := 2 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// 6. Record start time and invoke. This should trigger a dial attempt.
	startTime := time.Now()
	result := invoker.Invoke(ctx, inv)
	duration := time.Since(startTime)

	err = result.Error()

	t.Logf("invoke result err: %+v, duration: %v", err, duration)

	require.Error(t, err)
	assert.Truef(t, errors.Is(err, context.DeadlineExceeded), "context deadline exceeded")

	// 7. Ensure that the duration is at least the timeout duration
	assert.GreaterOrEqual(t, duration, timeout)
	assert.Less(t, duration, timeout+time.Second, "Invocation took too long, likely not timing out correctly")
}

// TestClientInvokeWithTimeout tests that a Triple client invocation correctly times out.
// Test certificates are dynamically generated at runtime to avoid committing private keys to the repository.
func TestClientInvokeWithTimeout(t *testing.T) {
	t.Run(
		"without TLS", func(t *testing.T) {
			testClientInvokeWithTimeout(t, nil)
		},
	)

	t.Run(
		"with TLS", func(t *testing.T) {
			// Generate test certificates dynamically
			caCertFile, serverCertFile, serverKeyFile, cleanup := generateTestCerts(t)
			defer cleanup()

			// Set TLS configuration with generated cert files
			tlsConfig := &global.TLSConfig{
				CACertFile:  caCertFile,
				TLSCertFile: serverCertFile,
				TLSKeyFile:  serverKeyFile,
			}

			testClientInvokeWithTimeout(t, tlsConfig)
		},
	)
}
