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
	"sync"
	"sync/atomic"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	invocation_impl "dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

// newShutdownTestFilter builds a Filter without starting processLogs, so the
// send/close synchronisation can be tested deterministically, without global
// state or background goroutines leaking into other tests.
func newShutdownTestFilter(buffer int) *Filter {
	ctx, cancel := context.WithCancel(context.Background())
	return &Filter{
		logChan:   make(chan Data, buffer),
		fileCache: make(map[string]*os.File),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// A send on a closed channel panics even inside a select with a default case:
// the send is considered ready, so default is never chosen. This is the race
// described in #3800 - Invoke may still be logging when Shutdown runs.
func TestLogIntoChannelAfterShutdownDoesNotPanic(t *testing.T) {
	f := newShutdownTestFilter(1)
	f.shutdown()

	require.NotPanics(t, func() {
		f.logIntoChannel(Data{accessLog: "true"})
	})
}

// Shutdown must not discard logs that were already queued, and must still
// close logChan: drainLogs relies on ok == false to finish (see #3564).
func TestShutdownKeepsLogsQueuedBeforeClose(t *testing.T) {
	f := newShutdownTestFilter(1)
	want := Data{accessLog: "true", data: map[string]string{"key": "value"}}
	f.logIntoChannel(want)
	f.shutdown()

	got, ok := <-f.logChan
	require.True(t, ok, "a log queued before shutdown must still be drainable")
	assert.Equal(t, want, got)

	_, ok = <-f.logChan
	assert.False(t, ok, "logChan must be closed so drainLogs can finish")
}

// Stress the send/close boundary: request goroutines keep invoking while
// shutdown closes logChan. Panics are recovered per goroutine so an unfixed
// implementation fails the assertion instead of crashing the test binary.
func TestInvokeConcurrentWithShutdownDoesNotPanic(t *testing.T) {
	const (
		goroutines = 32
		iterations = 200
	)
	// Large enough that the channel never fills, so the test exercises the
	// send/close race rather than the "channel is full" branch.
	f := newShutdownTestFilter(goroutines * iterations)
	invoker := &MockInvoker{url: common.NewURLWithOptions(
		common.WithParamsValue(constant.AccessLogFilterKey, "true"),
	)}
	inv := &invocation_impl.RPCInvocation{}

	var (
		wg         sync.WaitGroup
		panicCount atomic.Int32
	)
	start := make(chan struct{})
	for range goroutines {
		wg.Go(func() {
			defer func() {
				if r := recover(); r != nil {
					panicCount.Add(1)
				}
			}()
			<-start
			for range iterations {
				f.Invoke(context.Background(), invoker, inv)
			}
		})
	}
	close(start)
	f.shutdown()
	wg.Wait()

	assert.Zero(t, panicCount.Load(), "Invoke must not panic while shutdown closes logChan")
}
