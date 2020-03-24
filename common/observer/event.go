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

package observer

import (
	"fmt"
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// Event is align with Event interface in Java.
// it's the top abstraction
// Align with 2.7.5
type Event interface {
	fmt.Stringer
	GetSource() interface{}
	GetTimestamp() time.Time
}

// baseEvent is the base implementation of Event
// You should never use it directly
type BaseEvent struct {
	source    interface{}
	timestamp time.Time
}

// GetSource return the source
func (b *BaseEvent) GetSource() interface{} {
	return b.source
}

// GetTimestamp return the timestamp when the event is created
func (b *BaseEvent) GetTimestamp() time.Time {
	return b.timestamp
}

// String return a human readable string representing this event
func (b *BaseEvent) String() string {
	return fmt.Sprintf("baseEvent[source = %#v]", b.source)
}

func newBaseEvent(source interface{}) *BaseEvent {
	return &BaseEvent{
		source:    source,
		timestamp: time.Now(),
	}
}
