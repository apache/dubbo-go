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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTP3TransportOptions(t *testing.T) {
	opts := NewOptions(
		WithHttp3Enable(),
		WithHttp3Negotiation(false),
		WithHttp3KeepAlivePeriod(15*time.Second),
		WithHttp3MaxIdleTimeout(30*time.Second),
		WithHttp3MaxIncomingStreams(128),
		WithHttp3MaxIncomingUniStreams(64),
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
}
