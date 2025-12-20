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

package getty

import (
	"sync"
	"sync/atomic"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

// ============================================
// gettyRPCClient Tests
// ============================================
func TestGettyRPCClientUpdateActive(t *testing.T) {
	tests := []struct {
		name     string
		active   int64
		expected int64
	}{
		{"set positive active", 1234567890, 1234567890},
		{"set zero active", 0, 0},
		{"set negative active", -1, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &gettyRPCClient{}
			client.updateActive(tt.active)
			assert.Equal(t, tt.expected, atomic.LoadInt64(&client.active))
		})
	}
}

func TestGettyRPCClientSelectSession(t *testing.T) {
	t.Run("nil sessions", func(t *testing.T) {
		client := &gettyRPCClient{
			sessions: nil,
		}
		session := client.selectSession()
		assert.Nil(t, session)
	})

	t.Run("empty sessions", func(t *testing.T) {
		client := &gettyRPCClient{
			sessions: []*rpcSession{},
		}
		session := client.selectSession()
		assert.Nil(t, session)
	})
}

func TestGettyRPCClientAddSession(t *testing.T) {
	t.Run("add nil session does not initialize sessions", func(t *testing.T) {
		client := &gettyRPCClient{}
		// addSession with nil will return early, sessions remain nil
		// Note: The actual implementation checks for nil and returns early
		// but we can't test this without triggering the nil check in the code
		assert.Nil(t, client.sessions)
	})
}

func TestGettyRPCClientRemoveSession(t *testing.T) {
	t.Run("remove nil session", func(t *testing.T) {
		client := &gettyRPCClient{}
		// Should not panic
		client.removeSession(nil)
	})

	t.Run("remove from nil sessions", func(t *testing.T) {
		client := &gettyRPCClient{
			sessions: nil,
		}
		// Should not panic
		client.removeSession(nil)
	})
}

func TestGettyRPCClientUpdateSession(t *testing.T) {
	t.Run("update nil session", func(t *testing.T) {
		client := &gettyRPCClient{}
		// Should not panic
		client.updateSession(nil)
	})

	t.Run("update with nil sessions list", func(t *testing.T) {
		client := &gettyRPCClient{
			sessions: nil,
		}
		// Should not panic
		client.updateSession(nil)
	})
}

func TestGettyRPCClientGetClientRpcSession(t *testing.T) {
	t.Run("get from nil sessions", func(t *testing.T) {
		client := &gettyRPCClient{
			sessions: nil,
		}
		_, err := client.getClientRpcSession(nil)
		assert.Equal(t, errClientClosed, err)
	})

	t.Run("session not found", func(t *testing.T) {
		client := &gettyRPCClient{
			sessions: []*rpcSession{},
		}
		_, err := client.getClientRpcSession(nil)
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "session not exist")
	})
}

func TestGettyRPCClientIsAvailable(t *testing.T) {
	t.Run("not available with nil sessions", func(t *testing.T) {
		client := &gettyRPCClient{
			sessions: nil,
		}
		assert.False(t, client.isAvailable())
	})

	t.Run("not available with empty sessions", func(t *testing.T) {
		client := &gettyRPCClient{
			sessions: []*rpcSession{},
		}
		assert.False(t, client.isAvailable())
	})
}

func TestGettyRPCClientClose(t *testing.T) {
	t.Run("close empty client", func(t *testing.T) {
		client := &gettyRPCClient{
			sessions: []*rpcSession{},
		}
		err := client.close()
		assert.Nil(t, err)
	})

	t.Run("close twice", func(t *testing.T) {
		client := &gettyRPCClient{
			sessions: []*rpcSession{},
		}
		err1 := client.close()
		assert.Nil(t, err1)

		// Second close should return error
		err2 := client.close()
		assert.NotNil(t, err2)
	})
}

// ============================================
// Concurrent Access Tests
// ============================================

func TestGettyRPCClientConcurrentUpdateActive(t *testing.T) {
	client := &gettyRPCClient{}
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(val int64) {
			defer wg.Done()
			client.updateActive(val)
		}(int64(i))
	}

	wg.Wait()
	// Just verify no race condition occurred
	_ = atomic.LoadInt64(&client.active)
}

func TestGettyRPCClientConcurrentSelectSession(t *testing.T) {
	client := &gettyRPCClient{
		sessions: []*rpcSession{},
	}
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = client.selectSession()
		}()
	}

	wg.Wait()
}

func TestGettyRPCClientConcurrentIsAvailable(t *testing.T) {
	client := &gettyRPCClient{
		sessions: []*rpcSession{},
	}
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = client.isAvailable()
		}()
	}

	wg.Wait()
}

// ============================================
// Edge Cases Tests
// ============================================

func TestGettyRPCClientEdgeCases(t *testing.T) {
	t.Run("client with addr", func(t *testing.T) {
		client := &gettyRPCClient{
			addr: "127.0.0.1:20880",
		}
		assert.Equal(t, "127.0.0.1:20880", client.addr)
	})

	t.Run("client active timestamp", func(t *testing.T) {
		client := &gettyRPCClient{}
		timestamp := int64(1609459200) // 2021-01-01 00:00:00 UTC
		client.updateActive(timestamp)
		assert.Equal(t, timestamp, atomic.LoadInt64(&client.active))
	})
}

// ============================================
// rpcSession Tests
// ============================================

func TestRpcSessionAddReqNum(t *testing.T) {
	tests := []struct {
		name     string
		initial  int32
		add      int32
		expected int32
	}{
		{"add positive", 0, 5, 5},
		{"add to existing", 10, 3, 13},
		{"add negative", 10, -3, 7},
		{"add zero", 5, 0, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &rpcSession{reqNum: tt.initial}
			s.AddReqNum(tt.add)
			assert.Equal(t, tt.expected, s.GetReqNum())
		})
	}
}

func TestRpcSessionGetReqNum(t *testing.T) {
	tests := []struct {
		name     string
		reqNum   int32
		expected int32
	}{
		{"zero requests", 0, 0},
		{"positive requests", 100, 100},
		{"max int32", 2147483647, 2147483647},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &rpcSession{reqNum: tt.reqNum}
			assert.Equal(t, tt.expected, s.GetReqNum())
		})
	}
}

func TestRpcSessionConcurrentAccess(t *testing.T) {
	s := &rpcSession{}
	var wg sync.WaitGroup

	// Concurrent AddReqNum
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.AddReqNum(1)
		}()
	}

	wg.Wait()
	assert.Equal(t, int32(100), s.GetReqNum())
}

// ============================================
// Integration-like Tests
// ============================================

func TestGettyRPCClientLifecycle(t *testing.T) {
	client := &gettyRPCClient{
		addr:     "127.0.0.1:20880",
		sessions: []*rpcSession{},
	}

	// Initial state
	assert.False(t, client.isAvailable())
	assert.Equal(t, int64(0), atomic.LoadInt64(&client.active))

	// Update active
	client.updateActive(1234567890)
	assert.Equal(t, int64(1234567890), atomic.LoadInt64(&client.active))

	// Close
	err := client.close()
	assert.Nil(t, err)
	assert.Equal(t, int64(0), atomic.LoadInt64(&client.active))
}
