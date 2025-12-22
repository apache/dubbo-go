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

package metrics

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

var mockChan = make(chan MetricsEvent, 16)

type MockEvent struct {
	eventType string
}

func (m MockEvent) Type() string {
	if m.eventType != "" {
		return m.eventType
	}
	return "dubbo.metrics.mock"
}

func NewEmptyMockEvent() *MockEvent {
	return &MockEvent{}
}

func NewMockEventWithType(eventType string) *MockEvent {
	return &MockEvent{eventType: eventType}
}

func init() {
	Subscribe("dubbo.metrics.mock", mockChan)
	Publish(NewEmptyMockEvent())
}

func TestBusPublish(t *testing.T) {
	event := <-mockChan

	if mockEvent, ok := event.(*MockEvent); ok {
		assert.Equal(t, NewEmptyMockEvent(), mockEvent)
	}
}

func TestBusUnsubscribe(t *testing.T) {
	testChan := make(chan MetricsEvent, 1)
	eventType := "dubbo.metrics.test.unsubscribe"

	Subscribe(eventType, testChan)

	testEvent1 := NewMockEventWithType(eventType)
	Publish(testEvent1)

	select {
	case event := <-testChan:
		assert.Equal(t, testEvent1, event)
	default:
		t.Fatal("Expected to receive event before unsubscribe")
	}

	Unsubscribe(eventType)

	testEvent2 := NewMockEventWithType(eventType)
	Publish(testEvent2)

	select {
	case _, ok := <-testChan:
		if ok {
			t.Fatal("Channel should be closed after Unsubscribe")
		}
	default:
	}
}

func TestBusPublishChannelFull(t *testing.T) {
	testChan := make(chan MetricsEvent, 1)
	eventType := "dubbo.metrics.test.full"

	Subscribe(eventType, testChan)

	testEvent1 := NewMockEventWithType(eventType)
	testChan <- testEvent1

	testEvent2 := NewMockEventWithType(eventType)
	Publish(testEvent2)

	select {
	case event := <-testChan:
		assert.Equal(t, testEvent1, event)
		select {
		case <-testChan:
			t.Fatal("Expected second event to be dropped when channel is full")
		default:
		}
	default:
		t.Fatal("Expected to receive first event from channel")
	}

	Unsubscribe(eventType)
}
