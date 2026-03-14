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

package graceful_shutdown

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

type testClosingInstanceRemover struct {
	removedInstanceKeys []string
	result              bool
}

func (r *testClosingInstanceRemover) RemoveClosingInstance(instanceKey string) bool {
	r.removedInstanceKeys = append(r.removedInstanceKeys, instanceKey)
	return r.result
}

func TestClosingDirectoryRegistryRegisterFindAndUnregister(t *testing.T) {
	registry := newClosingDirectoryRegistry()
	remover1 := &testClosingInstanceRemover{result: true}
	remover2 := &testClosingInstanceRemover{result: true}

	registry.Register("org.apache.dubbo-go.TestService:1.0.0", remover1)
	registry.Register("org.apache.dubbo-go.TestService:1.0.0", remover2)

	removers := registry.Find("org.apache.dubbo-go.TestService:1.0.0")
	assert.Len(t, removers, 2)

	registry.Unregister("org.apache.dubbo-go.TestService:1.0.0", remover1)
	removers = registry.Find("org.apache.dubbo-go.TestService:1.0.0")
	assert.Len(t, removers, 1)

	registry.Unregister("org.apache.dubbo-go.TestService:1.0.0", remover2)
	assert.Empty(t, registry.Find("org.apache.dubbo-go.TestService:1.0.0"))
}

func TestClosingEventHandlerDispatchesByServiceKey(t *testing.T) {
	registry := newClosingDirectoryRegistry()
	handler := &closingEventHandler{registry: registry}

	targetRemover := &testClosingInstanceRemover{result: true}
	otherRemover := &testClosingInstanceRemover{result: true}

	registry.Register("org.apache.dubbo-go.TargetService:1.0.0", targetRemover)
	registry.Register("org.apache.dubbo-go.OtherService:1.0.0", otherRemover)

	removed := handler.HandleClosingEvent(ClosingEvent{
		ServiceKey:  "org.apache.dubbo-go.TargetService:1.0.0",
		InstanceKey: "target-instance",
	})

	assert.True(t, removed)
	assert.Equal(t, []string{"target-instance"}, targetRemover.removedInstanceKeys)
	assert.Empty(t, otherRemover.removedInstanceKeys)
}

func TestClosingEventHandlerRejectsIncompleteEvent(t *testing.T) {
	registry := newClosingDirectoryRegistry()
	handler := &closingEventHandler{registry: registry}
	defaultClosingAckTracker.reset()

	assert.False(t, handler.HandleClosingEvent(ClosingEvent{}))
	assert.False(t, handler.HandleClosingEvent(ClosingEvent{ServiceKey: "svc"}))
	assert.False(t, handler.HandleClosingEvent(ClosingEvent{InstanceKey: "instance"}))
}

func TestClosingEventHandlerRecordsActiveAckStats(t *testing.T) {
	registry := newClosingDirectoryRegistry()
	handler := &closingEventHandler{registry: registry}
	defaultClosingAckTracker.reset()

	remover := &testClosingInstanceRemover{result: true}
	registry.Register("org.apache.dubbo-go.TargetService:1.0.0", remover)

	assert.True(t, handler.HandleClosingEvent(ClosingEvent{
		Source:      "grpc-health-watch",
		ServiceKey:  "org.apache.dubbo-go.TargetService:1.0.0",
		InstanceKey: "target-instance",
		Address:     "127.0.0.1:20000",
	}))

	stats := DefaultClosingAckStats()
	assert.Equal(t, ClosingAckStats{
		Received: 1,
		Removed:  1,
		Missed:   0,
	}, stats["grpc-health-watch"])
}

func TestClosingEventHandlerRecordsActiveAckMisses(t *testing.T) {
	registry := newClosingDirectoryRegistry()
	handler := &closingEventHandler{registry: registry}
	defaultClosingAckTracker.reset()

	assert.False(t, handler.HandleClosingEvent(ClosingEvent{
		Source:      "triple-health-watch",
		ServiceKey:  "org.apache.dubbo-go.TargetService:1.0.0",
		InstanceKey: "missing-instance",
		Address:     "127.0.0.1:20000",
	}))

	stats := DefaultClosingAckStats()
	assert.Equal(t, ClosingAckStats{
		Received: 1,
		Removed:  0,
		Missed:   1,
	}, stats["triple-health-watch"])
}
