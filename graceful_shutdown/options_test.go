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

package graceful_shutdown

import (
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/global"
)

func TestDefaultOptions(t *testing.T) {
	opts := defaultOptions()
	assert.NotNil(t, opts)
	assert.NotNil(t, opts.Shutdown)
	assert.Equal(t, "60s", opts.Shutdown.Timeout)
	assert.Equal(t, "3s", opts.Shutdown.StepTimeout)
	assert.Equal(t, "3s", opts.Shutdown.ConsumerUpdateWaitTime)
	assert.Equal(t, "", opts.Shutdown.OfflineRequestWindowTimeout) // No default value
	assert.True(t, *opts.Shutdown.InternalSignal)
}

func TestNewOptions(t *testing.T) {
	// Test with default options
	opts := NewOptions()
	assert.NotNil(t, opts)
	assert.NotNil(t, opts.Shutdown)

	// Test with custom options
	customTimeout := 120 * time.Second
	customStepTimeout := 10 * time.Second
	customConsumerUpdateWaitTime := 5 * time.Second
	customOfflineRequestWindowTimeout := 2 * time.Second

	opts = NewOptions(
		WithTimeout(customTimeout),
		WithStepTimeout(customStepTimeout),
		WithConsumerUpdateWaitTime(customConsumerUpdateWaitTime),
		WithOfflineRequestWindowTimeout(customOfflineRequestWindowTimeout),
		WithoutInternalSignal(),
	)

	assert.Equal(t, customTimeout.String(), opts.Shutdown.Timeout)
	assert.Equal(t, customStepTimeout.String(), opts.Shutdown.StepTimeout)
	assert.Equal(t, customConsumerUpdateWaitTime.String(), opts.Shutdown.ConsumerUpdateWaitTime)
	assert.Equal(t, customOfflineRequestWindowTimeout.String(), opts.Shutdown.OfflineRequestWindowTimeout)
	assert.False(t, *opts.Shutdown.InternalSignal)
}

func TestOptionFunctions(t *testing.T) {
	// Test WithTimeout
	opts := defaultOptions()
	WithTimeout(120 * time.Second)(opts)
	assert.Equal(t, "2m0s", opts.Shutdown.Timeout)

	// Test WithStepTimeout
	WithStepTimeout(5 * time.Second)(opts)
	assert.Equal(t, "5s", opts.Shutdown.StepTimeout)

	// Test WithConsumerUpdateWaitTime
	WithConsumerUpdateWaitTime(10 * time.Second)(opts)
	assert.Equal(t, "10s", opts.Shutdown.ConsumerUpdateWaitTime)

	// Test WithOfflineRequestWindowTimeout
	WithOfflineRequestWindowTimeout(2 * time.Second)(opts)
	assert.Equal(t, "2s", opts.Shutdown.OfflineRequestWindowTimeout)

	// Test WithoutInternalSignal
	WithoutInternalSignal()(opts)
	assert.False(t, *opts.Shutdown.InternalSignal)

	// Test WithRejectRequest
	WithRejectRequest()(opts)
	assert.True(t, opts.Shutdown.RejectRequest.Load())

	// Test SetShutdownConfig
	customConfig := global.DefaultShutdownConfig()
	customConfig.Timeout = "300s"
	SetShutdownConfig(customConfig)(opts)
	assert.Equal(t, customConfig, opts.Shutdown)
}
