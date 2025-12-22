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

package nacos

import (
	"sync"
	"testing"
	"time"
)

import (
	nacosClient "github.com/dubbogo/gost/database/kv/nacos"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
)

type mockFacade struct {
	wg   sync.WaitGroup
	done chan struct{}
}

func (m *mockFacade) NacosClient() *nacosClient.NacosConfigClient   { return nil }
func (m *mockFacade) SetNacosClient(*nacosClient.NacosConfigClient) {}
func (m *mockFacade) WaitGroup() *sync.WaitGroup                    { return &m.wg }
func (m *mockFacade) GetDone() chan struct{}                        { return m.done }
func (m *mockFacade) GetURL() *common.URL                           { return nil }
func (m *mockFacade) Destroy()                                      {}
func (m *mockFacade) IsAvailable() bool                             { return true }

func TestHandleClientRestart(t *testing.T) {
	m := &mockFacade{done: make(chan struct{}, 1)}
	m.wg.Add(1)
	go HandleClientRestart(m)

	m.done <- struct{}{}

	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Test passed
	case <-time.After(5 * time.Second):
		t.Fatal("Test timed out waiting for HandleClientRestart to complete")
	}
}
