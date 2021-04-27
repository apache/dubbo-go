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

package registry

import (
	"fmt"
)

import (
	gxset "github.com/dubbogo/gost/container/set"
	gxpage "github.com/dubbogo/gost/hash/page"
)

const DefaultPageSize = 100

// ServiceDiscovery is the common operations of Service Discovery
type ServiceDiscovery interface {
	fmt.Stringer

	// ----------------- lifecycle -------------------

	// Destroy will destroy the service discovery.
	// If the discovery cannot be destroy, it will return an error.
	Destroy() error

	// ----------------- registration ----------------

	// Register will register an instance of ServiceInstance to registry
	Register(instance ServiceInstance) error

	// Update will update the data of the instance in registry
	Update(instance ServiceInstance) error

	// Unregister will unregister this instance from registry
	Unregister(instance ServiceInstance) error

	// ----------------- discovery -------------------
	// GetDefaultPageSize will return the default page size
	GetDefaultPageSize() int

	// GetServices will return the all service names.
	GetServices() *gxset.HashSet

	// GetInstances will return all service instances with serviceName
	GetInstances(serviceName string) []ServiceInstance

	// GetInstancesByPage will return a page containing instances of ServiceInstance with the serviceName
	// the page will start at offset
	GetInstancesByPage(serviceName string, offset int, pageSize int) gxpage.Pager

	// GetHealthyInstancesByPage will return a page containing instances of ServiceInstance.
	// The param healthy indices that the instance should be healthy or not.
	// The page will start at offset
	GetHealthyInstancesByPage(serviceName string, offset int, pageSize int, healthy bool) gxpage.Pager

	// Batch get all instances by the specified service names
	GetRequestInstances(serviceNames []string, offset int, requestedSize int) map[string]gxpage.Pager

	// ----------------- event ----------------------
	// AddListener adds a new ServiceInstancesChangedListenerImpl
	// see addServiceInstancesChangedListener in Java
	AddListener(listener ServiceInstancesChangedListener) error

	// DispatchEventByServiceName dispatches the ServiceInstancesChangedEvent to service instance whose name is serviceName
	DispatchEventByServiceName(serviceName string) error

	// DispatchEventForInstances dispatches the ServiceInstancesChangedEvent to target instances
	DispatchEventForInstances(serviceName string, instances []ServiceInstance) error

	// DispatchEvent dispatches the event
	DispatchEvent(event *ServiceInstancesChangedEvent) error
}
