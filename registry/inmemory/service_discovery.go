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

package inmemory

import (
	"github.com/dubbogo/gost/container/set"
	"github.com/dubbogo/gost/page"

	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/registry"
)

const (
	name = "in-memory"
)

func init() {

	instance := &InMemoryServiceDiscovery{
		instances: make(map[string]registry.ServiceInstance, 4),
	}

	extension.SetServiceDiscovery(name, func(url *common.URL) (discovery registry.ServiceDiscovery, err error) {
		return instance, nil
	})
}

// InMemoryServiceDiscovery is an implementation based on memory.
// Usually you will not use this implementation except for tests.
type InMemoryServiceDiscovery struct {
	instances map[string]registry.ServiceInstance
}

func (i *InMemoryServiceDiscovery) String() string {
	return name
}

// Destroy doesn't destroy the instance, it just clear the instances
func (i *InMemoryServiceDiscovery) Destroy() error {
	// reset to empty
	i.instances = make(map[string]registry.ServiceInstance, 4)
	return nil
}

// Register will store the instance using its id as key
func (i *InMemoryServiceDiscovery) Register(instance registry.ServiceInstance) error {
	i.instances[instance.GetId()] = instance
	return nil
}

// Update will act like register
func (i *InMemoryServiceDiscovery) Update(instance registry.ServiceInstance) error {
	return i.Register(instance)
}

// Unregister will remove the instance
func (i *InMemoryServiceDiscovery) Unregister(instance registry.ServiceInstance) error {
	delete(i.instances, instance.GetId())
	return nil
}

// GetDefaultPageSize will return the default page size
func (i *InMemoryServiceDiscovery) GetDefaultPageSize() int {
	return registry.DefaultPageSize
}

// GetServices will return all service names
func (i *InMemoryServiceDiscovery) GetServices() *gxset.HashSet {
	result := gxset.NewSet()
	for _, value := range i.instances {
		result.Add(value.GetServiceName())
	}
	return result
}

// GetInstances will find out all instances with serviceName
func (i *InMemoryServiceDiscovery) GetInstances(serviceName string) []registry.ServiceInstance {
	result := make([]registry.ServiceInstance, 0, len(i.instances))
	for _, value := range i.instances {
		if value.GetServiceName() == serviceName {
			result = append(result, value)
		}
	}
	return result
}

//
func (i *InMemoryServiceDiscovery) GetInstancesByPage(serviceName string, offset int, pageSize int) gxpage.Pager {
	panic("implement me")
}

func (i *InMemoryServiceDiscovery) GetHealthyInstancesByPage(serviceName string, offset int, pageSize int, healthy bool) gxpage.Pager {
	panic("implement me")
}

func (i *InMemoryServiceDiscovery) GetRequestInstances(serviceNames []string, offset int, requestedSize int) map[string]gxpage.Pager {
	panic("implement me")
}

func (i *InMemoryServiceDiscovery) AddListener(listener *registry.ServiceInstancesChangedListener) error {
	panic("implement me")
}

func (i *InMemoryServiceDiscovery) DispatchEventByServiceName(serviceName string) error {
	panic("implement me")
}

func (i *InMemoryServiceDiscovery) DispatchEventForInstances(serviceName string, instances []registry.ServiceInstance) error {
	panic("implement me")
}

func (i *InMemoryServiceDiscovery) DispatchEvent(event *registry.ServiceInstancesChangedEvent) error {
	panic("implement me")
}
