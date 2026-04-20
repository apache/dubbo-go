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
	"context"
	"errors"
	"os/signal"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

// MockFilter implements filter.Filter and config.Setter for testing
type MockFilter struct {
	mock.Mock
}

type testProtocol struct {
	destroy func()
}

func (p *testProtocol) Export(invoker base.Invoker) base.Exporter {
	return nil
}

func (p *testProtocol) Refer(url *common.URL) base.Invoker {
	return nil
}

func (p *testProtocol) Destroy() {
	if p.destroy != nil {
		p.destroy()
	}
}

type testRegistryProtocol struct {
	testProtocol
	unregister func()
}

func (p *testRegistryProtocol) UnregisterRegistries() {
	if p.unregister != nil {
		p.unregister()
	}
}

func getProtocolIfPresent(name string) (protocol base.Protocol, ok bool) {
	defer func() {
		if recover() != nil {
			protocol = nil
			ok = false
		}
	}()
	protocol = extension.GetProtocol(name)
	ok = protocol != nil
	return protocol, ok
}

func (m *MockFilter) Set(key string, value any) {
	m.Called(key, value)
}

func (m *MockFilter) Invoke(ctx context.Context, invoker base.Invoker, invocation base.Invocation) result.Result {
	return nil
}

func (m *MockFilter) OnResponse(ctx context.Context, result result.Result, invoker base.Invoker, invocation base.Invocation) result.Result {
	return nil
}

func resetShutdownTestState() {
	initOnce = sync.Once{}
	protocols = nil
	proMu = sync.Mutex{}

	shutdownConfigMu = sync.RWMutex{}
	shutdownConfig = nil
	shutdownOnce = sync.Once{}
	shutdownStarted = atomic.Bool{}
	shutdownDone = make(chan struct{})
	shutdownResult = nil
	signalNotify = signal.Notify
}

func TestInit(t *testing.T) {
	// Reset initOnce and protocols for testing
	resetShutdownTestState()

	// Register mock filters
	mockConsumerFilter := &MockFilter{}
	mockProviderFilter := &MockFilter{}

	// Expect Set method calls
	mockConsumerFilter.On("Set", mock.Anything, mock.Anything).Return()
	mockProviderFilter.On("Set", mock.Anything, mock.Anything).Return()

	// Register mock filters
	extension.SetFilter(constant.GracefulShutdownConsumerFilterKey, func() filter.Filter {
		return mockConsumerFilter
	})
	extension.SetFilter(constant.GracefulShutdownProviderFilterKey, func() filter.Filter {
		return mockProviderFilter
	})

	// Test with default options
	Init()

	// Test with custom options
	customTimeout := 120 * time.Second
	Init(WithTimeout(customTimeout))

	// Remove mock filters
	extension.UnregisterFilter(constant.GracefulShutdownConsumerFilterKey)
	extension.UnregisterFilter(constant.GracefulShutdownProviderFilterKey)
}

func TestShutdownClosesDoneAndRunsOnce(t *testing.T) {
	resetShutdownTestState()

	callbackName := "shutdown-run-once-test"
	originalCallback, callbackExists := extension.LookupGracefulShutdownCallback(callbackName)
	t.Cleanup(func() {
		extension.UnregisterGracefulShutdownCallback(callbackName)
		if callbackExists {
			extension.RegisterGracefulShutdownCallback(callbackName, originalCallback)
		}
	})

	var callbackCalls atomic.Int32
	extension.RegisterGracefulShutdownCallback(callbackName, func(ctx context.Context) error {
		callbackCalls.Add(1)
		return nil
	})

	cfg := global.DefaultShutdownConfig()
	internalSignal := false
	cfg.InternalSignal = &internalSignal
	cfg.ConsumerUpdateWaitTime = "0s"
	cfg.StepTimeout = "0s"
	cfg.NotifyTimeout = "10ms"
	cfg.OfflineRequestWindowTimeout = "0s"

	Init(SetShutdownConfig(cfg))

	firstDone := make(chan error, 1)
	secondDone := make(chan error, 1)
	go func() {
		firstDone <- Shutdown(context.Background())
	}()
	go func() {
		secondDone <- Shutdown(context.Background())
	}()

	require.NoError(t, <-firstDone)
	require.NoError(t, <-secondDone)
	assert.Equal(t, int32(1), callbackCalls.Load())

	select {
	case <-Done():
	default:
		t.Fatal("Done channel was not closed after Shutdown completed")
	}
}

func TestRegisterProtocol(t *testing.T) {
	// Reset protocols for testing
	protocols = make(map[string]struct{})
	proMu = sync.Mutex{}

	// Register some protocols
	RegisterProtocol("dubbo")
	RegisterProtocol("rest")
	RegisterProtocol("tri")

	// Check if protocols are registered correctly
	proMu.Lock()
	defer proMu.Unlock()

	assert.Contains(t, protocols, "dubbo")
	assert.Contains(t, protocols, "rest")
	assert.Contains(t, protocols, "tri")
	assert.Len(t, protocols, 3)
}

func TestRegisterProtocolInitializesMapWhenNeeded(t *testing.T) {
	protocols = nil
	proMu = sync.Mutex{}

	assert.NotPanics(t, func() {
		RegisterProtocol("grpc")
	})

	proMu.Lock()
	defer proMu.Unlock()
	assert.Contains(t, protocols, "grpc")
}

func TestTotalTimeout(t *testing.T) {
	// Test with default timeout
	config := global.DefaultShutdownConfig()
	timeout := totalTimeout(config)
	assert.Equal(t, defaultTimeout, timeout)

	// Test with custom timeout
	config.Timeout = "120s"
	timeout = totalTimeout(config)
	assert.Equal(t, 120*time.Second, timeout)

	// Test with invalid timeout
	config.Timeout = "invalid"
	timeout = totalTimeout(config)
	assert.Equal(t, defaultTimeout, timeout)

	// Test with timeout less than default
	config.Timeout = "30s"
	timeout = totalTimeout(config)
	assert.Equal(t, defaultTimeout, timeout) // Should use default if less than default
}

func TestParseDuration(t *testing.T) {
	// Test with valid duration
	res := parseDuration("10s", "test", 5*time.Second)
	assert.Equal(t, 10*time.Second, res)

	// Test with invalid duration
	res = parseDuration("invalid", "test", 5*time.Second)
	assert.Equal(t, 5*time.Second, res)

	// Test with empty string
	res = parseDuration("", "test", 5*time.Second)
	assert.Equal(t, 5*time.Second, res)
}

func TestWaitAndAcceptNewRequests(t *testing.T) {
	// Test with positive step timeout
	config := global.DefaultShutdownConfig()
	config.StepTimeout = "100ms"
	config.ProviderActiveCount.Store(0)

	start := time.Now()
	waitAndAcceptNewRequests(config)
	elapsed := time.Since(start)

	// Should wait for ConsumerUpdateWaitTime (default 3s) plus a little extra for processing
	assert.GreaterOrEqual(t, elapsed, 3*time.Second)

	// Test with negative step timeout (should skip waiting)
	config.StepTimeout = "-1s"
	start = time.Now()
	waitAndAcceptNewRequests(config)
	elapsed = time.Since(start)

	// Should only wait for ConsumerUpdateWaitTime
	assert.Less(t, elapsed, 3*time.Second+100*time.Millisecond)
}

func TestWaitForSendingAndReceivingRequests(t *testing.T) {
	// Test with active consumer requests
	config := global.DefaultShutdownConfig()
	config.StepTimeout = "100ms"
	config.ConsumerActiveCount.Store(1)

	start := time.Now()
	waitForSendingAndReceivingRequests(config)
	elapsed := time.Since(start)

	// Should wait for step timeout
	assert.GreaterOrEqual(t, elapsed, 100*time.Millisecond)

	// Test with no active consumer requests
	config.ConsumerActiveCount.Store(0)
	start = time.Now()
	waitForSendingAndReceivingRequests(config)
	elapsed = time.Since(start)

	// Should return immediately
	assert.Less(t, elapsed, 50*time.Millisecond)
}

func TestNotifyLongConnectionConsumersUsesIndependentTimeouts(t *testing.T) {
	firstName := "shutdown-timeout-first"
	secondName := "shutdown-timeout-second"

	originalFirst, firstExists := extension.LookupGracefulShutdownCallback(firstName)
	originalSecond, secondExists := extension.LookupGracefulShutdownCallback(secondName)
	t.Cleanup(func() {
		extension.UnregisterGracefulShutdownCallback(firstName)
		extension.UnregisterGracefulShutdownCallback(secondName)
		if firstExists {
			extension.RegisterGracefulShutdownCallback(firstName, originalFirst)
		}
		if secondExists {
			extension.RegisterGracefulShutdownCallback(secondName, originalSecond)
		}
	})

	var secondCalled atomic.Bool
	extension.RegisterGracefulShutdownCallback(firstName, func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})
	extension.RegisterGracefulShutdownCallback(secondName, func(ctx context.Context) error {
		secondCalled.Store(true)
		return nil
	})

	config := global.DefaultShutdownConfig()
	config.NotifyTimeout = "100ms"

	notifyLongConnectionConsumers(config)

	assert.True(t, secondCalled.Load())
}

func TestNotifyLongConnectionConsumersUsesShutdownNotifyTimeout(t *testing.T) {
	name := "shutdown-step-timeout"

	original, exists := extension.LookupGracefulShutdownCallback(name)
	t.Cleanup(func() {
		extension.UnregisterGracefulShutdownCallback(name)
		if exists {
			extension.RegisterGracefulShutdownCallback(name, original)
		}
	})

	extension.RegisterGracefulShutdownCallback(name, func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})

	config := global.DefaultShutdownConfig()
	config.NotifyTimeout = "100ms"

	start := time.Now()
	notifyLongConnectionConsumers(config)
	elapsed := time.Since(start)

	assert.GreaterOrEqual(t, elapsed, 100*time.Millisecond)
	assert.Less(t, elapsed, time.Second)
}

func TestNotifyLongConnectionConsumersRunsCallbacksInParallel(t *testing.T) {
	firstName := "shutdown-parallel-first"
	secondName := "shutdown-parallel-second"

	originalFirst, firstExists := extension.LookupGracefulShutdownCallback(firstName)
	originalSecond, secondExists := extension.LookupGracefulShutdownCallback(secondName)
	t.Cleanup(func() {
		extension.UnregisterGracefulShutdownCallback(firstName)
		extension.UnregisterGracefulShutdownCallback(secondName)
		if firstExists {
			extension.RegisterGracefulShutdownCallback(firstName, originalFirst)
		}
		if secondExists {
			extension.RegisterGracefulShutdownCallback(secondName, originalSecond)
		}
	})

	started := make(chan struct{}, 2)
	release := make(chan struct{})
	extension.RegisterGracefulShutdownCallback(firstName, func(ctx context.Context) error {
		started <- struct{}{}
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
	extension.RegisterGracefulShutdownCallback(secondName, func(ctx context.Context) error {
		started <- struct{}{}
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return errors.New("unexpected timeout")
		}
	})

	done := make(chan struct{})
	go func() {
		config := global.DefaultShutdownConfig()
		config.NotifyTimeout = "1s"
		notifyLongConnectionConsumers(config)
		close(done)
	}()

	for i := 0; i < 2; i++ {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("callbacks did not start in parallel")
		}
	}
	close(release)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("notifyLongConnectionConsumers did not finish")
	}
}

func TestBeforeShutdownNotifiesProtocolsBeforeDestroy(t *testing.T) {
	initOnce = sync.Once{}
	protocols = make(map[string]struct{})
	proMu = sync.Mutex{}

	events := make([]string, 0, 3)
	var eventsMu sync.Mutex
	record := func(event string) {
		eventsMu.Lock()
		defer eventsMu.Unlock()
		events = append(events, event)
	}

	originalRegistryProtocol, registryProtocolExists := getProtocolIfPresent(constant.RegistryProtocol)
	extension.SetProtocol(constant.RegistryProtocol, func() base.Protocol {
		return &testRegistryProtocol{unregister: func() { record("unregister-registry") }}
	})
	t.Cleanup(func() {
		if registryProtocolExists {
			extension.SetProtocol(constant.RegistryProtocol, func() base.Protocol { return originalRegistryProtocol })
			return
		}
		extension.UnregisterProtocol(constant.RegistryProtocol)
	})

	testProtocolName := "shutdown-order-test-protocol"
	extension.SetProtocol(testProtocolName, func() base.Protocol {
		return &testProtocol{destroy: func() { record("destroy-protocol") }}
	})
	t.Cleanup(func() {
		extension.UnregisterProtocol(testProtocolName)
	})

	originalCallback, callbackExists := extension.LookupGracefulShutdownCallback(testProtocolName)
	extension.RegisterGracefulShutdownCallback(testProtocolName, func(ctx context.Context) error {
		record("notify-protocol")
		return nil
	})
	t.Cleanup(func() {
		extension.UnregisterGracefulShutdownCallback(testProtocolName)
		if callbackExists {
			extension.RegisterGracefulShutdownCallback(testProtocolName, originalCallback)
		}
	})

	RegisterProtocol(testProtocolName)

	config := global.DefaultShutdownConfig()
	config.ConsumerUpdateWaitTime = "0s"
	config.StepTimeout = "100ms"
	config.OfflineRequestWindowTimeout = "0s"
	config.ProviderActiveCount.Store(0)
	config.ConsumerActiveCount.Store(0)

	beforeShutdown(config)

	assert.Equal(t, []string{"unregister-registry", "notify-protocol", "destroy-protocol"}, events)
}

func TestUnregisterRegistriesSkipsMissingRegistryProtocol(t *testing.T) {
	originalRegistryProtocol, registryProtocolExists := getProtocolIfPresent(constant.RegistryProtocol)
	extension.UnregisterProtocol(constant.RegistryProtocol)
	t.Cleanup(func() {
		if registryProtocolExists {
			extension.SetProtocol(constant.RegistryProtocol, func() base.Protocol { return originalRegistryProtocol })
		}
	})

	assert.NotPanics(t, func() {
		unregisterRegistries()
	})
}

func TestUnregisterRegistriesPrefersUnregisterOnlyCapability(t *testing.T) {
	originalRegistryProtocol, registryProtocolExists := getProtocolIfPresent(constant.RegistryProtocol)
	defer func() {
		if registryProtocolExists {
			extension.SetProtocol(constant.RegistryProtocol, func() base.Protocol { return originalRegistryProtocol })
			return
		}
		extension.UnregisterProtocol(constant.RegistryProtocol)
	}()

	called := make([]string, 0, 2)
	extension.SetProtocol(constant.RegistryProtocol, func() base.Protocol {
		return &testRegistryProtocol{
			testProtocol: testProtocol{
				destroy: func() {
					called = append(called, "destroy")
				},
			},
			unregister: func() {
				called = append(called, "unregister")
			},
		}
	})

	unregisterRegistries()

	assert.Equal(t, []string{"unregister"}, called)
}

func TestUnregisterRegistriesFallsBackToDestroy(t *testing.T) {
	originalRegistryProtocol, registryProtocolExists := getProtocolIfPresent(constant.RegistryProtocol)
	defer func() {
		if registryProtocolExists {
			extension.SetProtocol(constant.RegistryProtocol, func() base.Protocol { return originalRegistryProtocol })
			return
		}
		extension.UnregisterProtocol(constant.RegistryProtocol)
	}()

	called := false
	extension.SetProtocol(constant.RegistryProtocol, func() base.Protocol {
		return &testProtocol{
			destroy: func() {
				called = true
			},
		}
	})

	unregisterRegistries()

	assert.True(t, called)
}

func TestDestroyProtocolsSkipsMissingProtocol(t *testing.T) {
	protocols = map[string]struct{}{"missing-shutdown-protocol": {}}
	proMu = sync.Mutex{}

	assert.NotPanics(t, func() {
		destroyProtocols()
	})
}

func TestInvokeCustomShutdownCallbackDoesNotBlockForever(t *testing.T) {
	block := make(chan struct{})
	callback := func() {
		<-block
	}

	start := time.Now()
	invokeCustomShutdownCallback(100*time.Millisecond, callback)
	elapsed := time.Since(start)

	assert.Less(t, elapsed, time.Second)
}
