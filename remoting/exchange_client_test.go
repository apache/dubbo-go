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

package remoting

import (
	"errors"
	"sync"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
)

type mockClient struct {
	mu         sync.Mutex
	available  bool
	connectErr error
	connCount  int
}

func (m *mockClient) SetExchangeClient(client *ExchangeClient) {}
func (m *mockClient) Close()                                   {}
func (m *mockClient) IsAvailable() bool                        { m.mu.Lock(); defer m.mu.Unlock(); return m.available }

func (m *mockClient) Request(request *Request, timeout time.Duration, response *PendingResponse) error {
	return nil
}

func (m *mockClient) Connect(url *common.URL) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connCount++
	return m.connectErr
}

func testURL() *common.URL {
	return common.NewURLWithOptions(common.WithProtocol("dubbo"), common.WithIp("127.0.0.1"), common.WithPort("20880"))
}

func TestNewExchangeClient(t *testing.T) {
	t.Run("eager init", func(t *testing.T) {
		m := &mockClient{available: true}
		ec := NewExchangeClient(testURL(), m, 5*time.Second, false)
		assert.NotNil(t, ec)
		assert.True(t, m.connCount > 0)
	})

	t.Run("lazy init", func(t *testing.T) {
		m := &mockClient{available: true}
		ec := NewExchangeClient(testURL(), m, 5*time.Second, true)
		assert.NotNil(t, ec)
		assert.Equal(t, 0, m.connCount)
	})

	t.Run("connect fail", func(t *testing.T) {
		m := &mockClient{connectErr: errors.New("fail")}
		assert.Nil(t, NewExchangeClient(testURL(), m, 5*time.Second, false))
	})
}

func TestExchangeClientActiveNumber(t *testing.T) {
	ec := NewExchangeClient(testURL(), &mockClient{available: true}, 5*time.Second, true)
	assert.Equal(t, uint32(1), ec.GetActiveNumber())
	ec.IncreaseActiveNumber()
	assert.Equal(t, uint32(2), ec.GetActiveNumber())
	ec.DecreaseActiveNumber()
	assert.Equal(t, uint32(1), ec.GetActiveNumber())
}

func TestExchangeClientClose(t *testing.T) {
	m := &mockClient{available: true}
	ec := NewExchangeClient(testURL(), m, 5*time.Second, true)
	ec.Close()
	assert.False(t, ec.init)
}

func TestExchangeClientIsAvailable(t *testing.T) {
	m := &mockClient{available: true}
	ec := NewExchangeClient(testURL(), m, 5*time.Second, true)
	assert.True(t, ec.IsAvailable())
	m.mu.Lock()
	m.available = false
	m.mu.Unlock()
	assert.False(t, ec.IsAvailable())
}
