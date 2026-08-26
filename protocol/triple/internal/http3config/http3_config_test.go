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

package http3config

import (
	"testing"
	"time"
)

import (
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/quicvarint"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

import (
	"dubbo.apache.org/dubbo-go/v3/global"
)

func TestNewQUICConfig(t *testing.T) {
	t.Run("defaults_preserved_when_unset", func(t *testing.T) {
		quicConfig, err := NewQUICConfig(&global.Http3Config{}, nil)
		require.NoError(t, err)
		require.NotNil(t, quicConfig)
		assert.Zero(t, quicConfig.KeepAlivePeriod)
		assert.Zero(t, quicConfig.MaxIdleTimeout)
		assert.Zero(t, quicConfig.MaxIncomingStreams)
		assert.Zero(t, quicConfig.MaxIncomingUniStreams)
		assert.Zero(t, quicConfig.InitialStreamReceiveWindow)
		assert.Zero(t, quicConfig.MaxStreamReceiveWindow)
		assert.Zero(t, quicConfig.InitialConnectionReceiveWindow)
		assert.Zero(t, quicConfig.MaxConnectionReceiveWindow)
	})

	t.Run("explicit_fields_are_mapped", func(t *testing.T) {
		quicConfig, err := NewQUICConfig(&global.Http3Config{
			KeepAlivePeriod:                "15s",
			MaxIdleTimeout:                 "30s",
			MaxIncomingStreams:             128,
			MaxIncomingUniStreams:          64,
			InitialStreamReceiveWindow:     "512KiB",
			MaxStreamReceiveWindow:         "6MiB",
			InitialConnectionReceiveWindow: "1MiB",
			MaxConnectionReceiveWindow:     "16MiB",
		}, nil)
		require.NoError(t, err)
		require.NotNil(t, quicConfig)
		assert.Equal(t, 15*time.Second, quicConfig.KeepAlivePeriod)
		assert.Equal(t, 30*time.Second, quicConfig.MaxIdleTimeout)
		assert.Equal(t, int64(128), quicConfig.MaxIncomingStreams)
		assert.Equal(t, int64(64), quicConfig.MaxIncomingUniStreams)
		assert.Equal(t, uint64(524288), quicConfig.InitialStreamReceiveWindow)
		assert.Equal(t, uint64(6291456), quicConfig.MaxStreamReceiveWindow)
		assert.Equal(t, uint64(1048576), quicConfig.InitialConnectionReceiveWindow)
		assert.Equal(t, uint64(16777216), quicConfig.MaxConnectionReceiveWindow)
	})

	t.Run("invalid_keep_alive_period_returns_error", func(t *testing.T) {
		quicConfig, err := NewQUICConfig(&global.Http3Config{
			KeepAlivePeriod: "invalid",
		}, nil)
		require.Error(t, err)
		assert.Nil(t, quicConfig)
		assert.ErrorContains(t, err, "keep-alive-period")
	})

	t.Run("invalid_max_idle_timeout_returns_error", func(t *testing.T) {
		quicConfig, err := NewQUICConfig(&global.Http3Config{
			MaxIdleTimeout: "invalid",
		}, nil)
		require.Error(t, err)
		assert.Nil(t, quicConfig)
		assert.ErrorContains(t, err, "max-idle-timeout")
	})

	t.Run("invalid_receive_window_returns_error", func(t *testing.T) {
		quicConfig, err := NewQUICConfig(&global.Http3Config{
			MaxConnectionReceiveWindow: "invalid",
		}, nil)
		require.Error(t, err)
		assert.Nil(t, quicConfig)
		assert.ErrorContains(t, err, "max-connection-receive-window")
	})

	t.Run("receive_windows_must_fit_quic_varint", func(t *testing.T) {
		for _, test := range []struct {
			name  string
			field func(*global.Http3Config)
			want  string
		}{
			{
				name: "initial_stream",
				field: func(config *global.Http3Config) {
					config.InitialStreamReceiveWindow = "4611686018427387904"
				},
				want: "initial-stream-receive-window",
			},
			{
				name: "max_stream",
				field: func(config *global.Http3Config) {
					config.MaxStreamReceiveWindow = "4611686018427387904"
				},
				want: "max-stream-receive-window",
			},
			{
				name: "initial_connection",
				field: func(config *global.Http3Config) {
					config.InitialConnectionReceiveWindow = "4611686018427387904"
				},
				want: "initial-connection-receive-window",
			},
			{
				name: "max_connection",
				field: func(config *global.Http3Config) {
					config.MaxConnectionReceiveWindow = "4611686018427387904"
				},
				want: "max-connection-receive-window",
			},
		} {
			t.Run(test.name, func(t *testing.T) {
				config := &global.Http3Config{}
				test.field(config)

				quicConfig, err := NewQUICConfig(config, nil)
				require.Error(t, err)
				assert.Nil(t, quicConfig)
				require.ErrorContains(t, err, test.want)
				require.ErrorContains(t, err, "QUIC varint maximum")
			})
		}
	})

	t.Run("initial_receive_window_must_not_exceed_maximum", func(t *testing.T) {
		for _, test := range []struct {
			name      string
			configure func(*global.Http3Config)
			want      string
		}{
			{
				name: "stream",
				configure: func(config *global.Http3Config) {
					config.InitialStreamReceiveWindow = "16MiB"
					config.MaxStreamReceiveWindow = "1MiB"
				},
				want: "initial-stream-receive-window",
			},
			{
				name: "connection",
				configure: func(config *global.Http3Config) {
					config.InitialConnectionReceiveWindow = "32MiB"
					config.MaxConnectionReceiveWindow = "2MiB"
				},
				want: "initial-connection-receive-window",
			},
		} {
			t.Run(test.name, func(t *testing.T) {
				config := &global.Http3Config{
					InitialStreamReceiveWindow:     "1MiB",
					MaxStreamReceiveWindow:         "2MiB",
					InitialConnectionReceiveWindow: "1MiB",
					MaxConnectionReceiveWindow:     "2MiB",
				}
				test.configure(config)

				quicConfig, err := NewQUICConfig(config, nil)
				require.Error(t, err)
				assert.Nil(t, quicConfig)
				assert.ErrorContains(t, err, test.want)
			})
		}
	})

	t.Run("quic_varint_maximum_is_accepted", func(t *testing.T) {
		max := "4611686018427387903"
		quicConfig, err := NewQUICConfig(&global.Http3Config{
			InitialStreamReceiveWindow:     max,
			MaxStreamReceiveWindow:         max,
			InitialConnectionReceiveWindow: max,
			MaxConnectionReceiveWindow:     max,
		}, nil)
		require.NoError(t, err)
		assert.Equal(t, uint64(quicvarint.Max), quicConfig.InitialStreamReceiveWindow)
		assert.Equal(t, uint64(quicvarint.Max), quicConfig.MaxStreamReceiveWindow)
		assert.Equal(t, uint64(quicvarint.Max), quicConfig.InitialConnectionReceiveWindow)
		assert.Equal(t, uint64(quicvarint.Max), quicConfig.MaxConnectionReceiveWindow)
	})

	t.Run("yaml_receive_windows_are_validated", func(t *testing.T) {
		var config global.TripleConfig
		err := yaml.Unmarshal([]byte(`
http3:
  initial-stream-receive-window: "16MiB"
  max-stream-receive-window: "1MiB"
  initial-connection-receive-window: "32MiB"
  max-connection-receive-window: "2MiB"
`), &config)
		require.NoError(t, err)
		_, err = NewQUICConfig(config.Http3, nil)
		require.Error(t, err)
		assert.ErrorContains(t, err, "initial-stream-receive-window")
	})

	t.Run("nil_config_uses_defaults", func(t *testing.T) {
		defaults := &quic.Config{
			KeepAlivePeriod:                10 * time.Second,
			MaxIdleTimeout:                 20 * time.Second,
			InitialStreamReceiveWindow:     512 * 1024,
			MaxStreamReceiveWindow:         6 * 1024 * 1024,
			InitialConnectionReceiveWindow: 512 * 1024,
			MaxConnectionReceiveWindow:     15 * 1024 * 1024,
		}

		quicConfig, err := NewQUICConfig(nil, defaults)
		require.NoError(t, err)
		require.NotNil(t, quicConfig)
		assert.NotSame(t, defaults, quicConfig)
		assert.Equal(t, 10*time.Second, quicConfig.KeepAlivePeriod)
		assert.Equal(t, 20*time.Second, quicConfig.MaxIdleTimeout)
		assert.Equal(t, uint64(512*1024), quicConfig.InitialStreamReceiveWindow)
		assert.Equal(t, uint64(6*1024*1024), quicConfig.MaxStreamReceiveWindow)
		assert.Equal(t, uint64(512*1024), quicConfig.InitialConnectionReceiveWindow)
		assert.Equal(t, uint64(15*1024*1024), quicConfig.MaxConnectionReceiveWindow)
	})

	t.Run("defaults_are_used_when_fields_unset", func(t *testing.T) {
		defaults := &quic.Config{
			KeepAlivePeriod:                10 * time.Second,
			MaxIdleTimeout:                 20 * time.Second,
			InitialStreamReceiveWindow:     512 * 1024,
			MaxStreamReceiveWindow:         6 * 1024 * 1024,
			InitialConnectionReceiveWindow: 512 * 1024,
			MaxConnectionReceiveWindow:     15 * 1024 * 1024,
		}

		quicConfig, err := NewQUICConfig(&global.Http3Config{}, defaults)
		require.NoError(t, err)
		require.NotNil(t, quicConfig)
		assert.NotSame(t, defaults, quicConfig)
		assert.Equal(t, 10*time.Second, quicConfig.KeepAlivePeriod)
		assert.Equal(t, 20*time.Second, quicConfig.MaxIdleTimeout)
		assert.Equal(t, uint64(512*1024), quicConfig.InitialStreamReceiveWindow)
		assert.Equal(t, uint64(6*1024*1024), quicConfig.MaxStreamReceiveWindow)
		assert.Equal(t, uint64(512*1024), quicConfig.InitialConnectionReceiveWindow)
		assert.Equal(t, uint64(15*1024*1024), quicConfig.MaxConnectionReceiveWindow)
	})

	t.Run("explicit_fields_override_defaults", func(t *testing.T) {
		quicConfig, err := NewQUICConfig(&global.Http3Config{
			KeepAlivePeriod:                "15s",
			MaxIdleTimeout:                 "30s",
			InitialStreamReceiveWindow:     "262144",
			MaxStreamReceiveWindow:         "4194304",
			InitialConnectionReceiveWindow: "524288",
			MaxConnectionReceiveWindow:     "8388608",
		}, &quic.Config{
			KeepAlivePeriod:                10 * time.Second,
			MaxIdleTimeout:                 20 * time.Second,
			InitialStreamReceiveWindow:     512 * 1024,
			MaxStreamReceiveWindow:         6 * 1024 * 1024,
			InitialConnectionReceiveWindow: 512 * 1024,
			MaxConnectionReceiveWindow:     15 * 1024 * 1024,
		})
		require.NoError(t, err)
		require.NotNil(t, quicConfig)
		assert.Equal(t, 15*time.Second, quicConfig.KeepAlivePeriod)
		assert.Equal(t, 30*time.Second, quicConfig.MaxIdleTimeout)
		assert.Equal(t, uint64(262144), quicConfig.InitialStreamReceiveWindow)
		assert.Equal(t, uint64(4194304), quicConfig.MaxStreamReceiveWindow)
		assert.Equal(t, uint64(524288), quicConfig.InitialConnectionReceiveWindow)
		assert.Equal(t, uint64(8388608), quicConfig.MaxConnectionReceiveWindow)
	})

	t.Run("explicit_zero_duration_overrides_default", func(t *testing.T) {
		quicConfig, err := NewQUICConfig(&global.Http3Config{
			KeepAlivePeriod: "0s",
			MaxIdleTimeout:  "0s",
		}, &quic.Config{
			KeepAlivePeriod: 10 * time.Second,
			MaxIdleTimeout:  20 * time.Second,
		})
		require.NoError(t, err)
		require.NotNil(t, quicConfig)
		assert.Zero(t, quicConfig.KeepAlivePeriod)
		assert.Zero(t, quicConfig.MaxIdleTimeout)
	})
}
