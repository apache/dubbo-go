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

// Package etcdv3 contains tests for etcdv3 registry components.
package etcdv3

import (
	"errors"
	"path"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gxetcd "github.com/dubbogo/gost/database/kv/etcd/v3"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/remoting/etcdv3"
)

func newTestRegistry(t *testing.T) *etcdV3Registry {
	t.Helper()
	registryURL, err := common.NewURL("registry://127.0.0.1:2379")
	require.NoError(t, err)
	reg := &etcdV3Registry{}
	reg.InitBaseRegistry(registryURL, reg)
	reg.dataListener = NewRegistryDataListener()
	reg.listener = etcdv3.NewEventListener(nil)
	return reg
}

func newTestServiceURL(t *testing.T, rawURL string) *common.URL {
	t.Helper()
	serviceURL, err := common.NewURL(rawURL)
	require.NoError(t, err)
	return serviceURL
}

func TestEtcdV3RegistryDoUnregisterDeletesNode(t *testing.T) {
	client := &gxetcd.Client{}
	reg := newTestRegistry(t)
	reg.client = client

	oldValid := etcdClientValid
	oldDelete := etcdClientDelete
	defer func() {
		etcdClientValid = oldValid
		etcdClientDelete = oldDelete
	}()

	var deletedKey string
	etcdClientValid = func(got *gxetcd.Client) bool {
		assert.Same(t, client, got)
		return true
	}
	etcdClientDelete = func(got *gxetcd.Client, key string) error {
		assert.Same(t, client, got)
		deletedKey = key
		return nil
	}

	err := reg.DoUnregister("/dubbo/org.apache.DemoService/providers", "dubbo://127.0.0.1:20000/org.apache.DemoService")
	require.NoError(t, err)
	assert.Equal(t, path.Join("/dubbo/org.apache.DemoService/providers", "dubbo://127.0.0.1:20000/org.apache.DemoService"), deletedKey)
}

func TestEtcdV3RegistryDoUnregisterRequiresValidClient(t *testing.T) {
	reg := newTestRegistry(t)
	reg.client = &gxetcd.Client{}

	oldValid := etcdClientValid
	defer func() {
		etcdClientValid = oldValid
	}()
	etcdClientValid = func(*gxetcd.Client) bool {
		return false
	}

	err := reg.DoUnregister("/dubbo", "node")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "etcd client is not valid")
}

func TestEtcdV3RegistryDoUnregisterPropagatesDeleteError(t *testing.T) {
	reg := newTestRegistry(t)
	reg.client = &gxetcd.Client{}
	deleteErr := errors.New("delete failed")

	oldValid := etcdClientValid
	oldDelete := etcdClientDelete
	defer func() {
		etcdClientValid = oldValid
		etcdClientDelete = oldDelete
	}()
	etcdClientValid = func(*gxetcd.Client) bool {
		return true
	}
	etcdClientDelete = func(*gxetcd.Client, string) error {
		return deleteErr
	}

	err := reg.DoUnregister("/dubbo", "node")
	require.ErrorIs(t, err, deleteErr)
}

func TestEtcdV3RegistryDoUnsubscribeRemovesListener(t *testing.T) {
	reg := newTestRegistry(t)
	serviceURL := newTestServiceURL(t, "dubbo://127.0.0.1:20000/org.apache.DemoService?group=g&version=1.0.0")
	configListener := NewConfigurationListener(reg, serviceURL)
	reg.dataListener.SubscribeURL(serviceURL, configListener)

	listener, err := reg.DoUnsubscribe(serviceURL)
	require.NoError(t, err)
	assert.Same(t, configListener, listener)
	assert.Empty(t, reg.dataListener.subscribed)

	_, err = configListener.Next()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "listener has been closed")
}

func TestEtcdV3RegistryDoUnsubscribeMissingListener(t *testing.T) {
	reg := newTestRegistry(t)
	serviceURL := newTestServiceURL(t, "dubbo://127.0.0.1:20000/org.apache.DemoService")

	listener, err := reg.DoUnsubscribe(serviceURL)
	require.NoError(t, err)
	assert.Nil(t, listener)
}

func TestEtcdV3RegistryDoSubscribeRegistersListener(t *testing.T) {
	reg := newTestRegistry(t)
	serviceURL := newTestServiceURL(t, "dubbo://127.0.0.1:20000/org.apache.DemoService?group=g&version=1.0.0")

	oldListen := listenEtcdServiceEvent
	defer func() {
		listenEtcdServiceEvent = oldListen
	}()

	listened := make(chan string, 1)
	listenEtcdServiceEvent = func(_ *etcdv3.EventListener, key string, _ *dataListener) {
		listened <- key
	}

	listener, err := reg.DoSubscribe(serviceURL)
	require.NoError(t, err)
	require.NotNil(t, listener)

	select {
	case key := <-listened:
		assert.Equal(t, "/dubbo/org.apache.DemoService/providers", key)
	case <-time.After(time.Second):
		t.Fatal("expected subscription to start listening")
	}
	assert.Same(t, listener, reg.dataListener.subscribed[serviceURL.ServiceKey()])
	listener.Close()
}

func TestEtcdV3RegistryDoSubscribeRebuildsMissingEventListener(t *testing.T) {
	reg := newTestRegistry(t)
	reg.listener = nil
	reg.client = &gxetcd.Client{}
	serviceURL := newTestServiceURL(t, "dubbo://127.0.0.1:20000/org.apache.DemoService")

	oldListen := listenEtcdServiceEvent
	defer func() {
		listenEtcdServiceEvent = oldListen
	}()
	listenEtcdServiceEvent = func(_ *etcdv3.EventListener, _ string, _ *dataListener) {}

	listener, err := reg.DoSubscribe(serviceURL)
	require.NoError(t, err)
	require.NotNil(t, listener)
	assert.NotNil(t, reg.listener)
	listener.Close()
}

func TestEtcdV3RegistryDoSubscribeFailsWithMissingClient(t *testing.T) {
	reg := newTestRegistry(t)
	reg.listener = nil
	reg.client = nil
	serviceURL := newTestServiceURL(t, "dubbo://127.0.0.1:20000/org.apache.DemoService")

	listener, err := reg.DoSubscribe(serviceURL)
	require.Error(t, err)
	assert.Nil(t, listener)
	assert.Contains(t, err.Error(), "etcd client broken")
}

func TestEtcdV3RegistryCloseListenerClosesDataListener(t *testing.T) {
	reg := newTestRegistry(t)
	serviceURL := newTestServiceURL(t, "dubbo://127.0.0.1:20000/org.apache.DemoService")
	configListener := NewConfigurationListener(reg, serviceURL)
	reg.dataListener.SubscribeURL(serviceURL, configListener)

	reg.CloseListener()

	assert.True(t, reg.dataListener.closed)
	_, err := configListener.Next()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "listener has been closed")
}

func TestEtcdV3RegistryInitListenersRecoversSubscriptions(t *testing.T) {
	reg := newTestRegistry(t)
	serviceURL := newTestServiceURL(t, "dubbo://127.0.0.1:20000/org.apache.DemoService?group=g&version=1.0.0")
	oldConfigListener := NewConfigurationListener(reg, serviceURL)
	reg.dataListener.SubscribeURL(serviceURL, oldConfigListener)

	oldListen := listenEtcdServiceEvent
	defer func() {
		listenEtcdServiceEvent = oldListen
	}()

	listened := make(chan string, 1)
	listenEtcdServiceEvent = func(_ *etcdv3.EventListener, key string, _ *dataListener) {
		listened <- key
	}

	reg.InitListeners()

	select {
	case key := <-listened:
		assert.Equal(t, "/dubbo/org.apache.DemoService/providers", key)
	case <-time.After(time.Second):
		t.Fatal("expected recovered subscription to restart listening")
	}
	assert.NotNil(t, reg.dataListener.subscribed[serviceURL.ServiceKey()])
	assert.False(t, reg.dataListener.closed)

	_, err := oldConfigListener.Next()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "listener has been closed")
}
