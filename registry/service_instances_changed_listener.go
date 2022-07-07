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
	"reflect"
)

import (
	gxset "github.com/dubbogo/gost/container/set"
	"github.com/dubbogo/gost/gof/observer"
)

// ServiceInstancesChangedListener is the interface of the Service Discovery Changed Event Listener
type ServiceInstancesChangedListener interface {
	// OnEvent on ServiceInstancesChangedEvent the service instances change event
	OnEvent(e observer.Event) error
	// AddListenerAndNotify add notify listener and notify to listen service event
	AddListenerAndNotify(serviceKey string, notify NotifyListener)
	// RemoveListener remove notify listener
	RemoveListener(serviceKey string)
	// GetServiceNames return all listener service names
	GetServiceNames() *gxset.HashSet
	// Accept return true if the name is the same
	Accept(e observer.Event) bool
	// GetEventType returns ServiceInstancesChangedEvent
	GetEventType() reflect.Type
	// GetPriority returns -1, it will be the first invoked listener
	GetPriority() int
}
