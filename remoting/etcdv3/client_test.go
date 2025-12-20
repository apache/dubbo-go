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
// Mock clientFacade Implementation
// ============================================

type mockClientFacade struct {
	client     *gxetcd.Client
	clientLock sync.Mutex
	wg         sync.WaitGroup
	done       chan struct{}
	url        *common.URL
}

func newMockClientFacade() *mockClientFacade {
	return &mockClientFacade{
		done: make(chan struct{}),
	}
}

func (m *mockClientFacade) Client() *gxetcd.Client {
	return m.client
}

func (m *mockClientFacade) SetClient(client *gxetcd.Client) {
	m.client = client
}

func (m *mockClientFacade) ClientLock() *sync.Mutex {
	return &m.clientLock
}

func (m *mockClientFacade) WaitGroup() *sync.WaitGroup {
	return &m.wg
}

func (m *mockClientFacade) Done() chan struct{} {
	return m.done
}

func (m *mockClientFacade) RestartCallBack() bool {
	return true
}

func (m *mockClientFacade) GetURL() *common.URL {
	return m.url
}

func (m *mockClientFacade) IsAvailable() bool {
	return m.client != nil
}

func (m *mockClientFacade) Destroy() {
	close(m.done)
}

// ============================================
// ValidateClient Tests
// ============================================

func TestValidateClientWithNilClient(t *testing.T) {
	tests := []struct {
		name    string
		opts    []gxetcd.Option
		wantErr bool
	}{
		{
			name: "invalid endpoints - connection refused",
			opts: []gxetcd.Option{
				gxetcd.WithName("test-client"),
				gxetcd.WithEndpoints("127.0.0.1:23791"), // invalid port
				gxetcd.WithTimeout(100 * time.Millisecond),
			},
			wantErr: true,
		},
		{
			name: "empty endpoints",
			opts: []gxetcd.Option{
				gxetcd.WithName("test-client"),
				gxetcd.WithTimeout(100 * time.Millisecond),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			facade := newMockClientFacade()
			err := ValidateClient(facade, tt.opts...)
			if tt.wantErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestValidateClientOptionsApply(t *testing.T) {
	tests := []struct {
		name         string
		opts         []gxetcd.Option
		expectedName string
	}{
		{
			name: "with name option",
			opts: []gxetcd.Option{
				gxetcd.WithName("my-etcd-client"),
			},
			expectedName: "my-etcd-client",
		},
		{
			name:         "without options",
			opts:         []gxetcd.Option{},
			expectedName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := &gxetcd.Options{}
			for _, opt := range tt.opts {
				opt(options)
			}
			assert.Equal(t, tt.expectedName, options.Name)
		})
	}
}

// ============================================
// NewServiceDiscoveryClient Tests
// ============================================

func TestNewServiceDiscoveryClientOptions(t *testing.T) {
	tests := []struct {
		name              string
		opts              []gxetcd.Option
		expectedHeartbeat int
	}{
		{
			name:              "default heartbeat",
			opts:              []gxetcd.Option{},
			expectedHeartbeat: 1,
		},
		{
			name: "custom heartbeat",
			opts: []gxetcd.Option{
				gxetcd.WithHeartbeat(5),
			},
			expectedHeartbeat: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that options are applied correctly
			options := &gxetcd.Options{
				Heartbeat: 1, // default
			}
			for _, opt := range tt.opts {
				opt(options)
			}
			assert.Equal(t, tt.expectedHeartbeat, options.Heartbeat)
		})
	}
}

func TestNewServiceDiscoveryClientWithInvalidEndpoints(t *testing.T) {
	// This test verifies that NewServiceDiscoveryClient handles errors gracefully
	client := NewServiceDiscoveryClient(
		gxetcd.WithName("test-sd-client"),
		gxetcd.WithEndpoints("invalid:99999"),
		gxetcd.WithTimeout(50*time.Millisecond),
	)
	// Client may be nil due to connection failure
	// The function logs the error but doesn't panic
	_ = client
}

// ============================================
// Options Tests
// ============================================

func TestGxetcdOptions(t *testing.T) {
	tests := []struct {
		name     string
		opts     []gxetcd.Option
		validate func(*testing.T, *gxetcd.Options)
	}{
		{
			name: "with name",
			opts: []gxetcd.Option{
				gxetcd.WithName("test-name"),
			},
			validate: func(t *testing.T, o *gxetcd.Options) {
				assert.Equal(t, "test-name", o.Name)
			},
		},
		{
			name: "with endpoints",
			opts: []gxetcd.Option{
				gxetcd.WithEndpoints("127.0.0.1:2379", "127.0.0.1:2380"),
			},
			validate: func(t *testing.T, o *gxetcd.Options) {
				assert.Equal(t, 2, len(o.Endpoints))
				assert.Equal(t, "127.0.0.1:2379", o.Endpoints[0])
			},
		},
		{
			name: "with timeout",
			opts: []gxetcd.Option{
				gxetcd.WithTimeout(5 * time.Second),
			},
			validate: func(t *testing.T, o *gxetcd.Options) {
				assert.Equal(t, 5*time.Second, o.Timeout)
			},
		},
		{
			name: "with heartbeat",
			opts: []gxetcd.Option{
				gxetcd.WithHeartbeat(10),
			},
			validate: func(t *testing.T, o *gxetcd.Options) {
				assert.Equal(t, 10, o.Heartbeat)
			},
		},
		{
			name: "with all options",
			opts: []gxetcd.Option{
				gxetcd.WithName("full-client"),
				gxetcd.WithEndpoints("localhost:2379"),
				gxetcd.WithTimeout(3 * time.Second),
				gxetcd.WithHeartbeat(2),
			},
			validate: func(t *testing.T, o *gxetcd.Options) {
				assert.Equal(t, "full-client", o.Name)
				assert.Equal(t, 1, len(o.Endpoints))
				assert.Equal(t, 3*time.Second, o.Timeout)
				assert.Equal(t, 2, o.Heartbeat)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := &gxetcd.Options{}
			for _, opt := range tt.opts {
				opt(options)
			}
			tt.validate(t, options)
		})
	}
}

// ============================================
// ClientFacade Interface Tests
// ============================================

func TestMockClientFacadeImplementation(t *testing.T) {
	facade := newMockClientFacade()

	t.Run("initial state", func(t *testing.T) {
		assert.Nil(t, facade.Client())
		assert.NotNil(t, facade.ClientLock())
		assert.NotNil(t, facade.WaitGroup())
		assert.NotNil(t, facade.Done())
		assert.False(t, facade.IsAvailable())
	})

	t.Run("set client", func(t *testing.T) {
		// We can't create a real client without etcd, but we can test the setter
		facade.SetClient(nil)
		assert.Nil(t, facade.Client())
	})

	t.Run("restart callback", func(t *testing.T) {
		result := facade.RestartCallBack()
		assert.True(t, result)
	})

	t.Run("get url", func(t *testing.T) {
		assert.Nil(t, facade.GetURL())

		url := common.NewURLWithOptions(
			common.WithProtocol("etcd"),
			common.WithLocation("127.0.0.1:2379"),
		)
		facade.url = url
		assert.Equal(t, url, facade.GetURL())
	})
}

// ============================================
// Concurrent Access Tests
// ============================================

func TestValidateClientConcurrentAccess(t *testing.T) {
	facade := newMockClientFacade()
	var wg sync.WaitGroup

	// Simulate concurrent access to ValidateClient
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// This will fail due to invalid endpoints, but should not panic
			_ = ValidateClient(facade,
				gxetcd.WithName("concurrent-test"),
				gxetcd.WithEndpoints("invalid:9999"),
				gxetcd.WithTimeout(10*time.Millisecond),
			)
		}()
	}

	wg.Wait()
}

// ============================================
// Edge Cases Tests
// ============================================

func TestValidateClientEdgeCases(t *testing.T) {
	t.Run("nil options", func(t *testing.T) {
		facade := newMockClientFacade()
		// Should handle gracefully even with no options
		err := ValidateClient(facade)
		// Will fail because no endpoints provided
		assert.NotNil(t, err)
	})

	t.Run("multiple endpoints", func(t *testing.T) {
		options := &gxetcd.Options{}
		gxetcd.WithEndpoints(
			"127.0.0.1:2379",
			"127.0.0.1:2380",
			"127.0.0.1:2381",
		)(options)
		assert.Equal(t, 3, len(options.Endpoints))
	})

	t.Run("zero timeout", func(t *testing.T) {
		options := &gxetcd.Options{}
		gxetcd.WithTimeout(0)(options)
		assert.Equal(t, time.Duration(0), options.Timeout)
	})
}
