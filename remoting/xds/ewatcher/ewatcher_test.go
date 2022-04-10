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

package ewatcher

import (
	"sync"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/registry"
	mockRegistry "dubbo.apache.org/dubbo-go/v3/registry/mocks"
	"dubbo.apache.org/dubbo-go/v3/remoting"
	"dubbo.apache.org/dubbo-go/v3/xds/client/resource"
)

const (
	clusterNameFoo   = "outbound|20000||dubbo-go-app.svc.cluster.local"
	hostAddrFoo      = "dubbo-go-app.svc.cluster.local:20000"
	hostNameFoo      = "dubbo-go-app.svc.cluster.local"
	hostPortFoo      = "20000"
	interfaceNameFoo = "api.Greeter"
	serviceKeyFoo    = "provider::api.Greeter"
)

var (
	endpointHealthyUpdate = &resource.EndpointsUpdate{
		Localities: []resource.Locality{
			{
				Endpoints: []resource.Endpoint{
					{
						Address:      hostAddrFoo,
						HealthStatus: resource.EndpointHealthStatusHealthy,
					},
				},
			},
		},
	}

	endpointUnHealthyUpdate = &resource.EndpointsUpdate{
		Localities: []resource.Locality{
			{
				Endpoints: []resource.Endpoint{
					{
						Address:      hostAddrFoo,
						HealthStatus: resource.EndpointHealthStatusUnhealthy,
					},
				},
			},
		},
	}

	endpointUnknownUpdate = &resource.EndpointsUpdate{
		Localities: []resource.Locality{
			{
				Endpoints: []resource.Endpoint{
					{
						Address:      hostAddrFoo,
						HealthStatus: resource.EndpointHealthStatusUnknown,
					},
				},
			},
		},
	}
)

func TestNewEWatcherAndSetCancelFunction(t *testing.T) {
	hostAddrListenerMapFoo := map[string]map[string]registry.NotifyListener{
		hostAddrFoo: {
			serviceKeyFoo: &mockRegistry.NotifyListener{},
		},
	}
	ewatcher := NewEndpointWatcherCtxImpl(clusterNameFoo, hostAddrFoo, interfaceNameFoo, &sync.RWMutex{}, hostAddrListenerMapFoo)
	assert.NotNil(t, ewatcher)

	cancel := func() {}
	ewatcher.SetCancelFunction(cancel)
}

func TestNewEWatcherHandle(t *testing.T) {
	mockListener := newMockListener()
	hostAddrListenerMapFoo := map[string]map[string]registry.NotifyListener{
		hostAddrFoo: {
			serviceKeyFoo: mockListener,
		},
	}
	ewatcher := NewEndpointWatcherCtxImpl(clusterNameFoo, hostAddrFoo, interfaceNameFoo, &sync.RWMutex{}, hostAddrListenerMapFoo)
	assert.NotNil(t, ewatcher)

	ewatcher.Handle(*endpointHealthyUpdate, nil)
	assert.Equal(t, 1, len(mockListener.Events))
	assert.Equal(t, remoting.EventTypeUpdate, mockListener.Events[0].Action)
	assert.Equal(t, hostNameFoo, mockListener.Events[0].Service.Ip)
	assert.Equal(t, hostPortFoo, mockListener.Events[0].Service.Port)

	ewatcher.Handle(*endpointUnknownUpdate, nil)
	assert.Equal(t, 2, len(mockListener.Events))
	assert.Equal(t, remoting.EventTypeUpdate, mockListener.Events[1].Action)
	assert.Equal(t, hostNameFoo, mockListener.Events[1].Service.Ip)
	assert.Equal(t, hostPortFoo, mockListener.Events[1].Service.Port)

	ewatcher.Handle(*endpointUnHealthyUpdate, nil)
	assert.Equal(t, 3, len(mockListener.Events))
	assert.Equal(t, remoting.EventTypeDel, mockListener.Events[2].Action)
	assert.Equal(t, hostNameFoo, mockListener.Events[2].Service.Ip)
	assert.Equal(t, hostPortFoo, mockListener.Events[2].Service.Port)

}

func TestNewEWatcherDestroy(t *testing.T) {
	mockListener := newMockListener()
	hostAddrListenerMapFoo := map[string]map[string]registry.NotifyListener{
		hostAddrFoo: {
			serviceKeyFoo: mockListener,
		},
	}
	ewatcher := NewEndpointWatcherCtxImpl(clusterNameFoo, hostAddrFoo, interfaceNameFoo, &sync.RWMutex{}, hostAddrListenerMapFoo)
	assert.NotNil(t, ewatcher)

	ewatcher.SetCancelFunction(func() {})
	ewatcher.Destroy()

	assert.Equal(t, len(mockListener.Events), 1)
	assert.Equal(t, mockListener.Events[0].Action, remoting.EventTypeDel)
	assert.Equal(t, mockListener.Events[0].Service.Location, constant.MeshAnyAddrMatcher)
	assert.Equal(t, mockListener.Events[0].Service.GetParam(constant.MeshSubsetKey, ""), "")
	assert.Equal(t, mockListener.Events[0].Service.GetParam(constant.MeshClusterIDKey, ""), clusterNameFoo)
	assert.Equal(t, mockListener.Events[0].Service.GetParam(constant.MeshHostAddrKey, ""), hostAddrFoo)
}

func TestNewEWatcherDestroyWithNilCancel(t *testing.T) {
	mockListener := newMockListener()
	hostAddrListenerMapFoo := map[string]map[string]registry.NotifyListener{
		hostAddrFoo: {
			serviceKeyFoo: mockListener,
		},
	}
	ewatcher := NewEndpointWatcherCtxImpl(clusterNameFoo, hostAddrFoo, interfaceNameFoo, &sync.RWMutex{}, hostAddrListenerMapFoo)
	assert.NotNil(t, ewatcher)

	// test nil cancel
	ewatcher.Destroy()

	assert.Equal(t, len(mockListener.Events), 1)
	assert.Equal(t, mockListener.Events[0].Action, remoting.EventTypeDel)
	assert.Equal(t, mockListener.Events[0].Service.Location, constant.MeshAnyAddrMatcher)
	assert.Equal(t, mockListener.Events[0].Service.GetParam(constant.MeshSubsetKey, ""), "")
	assert.Equal(t, mockListener.Events[0].Service.GetParam(constant.MeshClusterIDKey, ""), clusterNameFoo)
	assert.Equal(t, mockListener.Events[0].Service.GetParam(constant.MeshHostAddrKey, ""), hostAddrFoo)
}

func TestGenerateRegistryEvent(t *testing.T) {
	// todo Add test

}

func newMockListener() *mockListener {
	return &mockListener{
		Events: make([]*registry.ServiceEvent, 0),
	}
}

type mockListener struct {
	Events []*registry.ServiceEvent
}

func (m *mockListener) Notify(event *registry.ServiceEvent) {
	m.Events = append(m.Events, event)
}

func (m *mockListener) NotifyAll(events []*registry.ServiceEvent, f func()) {
	m.Events = append(m.Events, events...)
}
