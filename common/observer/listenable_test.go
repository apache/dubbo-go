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
	"reflect"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestListenable(t *testing.T) {
	el := &TestEventListener{}
	bl := NewBaseListener()
	b := &bl
	b.AddEventListener(el)
	b.AddEventListener(el)
	al := b.GetAllEventListeners()
	assert.Equal(t, len(al), 1)
	assert.Equal(t, al[0].GetEventType(), reflect.TypeOf(TestEvent{}))
	b.RemoveEventListener(el)
	assert.Equal(t, len(b.GetAllEventListeners()), 0)
	var ts []EventListener
	ts = append(ts, el)
	b.AddEventListeners(ts)
	assert.Equal(t, len(al), 1)

}

type TestEvent struct {
	BaseEvent
}

type TestEventListener struct {
	EventListener
}

func (tel *TestEventListener) OnEvent(e Event) error {
	return nil
}

func (tel *TestEventListener) GetPriority() int {
	return -1
}

func (tel *TestEventListener) GetEventType() reflect.Type {
	return reflect.TypeOf(TestEvent{})
}
