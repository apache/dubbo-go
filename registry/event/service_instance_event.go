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
	"dubbo.apache.org/dubbo-go/v3/common/observer"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

// ServiceInstanceEvent means something happen to this ServiceInstance
// like register this service instance
type ServiceInstanceEvent struct {
	observer.BaseEvent
	serviceInstance registry.ServiceInstance
}

// NewServiceInstanceEvent create a ServiceInstanceEvent
func NewServiceInstanceEvent(source interface{}, instance registry.ServiceInstance) *ServiceInstanceEvent {
	return &ServiceInstanceEvent{
		BaseEvent:       *observer.NewBaseEvent(source),
		serviceInstance: instance,
	}
}

// getServiceInstance return the service instance
func (sie *ServiceInstanceEvent) getServiceInstance() registry.ServiceInstance {
	return sie.serviceInstance
}

// ServiceInstancePreRegisteredEvent
// this event will be dispatched before service instance be registered
type ServiceInstancePreRegisteredEvent struct {
	ServiceInstanceEvent
}

// ServiceInstancePreUnregisteredEvent
// this event will be dispatched before service instance be unregistered
type ServiceInstancePreUnregisteredEvent struct {
	ServiceInstanceEvent
}

// ServiceInstanceRegisteredEvent
// this event will be dispatched after service instance be registered
type ServiceInstanceRegisteredEvent struct {
	ServiceInstanceEvent
}

// ServiceInstanceRegisteredEvent
// this event will be dispatched after service instance be unregistered
type ServiceInstanceUnregisteredEvent struct {
	ServiceInstanceEvent
}

// NewServiceInstancePreRegisteredEvent create a ServiceInstancePreRegisteredEvent
func NewServiceInstancePreRegisteredEvent(source interface{}, instance registry.ServiceInstance) *ServiceInstancePreRegisteredEvent {
	return &ServiceInstancePreRegisteredEvent{*NewServiceInstanceEvent(source, instance)}
}

// NewServiceInstancePreUnregisteredEvent create a ServiceInstancePreUnregisteredEvent
func NewServiceInstancePreUnregisteredEvent(source interface{}, instance registry.ServiceInstance) *ServiceInstancePreUnregisteredEvent {
	return &ServiceInstancePreUnregisteredEvent{*NewServiceInstanceEvent(source, instance)}
}

// NewServiceInstanceRegisteredEvent create a ServiceInstanceRegisteredEvent
func NewServiceInstanceRegisteredEvent(source interface{}, instance registry.ServiceInstance) *ServiceInstanceRegisteredEvent {
	return &ServiceInstanceRegisteredEvent{*NewServiceInstanceEvent(source, instance)}
}

// NewServiceInstanceUnregisteredEvent create a ServiceInstanceUnregisteredEvent
func NewServiceInstanceUnregisteredEvent(source interface{}, instance registry.ServiceInstance) *ServiceInstanceUnregisteredEvent {
	return &ServiceInstanceUnregisteredEvent{*NewServiceInstanceEvent(source, instance)}
}
