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

package triple_protocol

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"math/big"
	"net"
	"net/http"
	"syscall"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
)

// newTestTLSConfig generates an in-memory self-signed certificate for the
// server-side TLS configuration. HTTP/3 requires TLS, so it is used by all
// HTTP/3 related tests.
func newTestTLSConfig(t *testing.T) *tls.Config {
	t.Helper()

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "localhost",
		},
		NotBefore:             time.Now().Add(-24 * time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
		BasicConstraintsValid: true,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	require.NoError(t, err)

	return &tls.Config{
		Certificates: []tls.Certificate{
			{Certificate: [][]byte{der}, PrivateKey: priv},
		},
	}
}

// getFreeAddr returns a free TCP address on loopback for the test server.
func getFreeAddr(t *testing.T) string {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := l.Addr().String()
	require.NoError(t, l.Close())

	return addr
}

// runServer starts the server in a goroutine and returns the channel that
// receives the error returned by Run. ListenAndServe is blocking, so the
// server must always be started this way in tests.
func runServer(srv *Server, protocol string, tlsConf *tls.Config) chan error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Run(protocol, tlsConf)
	}()
	return errCh
}

// isAddrInUse reports whether the error is a port conflict, i.e. the error
// chain unwraps to syscall.EADDRINUSE.
func isAddrInUse(err error) bool {
	return errors.Is(err, syscall.EADDRINUSE)
}

// startTestServer creates a server on a free port and starts it in a
// goroutine. It retries with a fresh port when binding fails.
func startTestServer(t *testing.T, protocol string, cfg *global.TripleConfig, tlsConf *tls.Config) (*Server, chan error) {
	t.Helper()

	for range 3 {
		srv := NewServer(getFreeAddr(t), cfg)
		errCh := runServer(srv, protocol, tlsConf)
		select {
		case err := <-errCh:
			if isAddrInUse(err) {
				continue
			}
			require.FailNow(t, "server failed to start", err)
		case <-time.After(100 * time.Millisecond):
		}
		return srv, errCh
	}
	require.FailNow(t, "failed to find a free port after 3 attempts")
	return nil, nil
}

// waitForTCPReady polls the address until a TCP connection can be established.
func waitForTCPReady(t *testing.T, addr string, timeout time.Duration) {
	t.Helper()

	require.Eventually(t, func() bool {
		conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if err != nil {
			return false
		}
		_ = conn.Close()
		return true
	}, timeout, 20*time.Millisecond)
}

// waitForHTTP3Stored waits until the HTTP/3 server has been stored, which
// means the startup path has passed the Store call. QUIC listens on UDP,
// so the TCP readiness probe does not apply here.
func waitForHTTP3Stored(t *testing.T, srv *Server) {
	t.Helper()

	require.Eventually(t, func() bool {
		return srv.http3Srv.Load() != nil
	}, 3*time.Second, 20*time.Millisecond)
}

// waitForServerExit waits for Run to return. The test fails if the server
// does not exit within the timeout.
func waitForServerExit(t *testing.T, errCh chan error, timeout time.Duration) error {
	t.Helper()

	select {
	case err := <-errCh:
		return err
	case <-time.After(timeout):
		require.FailNow(t, "server did not exit within", timeout)
		return nil
	}
}

func TestServer_HTTP2_StartAndStop(t *testing.T) {
	srv, errCh := startTestServer(t, constant.CallHTTP2, nil, nil)
	waitForTCPReady(t, srv.addr, 3*time.Second)

	require.NotNil(t, srv.httpSrv.Load())
	require.Nil(t, srv.http3Srv.Load())

	require.NoError(t, srv.Stop())
	require.ErrorIs(t, waitForServerExit(t, errCh, 5*time.Second), http.ErrServerClosed)
}

func TestServer_HTTP2_StartAndStopWithTLS(t *testing.T) {
	srv, errCh := startTestServer(t, constant.CallHTTP2, nil, newTestTLSConfig(t))
	waitForTCPReady(t, srv.addr, 3*time.Second)

	require.NotNil(t, srv.httpSrv.Load())
	require.Nil(t, srv.http3Srv.Load())

	require.NoError(t, srv.Stop())
	require.ErrorIs(t, waitForServerExit(t, errCh, 5*time.Second), http.ErrServerClosed)
}

func TestServer_HTTP3_StartAndStop(t *testing.T) {
	cfg := &global.TripleConfig{
		Http3: &global.Http3Config{Enable: true},
	}
	srv, errCh := startTestServer(t, constant.CallHTTP3, cfg, newTestTLSConfig(t))
	waitForHTTP3Stored(t, srv)

	require.NotNil(t, srv.http3Srv.Load())
	require.Nil(t, srv.httpSrv.Load())

	require.NoError(t, srv.Stop())
	require.ErrorIs(t, waitForServerExit(t, errCh, 5*time.Second), http.ErrServerClosed)
}

func TestServer_HTTP2AndHTTP3_StartAndStop(t *testing.T) {
	cfg := &global.TripleConfig{
		Http3: &global.Http3Config{Enable: true},
	}
	srv, errCh := startTestServer(t, constant.CallHTTP2AndHTTP3, cfg, newTestTLSConfig(t))
	waitForTCPReady(t, srv.addr, 3*time.Second)
	waitForHTTP3Stored(t, srv)

	require.NotNil(t, srv.httpSrv.Load())
	require.NotNil(t, srv.http3Srv.Load())

	require.NoError(t, srv.Stop())
	// startHttp2AndHttp3 swallows http.ErrServerClosed inside the errgroup,
	// so Run returns nil after the servers are closed.
	require.NoError(t, waitForServerExit(t, errCh, 5*time.Second))
}

func TestServer_HTTP2_StartAndGracefulStop(t *testing.T) {
	srv, errCh := startTestServer(t, constant.CallHTTP2, nil, nil)
	waitForTCPReady(t, srv.addr, 3*time.Second)

	require.NotNil(t, srv.httpSrv.Load())
	require.Nil(t, srv.http3Srv.Load())

	graceCtx, cancel := context.WithTimeout(context.Background(), constant.DefaultGracefulShutdownTimeout)
	defer cancel()
	require.NoError(t, srv.GracefulStop(graceCtx))
	require.ErrorIs(t, waitForServerExit(t, errCh, 5*time.Second), http.ErrServerClosed)
}

func TestServer_HTTP3_StartAndGracefulStop(t *testing.T) {
	cfg := &global.TripleConfig{
		Http3: &global.Http3Config{Enable: true},
	}
	srv, errCh := startTestServer(t, constant.CallHTTP3, cfg, newTestTLSConfig(t))
	waitForHTTP3Stored(t, srv)

	require.NotNil(t, srv.http3Srv.Load())
	require.Nil(t, srv.httpSrv.Load())

	graceCtx, cancel := context.WithTimeout(context.Background(), constant.DefaultGracefulShutdownTimeout)
	defer cancel()
	require.NoError(t, srv.GracefulStop(graceCtx))
	require.ErrorIs(t, waitForServerExit(t, errCh, 5*time.Second), http.ErrServerClosed)
}

func TestServer_HTTP2AndHTTP3_StartAndGracefulStop(t *testing.T) {
	cfg := &global.TripleConfig{
		Http3: &global.Http3Config{Enable: true},
	}
	srv, errCh := startTestServer(t, constant.CallHTTP2AndHTTP3, cfg, newTestTLSConfig(t))
	waitForTCPReady(t, srv.addr, 3*time.Second)
	waitForHTTP3Stored(t, srv)

	require.NotNil(t, srv.httpSrv.Load())
	require.NotNil(t, srv.http3Srv.Load())

	graceCtx, cancel := context.WithTimeout(context.Background(), constant.DefaultGracefulShutdownTimeout)
	defer cancel()
	require.NoError(t, srv.GracefulStop(graceCtx))
	// startHttp2AndHttp3 swallows http.ErrServerClosed inside the errgroup,
	// so Run returns nil after the servers are closed.
	require.NoError(t, waitForServerExit(t, errCh, 5*time.Second))
}

func TestServer_StopBeforeStart(t *testing.T) {
	srv := NewServer(getFreeAddr(t), nil)
	require.NoError(t, srv.Stop())
}

func TestServer_GracefulStopBeforeStart(t *testing.T) {
	srv := NewServer(getFreeAddr(t), nil)
	graceCtx, cancel := context.WithTimeout(context.Background(), constant.DefaultGracefulShutdownTimeout)
	defer cancel()
	require.NoError(t, srv.GracefulStop(graceCtx))
}

func TestServer_Run_HTTP3WithoutTLS(t *testing.T) {
	srv := NewServer(getFreeAddr(t), nil)
	err := srv.Run(constant.CallHTTP3, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must have TLS config")
}

func TestServer_Run_HTTP2AndHTTP3WithoutTLS(t *testing.T) {
	srv := NewServer(getFreeAddr(t), nil)
	err := srv.Run(constant.CallHTTP2AndHTTP3, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must have TLS config")
}

func TestServer_RunUnsupportedProtocol(t *testing.T) {
	srv := NewServer(getFreeAddr(t), nil)
	err := srv.Run("tcp", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported protocol")
}

// TestServer_RepeatedStartStop runs several Start/Stop cycles across all
// protocols. Each iteration creates a fresh Server, because an http.Server
// cannot be restarted after Close.
func TestServer_RepeatedStartStop(t *testing.T) {
	tlsConf := newTestTLSConfig(t)

	protocols := []struct {
		protocol string
		tlsConf  *tls.Config
	}{
		{protocol: constant.CallHTTP2},
		{protocol: constant.CallHTTP3, tlsConf: tlsConf},
		{protocol: constant.CallHTTP2AndHTTP3, tlsConf: tlsConf},
	}

	for range 3 {
		for _, tc := range protocols {
			cfg := &global.TripleConfig{}
			if tc.protocol == constant.CallHTTP3 || tc.protocol == constant.CallHTTP2AndHTTP3 {
				cfg.Http3 = &global.Http3Config{Enable: true}
			}
			srv, errCh := startTestServer(t, tc.protocol, cfg, tc.tlsConf)
			switch tc.protocol {
			case constant.CallHTTP2AndHTTP3:
				waitForTCPReady(t, srv.addr, 3*time.Second)
				waitForHTTP3Stored(t, srv)
			case constant.CallHTTP3:
				waitForHTTP3Stored(t, srv)
			default:
				waitForTCPReady(t, srv.addr, 3*time.Second)
			}

			require.NoError(t, srv.Stop())
			if tc.protocol == constant.CallHTTP2AndHTTP3 {
				require.NoError(t, waitForServerExit(t, errCh, 5*time.Second))
			} else {
				require.ErrorIs(t, waitForServerExit(t, errCh, 5*time.Second), http.ErrServerClosed)
			}
		}
	}
}
