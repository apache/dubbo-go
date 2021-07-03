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
	gxset "github.com/dubbogo/gost/container/set"
	gxpage "github.com/dubbogo/gost/hash/page"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/common/observer"
	"dubbo.apache.org/dubbo-go/v3/metadata/service"
	"dubbo.apache.org/dubbo-go/v3/metadata/service/local"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

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

// String returns serviceDiscovery.String()
func (epsd *EventPublishingServiceDiscovery) String() string {
	return epsd.serviceDiscovery.String()
}

// Destroy delegate function
func (epsd *EventPublishingServiceDiscovery) Destroy() error {
	f := func() error {
		return epsd.serviceDiscovery.Destroy()
	}
	return epsd.executeWithEvents(NewServiceDiscoveryDestroyingEvent(epsd, epsd.serviceDiscovery),
		f, NewServiceDiscoveryDestroyedEvent(epsd, epsd.serviceDiscovery))
}

// Register delegate function
func (epsd *EventPublishingServiceDiscovery) Register(instance registry.ServiceInstance) error {
	f := func() error {
		return epsd.serviceDiscovery.Register(instance)
	}
	return epsd.executeWithEvents(NewServiceInstancePreRegisteredEvent(epsd.serviceDiscovery, instance),
		f, NewServiceInstanceRegisteredEvent(epsd.serviceDiscovery, instance))
}

// Update returns the result of serviceDiscovery.Update
func (epsd *EventPublishingServiceDiscovery) Update(instance registry.ServiceInstance) error {
	f := func() error {
		return epsd.serviceDiscovery.Update(instance)
	}
	return epsd.executeWithEvents(nil, f, nil)
}

// Unregister unregister the instance and drop ServiceInstancePreUnregisteredEvent and ServiceInstanceUnregisteredEvent
func (epsd *EventPublishingServiceDiscovery) Unregister(instance registry.ServiceInstance) error {
	f := func() error {
		return epsd.serviceDiscovery.Unregister(instance)
	}
	return epsd.executeWithEvents(NewServiceInstancePreUnregisteredEvent(epsd.serviceDiscovery, instance),
		f, NewServiceInstanceUnregisteredEvent(epsd.serviceDiscovery, instance))
}

// GetDefaultPageSize returns the result of serviceDiscovery.GetDefaultPageSize
func (epsd *EventPublishingServiceDiscovery) GetDefaultPageSize() int {
	return epsd.serviceDiscovery.GetDefaultPageSize()
}

// GetServices returns the result of serviceDiscovery.GetServices
func (epsd *EventPublishingServiceDiscovery) GetServices() *gxset.HashSet {
	return epsd.serviceDiscovery.GetServices()
}

// GetInstances returns the result of serviceDiscovery.GetInstances
func (epsd *EventPublishingServiceDiscovery) GetInstances(serviceName string) []registry.ServiceInstance {
	return epsd.serviceDiscovery.GetInstances(serviceName)
}

// GetInstancesByPage returns the result of serviceDiscovery.GetInstancesByPage
func (epsd *EventPublishingServiceDiscovery) GetInstancesByPage(serviceName string, offset int, pageSize int) gxpage.Pager {
	return epsd.serviceDiscovery.GetInstancesByPage(serviceName, offset, pageSize)
}

// GetHealthyInstancesByPage returns the result of serviceDiscovery.GetHealthyInstancesByPage
func (epsd *EventPublishingServiceDiscovery) GetHealthyInstancesByPage(serviceName string, offset int, pageSize int, healthy bool) gxpage.Pager {
	return epsd.serviceDiscovery.GetHealthyInstancesByPage(serviceName, offset, pageSize, healthy)
}

// GetRequestInstances returns result from serviceDiscovery.GetRequestInstances
func (epsd *EventPublishingServiceDiscovery) GetRequestInstances(serviceNames []string, offset int, requestedSize int) map[string]gxpage.Pager {
	return epsd.serviceDiscovery.GetRequestInstances(serviceNames, offset, requestedSize)
}

// AddListener add event listener
func (epsd *EventPublishingServiceDiscovery) AddListener(listener registry.ServiceInstancesChangedListener) error {
	extension.GetGlobalDispatcher().AddEventListener(listener)
	return epsd.serviceDiscovery.AddListener(listener)
}

// DispatchEventByServiceName pass serviceName to serviceDiscovery
func (epsd *EventPublishingServiceDiscovery) DispatchEventByServiceName(serviceName string) error {
	return epsd.serviceDiscovery.DispatchEventByServiceName(serviceName)
}

// DispatchEventForInstances pass params to serviceDiscovery
func (epsd *EventPublishingServiceDiscovery) DispatchEventForInstances(serviceName string, instances []registry.ServiceInstance) error {
	return epsd.serviceDiscovery.DispatchEventForInstances(serviceName, instances)
}

// DispatchEvent pass the event to serviceDiscovery
func (epsd *EventPublishingServiceDiscovery) DispatchEvent(event *registry.ServiceInstancesChangedEvent) error {
	return epsd.serviceDiscovery.DispatchEvent(event)
}

// executeWithEvents dispatch before event and after event if return error will dispatch exception event
func (epsd *EventPublishingServiceDiscovery) executeWithEvents(beforeEvent observer.Event, f func() error, afterEvent observer.Event) error {
	globalDispatcher := extension.GetGlobalDispatcher()
	if beforeEvent != nil {
		globalDispatcher.Dispatch(beforeEvent)
	}
	if err := f(); err != nil {
		globalDispatcher.Dispatch(NewServiceDiscoveryExceptionEvent(epsd, epsd.serviceDiscovery, err))
		return err
	}
	if afterEvent != nil {
		globalDispatcher.Dispatch(afterEvent)
	}
	return nil
}

// getMetadataService returns metadata service instance
func getMetadataService() (service.MetadataService, error) {
	return local.GetLocalMetadataService()
}
