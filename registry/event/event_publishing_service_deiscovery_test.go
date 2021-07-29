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

package event

import (
	"reflect"
	"testing"
)

import (
	gxset "github.com/dubbogo/gost/container/set"
	gxpage "github.com/dubbogo/gost/hash/page"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/common/observer"
	"dubbo.apache.org/dubbo-go/v3/common/observer/dispatcher"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/metadata/mapping"
	_ "dubbo.apache.org/dubbo-go/v3/metadata/service/local"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

func TestEventPublishingServiceDiscovery_DispatchEvent(t *testing.T) {
	// extension.SetMetadataService("local", local.NewMetadataService)

	config.GetApplicationConfig().MetadataType = "local"

	extension.SetGlobalServiceNameMapping(func() mapping.ServiceNameMapping {
		return mapping.NewMockServiceNameMapping()
	})

	dc := NewEventPublishingServiceDiscovery(&ServiceDiscoveryA{})
	tsd := &TestServiceDiscoveryDestroyingEventListener{
		BaseListener: observer.NewBaseListener(),
	}
	tsd.SetT(t)
	tsi := &TestServiceInstancePreRegisteredEventListener{}
	tsi.SetT(t)
	extension.AddEventListener(func() observer.EventListener {
		return tsd
	})
	extension.AddEventListener(func() observer.EventListener {
		return tsi
	})
	extension.SetEventDispatcher("direct", dispatcher.NewDirectEventDispatcher)
	extension.SetAndInitGlobalDispatcher("direct")
	err := dc.Destroy()
	assert.Nil(t, err)
	si := &registry.DefaultServiceInstance{ID: "testServiceInstance"}
	err = dc.Register(si)
	assert.Nil(t, err)
}

type TestServiceDiscoveryDestroyingEventListener struct {
	suite.Suite
	observer.BaseListener
}

func (tel *TestServiceDiscoveryDestroyingEventListener) OnEvent(e observer.Event) error {
	e1, ok := e.(*ServiceDiscoveryDestroyingEvent)
	assert.Equal(tel.T(), ok, true)
	assert.Equal(tel.T(), "testServiceDiscovery", e1.GetOriginal().String())
	assert.Equal(tel.T(), "testServiceDiscovery", e1.GetServiceDiscovery().String())
	return nil
}

func (tel *TestServiceDiscoveryDestroyingEventListener) GetPriority() int {
	return -1
}

func (tel *TestServiceDiscoveryDestroyingEventListener) GetEventType() reflect.Type {
	return reflect.TypeOf(ServiceDiscoveryDestroyingEvent{})
}

type TestServiceInstancePreRegisteredEventListener struct {
	suite.Suite
	observer.BaseListener
}

func (tel *TestServiceInstancePreRegisteredEventListener) OnEvent(e observer.Event) error {
	e1, ok := e.(*ServiceInstancePreRegisteredEvent)
	assert.Equal(tel.T(), ok, true)
	assert.Equal(tel.T(), "testServiceInstance", e1.getServiceInstance().GetID())
	return nil
}

func (tel *TestServiceInstancePreRegisteredEventListener) GetPriority() int {
	return -1
}

func (tel *TestServiceInstancePreRegisteredEventListener) GetEventType() reflect.Type {
	return reflect.TypeOf(ServiceInstancePreRegisteredEvent{})
}

type ServiceDiscoveryA struct{}

// String return mockServiceDiscovery
func (msd *ServiceDiscoveryA) String() string {
	return "testServiceDiscovery"
}

// Destroy do nothing
func (msd *ServiceDiscoveryA) Destroy() error {
	return nil
}

func (msd *ServiceDiscoveryA) Register(instance registry.ServiceInstance) error {
	return nil
}

func (msd *ServiceDiscoveryA) Update(instance registry.ServiceInstance) error {
	return nil
}

func (msd *ServiceDiscoveryA) Unregister(instance registry.ServiceInstance) error {
	return nil
}

func (msd *ServiceDiscoveryA) GetDefaultPageSize() int {
	return 1
}

func (msd *ServiceDiscoveryA) GetServices() *gxset.HashSet {
	return nil
}

func (msd *ServiceDiscoveryA) GetInstances(serviceName string) []registry.ServiceInstance {
	return nil
}

func (msd *ServiceDiscoveryA) GetInstancesByPage(serviceName string, offset int, pageSize int) gxpage.Pager {
	return nil
}

func (msd *ServiceDiscoveryA) GetHealthyInstancesByPage(serviceName string, offset int, pageSize int, healthy bool) gxpage.Pager {
	return nil
}

func (msd *ServiceDiscoveryA) GetRequestInstances(serviceNames []string, offset int, requestedSize int) map[string]gxpage.Pager {
	return nil
}

func (msd *ServiceDiscoveryA) AddListener(listener registry.ServiceInstancesChangedListener) error {
	return nil
}

func (msd *ServiceDiscoveryA) DispatchEventByServiceName(serviceName string) error {
	return nil
}

func (msd *ServiceDiscoveryA) DispatchEventForInstances(serviceName string, instances []registry.ServiceInstance) error {
	return nil
}

func (msd *ServiceDiscoveryA) DispatchEvent(event *registry.ServiceInstancesChangedEvent) error {
	return nil
}
