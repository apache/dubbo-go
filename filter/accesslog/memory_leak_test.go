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
	"bytes"
	"context"
	"os"
	"runtime"
	"strconv"
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

// TestAccessLogFilterGoroutineShutdown verifies that newFilter starts the
// processLogs goroutine and that Shutdown terminates it. The start
// assertion requires the processLogs goroutine to appear in a
// runtime.Stack snapshot, so a future newFilter that forgets to start the
// goroutine fails the test. The shutdown assertion tracks that exact
// goroutine's ID instead of relying on the process-wide goroutine count,
// which is timing-sensitive under -race: a leftover goroutine from a
// previous test can be exiting concurrently with a newly started one,
// canceling out the count change.
func TestAccessLogFilterGoroutineShutdown(t *testing.T) {
	resetGlobalState()

	// Create filter (this should start the goroutine)
	filter := newFilter()
	assert.NotNil(t, filter)

	// The processLogs goroutine must be running; capture its goroutine ID.
	var gid int64
	assert.Eventually(t, func() bool {
		id, ok := processLogsGoroutineID()
		if ok {
			gid = id
		}
		return ok
	}, 30*time.Second, 10*time.Millisecond, "processLogs goroutine did not start")

	// Shutdown the filter; that exact goroutine must exit.
	Shutdown()
	assert.Eventually(t, func() bool {
		id, ok := processLogsGoroutineID()
		return !ok || id != gid
	}, 30*time.Second, 10*time.Millisecond, "processLogs goroutine did not exit after Shutdown")
}

// processLogsGoroutineID returns the goroutine ID of a running processLogs
// goroutine found in the runtime.Stack snapshot, and whether one was found.
func processLogsGoroutineID() (int64, bool) {
	buf := make([]byte, 64<<10)
	for {
		n := runtime.Stack(buf, true)
		if n < len(buf) {
			return findProcessLogsGoroutineID(buf[:n])
		}
		buf = make([]byte, len(buf)*2)
	}
}

// findProcessLogsGoroutineID scans the stack snapshot for the processLogs
// goroutine block and extracts its goroutine ID from the "goroutine <id>"
// header line.
func findProcessLogsGoroutineID(stack []byte) (int64, bool) {
	for block := range bytes.SplitSeq(stack, []byte("\n\n")) {
		if !bytes.Contains(block, []byte("accesslog.(*Filter).processLogs")) {
			continue
		}
		fields := bytes.Fields(bytes.SplitN(block, []byte("\n"), 2)[0])
		if len(fields) >= 2 {
			id, err := strconv.ParseInt(string(fields[1]), 10, 64)
			if err == nil {
				return id, true
			}
		}
	}
	return 0, false
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
