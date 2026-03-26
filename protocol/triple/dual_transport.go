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
	"crypto/tls"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	tri "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"golang.org/x/net/http2"
)

type originMode int

const (
	// originUnknown means the origin has not advertised HTTP/3 yet.
	originUnknown originMode = iota
	// originCandidate means the origin advertised HTTP/3 and is waiting for validation.
	originCandidate
	// originProbing means an out-of-band HTTP/3 probe is in flight.
	originProbing
	// originH3Healthy means later requests may be sent over HTTP/3.
	originH3Healthy
	// originCooldown means HTTP/3 recently failed and should be avoided for a while.
	originCooldown
)

// originState tracks whether the current upstream origin is ready for HTTP/3.
type originState struct {
	mode          originMode
	failures      int
	cooldownUntil time.Time
}

// dualTransport is a transport that can handle both HTTP/2 and HTTP/3
// It uses HTTP Alternative Services (Alt-Svc) for protocol negotiation
type dualTransport struct {
	http2Transport http.RoundTripper
	http3Transport http.RoundTripper
	// Cache for alternative services to avoid repeated lookups
	altSvcCache *tri.AltSvcCache

	state originState

	mu sync.Mutex

	probeTimeout time.Duration
	baseCooldown time.Duration
	maxCooldown  time.Duration
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
		probeTimeout:   3 * time.Second,
		baseCooldown:   4 * time.Second,
		maxCooldown:    1 * time.Minute,
	}
}

// RoundTrip implements http.RoundTripper interface with HTTP Alternative Services support
func (dt *dualTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	host := req.URL.Host

	if dt.shouldUseH3(host) {
		// Only use HTTP/3 after a separate probe marks the origin healthy.
		// If the HTTP/3 request fails, return the error directly instead of
		// replaying the same request over HTTP/2 with a partially consumed body.
		resp, err := dt.http3Transport.RoundTrip(req)
		if err == nil {
			dt.altSvcCache.UpdateFromHeaders(host, resp.Header)
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

	// Learn HTTP/3 availability from HTTP/2 responses and validate it with
	// an independent probe before routing later requests over HTTP/3.
	dt.observeH2Response(req.URL, resp.Header)
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

	now := time.Now()

	dt.mu.Lock()
	defer dt.mu.Unlock()

	switch dt.state.mode {
	case originH3Healthy:
		return true

	case originCooldown:
		// Wait for the cooldown window to expire before probing HTTP/3 again.
		if now.Before(dt.state.cooldownUntil) {
			return false
		}

		dt.state.mode = originCandidate
		dt.state.cooldownUntil = time.Time{}
		return false

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
	dt.state.failures = 0
	dt.state.cooldownUntil = time.Time{}
}

func (dt *dualTransport) markH3Failure(host string) {
	if host == "" {
		return
	}

	dt.mu.Lock()
	defer dt.mu.Unlock()

	dt.state.failures++
	dt.state.mode = originCooldown
	dt.state.cooldownUntil = time.Now().Add(dt.nextCooldown(dt.state.failures))
}

func (dt *dualTransport) observeH2Response(u *url.URL, headers http.Header) {
	if u == nil || u.Host == "" {
		return
	}

	dt.altSvcCache.UpdateFromHeaders(u.Host, headers)

	altSvc := dt.altSvcCache.Get(u.Host)
	if altSvc == nil || altSvc.Protocol != "h3" {
		return
	}

	now := time.Now()

	dt.mu.Lock()
	switch dt.state.mode {
	case originUnknown:
		dt.state.mode = originCandidate

	case originCooldown:
		if !now.Before(dt.state.cooldownUntil) {
			dt.state.mode = originCandidate
			dt.state.cooldownUntil = time.Time{}
		}

	case originCandidate, originProbing, originH3Healthy:

	}
	dt.mu.Unlock()

	dt.maybeStartProbe(u.Host, u)
}

func (dt *dualTransport) maybeStartProbe(host string, u *url.URL) {
	if host == "" || u == nil {
		return
	}

	altSvc := dt.altSvcCache.Get(host)
	if altSvc == nil || altSvc.Protocol != "h3" {
		return
	}

	now := time.Now()

	dt.mu.Lock()
	switch dt.state.mode {
	case originH3Healthy, originProbing:
		dt.mu.Unlock()
		return
	case originCooldown:
		if now.Before(dt.state.cooldownUntil) {
			dt.mu.Unlock()
			return
		}
		dt.state.mode = originCandidate
		dt.state.cooldownUntil = time.Time{}
	case originUnknown:
		dt.state.mode = originCandidate

	case originCandidate:
	}

	probeURL := &url.URL{
		Scheme: u.Scheme,
		Host:   host,
		Path:   "/",
	}
	// Validate HTTP/3 readiness out of band so the current business request
	// can stay on HTTP/2.
	dt.state.mode = originProbing
	dt.mu.Unlock()

	go dt.runProbe(host, probeURL)
}

func (dt *dualTransport) runProbe(host string, probeURL *url.URL) {
	ctx, cancel := context.WithTimeout(context.Background(), dt.probeTimeout)
	defer cancel()

	// Probe with an independent request so business request bodies never need
	// to be replayed across transports.
	req, err := http.NewRequestWithContext(ctx, http.MethodOptions, probeURL.String(), nil)
	if err != nil {
		dt.markH3Failure(host)
		return
	}
	resp, err := dt.http3Transport.RoundTrip(req)
	if err != nil {
		dt.markH3Failure(host)
		return
	}
	defer resp.Body.Close()

	_, _ = io.Copy(io.Discard, resp.Body)

	dt.markH3Success(host)
}

func (dt *dualTransport) nextCooldown(failures int) time.Duration {
	if failures <= 1 {
		return dt.baseCooldown
	}
	// Increase the cooldown window after repeated HTTP/3 failures, capped by maxCooldown.
	d := dt.baseCooldown
	for i := 1; i < failures; i++ {
		d *= 2
		if d >= dt.maxCooldown {
			return dt.maxCooldown
		}
	}
	return d
}
