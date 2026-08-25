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
	"path/filepath"
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
	filterMu.Lock()
	accessLogFilter = nil
	filterMu.Unlock()
	once = sync.Once{}
}

// TestAccessLogFilterGoroutineShutdown tests that the goroutine is properly shut down
func TestAccessLogFilterGoroutineShutdown(t *testing.T) {
	resetGlobalState()

	filter, ok := newFilter().(*Filter)
	if !ok {
		t.Fatal("newFilter should return a *Filter")
	}

	Shutdown()

	// After shutdown the goroutine should exit deterministically
	if !filter.waitProcessLogs(5 * time.Second) {
		t.Fatal("processLogs did not exit after shutdown")
	}
}

// TestAccessLogFilterConcurrentInvokeShutdown drives concurrent Invoke and
// Shutdown calls to verify the filter neither panics on a closed channel nor
// races on the global state when running under -race.
func TestAccessLogFilterConcurrentInvokeShutdown(t *testing.T) {
	resetGlobalState()

	filter := newFilter().(*Filter)
	url := common.NewURLWithOptions(
		common.WithParamsValue(constant.AccessLogFilterKey, filepath.Join(t.TempDir(), "access.log")),
	)
	invoker := &MockInvoker{url: url}
	invocation := &invocation_impl.RPCInvocation{}

	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			for range 100 {
				filter.Invoke(context.Background(), invoker, invocation)
			}
		})
	}
	wg.Go(func() {
		Shutdown()
	})

	wg.Wait()

	if !filter.waitProcessLogs(5 * time.Second) {
		t.Fatal("processLogs did not exit after concurrent shutdown")
	}
}

// TestAccessLogFilterConcurrentInitAndShutdown races the very first
// newFilter call against Shutdown from a shared barrier to verify that
// publishing under filterMu establishes a happens-before edge with readers:
// under -race this must not report a data race on accessLogFilter.
func TestAccessLogFilterConcurrentInitAndShutdown(t *testing.T) {
	for range 100 {
		resetGlobalState()

		start := make(chan struct{})
		var wg sync.WaitGroup
		wg.Go(func() {
			<-start
			newFilter()
		})
		wg.Go(func() {
			<-start
			// May legitimately observe nil when initialization has not
			// finished yet, so only the race detector is the oracle here.
			Shutdown()
		})
		close(start)
		wg.Wait()

		// When the racing Shutdown above observed nil, the filter created by
		// newFilter was never shut down; this extra call guarantees the
		// processLogs goroutine of every round is reaped before the next one.
		Shutdown()
	}
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
	for range 5 {
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
