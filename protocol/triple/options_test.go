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
	"testing"
	"time"
)

import (
	"github.com/quic-go/quic-go/quicvarint"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/internal/http3config"
)

func TestHTTP3TransportOptions(t *testing.T) {
	opts := NewOptions(
		WithHttp3Enable(),
		WithHttp3Negotiation(false),
		WithHttp3KeepAlivePeriod(15*time.Second),
		WithHttp3MaxIdleTimeout(30*time.Second),
		WithHttp3MaxIncomingStreams(128),
		WithHttp3MaxIncomingUniStreams(64),
		WithHttp3InitialStreamReceiveWindow(512*1024),
		WithHttp3MaxStreamReceiveWindow(6*1024*1024),
		WithHttp3InitialConnectionReceiveWindow(1024*1024),
		WithHttp3MaxConnectionReceiveWindow(16*1024*1024),
	)

	require.NotNil(t, opts)
	require.NotNil(t, opts.Triple)
	require.NotNil(t, opts.Triple.Http3)
	assert.True(t, opts.Triple.Http3.Enable)
	assert.False(t, opts.Triple.Http3.Negotiation)
	assert.Equal(t, "15s", opts.Triple.Http3.KeepAlivePeriod)
	assert.Equal(t, "30s", opts.Triple.Http3.MaxIdleTimeout)
	assert.Equal(t, int64(128), opts.Triple.Http3.MaxIncomingStreams)
	assert.Equal(t, int64(64), opts.Triple.Http3.MaxIncomingUniStreams)
	assert.Equal(t, "524288", opts.Triple.Http3.InitialStreamReceiveWindow)
	assert.Equal(t, "6291456", opts.Triple.Http3.MaxStreamReceiveWindow)
	assert.Equal(t, "1048576", opts.Triple.Http3.InitialConnectionReceiveWindow)
	assert.Equal(t, "16777216", opts.Triple.Http3.MaxConnectionReceiveWindow)
}

func TestHTTP3ReceiveWindowOptionsAreValidated(t *testing.T) {
	for _, test := range []struct {
		name    string
		options []Option
		field   string
	}{
		{
			name:    "initial_stream_overflow",
			options: []Option{WithHttp3InitialStreamReceiveWindow(quicvarint.Max + 1)},
			field:   "initial-stream-receive-window",
		},
		{
			name:    "max_stream_overflow",
			options: []Option{WithHttp3MaxStreamReceiveWindow(quicvarint.Max + 1)},
			field:   "max-stream-receive-window",
		},
		{
			name:    "initial_connection_overflow",
			options: []Option{WithHttp3InitialConnectionReceiveWindow(quicvarint.Max + 1)},
			field:   "initial-connection-receive-window",
		},
		{
			name:    "max_connection_overflow",
			options: []Option{WithHttp3MaxConnectionReceiveWindow(quicvarint.Max + 1)},
			field:   "max-connection-receive-window",
		},
		{
			name: "stream_initial_exceeds_maximum",
			options: []Option{
				WithHttp3InitialStreamReceiveWindow(16 * 1024 * 1024),
				WithHttp3MaxStreamReceiveWindow(1 * 1024 * 1024),
			},
			field: "initial-stream-receive-window",
		},
		{
			name: "connection_initial_exceeds_maximum",
			options: []Option{
				WithHttp3InitialConnectionReceiveWindow(32 * 1024 * 1024),
				WithHttp3MaxConnectionReceiveWindow(2 * 1024 * 1024),
			},
			field: "initial-connection-receive-window",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			opts := NewOptions(test.options...)
			_, err := http3config.NewQUICConfig(opts.Triple.Http3, nil)
			require.Error(t, err)
			assert.ErrorContains(t, err, test.field)
		})
	}

	opts := NewOptions(
		WithHttp3InitialStreamReceiveWindow(quicvarint.Max),
		WithHttp3MaxStreamReceiveWindow(quicvarint.Max),
		WithHttp3InitialConnectionReceiveWindow(quicvarint.Max),
		WithHttp3MaxConnectionReceiveWindow(quicvarint.Max),
	)
	quicConfig, err := http3config.NewQUICConfig(opts.Triple.Http3, nil)
	require.NoError(t, err)
	assert.Equal(t, uint64(quicvarint.Max), quicConfig.InitialStreamReceiveWindow)
	assert.Equal(t, uint64(quicvarint.Max), quicConfig.MaxStreamReceiveWindow)
	assert.Equal(t, uint64(quicvarint.Max), quicConfig.InitialConnectionReceiveWindow)
	assert.Equal(t, uint64(quicvarint.Max), quicConfig.MaxConnectionReceiveWindow)
}

func TestHTTP3DeprecatedAliases(t *testing.T) {
	opts := NewOptions(
		Http3Enable(),
		Http3Negotiation(false),
	)

	require.NotNil(t, opts)
	require.NotNil(t, opts.Triple)
	require.NotNil(t, opts.Triple.Http3)
	assert.True(t, opts.Triple.Http3.Enable)
	assert.False(t, opts.Triple.Http3.Negotiation)
}
