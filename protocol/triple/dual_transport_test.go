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
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	tri "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newTestResponse(status int, headers http.Header) *http.Response {
	if headers == nil {
		headers = make(http.Header)
	}
	return &http.Response{
		StatusCode: status,
		Header:     headers,
		Body:       io.NopCloser(strings.NewReader("")),
	}
}

func TestDualTransport_H2AltSvcStartsProbeAndPromotesH3Healthy(t *testing.T) {
	t.Parallel()

	dt := &dualTransport{
		http2Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			headers := make(http.Header)
			headers.Set("Alt-Svc", `h3=":443"; ma=86400`)
			return newTestResponse(http.StatusOK, headers), nil
		}),
		altSvcCache:  tri.NewAltSvcCache(),
		probeTimeout: 100 * time.Millisecond,
		baseCooldown: 4 * time.Second,
		maxCooldown:  1 * time.Minute,
	}

	probeCalled := make(chan struct{}, 1)
	dt.http3Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		assert.Equal(t, http.MethodOptions, req.Method)
		assert.Equal(t, "/", req.URL.Path)
		probeCalled <- struct{}{}
		return newTestResponse(http.StatusMethodNotAllowed, nil), nil
	})

	req, err := http.NewRequest(http.MethodPost, "https://example.com/service", nil)
	require.NoError(t, err)

	resp, err := dt.RoundTrip(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	select {
	case <-probeCalled:
	case <-time.After(time.Second):
		t.Fatal("probe was not triggered")
	}

	require.Eventually(t, func() bool {
		dt.mu.Lock()
		defer dt.mu.Unlock()
		return dt.state.mode == originH3Healthy
	}, time.Second, 10*time.Millisecond)
}

func TestDualTransport_ConcurrentH2DiscoveryStartsSingleProbe(t *testing.T) {
	t.Parallel()

	const numRequests = 8

	var h2Calls atomic.Int32
	var h3Calls atomic.Int32

	dt := &dualTransport{
		http2Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			h2Calls.Add(1)
			headers := make(http.Header)
			headers.Set("Alt-Svc", `h3=":443"; ma=86400`)
			return newTestResponse(http.StatusOK, headers), nil
		}),
		altSvcCache:  tri.NewAltSvcCache(),
		probeTimeout: 100 * time.Millisecond,
		baseCooldown: 4 * time.Second,
		maxCooldown:  1 * time.Minute,
	}

	probeStarted := make(chan struct{}, 1)
	releaseProbe := make(chan struct{})
	dt.http3Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		h3Calls.Add(1)
		assert.Equal(t, http.MethodOptions, req.Method)
		select {
		case probeStarted <- struct{}{}:
		default:
		}
		<-releaseProbe
		return newTestResponse(http.StatusMethodNotAllowed, nil), nil
	})

	var wg sync.WaitGroup
	wg.Add(numRequests)
	for i := 0; i < numRequests; i++ {
		req, err := http.NewRequest(http.MethodPost, "https://example.com/service", nil)
		require.NoError(t, err)

		go func(req *http.Request) {
			defer wg.Done()

			resp, err := dt.RoundTrip(req)
			if !assert.NoError(t, err) {
				return
			}
			if assert.NotNil(t, resp) {
				assert.Equal(t, http.StatusOK, resp.StatusCode)
			}
		}(req)
	}

	select {
	case <-probeStarted:
	case <-time.After(time.Second):
		t.Fatal("probe was not triggered")
	}

	wg.Wait()
	assert.Equal(t, int32(numRequests), h2Calls.Load())
	assert.Equal(t, int32(1), h3Calls.Load())

	close(releaseProbe)

	require.Eventually(t, func() bool {
		dt.mu.Lock()
		defer dt.mu.Unlock()
		return dt.state.mode == originH3Healthy
	}, time.Second, 10*time.Millisecond)
}

func TestDualTransport_ProbeFailureEntersCooldownAndKeepsHTTP2(t *testing.T) {
	t.Parallel()

	var h2Calls atomic.Int32
	var h3Calls atomic.Int32

	dt := &dualTransport{
		http2Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			h2Calls.Add(1)
			headers := make(http.Header)
			headers.Set("Alt-Svc", `h3=":443"; ma=86400`)
			return newTestResponse(http.StatusOK, headers), nil
		}),
		http3Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			h3Calls.Add(1)
			assert.Equal(t, http.MethodOptions, req.Method)
			return nil, errors.New("probe failed")
		}),
		altSvcCache:  tri.NewAltSvcCache(),
		probeTimeout: 100 * time.Millisecond,
		baseCooldown: 4 * time.Second,
		maxCooldown:  1 * time.Minute,
	}

	req, err := http.NewRequest(http.MethodPost, "https://example.com/service", nil)
	require.NoError(t, err)

	resp, err := dt.RoundTrip(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	require.Eventually(t, func() bool {
		dt.mu.Lock()
		defer dt.mu.Unlock()
		return dt.state.mode == originCooldown && dt.state.failures == 1 && !dt.state.cooldownUntil.IsZero()
	}, time.Second, 10*time.Millisecond)

	resp, err = dt.RoundTrip(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, int32(2), h2Calls.Load())
	assert.Equal(t, int32(1), h3Calls.Load())
}

func TestDualTransport_H3HealthyFailureDoesNotFallbackToHTTP2(t *testing.T) {
	t.Parallel()

	var h2Calls atomic.Int32
	var h3Calls atomic.Int32

	dt := &dualTransport{
		http2Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			h2Calls.Add(1)
			return newTestResponse(http.StatusOK, nil), nil
		}),
		http3Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			h3Calls.Add(1)
			return nil, errors.New("h3 failed")
		}),
		altSvcCache:  tri.NewAltSvcCache(),
		baseCooldown: 4 * time.Second,
		maxCooldown:  1 * time.Minute,
	}
	dt.altSvcCache.Set("example.com", &tri.AltSvcInfo{
		Protocol: "h3",
		Expires:  time.Now().Add(time.Hour),
	})
	dt.state.mode = originH3Healthy

	req, err := http.NewRequest(http.MethodPost, "https://example.com/service", nil)
	require.NoError(t, err)

	resp, err := dt.RoundTrip(req)
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, int32(1), h3Calls.Load())
	assert.Equal(t, int32(0), h2Calls.Load())

	dt.mu.Lock()
	defer dt.mu.Unlock()
	assert.Equal(t, originCooldown, dt.state.mode)
	assert.False(t, dt.state.cooldownUntil.IsZero())
}

func TestDualTransport_H3ContextErrorDoesNotEnterCooldown(t *testing.T) {
	t.Parallel()

	var h2Calls atomic.Int32
	var h3Calls atomic.Int32

	dt := &dualTransport{
		http2Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			h2Calls.Add(1)
			return newTestResponse(http.StatusOK, nil), nil
		}),
		http3Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			h3Calls.Add(1)
			return nil, req.Context().Err()
		}),
		altSvcCache:  tri.NewAltSvcCache(),
		baseCooldown: 4 * time.Second,
		maxCooldown:  1 * time.Minute,
	}
	dt.altSvcCache.Set("example.com", &tri.AltSvcInfo{
		Protocol: "h3",
		Expires:  time.Now().Add(time.Hour),
	})
	dt.state.mode = originH3Healthy

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://example.com/service", nil)
	require.NoError(t, err)

	resp, err := dt.RoundTrip(req)
	require.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, resp)
	assert.Equal(t, int32(1), h3Calls.Load())
	assert.Equal(t, int32(0), h2Calls.Load())

	dt.mu.Lock()
	defer dt.mu.Unlock()
	assert.Equal(t, originH3Healthy, dt.state.mode)
	assert.Equal(t, 0, dt.state.failures)
	assert.True(t, dt.state.cooldownUntil.IsZero())
}

func TestDualTransport_CooldownUsesHTTP2(t *testing.T) {
	t.Parallel()

	var h2Calls atomic.Int32
	var h3Calls atomic.Int32

	dt := &dualTransport{
		http2Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			h2Calls.Add(1)
			return newTestResponse(http.StatusOK, nil), nil
		}),
		http3Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			h3Calls.Add(1)
			return newTestResponse(http.StatusOK, nil), nil
		}),
		altSvcCache:  tri.NewAltSvcCache(),
		baseCooldown: 4 * time.Second,
		maxCooldown:  1 * time.Minute,
	}
	dt.altSvcCache.Set("example.com", &tri.AltSvcInfo{
		Protocol: "h3",
		Expires:  time.Now().Add(time.Hour),
	})
	dt.state.mode = originCooldown
	dt.state.cooldownUntil = time.Now().Add(time.Minute)

	req, err := http.NewRequest(http.MethodPost, "https://example.com/service", nil)
	require.NoError(t, err)

	resp, err := dt.RoundTrip(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, int32(1), h2Calls.Load())
	assert.Equal(t, int32(0), h3Calls.Load())
}

func TestDualTransport_CooldownExpiryStartsProbeAndPromotesH3Healthy(t *testing.T) {
	t.Parallel()

	var h2Calls atomic.Int32
	var h3Calls atomic.Int32

	dt := &dualTransport{
		http2Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			h2Calls.Add(1)
			headers := make(http.Header)
			headers.Set("Alt-Svc", `h3=":443"; ma=86400`)
			return newTestResponse(http.StatusOK, headers), nil
		}),
		altSvcCache:  tri.NewAltSvcCache(),
		probeTimeout: 100 * time.Millisecond,
		baseCooldown: 4 * time.Second,
		maxCooldown:  1 * time.Minute,
	}

	probeCalled := make(chan struct{}, 1)
	dt.http3Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		h3Calls.Add(1)
		assert.Equal(t, http.MethodOptions, req.Method)
		assert.Equal(t, "/", req.URL.Path)
		probeCalled <- struct{}{}
		return newTestResponse(http.StatusMethodNotAllowed, nil), nil
	})

	dt.altSvcCache.Set("example.com", &tri.AltSvcInfo{
		Protocol: "h3",
		Expires:  time.Now().Add(time.Hour),
	})
	dt.state.mode = originCooldown
	dt.state.failures = 1
	dt.state.cooldownUntil = time.Now().Add(-time.Second)

	req, err := http.NewRequest(http.MethodPost, "https://example.com/service", nil)
	require.NoError(t, err)

	resp, err := dt.RoundTrip(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, int32(1), h2Calls.Load())

	select {
	case <-probeCalled:
	case <-time.After(time.Second):
		t.Fatal("probe was not triggered after cooldown expiry")
	}

	require.Eventually(t, func() bool {
		dt.mu.Lock()
		defer dt.mu.Unlock()
		return dt.state.mode == originH3Healthy && dt.state.failures == 0 && dt.state.cooldownUntil.IsZero()
	}, time.Second, 10*time.Millisecond)
	assert.Equal(t, int32(1), h3Calls.Load())
}
