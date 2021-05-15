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
	"reflect"
	"sync"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/common/observer"
)

func init() {
	extension.AddEventListener(GetCustomizableServiceInstanceListener)
}

// customizableServiceInstanceListener is singleton
type customizableServiceInstanceListener struct{}

// GetPriority return priority 9999,
// 9999 is big enough to make sure it will be last invoked
func (c *customizableServiceInstanceListener) GetPriority() int {
	return 9999
}

// OnEvent if the event is ServiceInstancePreRegisteredEvent
// it will iterate all ServiceInstanceCustomizer instances
// or it will do nothing
func (c *customizableServiceInstanceListener) OnEvent(e observer.Event) error {
	if preRegEvent, ok := e.(*ServiceInstancePreRegisteredEvent); ok {
		for _, cus := range extension.GetCustomizers() {
			cus.Customize(preRegEvent.serviceInstance)
		}
	}
	return nil
}

// GetEventType will return ServiceInstancePreRegisteredEvent
func (c *customizableServiceInstanceListener) GetEventType() reflect.Type {
	return reflect.TypeOf(&ServiceInstancePreRegisteredEvent{})
}

var (
	customizableServiceInstanceListenerInstance *customizableServiceInstanceListener
	customizableServiceInstanceListenerOnce     sync.Once
)

// GetCustomizableServiceInstanceListener returns an instance
// if the instance was not initialized, we create one
func GetCustomizableServiceInstanceListener() observer.EventListener {
	customizableServiceInstanceListenerOnce.Do(func() {
		customizableServiceInstanceListenerInstance = &customizableServiceInstanceListener{}
	})
	return customizableServiceInstanceListenerInstance
}
