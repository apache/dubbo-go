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
	"crypto/tls"
	"net/http"
	"sync"
	"time"

	tri "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"golang.org/x/net/http2"
)

type originMode int

const (
	originUnknown originMode = iota
	originCandidate
	originProbing
	originH3Healthy
	originCooldown
)

type originState struct {
	mode     originMode
	failures int
}

// dualTransport is a transport that can handle both HTTP/2 and HTTP/3
// It uses HTTP Alternative Services (Alt-Svc) for protocol negotiation
type dualTransport struct {
	http2Transport http.RoundTripper
	http3Transport http.RoundTripper
	// Cache for alternative services to avoid repeated lookups
	altSvcCache *tri.AltSvcCache

	state originState

	mu sync.RWMutex
}

// newDualTransport creates a new dual transport that supports both HTTP/2 and HTTP/3
func newDualTransport(tlsConfig *tls.Config, keepAliveInterval, keepAliveTimeout time.Duration) http.RoundTripper {
	http2Transport := &http2.Transport{
		TLSClientConfig: tlsConfig,
		ReadIdleTimeout: keepAliveInterval,
		PingTimeout:     keepAliveTimeout,
	}

	http3Transport := &http3.Transport{
		TLSClientConfig: tlsConfig,
		QUICConfig: &quic.Config{
			KeepAlivePeriod: keepAliveInterval,
			MaxIdleTimeout:  keepAliveTimeout,
		},
	}

	return &dualTransport{
		http2Transport: http2Transport,
		http3Transport: http3Transport,
		altSvcCache:    tri.NewAltSvcCache(),
	}
}

// RoundTrip implements http.RoundTripper interface with HTTP Alternative Services support
func (dt *dualTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	host := req.URL.Host

	if dt.shouldUseH3(host) {
		resp, err := dt.http3Transport.RoundTrip(req)
		if err == nil {
			dt.markH3Success(host)
			return resp, nil
		}
		dt.markH3Failure(host)
		return nil, err
	}

	resp, err := dt.http2Transport.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	dt.observeH2Response(host, resp.Header)
	return resp, nil
}

func (dt *dualTransport) shouldUseH3(host string) bool {
	if host == "" {
		return false
	}

	altSvc := dt.altSvcCache.Get(host)
	if altSvc == nil || altSvc.Protocol != "h3" {
		return false
	}

	dt.mu.Lock()
	defer dt.mu.Unlock()

	switch dt.state.mode {
	case originH3Healthy:
		return true
	case originCooldown:

	case originUnknown, originCandidate, originProbing:
		return false
	default:
		return false
	}

}

func (dt *dualTransport) markH3Success(host string) {
	if host == "" {
		return
	}

	dt.mu.Lock()
	defer dt.mu.Unlock()

	dt.state.mode = originH3Healthy
}

func (dt *dualTransport) markH3Failure(host string) {
	if host == "" {
		return
	}

	dt.mu.Lock()
	defer dt.mu.Unlock()

	dt.state.mode = originCooldown
}

func (dt *dualTransport) observeH2Response(host string, headers http.Header) {
	if host == "" {
		return
	}

	dt.altSvcCache.UpdateFromHeaders(host, headers)

	altSvc := dt.altSvcCache.Get(host)
	if altSvc == nil || altSvc.Protocol != "h3" {
		return
	}

	dt.mu.Lock()
	switch dt.state.mode {
	case originUnknown:
		dt.state.mode = originCandidate

	case originCooldown:

	case originCandidate, originProbing, originH3Healthy:

	}
	dt.mu.Unlock()

	dt.maybeStartProbe()
}

func (dt *dualTransport) maybeStartProbe()
