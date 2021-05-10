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

package dispatcher

import (
	"fmt"
	"reflect"
	"testing"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/observer"
)

func TestDirectEventDispatcher_Dispatch(t *testing.T) {
	ded := NewDirectEventDispatcher()
	ded.AddEventListener(&TestEventListener{
		BaseListener: observer.NewBaseListener(),
	})
	ded.AddEventListener(&TestEventListener1{})
	ded.Dispatch(&TestEvent{})
	ded.Dispatch(nil)
}

type TestEvent struct {
	observer.BaseEvent
}

type TestEventListener struct {
	observer.BaseListener
	observer.EventListener
}

func (tel *TestEventListener) OnEvent(e observer.Event) error {
	fmt.Println("TestEventListener")
	return nil
}

func (tel *TestEventListener) GetPriority() int {
	return -1
}

func (tel *TestEventListener) GetEventType() reflect.Type {
	return reflect.TypeOf(&TestEvent{})
}

type TestEventListener1 struct {
	observer.EventListener
}

func (tel *TestEventListener1) OnEvent(e observer.Event) error {
	fmt.Println("TestEventListener1")
	return nil
}

func (tel *TestEventListener1) GetPriority() int {
	return 1
}

func (tel *TestEventListener1) GetEventType() reflect.Type {
	return reflect.TypeOf(TestEvent{})
}
