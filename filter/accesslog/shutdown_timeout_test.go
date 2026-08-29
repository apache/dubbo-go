//go:build unix

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
	"strings"
	"syscall"
	"testing"
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	invocation_impl "dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

// TestAccessLogFilterShutdownTimeoutBlockedWriter verifies the shutdown
// timeout path: when processLogs is stuck inside a blocking write past the
// wait timeout, Shutdown must leave the cached file handle untouched and
// defer closing it to a background cleanup that runs after the writer exits.
func TestAccessLogFilterShutdownTimeoutBlockedWriter(t *testing.T) {
	resetGlobalState()

	fifo := filepath.Join(t.TempDir(), "access.log")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Fatalf("mkfifo: %v", err)
	}

	// Open the read end and keep it open without consuming, so writes into
	// the FIFO block once the kernel pipe buffer is full. The read end must
	// be opened with O_NONBLOCK: a blocking open on a FIFO waits for a
	// writer to show up and would hang the test right here.
	reader, err := os.OpenFile(fifo, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		t.Fatalf("open read end of fifo: %v", err)
	}

	filter := newFilter().(*Filter)
	filter.shutdownWaitTimeout = 200 * time.Millisecond

	// A large payload per entry fills the pipe buffer within one or two
	// writes, which keeps the blocking deterministic.
	bigPayload := strings.Repeat("x", 128*1024)
	invocation := invocation_impl.NewRPCInvocationWithOptions(
		invocation_impl.WithAttachment(constant.MethodKey, bigPayload),
	)
	url := common.NewURLWithOptions(
		common.WithParamsValue(constant.AccessLogFilterKey, fifo),
	)
	invoker := &MockInvoker{url: url}
	for range 8 {
		filter.Invoke(context.Background(), invoker, invocation)
	}

	// Wait until processLogs has opened the FIFO and cached the handle,
	// otherwise Shutdown would legitimately observe an empty cache.
	deadline := time.Now().Add(2 * time.Second)
	for {
		filter.fileLock.RLock()
		_, exists := filter.fileCache[fifo]
		filter.fileLock.RUnlock()
		if exists {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("writer never opened the fifo")
		}
		time.Sleep(10 * time.Millisecond)
	}
	// Give the writer time to block inside WriteString.
	time.Sleep(300 * time.Millisecond)

	// Hits the timeout path after 200ms because the writer never exits.
	Shutdown()

	filter.fileLock.RLock()
	cached := len(filter.fileCache)
	filter.fileLock.RUnlock()
	if cached == 0 {
		t.Fatal("file handles were closed while the writer was still blocked")
	}

	// The filter opens log files with O_RDWR and therefore holds its own
	// read end, so closing ours can never produce EPIPE. Unblock the writer
	// by draining the FIFO instead: once all queued entries are written, the
	// already canceled context stops processLogs and the deferred background
	// cleanup closes the cached files, which ends the draining goroutine.
	reader.Close()
	drainer, err := os.OpenFile(fifo, os.O_RDONLY, 0)
	if err != nil {
		t.Fatalf("open draining read end of fifo: %v", err)
	}
	defer drainer.Close()
	go func() {
		buf := make([]byte, 32*1024)
		for {
			if _, err := drainer.Read(buf); err != nil {
				return // EOF once every writer closed the FIFO
			}
		}
	}()

	if !filter.waitProcessLogs(5 * time.Second) {
		t.Fatal("processLogs did not exit after unblocking the writer")
	}

	// The background cleanup closes the files shortly after the writer exits;
	// poll instead of asserting right away.
	deadline = time.Now().Add(2 * time.Second)
	for {
		filter.fileLock.RLock()
		cached = len(filter.fileCache)
		filter.fileLock.RUnlock()
		if cached == 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("file cache should be empty once the background cleanup finished")
		}
		time.Sleep(10 * time.Millisecond)
	}
}
