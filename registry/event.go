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
	"math/rand"
	"time"
)

import (
	"github.com/dubbogo/gost/gof/observer"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

type KeyFunc func(*common.URL) string

func init() {
	rand.Seed(time.Now().UnixNano())
}

// ServiceEvent includes create, update, delete event
type ServiceEvent struct {
	Action  remoting.EventType
	Service *common.URL
	key     string // store the key for Service.Key()
	updated bool   // If the url is updated, such as Merged.
	KeyFunc KeyFunc
}

// String return the description of event
func (e *ServiceEvent) String() string {
	return fmt.Sprintf("ServiceEvent{Action{%s}, Path{%s}, Key{%s}}", e.Action, e.Service, e.key)
}

// Update updates the url with the merged URL. Work with Updated() can reduce the process
// of some merging URL.
func (e *ServiceEvent) Update(url *common.URL) {
	e.Service = url
	e.updated = true
}

// Updated checks if the url is updated. If the serviceEvent is updated, then it don't need
// merge url again.
func (e *ServiceEvent) Updated() bool {
	return e.updated
}

// Key generates the key for service.Key(). It is cached once.
func (e *ServiceEvent) Key() string {
	if len(e.key) > 0 {
		return e.key
	}
	if e.KeyFunc == nil {
		e.key = e.Service.GetCacheInvokerMapKey()
	} else {
		e.key = e.KeyFunc(e.Service)
	}
	return e.key
}

// ServiceInstancesChangedEvent represents service instances make some changing
type ServiceInstancesChangedEvent struct {
	observer.BaseEvent
	ServiceName string
	Instances   []ServiceInstance
}

// String return the description of the event
func (s *ServiceInstancesChangedEvent) String() string {
	return fmt.Sprintf("ServiceInstancesChangedEvent[source=%s]", s.ServiceName)
}

// NewServiceInstancesChangedEvent will create the ServiceInstanceChangedEvent instance
func NewServiceInstancesChangedEvent(serviceName string, instances []ServiceInstance) *ServiceInstancesChangedEvent {
	return &ServiceInstancesChangedEvent{
		BaseEvent: observer.BaseEvent{
			Source:    serviceName,
			Timestamp: time.Now(),
		},
		ServiceName: serviceName,
		Instances:   instances,
	}
}
