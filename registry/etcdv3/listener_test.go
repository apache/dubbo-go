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
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

// mockConfigListener implements config_center.ConfigurationListener for testing
type mockConfigListener struct {
	events []*config_center.ConfigChangeEvent
}

func (m *mockConfigListener) Process(configType *config_center.ConfigChangeEvent) {
	m.events = append(m.events, configType)
}

// ---------- dataListener tests ----------

func TestNewRegistryDataListener(t *testing.T) {
	dl := NewRegistryDataListener()
	assert.NotNil(t, dl)
	assert.NotNil(t, dl.subscribed)
	assert.False(t, dl.closed)
}

func TestDataListenerSubscribeURL(t *testing.T) {
	dl := NewRegistryDataListener()
	url, _ := common.NewURL("dubbo://127.0.0.1:20000/com.example.Service?interface=com.example.Service")
	listener := &mockConfigListener{}

	dl.SubscribeURL(url, listener)
	assert.Equal(t, listener, dl.subscribed[url.ServiceKey()])
	assert.NotContains(t, dl.subscribed, "nonexistent")
}

func TestDataListenerSubscribeURLAfterClose(t *testing.T) {
	dl := NewRegistryDataListener()
	dl.closed = true
	url, _ := common.NewURL("dubbo://127.0.0.1:20000/com.example.Service?interface=com.example.Service")

	dl.SubscribeURL(url, &mockConfigListener{})
	assert.Empty(t, dl.subscribed)
}

func TestDataListenerUnSubscribeURL(t *testing.T) {
	reg := &etcdV3Registry{}
	url, _ := common.NewURL("dubbo://127.0.0.1:20000/com.example.Service?interface=com.example.Service")
	dl := NewRegistryDataListener()

	cl := NewConfigurationListener(reg)
	dl.SubscribeURL(url, cl)

	returned := dl.UnSubscribeURL(url)
	assert.NotNil(t, returned)
	assert.True(t, cl.isClosed)
	assert.NotContains(t, dl.subscribed, url.ServiceKey())
}

func TestDataListenerUnSubscribeURLNotSubscribed(t *testing.T) {
	dl := NewRegistryDataListener()
	url, _ := common.NewURL("dubbo://127.0.0.1:20000/com.example.Service?interface=com.example.Service")

	returned := dl.UnSubscribeURL(url)
	assert.Nil(t, returned)
}

func TestDataListenerDataChangeMatch(t *testing.T) {
	dl := NewRegistryDataListener()
	url, _ := common.NewURL("dubbo://127.0.0.1:20000/com.example.Service?interface=com.example.Service")
	listener := &mockConfigListener{}
	dl.SubscribeURL(url, listener)

	event := remoting.Event{
		Path:   "/dubbo/com.example.Service/providers/dubbo://127.0.0.1:20000/com.example.Service?interface=com.example.Service",
		Action: remoting.EventTypeAdd,
	}

	got := dl.DataChange(event)
	assert.True(t, got)
	assert.Len(t, listener.events, 1)
	assert.Equal(t, event.Action, listener.events[0].ConfigType)
}

func TestDataListenerDataChangeNoProviderPath(t *testing.T) {
	dl := NewRegistryDataListener()

	event := remoting.Event{
		Path:   "/dubbo/com.example.Service/consumers/some-url",
		Action: remoting.EventTypeAdd,
	}

	got := dl.DataChange(event)
	assert.False(t, got)
}

func TestDataListenerDataChangeNoMatch(t *testing.T) {
	dl := NewRegistryDataListener()
	url, _ := common.NewURL("dubbo://127.0.0.1:20000/com.example.Service1?interface=com.example.Service1")
	listener := &mockConfigListener{}
	dl.SubscribeURL(url, listener)

	// event for a different service
	event := remoting.Event{
		Path:   "/dubbo/com.example.Service2/providers/dubbo://127.0.0.1:20000/com.example.Service2?interface=com.example.Service2",
		Action: remoting.EventTypeAdd,
	}

	got := dl.DataChange(event)
	assert.False(t, got)
	assert.Empty(t, listener.events)
}

func TestDataListenerDataChangeAfterClose(t *testing.T) {
	dl := NewRegistryDataListener()
	dl.closed = true

	event := remoting.Event{
		Path:   "/dubbo/com.example.Service/providers/dubbo://127.0.0.1:20000/com.example.Service?interface=com.example.Service",
		Action: remoting.EventTypeAdd,
	}

	got := dl.DataChange(event)
	assert.False(t, got)
}

func TestDataListenerClose(t *testing.T) {
	reg := &etcdV3Registry{}
	dl := NewRegistryDataListener()
	url1, _ := common.NewURL("dubbo://127.0.0.1:20000/com.example.Service1?interface=com.example.Service1")
	url2, _ := common.NewURL("dubbo://127.0.0.1:20000/com.example.Service2?interface=com.example.Service2")

	c1 := NewConfigurationListener(reg)
	c2 := NewConfigurationListener(reg)
	dl.SubscribeURL(url1, c1)
	dl.SubscribeURL(url2, c2)
	assert.False(t, dl.closed)

	dl.Close()
	assert.True(t, dl.closed)
	assert.True(t, c1.isClosed)
	assert.True(t, c2.isClosed)
}

func TestDataListenerCloseEmpty(t *testing.T) {
	dl := NewRegistryDataListener()
	dl.Close()
	assert.True(t, dl.closed)
}

// ---------- configurationListener tests ----------

func TestNewConfigurationListener(t *testing.T) {
	reg := &etcdV3Registry{}
	cl := NewConfigurationListener(reg)

	assert.NotNil(t, cl)
	assert.False(t, cl.isClosed)
	assert.NotNil(t, cl.close)
	assert.NotNil(t, cl.events)
}

func TestConfigurationListenerProcessAndNext(t *testing.T) {
	reg := &etcdV3Registry{}
	cl := NewConfigurationListener(reg)

	serviceURL, _ := common.NewURL("dubbo://127.0.0.1:20000/com.example.Service?interface=com.example.Service")
	event := &config_center.ConfigChangeEvent{
		Key:        "test-key",
		Value:      serviceURL,
		ConfigType: remoting.EventTypeAdd,
	}

	cl.Process(event)

	done := make(chan struct{})
	var result *testResult
	go func() {
		r, err := cl.Next()
		result = &testResult{serviceEvent: r, err: err}
		close(done)
	}()

	select {
	case <-done:
		assert.NoError(t, result.err)
		assert.NotNil(t, result.serviceEvent)
		assert.Equal(t, remoting.EventTypeAdd, result.serviceEvent.Action)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for Next()")
	}
}

type testResult struct {
	serviceEvent *registry.ServiceEvent
	err          error
}

func TestConfigurationListenerClose(t *testing.T) {
	reg := &etcdV3Registry{}
	cl := NewConfigurationListener(reg)

	assert.False(t, cl.isClosed)
	cl.Close()
	assert.True(t, cl.isClosed)

	// calling Close twice should not panic (closeOnce)
	cl.Close()
}

func TestConfigurationListenerNextReturnsErrorAfterClose(t *testing.T) {
	reg := &etcdV3Registry{}
	cl := NewConfigurationListener(reg)

	cl.Close()
	_, err := cl.Next()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "listener has been closed")
}
