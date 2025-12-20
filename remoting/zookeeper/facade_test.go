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

package zookeeper

import (
	"sync"
	"testing"
	"time"
)

import (
	gxzookeeper "github.com/dubbogo/gost/database/kv/zk"

	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
)

// ============================================
// Mock ZkClientFacade for Facade Tests
// ============================================
type facadeTestZkClientFacade struct {
	zkClient         *gxzookeeper.ZookeeperClient
	zkLock           sync.Mutex
	wg               sync.WaitGroup
	done             chan struct{}
	url              *common.URL
	restartCallCount int
	mu               sync.Mutex
}

func newFacadeTestZkClientFacade() *facadeTestZkClientFacade {
	return &facadeTestZkClientFacade{
		done: make(chan struct{}),
	}
}

func (f *facadeTestZkClientFacade) ZkClient() *gxzookeeper.ZookeeperClient {
	return f.zkClient
}

func (f *facadeTestZkClientFacade) SetZkClient(client *gxzookeeper.ZookeeperClient) {
	f.zkClient = client
}

func (f *facadeTestZkClientFacade) ZkClientLock() *sync.Mutex {
	return &f.zkLock
}

func (f *facadeTestZkClientFacade) WaitGroup() *sync.WaitGroup {
	return &f.wg
}

func (f *facadeTestZkClientFacade) Done() chan struct{} {
	return f.done
}

func (f *facadeTestZkClientFacade) RestartCallBack() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.restartCallCount++
	return true
}

func (f *facadeTestZkClientFacade) GetRestartCallCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.restartCallCount
}

func (f *facadeTestZkClientFacade) GetURL() *common.URL {
	return f.url
}

func (f *facadeTestZkClientFacade) Destroy() {
	close(f.done)
}

// ============================================
// ZkClientFacade Interface Tests
// ============================================

func TestZkClientFacadeInterface(t *testing.T) {
	t.Run("interface methods", func(t *testing.T) {
		facade := newFacadeTestZkClientFacade()

		// Test ZkClient() and SetZkClient()
		assert.Nil(t, facade.ZkClient())

		// Test ZkClientLock()
		lock := facade.ZkClientLock()
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
// Facade Pattern Tests
// ============================================

func TestFacadePatternImplementation(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *facadeTestZkClientFacade
		validate func(*testing.T, *facadeTestZkClientFacade)
	}{
		{
			name: "initial state",
			setup: func() *facadeTestZkClientFacade {
				return newFacadeTestZkClientFacade()
			},
			validate: func(t *testing.T, f *facadeTestZkClientFacade) {
				assert.Nil(t, f.ZkClient())
				assert.NotNil(t, f.ZkClientLock())
				assert.NotNil(t, f.WaitGroup())
				assert.NotNil(t, f.Done())
			},
		},
		{
			name: "restart callback tracking",
			setup: func() *facadeTestZkClientFacade {
				f := newFacadeTestZkClientFacade()
				f.RestartCallBack()
				f.RestartCallBack()
				f.RestartCallBack()
				return f
			},
			validate: func(t *testing.T, f *facadeTestZkClientFacade) {
				assert.Equal(t, 3, f.GetRestartCallCount())
			},
		},
		{
			name: "with url",
			setup: func() *facadeTestZkClientFacade {
				f := newFacadeTestZkClientFacade()
				f.url = common.NewURLWithOptions(
					common.WithProtocol("zookeeper"),
					common.WithLocation("127.0.0.1:2181"),
				)
				return f
			},
			validate: func(t *testing.T, f *facadeTestZkClientFacade) {
				assert.NotNil(t, f.GetURL())
				assert.Equal(t, "zookeeper", f.GetURL().Protocol)
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
	facade := newFacadeTestZkClientFacade()
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
	facade := newFacadeTestZkClientFacade()
	var wg sync.WaitGroup
	counter := 0

	// Test that ZkClientLock properly synchronizes access
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			lock := facade.ZkClientLock()
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
	facade := newFacadeTestZkClientFacade()

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
		facade := newFacadeTestZkClientFacade()
		select {
		case <-facade.Done():
			t.Fatal("Done channel should not be closed initially")
		default:
			// Expected
		}
	})

	t.Run("done channel closed after destroy", func(t *testing.T) {
		facade := newFacadeTestZkClientFacade()
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
	t.Run("nil url", func(t *testing.T) {
		facade := newFacadeTestZkClientFacade()
		assert.Nil(t, facade.GetURL())
	})

	t.Run("set nil client", func(t *testing.T) {
		facade := newFacadeTestZkClientFacade()
		facade.SetZkClient(nil)
		assert.Nil(t, facade.ZkClient())
	})
}

// ============================================
// Integration-like Tests
// ============================================

func TestFacadeLifecycle(t *testing.T) {
	facade := newFacadeTestZkClientFacade()

	// Simulate lifecycle
	t.Run("startup", func(t *testing.T) {
		assert.Nil(t, facade.ZkClient())
	})

	t.Run("set url", func(t *testing.T) {
		facade.url = common.NewURLWithOptions(
			common.WithProtocol("zookeeper"),
			common.WithLocation("127.0.0.1:2181"),
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

// ============================================
// URL Configuration Tests
// ============================================

func TestFacadeURLConfigurations(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
		ip       string
		port     string
		params   map[string]string
	}{
		{
			name:     "basic zookeeper url",
			protocol: "zookeeper",
			ip:       "127.0.0.1",
			port:     "2181",
			params:   nil,
		},
		{
			name:     "zookeeper with timeout",
			protocol: "zookeeper",
			ip:       "127.0.0.1",
			port:     "2181",
			params:   map[string]string{"timeout": "5s"},
		},
		{
			name:     "zookeeper with different port",
			protocol: "zookeeper",
			ip:       "192.168.1.100",
			port:     "2182",
			params:   map[string]string{"timeout": "10s", "session.timeout": "30s"},
		},
		{
			name:     "zookeeper with auth",
			protocol: "zookeeper",
			ip:       "127.0.0.1",
			port:     "2181",
			params:   map[string]string{"username": "admin", "password": "secret"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			facade := newFacadeTestZkClientFacade()
			facade.url = common.NewURLWithOptions(
				common.WithProtocol(tt.protocol),
				common.WithIp(tt.ip),
				common.WithPort(tt.port),
			)
			for k, v := range tt.params {
				facade.url.SetParam(k, v)
			}
			assert.NotNil(t, facade.GetURL())
			assert.Equal(t, tt.protocol, facade.GetURL().Protocol)
			assert.Equal(t, tt.ip+":"+tt.port, facade.GetURL().Location)
		})
	}
}

// ============================================
// Benchmark Tests
// ============================================

func BenchmarkFacadeRestartCallback(b *testing.B) {
	facade := newFacadeTestZkClientFacade()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		facade.RestartCallBack()
	}
}

func BenchmarkFacadeLockUnlock(b *testing.B) {
	facade := newFacadeTestZkClientFacade()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lock := facade.ZkClientLock()
		lock.Lock()
		lock.Unlock()
	}
}

func BenchmarkFacadeGetURL(b *testing.B) {
	facade := newFacadeTestZkClientFacade()
	facade.url = common.NewURLWithOptions(
		common.WithProtocol("zookeeper"),
		common.WithLocation("127.0.0.1:2181"),
	)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		facade.GetURL()
	}
}

func BenchmarkFacadeConcurrentRestartCallback(b *testing.B) {
	facade := newFacadeTestZkClientFacade()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			facade.RestartCallBack()
		}
	})
}

// ============================================
// State Transition Tests
// ============================================

func TestFacadeStateTransitions(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*facadeTestZkClientFacade)
		validate func(*testing.T, *facadeTestZkClientFacade)
	}{
		{
			name: "initial to configured",
			setup: func(f *facadeTestZkClientFacade) {
				f.url = common.NewURLWithOptions(
					common.WithProtocol("zookeeper"),
					common.WithLocation("127.0.0.1:2181"),
				)
			},
			validate: func(t *testing.T, f *facadeTestZkClientFacade) {
				assert.NotNil(t, f.GetURL())
				assert.Nil(t, f.ZkClient())
			},
		},
		{
			name: "configured with multiple restarts",
			setup: func(f *facadeTestZkClientFacade) {
				f.url = common.NewURLWithOptions(
					common.WithProtocol("zookeeper"),
					common.WithLocation("127.0.0.1:2181"),
				)
				for i := 0; i < 5; i++ {
					f.RestartCallBack()
				}
			},
			validate: func(t *testing.T, f *facadeTestZkClientFacade) {
				assert.Equal(t, 5, f.GetRestartCallCount())
			},
		},
		{
			name: "destroyed state",
			setup: func(f *facadeTestZkClientFacade) {
				f.Destroy()
			},
			validate: func(t *testing.T, f *facadeTestZkClientFacade) {
				select {
				case <-f.Done():
					// Expected - channel is closed
				default:
					t.Fatal("Done channel should be closed")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			facade := newFacadeTestZkClientFacade()
			tt.setup(facade)
			tt.validate(t, facade)
		})
	}
}

// ============================================
// Multiple Facade Instances Tests
// ============================================

func TestMultipleFacadeInstances(t *testing.T) {
	facades := make([]*facadeTestZkClientFacade, 10)
	for i := 0; i < 10; i++ {
		facades[i] = newFacadeTestZkClientFacade()
		facades[i].url = common.NewURLWithOptions(
			common.WithProtocol("zookeeper"),
			common.WithLocation("127.0.0.1:2181"),
		)
	}

	// Each facade should be independent
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				facades[idx].RestartCallBack()
			}
		}(i)
	}
	wg.Wait()

	// Verify each facade has its own count
	for i := 0; i < 10; i++ {
		assert.Equal(t, 10, facades[i].GetRestartCallCount())
	}
}
