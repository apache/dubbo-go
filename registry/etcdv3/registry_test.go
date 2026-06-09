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
	"testing"
)

import (
	gxetcd "github.com/dubbogo/gost/database/kv/etcd/v3"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/remoting"
	remotingEtcdv3 "dubbo.apache.org/dubbo-go/v3/remoting/etcdv3"
)

func TestEtcdV3RegistryDoUnregisterRejectsInvalidClient(t *testing.T) {
	reg := newTestEtcdRegistry(t)

	err := reg.DoUnregister("/dubbo", "org.apache.demo.UserProvider")

	require.Error(t, err)
	assert.ErrorContains(t, err, "etcd client is not valid")
}

func TestEtcdV3RegistryDoUnsubscribeClosesServiceListener(t *testing.T) {
	reg := newTestEtcdRegistry(t)
	reg.listener = remotingEtcdv3.NewEventListener(nil)
	reg.dataListener = NewRegistryDataListener()

	serviceURL := mustURL(t, "dubbo://127.0.0.1:20000/org.apache.demo.UserProvider?group=g&version=v")
	listener := NewConfigurationListener(reg, serviceURL)
	reg.dataListener.SubscribeURL(serviceURL, listener)

	removed, err := reg.DoUnsubscribe(serviceURL)

	require.NoError(t, err)
	require.Same(t, listener, removed)
	assert.True(t, listener.isClosed)
	_, ok := reg.dataListener.subscribed[serviceURL.ServiceKey()]
	assert.False(t, ok)
}

func TestEtcdV3RegistryInitListenersWithoutExistingSubscriptions(t *testing.T) {
	reg := newTestEtcdRegistry(t)

	reg.InitListeners()

	assert.NotNil(t, reg.listener)
	require.NotNil(t, reg.dataListener)
	assert.Empty(t, reg.dataListener.subscribed)
}

func TestEtcdV3RegistryInitListenersRecoversExistingSubscriptions(t *testing.T) {
	reg := newTestEtcdRegistry(t)
	restore := stubListenServiceEvent()
	defer restore()
	reg.dataListener = NewRegistryDataListener()

	serviceURL := mustURL(t, "dubbo://127.0.0.1:20000/org.apache.demo.UserProvider?group=g&version=v")
	oldListener := NewConfigurationListener(reg, serviceURL)
	reg.dataListener.SubscribeURL(serviceURL, oldListener)
	oldDataListener := reg.dataListener

	reg.InitListeners()

	require.True(t, oldDataListener.closed)
	require.True(t, oldListener.isClosed)
	require.NotNil(t, reg.dataListener)
	recovered := reg.dataListener.subscribed[serviceURL.ServiceKey()]
	require.NotNil(t, recovered)
	require.NotSame(t, oldListener, recovered)
	recovered.(*configurationListener).Close()
}

func TestEtcdV3RegistryInitListenersSkipsInvalidSubscriptions(t *testing.T) {
	reg := newTestEtcdRegistry(t)
	reg.dataListener = NewRegistryDataListener()
	serviceURL := mustURL(t, "dubbo://127.0.0.1:20000/org.apache.demo.UserProvider?group=g&version=v")
	reg.dataListener.SubscribeURL(serviceURL, &MockDataListener{})

	reg.InitListeners()

	require.NotNil(t, reg.dataListener)
	assert.Empty(t, reg.dataListener.subscribed)
}

func TestEtcdV3RegistryCloseListenerClosesDataListener(t *testing.T) {
	reg := newTestEtcdRegistry(t)
	reg.dataListener = NewRegistryDataListener()

	reg.CloseListener()

	assert.True(t, reg.dataListener.closed)
}

func TestEtcdV3RegistryDoSubscribeReturnsExistingListener(t *testing.T) {
	reg := newTestEtcdRegistry(t)
	reg.dataListener = NewRegistryDataListener()

	serviceURL := mustURL(t, "dubbo://127.0.0.1:20000/org.apache.demo.UserProvider?group=g&version=v")
	listener := NewConfigurationListener(reg, serviceURL)
	defer listener.Close()
	reg.dataListener.SubscribeURL(serviceURL, listener)

	existing, err := reg.DoSubscribe(serviceURL)

	require.NoError(t, err)
	assert.Same(t, listener, existing)
}

func TestEtcdV3RegistryDoSubscribeRejectsClosedExistingListener(t *testing.T) {
	reg := newTestEtcdRegistry(t)
	reg.dataListener = NewRegistryDataListener()

	serviceURL := mustURL(t, "dubbo://127.0.0.1:20000/org.apache.demo.UserProvider?group=g&version=v")
	listener := NewConfigurationListener(reg, serviceURL)
	listener.Close()
	reg.dataListener.SubscribeURL(serviceURL, listener)

	existing, err := reg.DoSubscribe(serviceURL)

	require.Error(t, err)
	assert.Nil(t, existing)
	assert.ErrorContains(t, err, "configListener already been closed")
}

func TestEtcdV3RegistryDoSubscribeRejectsBrokenClient(t *testing.T) {
	reg := newTestEtcdRegistry(t)
	serviceURL := mustURL(t, "dubbo://127.0.0.1:20000/org.apache.demo.UserProvider?group=g&version=v")

	listener, err := reg.DoSubscribe(serviceURL)

	require.Error(t, err)
	assert.Nil(t, listener)
	assert.ErrorContains(t, err, "etcd client broken")
	require.NotNil(t, reg.dataListener)
}

func TestEtcdV3RegistryDoSubscribeRegistersNewListener(t *testing.T) {
	reg := newTestEtcdRegistry(t)
	restore := stubListenServiceEvent()
	defer restore()
	reg.client = &gxetcd.Client{}
	serviceURL := mustURL(t, "dubbo://127.0.0.1:20000/org.apache.demo.UserProvider?group=g&version=v")

	listener, err := reg.DoSubscribe(serviceURL)

	require.NoError(t, err)
	require.NotNil(t, listener)
	defer listener.(*configurationListener).Close()
	assert.NotNil(t, reg.listener)
	assert.Same(t, listener, reg.dataListener.subscribed[serviceURL.ServiceKey()])
}

func TestEtcdV3RegistryDoUnsubscribeRejectsNilDataListener(t *testing.T) {
	reg := newTestEtcdRegistry(t)
	serviceURL := mustURL(t, "dubbo://127.0.0.1:20000/org.apache.demo.UserProvider?group=g&version=v")

	listener, err := reg.DoUnsubscribe(serviceURL)

	require.Error(t, err)
	assert.Nil(t, listener)
	assert.ErrorContains(t, err, "etcd data listener is nil")
}

func TestEtcdV3RegistryDoUnsubscribeRejectsNilEventListener(t *testing.T) {
	reg := newTestEtcdRegistry(t)
	reg.dataListener = NewRegistryDataListener()
	serviceURL := mustURL(t, "dubbo://127.0.0.1:20000/org.apache.demo.UserProvider?group=g&version=v")

	listener, err := reg.DoUnsubscribe(serviceURL)

	require.Error(t, err)
	assert.Nil(t, listener)
	assert.ErrorContains(t, err, "etcd event listener is nil")
}

func TestEtcdV3RegistryDoUnsubscribeIgnoresMissingSubscription(t *testing.T) {
	reg := newTestEtcdRegistry(t)
	reg.listener = remotingEtcdv3.NewEventListener(nil)
	reg.dataListener = NewRegistryDataListener()
	serviceURL := mustURL(t, "dubbo://127.0.0.1:20000/org.apache.demo.UserProvider?group=g&version=v")

	listener, err := reg.DoUnsubscribe(serviceURL)

	require.NoError(t, err)
	assert.Nil(t, listener)
}

func TestEtcdV3RegistryDoUnsubscribeRejectsClosedSubscription(t *testing.T) {
	reg := newTestEtcdRegistry(t)
	reg.listener = remotingEtcdv3.NewEventListener(nil)
	reg.dataListener = NewRegistryDataListener()

	serviceURL := mustURL(t, "dubbo://127.0.0.1:20000/org.apache.demo.UserProvider?group=g&version=v")
	listener := NewConfigurationListener(reg, serviceURL)
	listener.Close()
	reg.dataListener.SubscribeURL(serviceURL, listener)

	removed, err := reg.DoUnsubscribe(serviceURL)

	require.Error(t, err)
	assert.Nil(t, removed)
	assert.ErrorContains(t, err, "has already been closed")
}

func TestEtcdProviderPath(t *testing.T) {
	serviceURL := mustURL(t, "dubbo://127.0.0.1:20000/org.apache.demo.UserProvider?group=g&version=v")

	assert.Equal(t, "/dubbo/org.apache.demo.UserProvider/providers", etcdProviderPath(serviceURL))
}

func stubListenServiceEvent() func() {
	original := listenServiceEvent
	listenServiceEvent = func(_ *remotingEtcdv3.EventListener, _ string, _ remoting.DataListener) {}
	return func() {
		listenServiceEvent = original
	}
}
