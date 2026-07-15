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

package apollo

import (
	"sync"
	"testing"
)

import (
	"github.com/apolloconfig/agollo/v4"
	"github.com/apolloconfig/agollo/v4/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	cc "dubbo.apache.org/dubbo-go/v3/config_center"
)

type trackingApolloClient struct {
	agollo.Client

	mu              sync.Mutex
	active          map[storage.ChangeListener]struct{}
	addCount        int
	removeCount     int
	onAdd           func(storage.ChangeListener)
	removeStarted   chan struct{}
	releaseRemove   chan struct{}
	removeStartOnce sync.Once
}

func newTrackingApolloClient() *trackingApolloClient {
	return &trackingApolloClient{
		active: make(map[storage.ChangeListener]struct{}),
	}
}

func (c *trackingApolloClient) AddChangeListener(listener storage.ChangeListener) {
	c.mu.Lock()
	c.active[listener] = struct{}{}
	c.addCount++
	onAdd := c.onAdd
	c.mu.Unlock()
	if onAdd != nil {
		onAdd(listener)
	}
}

func (c *trackingApolloClient) RemoveChangeListener(listener storage.ChangeListener) {
	if c.removeStarted != nil {
		c.removeStartOnce.Do(func() {
			close(c.removeStarted)
		})
		<-c.releaseRemove
	}
	c.mu.Lock()
	delete(c.active, listener)
	c.removeCount++
	c.mu.Unlock()
}

func (c *trackingApolloClient) state() (int, int, int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.addCount, c.removeCount, len(c.active)
}

func TestApolloConfigurationAddsListenerBeforeRegistration(t *testing.T) {
	client := newTrackingApolloClient()
	listener := &countingListener{}
	client.onAdd = func(registered storage.ChangeListener) {
		registered.OnNewestChange(testFullChangeEvent())
	}
	configuration := &apolloConfiguration{client: client}

	configuration.AddListener("application", listener)

	assert.Equal(t, int64(1), listener.count.Load())
	addCount, removeCount, activeCount := client.state()
	assert.Equal(t, 1, addCount)
	assert.Zero(t, removeCount)
	assert.Equal(t, 1, activeCount)
}

func TestApolloConfigurationAllowsReentrantRemovalDuringRegistration(t *testing.T) {
	client := newTrackingApolloClient()
	configuration := &apolloConfiguration{client: client}
	const key = "application"
	storedKey := cc.NewOptions().Center.Group + key
	var listener *callbackListener
	listener = &callbackListener{process: func(*cc.ConfigChangeEvent) {
		configuration.RemoveListener(key, listener)
	}}
	client.onAdd = func(registered storage.ChangeListener) {
		registered.OnNewestChange(testFullChangeEvent())
	}

	configuration.AddListener(key, listener)

	_, ok := configuration.listeners.Load(storedKey)
	assert.False(t, ok)
	addCount, removeCount, activeCount := client.state()
	assert.Equal(t, 1, addCount)
	assert.Equal(t, 1, removeCount)
	assert.Zero(t, activeCount)
}

func TestApolloConfigurationListenerLifecycle(t *testing.T) {
	client := newTrackingApolloClient()
	configuration := &apolloConfiguration{client: client}
	first := &countingListener{}
	second := &countingListener{}
	const key = "application"
	storedKey := cc.NewOptions().Center.Group + key

	configuration.AddListener(key, first)
	configuration.AddListener(key, second)
	configuration.RemoveListener(key, first)

	stored, ok := configuration.listeners.Load(storedKey)
	require.True(t, ok)
	assert.False(t, stored.(*apolloListener).IsEmpty())
	addCount, removeCount, activeCount := client.state()
	assert.Equal(t, 1, addCount)
	assert.Zero(t, removeCount)
	assert.Equal(t, 1, activeCount)

	configuration.RemoveListener(key, second)
	_, ok = configuration.listeners.Load(storedKey)
	assert.False(t, ok)
	addCount, removeCount, activeCount = client.state()
	assert.Equal(t, 1, addCount)
	assert.Equal(t, 1, removeCount)
	assert.Zero(t, activeCount)

	configuration.AddListener(key, first)
	_, ok = configuration.listeners.Load(storedKey)
	assert.True(t, ok)
	addCount, removeCount, activeCount = client.state()
	assert.Equal(t, 2, addCount)
	assert.Equal(t, 1, removeCount)
	assert.Equal(t, 1, activeCount)
}

func TestApolloConfigurationSerializesRemoveAndReAdd(t *testing.T) {
	client := newTrackingApolloClient()
	configuration := &apolloConfiguration{client: client}
	first := &countingListener{}
	second := &countingListener{}
	const key = "application"
	storedKey := cc.NewOptions().Center.Group + key
	configuration.AddListener(key, first)
	previous, ok := configuration.listeners.Load(storedKey)
	require.True(t, ok)

	client.removeStarted = make(chan struct{})
	client.releaseRemove = make(chan struct{})
	removeDone := make(chan struct{})
	go func() {
		configuration.RemoveListener(key, first)
		close(removeDone)
	}()
	waitForSignal(t, client.removeStarted)

	addDone := make(chan struct{})
	go func() {
		configuration.AddListener(key, second)
		close(addDone)
	}()
	waitForSignal(t, addDone)
	close(client.releaseRemove)
	waitForSignal(t, removeDone)

	current, ok := configuration.listeners.Load(storedKey)
	require.True(t, ok)
	assert.NotSame(t, previous, current)
	assert.False(t, current.(*apolloListener).IsEmpty())
	addCount, removeCount, activeCount := client.state()
	assert.Equal(t, 2, addCount)
	assert.Equal(t, 1, removeCount)
	assert.Equal(t, 1, activeCount)
}
