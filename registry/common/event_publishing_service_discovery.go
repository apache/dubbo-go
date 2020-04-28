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
	gxset "github.com/dubbogo/gost/container/set"
	gxpage "github.com/dubbogo/gost/page"
)

import (
	"github.com/apache/dubbo-go/registry"
)

// EventPublishingServiceDiscovery will enhance Service Discovery
// Publish some event about service discovery
type EventPublishingServiceDiscovery struct {
	serviceDiscovery *registry.ServiceDiscovery
}

// NewEventPublishingServiceDiscovery is a constructor
func NewEventPublishingServiceDiscovery(serviceDiscovery *registry.ServiceDiscovery) *EventPublishingServiceDiscovery {
	return &EventPublishingServiceDiscovery{serviceDiscovery: serviceDiscovery}
}

func (epsd *EventPublishingServiceDiscovery) String() string {
	panic("implement me")
}

func (epsd *EventPublishingServiceDiscovery) Destroy() error {
	panic("implement me")
}

func (epsd *EventPublishingServiceDiscovery) Register(instance registry.ServiceInstance) error {
	panic("implement me")
}

func (epsd *EventPublishingServiceDiscovery) Update(instance registry.ServiceInstance) error {
	panic("implement me")
}

func (epsd *EventPublishingServiceDiscovery) Unregister(instance registry.ServiceInstance) error {
	panic("implement me")
}

func (epsd *EventPublishingServiceDiscovery) GetDefaultPageSize() int {
	panic("implement me")
}

func (epsd *EventPublishingServiceDiscovery) GetServices() *gxset.HashSet {
	panic("implement me")
}

func (epsd *EventPublishingServiceDiscovery) GetInstances(serviceName string) []registry.ServiceInstance {
	panic("implement me")
}

func (epsd *EventPublishingServiceDiscovery) GetInstancesByPage(serviceName string, offset int, pageSize int) gxpage.Pager {
	panic("implement me")
}

func (epsd *EventPublishingServiceDiscovery) GetHealthyInstancesByPage(serviceName string, offset int, pageSize int, healthy bool) gxpage.Pager {
	panic("implement me")
}

func (epsd *EventPublishingServiceDiscovery) GetRequestInstances(serviceNames []string, offset int, requestedSize int) map[string]gxpage.Pager {
	panic("implement me")
}

func (epsd *EventPublishingServiceDiscovery) AddListener(listener *registry.ServiceInstancesChangedListener) error {
	panic("implement me")
}

func (epsd *EventPublishingServiceDiscovery) DispatchEventByServiceName(serviceName string) error {
	panic("implement me")
}

func (epsd *EventPublishingServiceDiscovery) DispatchEventForInstances(serviceName string, instances []registry.ServiceInstance) error {
	panic("implement me")
}

func (epsd *EventPublishingServiceDiscovery) DispatchEvent(event *registry.ServiceInstancesChangedEvent) error {
	panic("implement me")
}
