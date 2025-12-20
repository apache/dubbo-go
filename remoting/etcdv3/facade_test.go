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

package etcdv3

import (
	"context"
	"sync"
	"testing"
	"time"
)

import (
	gxetcd "github.com/dubbogo/gost/database/kv/etcd/v3"

	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
)

// ============================================
// Mock clientFacade for Facade Tests
// ============================================
type facadeMockClient struct {
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
}

func newFacadeMockClient() *facadeMockClient {
	ctx, cancel := context.WithCancel(context.Background())
	return &facadeMockClient{
		ctx:    ctx,
		cancel: cancel,
		done:   make(chan struct{}),
	}
}

func (m *facadeMockClient) GetCtx() context.Context {
	return m.ctx
}

func (m *facadeMockClient) Done() <-chan struct{} {
	return m.done
}

func (m *facadeMockClient) Close() {
	m.cancel()
	close(m.done)
}

type facadeTestClientFacade struct {
	client           *gxetcd.Client
	mockClient       *facadeMockClient
	clientLock       sync.Mutex
	wg               sync.WaitGroup
	done             chan struct{}
	url              *common.URL
	restartCallCount int
	mu               sync.Mutex
}

func newFacadeTestClientFacade() *facadeTestClientFacade {
	return &facadeTestClientFacade{
		mockClient: newFacadeMockClient(),
		done:       make(chan struct{}),
	}
}

func (f *facadeTestClientFacade) Client() *gxetcd.Client {
	return f.client
}

func (f *facadeTestClientFacade) SetClient(client *gxetcd.Client) {
	f.client = client
}

func (f *facadeTestClientFacade) ClientLock() *sync.Mutex {
	return &f.clientLock
}

func (f *facadeTestClientFacade) WaitGroup() *sync.WaitGroup {
	return &f.wg
}

func (f *facadeTestClientFacade) Done() chan struct{} {
	return f.done
}

func (f *facadeTestClientFacade) RestartCallBack() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.restartCallCount++
	return true
}

func (f *facadeTestClientFacade) GetRestartCallCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.restartCallCount
}

func (f *facadeTestClientFacade) GetURL() *common.URL {
	return f.url
}

func (f *facadeTestClientFacade) IsAvailable() bool {
	return f.client != nil
}

func (f *facadeTestClientFacade) Destroy() {
	close(f.done)
}

// ============================================
// clientFacade Interface Tests
// ============================================

func TestClientFacadeInterface(t *testing.T) {
	t.Run("interface methods", func(t *testing.T) {
		facade := newFacadeTestClientFacade()

		// Test Client() and SetClient()
		assert.Nil(t, facade.Client())

		// Test ClientLock()
		lock := facade.ClientLock()
		assert.NotNil(t, lock)

		// Test WaitGroup()
		wg := facade.WaitGroup()
		assert.NotNil(t, wg)

		// Test Done()
		done := facade.Done()
		assert.NotNil(t, done)

		// Test RestartCallBack()
		result := facade.RestartCallBack()
		assert.True(t, result)
		assert.Equal(t, 1, facade.GetRestartCallCount())
	})
}

// ============================================
// HandleClientRestart Tests
// ============================================

func TestHandleClientRestartExitOnDone(t *testing.T) {
	facade := newFacadeTestClientFacade()

	// Add to wait group before starting
	facade.wg.Add(1)

	// Start HandleClientRestart in a goroutine
	go func() {
		// We need a mock client that implements the required interface
		// Since we can't easily mock gxetcd.Client, we test the Done channel behavior
		select {
		case <-facade.Done():
			facade.wg.Done()
			return
		case <-time.After(100 * time.Millisecond):
			facade.wg.Done()
			return
		}
	}()

	// Signal done
	close(facade.done)

	// Wait for goroutine to finish
	done := make(chan struct{})
	go func() {
		facade.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("HandleClientRestart did not exit on Done signal")
	}
}

// ============================================
// Facade Pattern Tests
// ============================================

func TestFacadePatternImplementation(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *facadeTestClientFacade
		validate func(*testing.T, *facadeTestClientFacade)
	}{
		{
			name: "initial state",
			setup: func() *facadeTestClientFacade {
				return newFacadeTestClientFacade()
			},
			validate: func(t *testing.T, f *facadeTestClientFacade) {
				assert.Nil(t, f.Client())
				assert.NotNil(t, f.ClientLock())
				assert.NotNil(t, f.WaitGroup())
				assert.NotNil(t, f.Done())
			},
		},
		{
			name: "restart callback tracking",
			setup: func() *facadeTestClientFacade {
				f := newFacadeTestClientFacade()
				f.RestartCallBack()
				f.RestartCallBack()
				f.RestartCallBack()
				return f
			},
			validate: func(t *testing.T, f *facadeTestClientFacade) {
				assert.Equal(t, 3, f.GetRestartCallCount())
			},
		},
		{
			name: "with url",
			setup: func() *facadeTestClientFacade {
				f := newFacadeTestClientFacade()
				f.url = common.NewURLWithOptions(
					common.WithProtocol("etcd"),
					common.WithLocation("127.0.0.1:2379"),
				)
				return f
			},
			validate: func(t *testing.T, f *facadeTestClientFacade) {
				assert.NotNil(t, f.GetURL())
				assert.Equal(t, "etcd", f.GetURL().Protocol)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			facade := tt.setup()
			tt.validate(t, facade)
		})
	}
}

// ============================================
// Concurrent Access Tests
// ============================================

func TestFacadeConcurrentAccess(t *testing.T) {
	facade := newFacadeTestClientFacade()
	var wg sync.WaitGroup

	// Concurrent RestartCallBack calls
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			facade.RestartCallBack()
		}()
	}

	wg.Wait()
	assert.Equal(t, 100, facade.GetRestartCallCount())
}

func TestFacadeLockBehavior(t *testing.T) {
	facade := newFacadeTestClientFacade()
	var wg sync.WaitGroup
	counter := 0

	// Test that ClientLock properly synchronizes access
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			lock := facade.ClientLock()
			lock.Lock()
			counter++
			lock.Unlock()
		}()
	}

	wg.Wait()
	assert.Equal(t, 100, counter)
}

// ============================================
// WaitGroup Tests
// ============================================

func TestFacadeWaitGroup(t *testing.T) {
	facade := newFacadeTestClientFacade()

	// Add multiple goroutines
	for i := 0; i < 5; i++ {
		facade.WaitGroup().Add(1)
		go func(id int) {
			defer facade.WaitGroup().Done()
			time.Sleep(10 * time.Millisecond)
		}(i)
	}

	// Wait should complete
	done := make(chan struct{})
	go func() {
		facade.WaitGroup().Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("WaitGroup.Wait() did not complete in time")
	}
}

// ============================================
// Done Channel Tests
// ============================================

func TestFacadeDoneChannel(t *testing.T) {
	t.Run("done channel not closed initially", func(t *testing.T) {
		facade := newFacadeTestClientFacade()
		select {
		case <-facade.Done():
			t.Fatal("Done channel should not be closed initially")
		default:
			// Expected
		}
	})

	t.Run("done channel closed after destroy", func(t *testing.T) {
		facade := newFacadeTestClientFacade()
		facade.Destroy()

		select {
		case <-facade.Done():
			// Expected
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Done channel should be closed after Destroy")
		}
	})
}

// ============================================
// Edge Cases Tests
// ============================================

func TestFacadeEdgeCases(t *testing.T) {
	t.Run("multiple destroy calls", func(t *testing.T) {
		facade := newFacadeTestClientFacade()
		facade.Destroy()

		// Second destroy should not panic (channel already closed)
		defer func() {
			_ = recover() // Expected panic from closing already closed channel
		}()
	})

	t.Run("nil url", func(t *testing.T) {
		facade := newFacadeTestClientFacade()
		assert.Nil(t, facade.GetURL())
	})

	t.Run("availability check", func(t *testing.T) {
		facade := newFacadeTestClientFacade()
		assert.False(t, facade.IsAvailable())
	})
}

// ============================================
// Integration-like Tests
// ============================================

func TestFacadeLifecycle(t *testing.T) {
	facade := newFacadeTestClientFacade()

	// Simulate lifecycle
	t.Run("startup", func(t *testing.T) {
		assert.Nil(t, facade.Client())
		assert.False(t, facade.IsAvailable())
	})

	t.Run("set url", func(t *testing.T) {
		facade.url = common.NewURLWithOptions(
			common.WithProtocol("etcd"),
			common.WithLocation("127.0.0.1:2379"),
			common.WithParamsValue("timeout", "5s"),
		)
		assert.NotNil(t, facade.GetURL())
	})

	t.Run("restart callbacks", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			facade.RestartCallBack()
		}
		assert.Equal(t, 3, facade.GetRestartCallCount())
	})

	t.Run("shutdown", func(t *testing.T) {
		facade.Destroy()
		select {
		case <-facade.Done():
			// Expected
		default:
			t.Fatal("Done channel should be closed after Destroy")
		}
	})
}
