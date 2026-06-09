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
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

type MockDataListener struct{}

func (*MockDataListener) Process(configType *config_center.ConfigChangeEvent) {}

func TestDataListenerDataChangeDispatchesToSubscribedService(t *testing.T) {
	reg := newTestEtcdRegistry(t)
	serviceURL := mustURL(t, "dubbo://127.0.0.1:20000/org.apache.demo.UserProvider?group=g&version=v")
	listener := NewConfigurationListener(reg, serviceURL)
	defer listener.Close()

	dataListener := NewRegistryDataListener()
	dataListener.SubscribeURL(serviceURL, listener)

	ok := dataListener.DataChange(remoting.Event{
		Path:   "/dubbo/org.apache.demo.UserProvider/providers/" + url.QueryEscape(serviceURL.String()),
		Action: remoting.EventTypeAdd,
	})
	require.True(t, ok)

	event, err := listener.Next()
	require.NoError(t, err)
	assert.Equal(t, remoting.EventTypeAdd, event.Action)
	assert.Equal(t, serviceURL.ServiceKey(), event.Service.ServiceKey())
}

func TestDataListenerUnsubscribeStopsDispatch(t *testing.T) {
	reg := newTestEtcdRegistry(t)
	serviceURL := mustURL(t, "dubbo://127.0.0.1:20000/org.apache.demo.UserProvider?group=g&version=v")
	listener := NewConfigurationListener(reg, serviceURL)

	dataListener := NewRegistryDataListener()
	dataListener.SubscribeURL(serviceURL, listener)
	removed := dataListener.UnSubscribeURL(serviceURL)

	require.Same(t, listener, removed)
	require.True(t, listener.isClosed)
	assert.False(t, dataListener.DataChange(remoting.Event{
		Path:   "/dubbo/org.apache.demo.UserProvider/providers/" + url.QueryEscape(serviceURL.String()),
		Action: remoting.EventTypeAdd,
	}))
}

func TestDataListenerCloseClosesSubscriptions(t *testing.T) {
	reg := newTestEtcdRegistry(t)
	serviceURL := mustURL(t, "dubbo://127.0.0.1:20000/org.apache.demo.UserProvider?group=g&version=v")
	listener := NewConfigurationListener(reg, serviceURL)

	dataListener := NewRegistryDataListener()
	dataListener.SubscribeURL(serviceURL, listener)
	dataListener.Close()

	require.True(t, dataListener.closed)
	require.True(t, listener.isClosed)
}

func newTestEtcdRegistry(t *testing.T) *etcdV3Registry {
	t.Helper()
	registryURL := mustURL(t, "etcdv3://127.0.0.1:2379")
	reg := &etcdV3Registry{}
	reg.InitBaseRegistry(registryURL, reg)
	return reg
}

func mustURL(t *testing.T, rawURL string) *common.URL {
	t.Helper()
	parsedURL, err := common.NewURL(rawURL)
	require.NoError(t, err)
	return parsedURL
}
