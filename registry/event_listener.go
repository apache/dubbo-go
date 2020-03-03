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
	gxsort "github.com/dubbogo/gost/sort"
)

// EventListener is an new interface used to align with dubbo 2.7.5
// It contains the Prioritized means that the listener has its priority
type EventListener interface {
	gxsort.Prioritizer
	// OnEvent handle this event
	OnEvent(e Event) error
}

// ConditionalEventListener only handle the event which it can handle
type ConditionalEventListener interface {
	EventListener
	// Accept will make the decision whether it should handle this event
	Accept(e Event) bool
}

// TODO (implement ConditionalEventListener)
type ServiceInstancesChangedListener struct {
}
