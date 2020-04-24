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

type ServiceInstancesChangedListener struct {
	ServiceName string
	observer.ConditionalEventListener
	ChangedNotify ChangedNotify
}

func (sicl *ServiceInstancesChangedListener) OnEvent(e ServiceInstancesChangedEvent) error {
	sicl.ChangedNotify.Notify(e)
	return nil
}

func (sicl *ServiceInstancesChangedListener) GetPriority() int {
	return -1
}

func (sicl *ServiceInstancesChangedListener) GetEventType() reflect.Type {
	return reflect.TypeOf(&ServiceInstancesChangedEvent{})
}

func (sicl *ServiceInstancesChangedListener) Accept(e ServiceInstancesChangedEvent) bool {
	return e.ServiceName == sicl.ServiceName
}

type ChangedNotify interface {
	Notify(e ServiceInstancesChangedEvent)
}
