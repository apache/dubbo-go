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
	"sync"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
)

type mockServer struct {
	mu          sync.Mutex
	startCalled int
	stopCalled  int
}

func (m *mockServer) Start() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.startCalled++
}

func (m *mockServer) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopCalled++
}

// Compile-time check
var _ Server = (*mockServer)(nil)

func TestNewExchangeServer(t *testing.T) {
	url := common.NewURLWithOptions(
		common.WithProtocol("dubbo"),
		common.WithIp("127.0.0.1"),
		common.WithPort("20880"),
	)
	mockS := &mockServer{}

	es := NewExchangeServer(url, mockS)
	assert.NotNil(t, es)
	assert.Equal(t, url, es.URL)
	assert.Equal(t, mockS, es.Server)
}

func TestExchangeServerStartStop(t *testing.T) {
	url := common.NewURLWithOptions(
		common.WithProtocol("dubbo"),
		common.WithIp("127.0.0.1"),
		common.WithPort("20880"),
	)
	mockS := &mockServer{}
	es := NewExchangeServer(url, mockS)

	es.Start()
	assert.Equal(t, 1, mockS.startCalled)

	es.Stop()
	assert.Equal(t, 1, mockS.stopCalled)
}

func TestExchangeServerConcurrent(t *testing.T) {
	url := common.NewURLWithOptions(
		common.WithProtocol("dubbo"),
		common.WithIp("127.0.0.1"),
		common.WithPort("20880"),
	)
	mockS := &mockServer{}
	es := NewExchangeServer(url, mockS)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() { defer wg.Done(); es.Start() }()
		go func() { defer wg.Done(); es.Stop() }()
	}
	wg.Wait()

	assert.Equal(t, 50, mockS.startCalled)
	assert.Equal(t, 50, mockS.stopCalled)
}
