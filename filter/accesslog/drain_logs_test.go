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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestDrainLogsReturnsAfterChannelClosed verifies that drainLogs does not hang
// when the log channel is already closed (the !ok -> return path).
func TestDrainLogsReturnsAfterChannelClosed(t *testing.T) {
	f := &Filter{logChan: make(chan Data), ctx: context.Background()}
	close(f.logChan)

	done := make(chan struct{})
	go func() {
		f.drainLogs()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("drainLogs did not return after the channel was closed")
	}
}

// TestDrainLogsBlocksOnEmptyOpenChannel is a regression test for #3558: the
// previous implementation returned immediately on an empty channel because of a
// `default: return` branch, which made the 5s timeout guard dead. After the fix
// drainLogs must block on an empty, open channel until it is closed.
func TestDrainLogsBlocksOnEmptyOpenChannel(t *testing.T) {
	f := &Filter{logChan: make(chan Data), ctx: context.Background()}

	done := make(chan struct{})
	go func() {
		f.drainLogs()
		close(done)
	}()

	// It should still be running (blocking) shortly after start.
	select {
	case <-done:
		t.Fatal("drainLogs returned immediately on an empty open channel (dead guard bug)")
	case <-time.After(300 * time.Millisecond):
		// expected: still blocking, the 5s guard is now live
	}

	// Release it and make sure it eventually returns.
	close(f.logChan)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("drainLogs did not return after the channel was closed")
	}
}

// TestDrainLogsDrainsBufferedData verifies that all buffered log entries are
// flushed (and written to the configured file) before drainLogs returns.
func TestDrainLogsDrainsBufferedData(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "access.log")
	f := &Filter{
		logChan:   make(chan Data, 3),
		ctx:       context.Background(),
		fileCache: make(map[string]*os.File),
	}

	for i := 0; i < 3; i++ {
		f.logChan <- Data{
			accessLog: tmp,
			data:      map[string]string{"k": "v"},
		}
	}
	close(f.logChan)

	f.drainLogs()

	content, err := os.ReadFile(tmp)
	assert.NoError(t, err)
	assert.Equal(t, 3, strings.Count(string(content), "\n"),
		"all buffered log entries should be written")
}
