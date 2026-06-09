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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
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
