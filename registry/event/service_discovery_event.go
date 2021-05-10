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

// ServiceDiscoveryEvent means that something happens to service discovery instance
type ServiceDiscoveryEvent struct {
	observer.BaseEvent
	original registry.ServiceDiscovery
}

// NewServiceDiscoveryEvent returns an instance
func NewServiceDiscoveryEvent(discovery registry.ServiceDiscovery, original registry.ServiceDiscovery) *ServiceDiscoveryEvent {
	return &ServiceDiscoveryEvent{
		BaseEvent: *observer.NewBaseEvent(discovery),
		original:  original,
	}
}

// GetServiceDiscovery returns the event source
func (sde *ServiceDiscoveryEvent) GetServiceDiscovery() registry.ServiceDiscovery {
	return sde.GetSource().(registry.ServiceDiscovery)
}

// GetOriginal actually I think we can remove this method.
func (sde *ServiceDiscoveryEvent) GetOriginal() registry.ServiceDiscovery {
	return sde.original
}

// ServiceDiscoveryDestroyingEvent
// this event will be dispatched before service discovery be destroyed
type ServiceDiscoveryDestroyingEvent struct {
	ServiceDiscoveryEvent
}

// ServiceDiscoveryExceptionEvent
// this event will be dispatched when the error occur in service discovery
type ServiceDiscoveryExceptionEvent struct {
	ServiceDiscoveryEvent
	err error
}

// ServiceDiscoveryInitializedEvent
// this event will be dispatched after service discovery initialize
type ServiceDiscoveryInitializedEvent struct {
	ServiceDiscoveryEvent
}

// ServiceDiscoveryInitializingEvent
// this event will be dispatched before service discovery initialize
type ServiceDiscoveryInitializingEvent struct {
	ServiceDiscoveryEvent
}

// ServiceDiscoveryDestroyedEvent
// this event will be dispatched after service discovery be destroyed
type ServiceDiscoveryDestroyedEvent struct {
	ServiceDiscoveryEvent
}

// NewServiceDiscoveryDestroyingEvent create a ServiceDiscoveryDestroyingEvent
func NewServiceDiscoveryDestroyingEvent(discovery registry.ServiceDiscovery, original registry.ServiceDiscovery) *ServiceDiscoveryDestroyingEvent {
	return &ServiceDiscoveryDestroyingEvent{*NewServiceDiscoveryEvent(discovery, original)}
}

// NewServiceDiscoveryExceptionEvent create a ServiceDiscoveryExceptionEvent
func NewServiceDiscoveryExceptionEvent(discovery registry.ServiceDiscovery, original registry.ServiceDiscovery, err error) *ServiceDiscoveryExceptionEvent {
	return &ServiceDiscoveryExceptionEvent{*NewServiceDiscoveryEvent(discovery, original), err}
}

// NewServiceDiscoveryInitializedEvent create a ServiceDiscoveryInitializedEvent
func NewServiceDiscoveryInitializedEvent(discovery registry.ServiceDiscovery, original registry.ServiceDiscovery) *ServiceDiscoveryInitializedEvent {
	return &ServiceDiscoveryInitializedEvent{*NewServiceDiscoveryEvent(discovery, original)}
}

// NewServiceDiscoveryInitializingEvent create a ServiceDiscoveryInitializingEvent
func NewServiceDiscoveryInitializingEvent(discovery registry.ServiceDiscovery, original registry.ServiceDiscovery) *ServiceDiscoveryInitializingEvent {
	return &ServiceDiscoveryInitializingEvent{*NewServiceDiscoveryEvent(discovery, original)}
}

// NewServiceDiscoveryDestroyedEvent create a ServiceDiscoveryDestroyedEvent
func NewServiceDiscoveryDestroyedEvent(discovery registry.ServiceDiscovery, original registry.ServiceDiscovery) *ServiceDiscoveryDestroyedEvent {
	return &ServiceDiscoveryDestroyedEvent{*NewServiceDiscoveryEvent(discovery, original)}
}
