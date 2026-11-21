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

package getty

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

import (
	getty "github.com/apache/dubbo-getty"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

func TestInitServer(t *testing.T) {
	url, err := common.NewURL("dubbo://127.0.0.1:20003/test")
	require.NoError(t, err)
	url.SetAttribute(constant.ProtocolConfigKey, map[string]*global.ProtocolConfig{
		"dubbo": {
			Name: "dubbo",
			Ip:   "127.0.0.1",
			Port: "20003",
		},
	})
	url.SetAttribute(constant.ApplicationKey, global.ApplicationConfig{})
	initServer(url)
}

func TestInitServerTLS(t *testing.T) {
	newURL := func() *common.URL {
		url, err := common.NewURL("dubbo://127.0.0.1:20003/test")
		require.NoError(t, err)
		url.SetAttribute(constant.ProtocolConfigKey, map[string]*global.ProtocolConfig{
			"dubbo": {
				Name:   "dubbo",
				Ip:     "127.0.0.1",
				Port:   "20003",
				Params: map[string]any{},
			},
		})
		url.SetAttribute(constant.ApplicationKey, global.ApplicationConfig{})
		return url
	}

	t.Run("valid TLS config enables SSLEnabled and TLSBuilder", func(t *testing.T) {
		srvConf = GetDefaultServerConfig()
		url := newURL()
		url.SetAttribute(constant.TLSConfigKey, &global.TLSConfig{
			TLSCertFile: "/path/to/server.crt",
			TLSKeyFile:  "/path/to/server.key",
		})
		initServer(url)
		assert.True(t, srvConf.SSLEnabled)
		assert.NotNil(t, srvConf.TLSBuilder)
	})

	t.Run("invalid TLS config keeps TLS disabled", func(t *testing.T) {
		srvConf = GetDefaultServerConfig()
		url := newURL()
		url.SetAttribute(constant.TLSConfigKey, &global.TLSConfig{
			TLSCertFile: "",
			TLSKeyFile:  "",
		})
		initServer(url)
		assert.False(t, srvConf.SSLEnabled)
	})

	t.Run("wrong TLSConfigKey type returns early without panic", func(t *testing.T) {
		srvConf = GetDefaultServerConfig()
		url := newURL()
		url.SetAttribute(constant.TLSConfigKey, "not a *global.TLSConfig")
		initServer(url)
		assert.False(t, srvConf.SSLEnabled)
	})
}

type serverConfigReadingParams struct {
	sessionNumber int
	called        bool
	observed      ServerConfig
	entered       chan struct{}
	release       chan struct{}
}

func (p *serverConfigReadingParams) MarshalYAML() (any, error) {
	p.observed = GetServerConfig()
	p.called = true
	if p.entered != nil {
		close(p.entered)
		<-p.release
	}
	return map[string]any{"session-number": p.sessionNumber}, nil
}

func TestNewServerParamsCanReadServerConfig(t *testing.T) {
	t.Run("reads defaults", func(t *testing.T) {
		preserveServerConfig(t)
		conf := *GetDefaultServerConfig()
		SetServerConfig(conf)
		params := &serverConfigReadingParams{sessionNumber: 123}

		server := NewServer(newServerConfigTestURL(t, params), nil)

		assert.True(t, params.called)
		assert.Equal(t, conf, params.observed)
		conf.SessionNumber = params.sessionNumber
		assert.Equal(t, conf, server.conf)
		assert.Equal(t, conf, GetServerConfig())
	})

	t.Run("preserves concurrent updates", func(t *testing.T) {
		preserveServerConfig(t)
		SetServerConfig(*GetDefaultServerConfig())
		params := &serverConfigReadingParams{
			sessionNumber: 124,
			entered:       make(chan struct{}),
			release:       make(chan struct{}),
		}
		latest := *GetDefaultServerConfig()
		latest.SessionTimeout = "90s"
		latest.GrPoolSize = 3
		latest.GettySessionParam.SessionName = "concurrent"
		latest.GettySessionParam.MaxMsgLen = 4096
		require.NoError(t, latest.CheckValidity())
		updated := make(chan struct{})
		go func() {
			defer close(updated)
			<-params.entered
			SetServerConfig(latest)
			close(params.release)
		}()

		server := NewServer(newServerConfigTestURL(t, params), nil)
		<-updated

		assert.True(t, params.called)
		latest.SessionNumber = params.sessionNumber
		assert.Equal(t, latest, server.conf)
		assert.Equal(t, latest, GetServerConfig())
	})
}

func TestSetServerConfigCopiesBuiltinTLSBuilder(t *testing.T) {
	preserveServerConfig(t)
	original := serverConfigTestTLSBuilder("original")
	expected := original
	conf := *GetDefaultServerConfig()
	conf.TLSBuilder = &original
	SetServerConfig(conf)
	url := newServerConfigTestURL(t, nil)
	existing := NewServer(url, nil)

	original = serverConfigTestTLSBuilder("changed")

	assert.Equal(t, expected, *requireServerTLSBuilder(t, GetServerConfig().TLSBuilder))
	assert.Equal(t, expected, *requireServerTLSBuilder(t, existing.conf.TLSBuilder))
	created := NewServer(url, nil)
	assert.Equal(t, expected, *requireServerTLSBuilder(t, created.conf.TLSBuilder))
}

func TestGetServerConfigCopiesBuiltinTLSBuilder(t *testing.T) {
	preserveServerConfig(t)
	original := serverConfigTestTLSBuilder("original")
	expected := original
	conf := *GetDefaultServerConfig()
	conf.TLSBuilder = &original
	SetServerConfig(conf)
	url := newServerConfigTestURL(t, nil)
	existing := NewServer(url, nil)
	first := GetServerConfig()
	second := GetServerConfig()
	firstBuilder := requireServerTLSBuilder(t, first.TLSBuilder)
	secondBuilder := requireServerTLSBuilder(t, second.TLSBuilder)
	srvConfLock.RLock()
	retainedBuilder := srvConf.TLSBuilder
	srvConfLock.RUnlock()
	assert.NotSame(t, retainedBuilder, existing.conf.TLSBuilder)
	assert.NotSame(t, firstBuilder, secondBuilder)

	changed := serverConfigTestTLSBuilder("changed")
	*firstBuilder = changed

	assert.Equal(t, expected, *secondBuilder)
	assert.Equal(t, expected, *requireServerTLSBuilder(t, GetServerConfig().TLSBuilder))
	assert.Equal(t, expected, *requireServerTLSBuilder(t, existing.conf.TLSBuilder))

	SetServerConfig(first)
	created := NewServer(url, nil)
	assert.Equal(t, changed, *requireServerTLSBuilder(t, GetServerConfig().TLSBuilder))
	assert.Equal(t, changed, *requireServerTLSBuilder(t, created.conf.TLSBuilder))
	assert.Equal(t, expected, *requireServerTLSBuilder(t, existing.conf.TLSBuilder))

	*firstBuilder = serverConfigTestTLSBuilder("changed again")

	assert.Equal(t, changed, *requireServerTLSBuilder(t, GetServerConfig().TLSBuilder))
	assert.Equal(t, changed, *requireServerTLSBuilder(t, created.conf.TLSBuilder))
	assert.Equal(t, expected, *requireServerTLSBuilder(t, existing.conf.TLSBuilder))
	assert.Equal(t, changed, *requireServerTLSBuilder(t, NewServer(url, nil).conf.TLSBuilder))
}

func TestServerStartIsolatesBuiltinTLSBuilder(t *testing.T) {
	for _, tc := range []struct {
		name       string
		fromGetter bool
	}{
		{name: "original caller"},
		{name: "getter result", fromGetter: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			preserveServerConfig(t)
			dir := t.TempDir()
			certPaths := [2]string{filepath.Join(dir, "cert-a.pem"), filepath.Join(dir, "cert-b.pem")}
			keyPath := filepath.Join(dir, "key.pem")
			builder := &getty.ServerTlsConfigBuilder{
				ServerKeyCertChainPath:        certPaths[0],
				ServerPrivateKeyPath:          keyPath,
				ServerTrustCertCollectionPath: certPaths[0],
			}
			conf := *GetDefaultServerConfig()
			conf.SSLEnabled = true
			conf.TLSBuilder = builder
			conf.GrPoolSize = 1
			SetServerConfig(conf)
			server := NewServer(newServerConfigTestURL(t, nil), nil)
			if tc.fromGetter {
				builder = requireServerTLSBuilder(t, GetServerConfig().TLSBuilder)
			}

			certPEM, keyPEM := newServerConfigTestCertificate(t)
			require.NoError(t, os.WriteFile(certPaths[0], certPEM, 0600))
			require.NoError(t, os.WriteFile(certPaths[1], certPEM, 0600))
			require.NoError(t, os.WriteFile(keyPath, keyPEM, 0600))

			start := make(chan struct{})
			ready := make(chan struct{})
			done := make(chan struct{})
			go func() {
				defer close(done)
				<-start
				close(ready)
				for i := range 1 << 20 {
					builder.ServerKeyCertChainPath = certPaths[i%2]
					if i%256 == 0 {
						runtime.Gosched()
					}
				}
			}()
			defer func() { <-done }()
			close(start)
			<-ready
			server.Start()
			<-done
			t.Cleanup(server.Stop)

			requireServerTLSHandshake(t, server, certPEM, keyPEM)
		})
	}
}

type serverConfigTestTLSBuilderProbe struct {
	calls int
}

func (b *serverConfigTestTLSBuilderProbe) BuildTlsConfig() (*tls.Config, error) {
	b.calls++
	return &tls.Config{}, nil
}

func TestServerConfigTLSBuilderCompatibility(t *testing.T) {
	var typedNil *getty.ServerTlsConfigBuilder
	for _, tc := range []struct {
		name    string
		builder getty.TlsConfigBuilder
	}{
		{name: "nil interface"},
		{name: "typed nil built-in pointer", builder: typedNil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			preserveServerConfig(t)
			conf := *GetDefaultServerConfig()
			conf.SSLEnabled = true
			conf.TLSBuilder = tc.builder
			SetServerConfig(conf)

			assert.Equal(t, tc.builder, GetServerConfig().TLSBuilder)
			server := NewServer(newServerConfigTestURL(t, nil), nil)
			assert.Equal(t, tc.builder, server.conf.TLSBuilder)
		})
	}

	t.Run("custom builder retains identity and defers construction", func(t *testing.T) {
		preserveServerConfig(t)
		builder := &serverConfigTestTLSBuilderProbe{}
		conf := *GetDefaultServerConfig()
		conf.SSLEnabled = true
		conf.TLSBuilder = builder
		SetServerConfig(conf)
		assert.Zero(t, builder.calls)
		assert.Same(t, builder, GetServerConfig().TLSBuilder)
		assert.Zero(t, builder.calls)
		server := NewServer(newServerConfigTestURL(t, nil), nil)
		assert.Same(t, builder, server.conf.TLSBuilder)
		assert.Zero(t, builder.calls)
	})
}

func preserveServerConfig(t *testing.T) {
	t.Helper()
	srvConfLock.RLock()
	original := srvConf
	srvConfLock.RUnlock()
	t.Cleanup(func() {
		srvConfLock.Lock()
		srvConf = original
		srvConfLock.Unlock()
	})
}

func newServerConfigTestURL(t *testing.T, params any) *common.URL {
	t.Helper()
	remoting.RegistryCodec("dubbo", &DubboTestCodec{})
	url, err := common.NewURL("dubbo://127.0.0.1:0/test")
	require.NoError(t, err)
	url.SetAttribute(constant.ProtocolConfigKey, map[string]*global.ProtocolConfig{
		"dubbo": {Name: "dubbo", Ip: "127.0.0.1", Port: "0", Params: params},
	})
	url.SetAttribute(constant.ApplicationKey, global.ApplicationConfig{})
	return url
}

func serverConfigTestTLSBuilder(prefix string) getty.ServerTlsConfigBuilder {
	return getty.ServerTlsConfigBuilder{
		ServerKeyCertChainPath:        prefix + "-cert.pem",
		ServerPrivateKeyPath:          prefix + "-key.pem",
		ServerKeyPassword:             prefix + "-password",
		ServerTrustCertCollectionPath: prefix + "-ca.pem",
	}
}

func requireServerTLSBuilder(t *testing.T, builder getty.TlsConfigBuilder) *getty.ServerTlsConfigBuilder {
	t.Helper()
	builtin, ok := builder.(*getty.ServerTlsConfigBuilder)
	require.True(t, ok)
	require.NotNil(t, builtin)
	return builtin
}

func newServerConfigTestCertificate(t *testing.T) ([]byte, []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	certificate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "getty test"},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(time.Hour),
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
		BasicConstraintsValid: true,
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, certificate, certificate, &key.PublicKey, key)
	require.NoError(t, err)
	keyDER, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
}

func requireServerTLSHandshake(t *testing.T, server *Server, certPEM, keyPEM []byte) {
	t.Helper()
	stream, ok := server.tcpServer.(getty.StreamServer)
	require.True(t, ok)
	listener := stream.Listener()
	require.NotNil(t, listener)
	certificate, err := tls.X509KeyPair(certPEM, keyPEM)
	require.NoError(t, err)
	roots := x509.NewCertPool()
	require.True(t, roots.AppendCertsFromPEM(certPEM))
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 5 * time.Second}, "tcp", listener.Addr().String(), &tls.Config{
		Certificates: []tls.Certificate{certificate},
		RootCAs:      roots,
		MinVersion:   tls.VersionTLS12,
	})
	require.NoError(t, err)
	defer conn.Close()
	assert.True(t, conn.ConnectionState().HandshakeComplete)
}
