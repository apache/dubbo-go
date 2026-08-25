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
// newFilter call against Shutdown from a shared barrier. The single Shutdown
// call must complete the whole lifecycle on its own — once.Do(doInit) makes
// it wait for the in-flight initialization before shutting the filter down —
// so after both calls finish, the processLogs goroutine must have exited
// without any compensating Shutdown.
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
			Shutdown()
		})
		close(start)
		wg.Wait()

		filter, ok := newFilter().(*Filter)
		if !ok {
			t.Fatal("newFilter should return a *Filter")
		}
		if !filter.waitProcessLogs(5 * time.Second) {
			t.Fatal("processLogs did not exit after the concurrent Shutdown returned")
		}
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

	// Wait until the consumer goroutine has opened and cached the log file;
	// a fixed sleep here would be flaky under CI load.
	deadline := time.Now().Add(2 * time.Second)
	var cachedFile *os.File
	var exists bool
	for {
		filter.fileLock.RLock()
		cachedFile, exists = filter.fileCache[tempFile]
		filter.fileLock.RUnlock()
		if exists {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("log file was never cached by the consumer")
		}
		time.Sleep(10 * time.Millisecond)
	}

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
