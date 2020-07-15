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
	"github.com/apache/dubbo-go/common/observer"
)

// The Service Discovery Changed  Event Listener
type ServiceInstancesChangedListener struct {
	ServiceName   string
	ChangedNotify observer.ChangedNotify
}

// OnEvent on ServiceInstancesChangedEvent the service instances change event
func (lstn *ServiceInstancesChangedListener) OnEvent(e observer.Event) error {
	lstn.ChangedNotify.Notify(e)
	return nil
}

// Accept return true if the name is the same
func (lstn *ServiceInstancesChangedListener) Accept(e observer.Event) bool {
	if ce, ok := e.(*ServiceInstancesChangedEvent); ok {
		return ce.ServiceName == lstn.ServiceName
	}
	return false
}

// GetPriority returns -1, it will be the first invoked listener
func (lstn *ServiceInstancesChangedListener) GetPriority() int {
	return -1
}

// GetEventType returns ServiceInstancesChangedEvent
func (lstn *ServiceInstancesChangedListener) GetEventType() reflect.Type {
	return reflect.TypeOf(&ServiceInstancesChangedEvent{})
}
