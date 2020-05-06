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

package common

import (
	"github.com/apache/dubbo-go/common/extension"
	gxset "github.com/dubbogo/gost/container/set"
	gxpage "github.com/dubbogo/gost/page"
)

import (
	"github.com/apache/dubbo-go/registry"
)

var dispatcher = extension.GetGlobalDispatcher()

// EventPublishingServiceDiscovery will enhance Service Discovery
// Publish some event about service discovery
type EventPublishingServiceDiscovery struct {
	serviceDiscovery registry.ServiceDiscovery
}

// NewEventPublishingServiceDiscovery is a constructor
func NewEventPublishingServiceDiscovery(serviceDiscovery registry.ServiceDiscovery) *EventPublishingServiceDiscovery {
	return &EventPublishingServiceDiscovery{
		serviceDiscovery: serviceDiscovery,
	}
}

func (epsd *EventPublishingServiceDiscovery) String() string {
	return epsd.serviceDiscovery.String()
}

func (epsd *EventPublishingServiceDiscovery) Destroy() error {
	dispatcher.Dispatch(NewServiceDiscoveryDestroyingEvent(epsd, epsd.serviceDiscovery))
	if err := epsd.serviceDiscovery.Destroy(); err != nil {
		dispatcher.Dispatch(NewServiceDiscoveryExceptionEvent(epsd, epsd.serviceDiscovery, err))
		return err
	}
	dispatcher.Dispatch(NewServiceDiscoveryDestroyedEvent(epsd, epsd.serviceDiscovery))
	return nil
}

func (epsd *EventPublishingServiceDiscovery) Register(instance registry.ServiceInstance) error {
	dispatcher.Dispatch(NewServiceInstancePreRegisteredEvent(epsd.serviceDiscovery, instance))
	if err := epsd.serviceDiscovery.Register(instance); err != nil {
		dispatcher.Dispatch(NewServiceDiscoveryExceptionEvent(epsd, epsd.serviceDiscovery, err))
		return err
	}
	dispatcher.Dispatch(NewServiceInstanceRegisteredEvent(epsd.serviceDiscovery, instance))
	return nil
}

func (epsd *EventPublishingServiceDiscovery) Update(instance registry.ServiceInstance) error {
	if err := epsd.serviceDiscovery.Update(instance); err != nil {
		dispatcher.Dispatch(NewServiceDiscoveryExceptionEvent(epsd, epsd.serviceDiscovery, err))
		return err
	}
	return nil
}

func (epsd *EventPublishingServiceDiscovery) Unregister(instance registry.ServiceInstance) error {
	dispatcher.Dispatch(NewServiceInstancePreUnregisteredEvent(epsd.serviceDiscovery, instance))
	if err := epsd.serviceDiscovery.Register(instance); err != nil {
		dispatcher.Dispatch(NewServiceDiscoveryExceptionEvent(epsd, epsd.serviceDiscovery, err))
		return err
	}
	dispatcher.Dispatch(NewServiceInstanceUnregisteredEvent(epsd.serviceDiscovery, instance))
	return nil
}

func (epsd *EventPublishingServiceDiscovery) GetDefaultPageSize() int {
	return epsd.serviceDiscovery.GetDefaultPageSize()
}

func (epsd *EventPublishingServiceDiscovery) GetServices() *gxset.HashSet {
	return epsd.serviceDiscovery.GetServices()
}

func (epsd *EventPublishingServiceDiscovery) GetInstances(serviceName string) []registry.ServiceInstance {
	return epsd.serviceDiscovery.GetInstances(serviceName)
}

func (epsd *EventPublishingServiceDiscovery) GetInstancesByPage(serviceName string, offset int, pageSize int) gxpage.Pager {
	return epsd.serviceDiscovery.GetInstancesByPage(serviceName, offset, pageSize)
}

func (epsd *EventPublishingServiceDiscovery) GetHealthyInstancesByPage(serviceName string, offset int, pageSize int, healthy bool) gxpage.Pager {
	return epsd.serviceDiscovery.GetHealthyInstancesByPage(serviceName, offset, pageSize, healthy)
}

func (epsd *EventPublishingServiceDiscovery) GetRequestInstances(serviceNames []string, offset int, requestedSize int) map[string]gxpage.Pager {
	return epsd.serviceDiscovery.GetRequestInstances(serviceNames, offset, requestedSize)
}

func (epsd *EventPublishingServiceDiscovery) AddListener(listener *registry.ServiceInstancesChangedListener) error {
	dispatcher.AddEventListener(listener)
	return epsd.serviceDiscovery.AddListener(listener)
}

func (epsd *EventPublishingServiceDiscovery) DispatchEventByServiceName(serviceName string) error {
	return epsd.DispatchEventByServiceName(serviceName)
}

func (epsd *EventPublishingServiceDiscovery) DispatchEventForInstances(serviceName string, instances []registry.ServiceInstance) error {
	return epsd.serviceDiscovery.DispatchEventForInstances(serviceName, instances)
}

func (epsd *EventPublishingServiceDiscovery) DispatchEvent(event *registry.ServiceInstancesChangedEvent) error {
	return epsd.serviceDiscovery.DispatchEvent(event)
}
