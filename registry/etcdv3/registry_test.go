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
	"path"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

func newTestEtcdV3Registry(t *testing.T) *etcdV3Registry {
	url, _ := common.NewURL("dubbo://127.0.0.1:2379")
	reg, err := newETCDV3Registry(url)
	require.NoError(t, err)
	r := reg.(*etcdV3Registry)
	t.Cleanup(func() { r.Destroy() })
	return r
}

// ---------- DoRegister tests ----------

func TestDoRegister(t *testing.T) {
	r := newTestEtcdV3Registry(t)

	root := "/dubbo/com.example.Service/providers"
	node := "dubbo%3A%2F%2F127.0.0.1%3A20000%2Fcom.example.Service%3Finterface%3Dcom.example.Service"
	key := path.Join(root, node)
	defer r.client.Delete(key)

	err := r.DoRegister(root, node)
	require.NoError(t, err)

	// Verify the key was created in etcd with a lease
	val, err := r.client.Get(key)
	assert.NoError(t, err)
	assert.Empty(t, val)
}

// ---------- DoUnregister tests ----------

func TestDoUnregister(t *testing.T) {
	r := newTestEtcdV3Registry(t)

	root := "/dubbo/com.example.Service/providers"
	node := "test-node"
	key := path.Join(root, node)

	// Pre-create the key via the etcd client
	err := r.client.Put(key, "")
	require.NoError(t, err)

	// Verify it exists first
	_, err = r.client.Get(key)
	require.NoError(t, err)

	err = r.DoUnregister(root, node)
	require.NoError(t, err)

	// Verify the key was deleted from etcd (Get now returns error)
	_, err = r.client.Get(key)
	assert.Error(t, err)
}

// ---------- DoSubscribe tests ----------

func TestDoSubscribe(t *testing.T) {
	r := newTestEtcdV3Registry(t)

	conf, _ := common.NewURL("dubbo://127.0.0.1:20000/com.example.Service?interface=com.example.Service")

	// Put a provider key before subscribing so ListenServiceEvent's initial
	// GetChildren finds it and fires a DataChange event.
	providerKey := "/dubbo/com.example.Service/providers/dubbo%3A%2F%2F127.0.0.1%3A20000%2Fcom.example.Service%3Finterface%3Dcom.example.Service"
	err := r.client.Put(providerKey, "")
	require.NoError(t, err)
	defer r.client.Delete(providerKey)

	listener, err := r.DoSubscribe(conf)
	require.NoError(t, err)
	require.NotNil(t, listener)

	// The GetChildren call inside ListenServiceEvent should fire DataChange
	// for the existing provider key, which propagates to listener.Next().
	done := make(chan struct{})
	var se *registry.ServiceEvent
	go func() {
		se, err = listener.Next()
		close(done)
	}()

	select {
	case <-done:
		assert.NoError(t, err)
		assert.NotNil(t, se)
		assert.Equal(t, remoting.EventTypeAdd, se.Action)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for subscribe event")
	}
}

func TestDoSubscribeReusesExistingListener(t *testing.T) {
	r := newTestEtcdV3Registry(t)

	conf, _ := common.NewURL("dubbo://127.0.0.1:20000/com.example.Service?interface=com.example.Service")

	providerKey := "/dubbo/com.example.Service/providers/dubbo%3A%2F%2F127.0.0.1%3A20000%2Fcom.example.Service%3Finterface%3Dcom.example.Service"
	err := r.client.Put(providerKey, "")
	require.NoError(t, err)
	defer r.client.Delete(providerKey)

	listener1, err := r.DoSubscribe(conf)
	require.NoError(t, err)

	// Consume the initial GetChildren event
	done := make(chan struct{})
	go func() {
		listener1.Next()
		close(done)
	}()
	<-done

	// Second subscribe should return the same listener
	listener2, err := r.DoSubscribe(conf)
	assert.NoError(t, err)
	assert.Equal(t, listener1, listener2)
}

func TestDoSubscribeExistingListenerClosed(t *testing.T) {
	r := newTestEtcdV3Registry(t)

	cl := NewConfigurationListener(r)
	cl.Close()
	conf, _ := common.NewURL("dubbo://127.0.0.1:20000/com.example.Service?interface=com.example.Service")
	r.dataListener.SubscribeURL(conf, cl)

	_, err := r.DoSubscribe(conf)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "configListener already been closed")
}

// ---------- DoUnsubscribe tests ----------

func TestDoUnsubscribe(t *testing.T) {
	r := newTestEtcdV3Registry(t)

	conf, _ := common.NewURL("dubbo://127.0.0.1:20000/com.example.Service?interface=com.example.Service")

	providerKey := "/dubbo/com.example.Service/providers/dubbo%3A%2F%2F127.0.0.1%3A20000%2Fcom.example.Service%3Finterface%3Dcom.example.Service"
	err := r.client.Put(providerKey, "")
	require.NoError(t, err)
	defer r.client.Delete(providerKey)

	_, err = r.DoSubscribe(conf)
	require.NoError(t, err)

	cl, err := r.DoUnsubscribe(conf)
	assert.NoError(t, err)
	assert.NotNil(t, cl)
	assert.True(t, cl.(*configurationListener).isClosed)

	// Verify removed from dataListener
	assert.NotContains(t, r.dataListener.subscribed, conf.ServiceKey())
}

func TestDoUnsubscribeNotSubscribed(t *testing.T) {
	r := newTestEtcdV3Registry(t)

	conf, _ := common.NewURL("dubbo://127.0.0.1:20000/com.example.Service?interface=com.example.Service")

	listener, err := r.DoUnsubscribe(conf)
	assert.NoError(t, err)
	assert.Nil(t, listener)
}

func TestDoUnsubscribeAlreadyClosed(t *testing.T) {
	r := newTestEtcdV3Registry(t)

	conf, _ := common.NewURL("dubbo://127.0.0.1:20000/com.example.Service?interface=com.example.Service")

	providerKey := "/dubbo/com.example.Service/providers/dubbo%3A%2F%2F127.0.0.1%3A20000%2Fcom.example.Service%3Finterface%3Dcom.example.Service"
	err := r.client.Put(providerKey, "")
	require.NoError(t, err)
	defer r.client.Delete(providerKey)

	_, err = r.DoSubscribe(conf)
	require.NoError(t, err)

	// Manually close the listener
	cl, ok := r.dataListener.subscribed[conf.ServiceKey()].(*configurationListener)
	require.True(t, ok)
	cl.Close()

	// Unsubscribe should error because the existing listener is already closed
	_, err = r.DoUnsubscribe(conf)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "configListener for service com.example.Service has already been closed")
}

// ---------- InitListeners tests ----------

func TestInitListeners(t *testing.T) {
	r := newTestEtcdV3Registry(t)

	assert.NotNil(t, r.listener)
	assert.NotNil(t, r.dataListener)
	assert.NotNil(t, r.dataListener.subscribed)
}

// ---------- CloseListener tests ----------

func TestCloseListener(t *testing.T) {
	r := newTestEtcdV3Registry(t)

	cl := NewConfigurationListener(r)
	url, _ := common.NewURL("dubbo://127.0.0.1:20000/com.example.Service?interface=com.example.Service")
	r.dataListener.SubscribeURL(url, cl)

	r.CloseListener()
	assert.True(t, cl.isClosed)
}

func TestCloseListenerNil(t *testing.T) {
	r := &etcdV3Registry{}
	// dataListener is nil, should not panic
	r.CloseListener()
}

// ---------- Registry interface conformance ----------

func TestRegistryInterface(t *testing.T) {
	var _ registry.Registry = (*etcdV3Registry)(nil)
}
