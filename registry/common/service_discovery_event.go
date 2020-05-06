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
	"github.com/apache/dubbo-go/common/observer"
	"github.com/apache/dubbo-go/registry"
)

type ServiceDiscoveryEvent struct {
	observer.BaseEvent
	original registry.ServiceDiscovery
	err      error
}

func NewServiceDiscoveryEventWithoutError(discovery registry.ServiceDiscovery, original registry.ServiceDiscovery) *ServiceDiscoveryEvent {
	return &ServiceDiscoveryEvent{
		BaseEvent: *observer.NewBaseEvent(discovery),
		original:  original,
	}
}

func NewServiceDiscoveryEvent(discovery registry.ServiceDiscovery, original registry.ServiceDiscovery, err error) *ServiceDiscoveryEvent {
	return &ServiceDiscoveryEvent{
		BaseEvent: *observer.NewBaseEvent(discovery),
		original:  original,
		err:       err,
	}
}

func (sde *ServiceDiscoveryEvent) GetServiceDiscovery() registry.ServiceDiscovery {
	return sde.GetSource().(registry.ServiceDiscovery)
}

func (sde *ServiceDiscoveryEvent) GetOriginal() registry.ServiceDiscovery {
	return sde.original
}

type ServiceDiscoveryDestroyingEvent ServiceDiscoveryEvent

type ServiceDiscoveryExceptionEvent ServiceDiscoveryEvent

type ServiceDiscoveryInitializedEvent ServiceDiscoveryEvent

type ServiceDiscoveryInitializingEvent ServiceDiscoveryEvent

type ServiceDiscoveryDestroyedEvent ServiceDiscoveryEvent

// NewServiceDiscoveryDestroyingEvent create a ServiceDiscoveryDestroyingEvent
func NewServiceDiscoveryDestroyingEvent(discovery registry.ServiceDiscovery, original registry.ServiceDiscovery) *ServiceDiscoveryDestroyingEvent {
	return (*ServiceDiscoveryDestroyingEvent)(NewServiceDiscoveryEventWithoutError(discovery, original))
}

// NewServiceDiscoveryExceptionEvent create a ServiceDiscoveryExceptionEvent
func NewServiceDiscoveryExceptionEvent(discovery registry.ServiceDiscovery, original registry.ServiceDiscovery, err error) *ServiceDiscoveryExceptionEvent {
	return (*ServiceDiscoveryExceptionEvent)(NewServiceDiscoveryEvent(discovery, original, err))
}

// NewServiceDiscoveryInitializedEvent create a ServiceDiscoveryInitializedEvent
func NewServiceDiscoveryInitializedEvent(discovery registry.ServiceDiscovery, original registry.ServiceDiscovery) *ServiceDiscoveryInitializedEvent {
	return (*ServiceDiscoveryInitializedEvent)(NewServiceDiscoveryEventWithoutError(discovery, original))
}

// NewServiceDiscoveryInitializingEvent create a ServiceDiscoveryInitializingEvent
func NewServiceDiscoveryInitializingEvent(discovery registry.ServiceDiscovery, original registry.ServiceDiscovery) *ServiceDiscoveryInitializingEvent {
	return (*ServiceDiscoveryInitializingEvent)(NewServiceDiscoveryEventWithoutError(discovery, original))
}

// NewServiceDiscoveryDestroyedEvent create a ServiceDiscoveryDestroyedEvent
func NewServiceDiscoveryDestroyedEvent(discovery registry.ServiceDiscovery, original registry.ServiceDiscovery) *ServiceDiscoveryDestroyedEvent {
	return (*ServiceDiscoveryDestroyedEvent)(NewServiceDiscoveryEventWithoutError(discovery, original))
}
