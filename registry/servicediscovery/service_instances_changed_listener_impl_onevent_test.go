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

package servicediscovery

import (
	"testing"
	"time"
)

import (
	gxset "github.com/dubbogo/gost/container/set"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

// noopNotifyListener is a registry.NotifyListener that does nothing.
type noopNotifyListener struct{}

func (noopNotifyListener) Notify(*registry.ServiceEvent)              {}
func (noopNotifyListener) NotifyAll([]*registry.ServiceEvent, func()) {}

// reentrantNotifyListener's NotifyAll re-enters lstn.AddListenerAndNotify,
// which acquires lstn.mutex. Before #3527, OnEvent held lstn.mutex across the
// NotifyAll dispatch, so this self-deadlocked (sync.Mutex is not reentrant).
type reentrantNotifyListener struct {
	lstn      *ServiceInstancesChangedListenerImpl
	triggered chan struct{}
}

func (r *reentrantNotifyListener) Notify(*registry.ServiceEvent) {}
func (r *reentrantNotifyListener) NotifyAll(_ []*registry.ServiceEvent, _ func()) {
	// Re-enter the same listener under the same mutex that OnEvent used to hold.
	r.lstn.AddListenerAndNotify("reentrant-key", noopNotifyListener{})
	close(r.triggered)
}

// TestOnEvent_NoDeadlockOnReentrantNotify verifies that a NotifyAll callback
// re-entering AddListenerAndNotify on the same listener no longer self-deadlocks.
// Regression for #3527 (OnEvent held lstn.mutex across the external callback).
func TestOnEvent_NoDeadlockOnReentrantNotify(t *testing.T) {
	lstn := &ServiceInstancesChangedListenerImpl{
		app:                "test-app",
		registryId:         "test-reg",
		serviceNames:       gxset.NewSet("svc"),
		listeners:          make(map[string]registry.NotifyListener),
		serviceUrls:        make(map[string][]*common.URL),
		revisionToMetadata: make(map[string]*info.MetadataInfo),
		allInstances:       make(map[string][]registry.ServiceInstance),
	}
	triggered := make(chan struct{})
	lstn.listeners["k"] = &reentrantNotifyListener{lstn: lstn, triggered: triggered}

	// An event with no instances exercises the notify dispatch without needing
	// the metadata cache; each listener is notified with an empty event list.
	done := make(chan error, 1)
	go func() { done <- lstn.OnEvent(registry.NewServiceInstancesChangedEvent("svc", nil)) }()

	select {
	case <-triggered:
		// NotifyAll ran and re-entered AddListenerAndNotify without deadlocking.
	case <-time.After(2 * time.Second):
		t.Fatal("OnEvent deadlocked: NotifyAll callback could not re-enter AddListenerAndNotify")
	}
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("OnEvent did not return")
	}

	// the re-entrant registration must have taken effect
	assert.Contains(t, lstn.listeners, "reentrant-key")
}
