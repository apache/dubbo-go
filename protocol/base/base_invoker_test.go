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

package base

import (
	"context"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
)

func TestBaseInvoker(t *testing.T) {
	url, err := common.NewURL("dubbo://localhost:9090")
	require.NoError(t, err)

	ivk := NewBaseInvoker(url)
	assert.NotNil(t, ivk.GetURL())
	assert.True(t, ivk.IsAvailable())
	assert.False(t, ivk.IsDestroyed())

	ivk.Destroy()
	assert.False(t, ivk.IsAvailable())
	assert.True(t, ivk.IsDestroyed())
}

func TestBaseInvokerWithFullURL(t *testing.T) {
	url, err := common.NewURL("dubbo://localhost:20880/com.example.Service?version=1.0.0&group=test")
	require.NoError(t, err)

	ivk := NewBaseInvoker(url)

	// Test GetURL
	returnedURL := ivk.GetURL()
	assert.NotNil(t, returnedURL)
	assert.Equal(t, "dubbo", returnedURL.Protocol)
	assert.Equal(t, "localhost", returnedURL.Ip)
	assert.Equal(t, "20880", returnedURL.Port)

	// Test initial state
	assert.True(t, ivk.IsAvailable())
	assert.False(t, ivk.IsDestroyed())

	// Test String method before destroy
	str := ivk.String()
	assert.Contains(t, str, "dubbo")
	assert.Contains(t, str, "localhost")
	assert.Contains(t, str, "20880")

	// Test Destroy
	ivk.Destroy()
	assert.False(t, ivk.IsAvailable())
	assert.True(t, ivk.IsDestroyed())
	assert.NotNil(t, ivk.GetURL())

	// Test String method after destroy (url should still be available)
	str = ivk.String()
	assert.Contains(t, str, "dubbo")
	assert.Contains(t, str, "localhost")
}

func TestBaseInvokerInvoke(t *testing.T) {
	url, err := common.NewURL("dubbo://localhost:9090/test.Service")
	require.NoError(t, err)

	ivk := NewBaseInvoker(url)

	// Create a mock invocation
	ctx := context.Background()

	// Invoke method should return an empty RPCResult
	result := ivk.Invoke(ctx, nil)
	assert.NotNil(t, result)
}

func TestBaseInvokerMultipleDestroy(t *testing.T) {
	url, err := common.NewURL("dubbo://localhost:9090")
	require.NoError(t, err)

	ivk := NewBaseInvoker(url)

	// First destroy
	ivk.Destroy()
	assert.True(t, ivk.IsDestroyed())
	assert.False(t, ivk.IsAvailable())

	// Second destroy should not cause panic
	ivk.Destroy()
	assert.True(t, ivk.IsDestroyed())
	assert.False(t, ivk.IsAvailable())
}

func TestBaseInvokerStringWithDifferentURLs(t *testing.T) {
	tests := []struct {
		name     string
		urlStr   string
		contains []string
	}{
		{
			name:     "Standard URL",
			urlStr:   "dubbo://localhost:8080/test.Service",
			contains: []string{"dubbo", "localhost", "8080", "test.Service"},
		},
		{
			name:     "URL with path",
			urlStr:   "tri://localhost:9090/org.apache.Service",
			contains: []string{"tri", "localhost", "9090", "org.apache.Service"},
		},
		{
			name:     "URL with port only",
			urlStr:   "dubbo://:8888/Service",
			contains: []string{"dubbo", "8888"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, err := common.NewURL(tt.urlStr)
			require.NoError(t, err)

			ivk := NewBaseInvoker(url)
			str := ivk.String()

			for _, contain := range tt.contains {
				assert.Contains(t, str, contain)
			}
		})
	}
}
