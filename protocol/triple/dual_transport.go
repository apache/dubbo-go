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
	"errors"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"

	"golang.org/x/net/http2"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	tri "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
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

const (
	defaultH3ProbeTimeout = 3 * time.Second
	defaultH3BaseCooldown = 4 * time.Second
	defaultH3MaxCooldown  = 1 * time.Minute
)

// originState tracks whether the current upstream origin is ready for HTTP/3.
type originState struct {
	mode          originMode
	failures      int
	cooldownUntil time.Time
}

// dualTransport keeps HTTP/2 as the stable path and only uses HTTP/3 after discovery and probe.
type dualTransport struct {
	http2Transport http.RoundTripper
	http3Transport http.RoundTripper
	// Cache for alternative services to avoid repeated lookups
	altSvcCache *tri.AltSvcCache

	// state tracks HTTP/3 readiness for the bound upstream origin.
	state originState

	// mu protects state and serializes transitions between unknown/probing/healthy/cooldown.
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
		probeTimeout:   defaultH3ProbeTimeout,
		baseCooldown:   defaultH3BaseCooldown,
		maxCooldown:    defaultH3MaxCooldown,
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
			dt.markH3Success(host)
			return resp, nil
		}
		if dt.shouldMarkH3Failure(req, err) {
			logger.Warnf("HTTP/3 request failed for %s: %v", req.URL.String(), err)
			dt.markH3Failure(host)
		}
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

// shouldUseH3 only decides whether the current request may use HTTP/3.
func (dt *dualTransport) shouldUseH3(host string) bool {
	if host == "" {
		return false
	}

	altSvc := dt.altSvcCache.Get(host)
	if altSvc == nil || altSvc.Protocol != constant.AltSvcProtocolH3 {
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

		logger.Debugf("HTTP/3 cooldown expired for %s", host)
		dt.state.mode = originCandidate
		dt.state.cooldownUntil = time.Time{}
		return false

	case originUnknown, originCandidate, originProbing:
		return false
	}
	return false
}

func (dt *dualTransport) shouldMarkH3Failure(req *http.Request, err error) bool {
	if req.Context().Err() != nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	return true
}

func (dt *dualTransport) markH3Success(host string) {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	dt.state.mode = originH3Healthy
	dt.state.failures = 0
	dt.state.cooldownUntil = time.Time{}
	logger.Debugf("HTTP/3 ready for %s", host)
}

func (dt *dualTransport) markH3Failure(host string) {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	dt.state.failures++
	dt.state.mode = originCooldown
	dt.state.cooldownUntil = time.Now().Add(dt.nextCooldown(dt.state.failures))
	logger.Debugf(
		"HTTP/3 cooldown for %s until %s after %d failure(s)",
		host,
		dt.state.cooldownUntil.Format(time.RFC3339),
		dt.state.failures,
	)
}

// observeH2Response learns Alt-Svc from the current HTTP/2 response for later requests.
func (dt *dualTransport) observeH2Response(u *url.URL, headers http.Header) {
	if u == nil || u.Host == "" {
		return
	}

	dt.altSvcCache.UpdateFromHeaders(u.Host, headers)

	altSvc := dt.altSvcCache.Get(u.Host)
	if altSvc == nil || altSvc.Protocol != constant.AltSvcProtocolH3 {
		return
	}

	now := time.Now()

	dt.mu.Lock()
	switch dt.state.mode {
	case originUnknown:
		dt.state.mode = originCandidate
		logger.Debugf("HTTP/3 advertised by %s", u.Host)

	case originCooldown:
		if !now.Before(dt.state.cooldownUntil) {
			dt.state.mode = originCandidate
			dt.state.cooldownUntil = time.Time{}
			logger.Debugf("HTTP/3 reprobe enabled for %s", u.Host)
		}

	case originCandidate, originProbing, originH3Healthy:
		// Already aware of HTTP/3, nothing to do
	}
	dt.mu.Unlock()

	dt.maybeStartProbe(u)
}

func (dt *dualTransport) maybeStartProbe(u *url.URL) {
	if u == nil || u.Host == "" {
		return
	}
	host := u.Host

	altSvc := dt.altSvcCache.Get(host)
	if altSvc == nil || altSvc.Protocol != constant.AltSvcProtocolH3 {
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
	case originUnknown, originCandidate:
		dt.state.mode = originCandidate
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

	logger.Debugf("Start HTTP/3 probe for %s via %s", host, probeURL.String())
	go dt.runProbe(probeURL)
}

func (dt *dualTransport) runProbe(probeURL *url.URL) {
	if probeURL == nil || probeURL.Host == "" {
		return
	}
	host := probeURL.Host

	ctx, cancel := context.WithTimeout(context.Background(), dt.probeTimeout)
	defer cancel()

	// Probe with an independent request so business request bodies never need
	// to be replayed across transports.
	req, err := http.NewRequestWithContext(ctx, http.MethodOptions, probeURL.String(), nil)
	if err != nil {
		logger.Debugf("Create HTTP/3 probe request failed for %s: %v", host, err)
		dt.markH3Failure(host)
		return
	}
	resp, err := dt.http3Transport.RoundTrip(req)
	if err != nil {
		logger.Debugf("HTTP/3 probe failed for %s: %v", host, err)
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
