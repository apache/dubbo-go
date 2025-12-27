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

package config

import (
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

func TestShutdownConfigGetTimeout(t *testing.T) {
	config := ShutdownConfig{}
	assert.False(t, config.RejectRequest.Load())

	config = ShutdownConfig{
		Timeout:                     "60s",
		StepTimeout:                 "10s",
		OfflineRequestWindowTimeout: "30s",
	}

	assert.Equal(t, 60*time.Second, config.GetTimeout())
	assert.Equal(t, 10*time.Second, config.GetStepTimeout())
	assert.Equal(t, 30*time.Second, config.GetOfflineRequestWindowTimeout())
	config = ShutdownConfig{
		Timeout:                     "34ms",
		StepTimeout:                 "79ms",
		OfflineRequestWindowTimeout: "13ms",
	}

	assert.Equal(t, 34*time.Millisecond, config.GetTimeout())
	assert.Equal(t, 79*time.Millisecond, config.GetStepTimeout())
	assert.Equal(t, 13*time.Millisecond, config.GetOfflineRequestWindowTimeout())

	// test default
	config = ShutdownConfig{}

	assert.Equal(t, defaultTimeout, config.GetTimeout())
	assert.Equal(t, defaultStepTimeout, config.GetStepTimeout())
	assert.Equal(t, defaultOfflineRequestWindowTimeout, config.GetOfflineRequestWindowTimeout())
}

func TestNewShutDownConfigBuilder(t *testing.T) {
	config := NewShutDownConfigBuilder().
		SetTimeout("10s").
		SetStepTimeout("15s").
		SetOfflineRequestWindowTimeout("13s").
		SetRejectRequestHandler("handler").
		SetRejectRequest(true).
		SetInternalSignal(false).
		Build()

	assert.Equal(t, constant.ShutdownConfigPrefix, config.Prefix())

	timeout := config.GetTimeout()
	assert.Equal(t, 10*time.Second, timeout)

	stepTimeout := config.GetStepTimeout()
	assert.Equal(t, 15*time.Second, stepTimeout)

	offlineRequestWindowTimeout := config.GetOfflineRequestWindowTimeout()
	assert.Equal(t, 13*time.Second, offlineRequestWindowTimeout)
	err := config.Init()
	require.NoError(t, err)

	waitTime := config.GetConsumerUpdateWaitTime()
	assert.Equal(t, 3*time.Second, waitTime)

	assert.False(t, config.GetInternalSignal())
}

func TestGetInternalSignal(t *testing.T) {
	config := NewShutDownConfigBuilder().
		SetTimeout("10s").
		SetStepTimeout("15s").
		SetOfflineRequestWindowTimeout("13s").
		SetRejectRequestHandler("handler").
		SetRejectRequest(true).
		Build()

	assert.True(t, config.GetInternalSignal())
}
