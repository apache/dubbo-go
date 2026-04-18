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

package dubbo

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/global"
)

func TestCompatHttp3Config(t *testing.T) {
	t.Run("global_to_config", func(t *testing.T) {
		src := &global.Http3Config{
			Enable:                true,
			Negotiation:           false,
			KeepAlivePeriod:       "15s",
			MaxIdleTimeout:        "30s",
			MaxIncomingStreams:    128,
			MaxIncomingUniStreams: 64,
		}

		got := compatHttp3Config(src)
		assert.NotNil(t, got)
		assert.Equal(t, src.Enable, got.Enable)
		assert.Equal(t, src.Negotiation, got.Negotiation)
		assert.Equal(t, src.KeepAlivePeriod, got.KeepAlivePeriod)
		assert.Equal(t, src.MaxIdleTimeout, got.MaxIdleTimeout)
		assert.Equal(t, src.MaxIncomingStreams, got.MaxIncomingStreams)
		assert.Equal(t, src.MaxIncomingUniStreams, got.MaxIncomingUniStreams)
	})

	t.Run("config_to_global", func(t *testing.T) {
		src := &config.Http3Config{
			Enable:                true,
			Negotiation:           false,
			KeepAlivePeriod:       "15s",
			MaxIdleTimeout:        "30s",
			MaxIncomingStreams:    128,
			MaxIncomingUniStreams: 64,
		}

		got := compatGlobalHttp3Config(src)
		assert.NotNil(t, got)
		assert.Equal(t, src.Enable, got.Enable)
		assert.Equal(t, src.Negotiation, got.Negotiation)
		assert.Equal(t, src.KeepAlivePeriod, got.KeepAlivePeriod)
		assert.Equal(t, src.MaxIdleTimeout, got.MaxIdleTimeout)
		assert.Equal(t, src.MaxIncomingStreams, got.MaxIncomingStreams)
		assert.Equal(t, src.MaxIncomingUniStreams, got.MaxIncomingUniStreams)
	})

	t.Run("nil", func(t *testing.T) {
		assert.Nil(t, compatHttp3Config(nil))
		assert.Nil(t, compatGlobalHttp3Config(nil))
	})
}
