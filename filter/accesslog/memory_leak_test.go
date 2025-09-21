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

package accesslog

import (
	"context"
	"os"
	"runtime"
	"sync"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	invocation_impl "dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

// resetGlobalState resets the global state for testing
func resetGlobalState() {
	once.Do(func() {}) // Trigger once
	accessLogFilter = nil
	once = sync.Once{}
}

// TestAccessLogFilterGoroutineShutdown tests that the goroutine is properly shut down
func TestAccessLogFilterGoroutineShutdown(t *testing.T) {
	resetGlobalState()

	// Count goroutines before
	initialGoroutines := runtime.NumGoroutine()

	// Create filter (this should start the goroutine)
	filter := newFilter()
	assert.NotNil(t, filter)

	// Give the goroutine time to start
	time.Sleep(100 * time.Millisecond)
	postCreateGoroutines := runtime.NumGoroutine()

	// Should have at least one more goroutine
	assert.Greater(t, postCreateGoroutines, initialGoroutines)

	// Shutdown the filter
	Shutdown()

	// Give goroutine time to exit
	time.Sleep(100 * time.Millisecond)
	runtime.GC() // Force garbage collection

	postShutdownGoroutines := runtime.NumGoroutine()

	// Goroutine count should be back to original or less
	assert.LessOrEqual(t, postShutdownGoroutines, initialGoroutines+1,
		"Goroutines should be cleaned up after shutdown")
}

// TestAccessLogFilterFileHandleManagement tests proper file handle management
func TestAccessLogFilterFileHandleManagement(t *testing.T) {
	resetGlobalState()

	tempFile := "/tmp/test_access_log.log"
	defer os.Remove(tempFile)

	// Create filter
	filter := newFilter().(*Filter)

	// Create test URL and invocation
	url := common.NewURLWithOptions(
		common.WithParamsValue(constant.AccessLogFilterKey, tempFile),
	)

	invoker := &MockInvoker{url: url}
	invocation := &invocation_impl.RPCInvocation{}

	// Invoke multiple times to test file handle caching
	for i := 0; i < 5; i++ {
		filter.Invoke(context.Background(), invoker, invocation)
	}

	// Wait for logs to be processed
	time.Sleep(100 * time.Millisecond)

	// Check that file is in cache
	filter.fileLock.RLock()
	cachedFile, exists := filter.fileCache[tempFile]
	filter.fileLock.RUnlock()

	assert.True(t, exists, "File should be cached")
	assert.NotNil(t, cachedFile, "Cached file should not be nil")

	// Shutdown and verify files are closed
	Shutdown()

	// Check that cache is cleared
	filter.fileLock.RLock()
	cacheSize := len(filter.fileCache)
	filter.fileLock.RUnlock()

	assert.Equal(t, 0, cacheSize, "File cache should be empty after shutdown")
}

// MockInvoker for testing
type MockInvoker struct {
	base.BaseInvoker
	url *common.URL
}

func (m *MockInvoker) GetURL() *common.URL {
	return m.url
}

func (m *MockInvoker) Invoke(ctx context.Context, invocation base.Invocation) result.Result {
	return &result.RPCResult{}
}
