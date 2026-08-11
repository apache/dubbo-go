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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	})

	t.Run("explicit_fields_are_mapped", func(t *testing.T) {
		quicConfig, err := NewQUICConfig(&global.Http3Config{
			KeepAlivePeriod:       "15s",
			MaxIdleTimeout:        "30s",
			MaxIncomingStreams:    128,
			MaxIncomingUniStreams: 64,
		}, nil)
		require.NoError(t, err)
		require.NotNil(t, quicConfig)
		assert.Equal(t, 15*time.Second, quicConfig.KeepAlivePeriod)
		assert.Equal(t, 30*time.Second, quicConfig.MaxIdleTimeout)
		assert.Equal(t, int64(128), quicConfig.MaxIncomingStreams)
		assert.Equal(t, int64(64), quicConfig.MaxIncomingUniStreams)
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
	t.Run("nil_config_uses_defaults", func(t *testing.T) {
		defaults := &quic.Config{
			KeepAlivePeriod: 10 * time.Second,
			MaxIdleTimeout:  20 * time.Second,
		}

		quicConfig, err := NewQUICConfig(nil, defaults)
		require.NoError(t, err)
		require.NotNil(t, quicConfig)
		assert.NotSame(t, defaults, quicConfig)
		assert.Equal(t, 10*time.Second, quicConfig.KeepAlivePeriod)
		assert.Equal(t, 20*time.Second, quicConfig.MaxIdleTimeout)
	})

	t.Run("defaults_are_used_when_fields_unset", func(t *testing.T) {
		defaults := &quic.Config{
			KeepAlivePeriod: 10 * time.Second,
			MaxIdleTimeout:  20 * time.Second,
		}

		quicConfig, err := NewQUICConfig(&global.Http3Config{}, defaults)
		require.NoError(t, err)
		require.NotNil(t, quicConfig)
		assert.NotSame(t, defaults, quicConfig)
		assert.Equal(t, 10*time.Second, quicConfig.KeepAlivePeriod)
		assert.Equal(t, 20*time.Second, quicConfig.MaxIdleTimeout)
	})

	t.Run("explicit_fields_override_defaults", func(t *testing.T) {
		quicConfig, err := NewQUICConfig(&global.Http3Config{
			KeepAlivePeriod: "15s",
			MaxIdleTimeout:  "30s",
		}, &quic.Config{
			KeepAlivePeriod: 10 * time.Second,
			MaxIdleTimeout:  20 * time.Second,
		})
		require.NoError(t, err)
		require.NotNil(t, quicConfig)
		assert.Equal(t, 15*time.Second, quicConfig.KeepAlivePeriod)
		assert.Equal(t, 30*time.Second, quicConfig.MaxIdleTimeout)
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
