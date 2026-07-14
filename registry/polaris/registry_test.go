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

package polaris

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

import (
	"github.com/golang/protobuf/ptypes/wrappers"

	api "github.com/polarismesh/polaris-go"
	"github.com/polarismesh/polaris-go/pkg/model"
	"github.com/polarismesh/polaris-go/pkg/model/local"
	"github.com/polarismesh/polaris-go/pkg/model/pb"
	namingpb "github.com/polarismesh/polaris-go/pkg/model/pb/v1"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

const (
	testPolarisNamespace   = "test"
	testPolarisRevisionKey = "revision"
)

type fakePolarisConsumer struct {
	api.ConsumerAPI
	instanceResponses [][]model.Instance
	getCalls          int
	watchInstances    []model.Instance
	watchEvents       []model.SubScribeEvent
	watchErr          error
	watchCalls        int
	watchRequest      *api.WatchServiceRequest
}

func (f *fakePolarisConsumer) GetInstances(_ *api.GetInstancesRequest) (*model.InstancesResponse, error) {
	var instances []model.Instance
	if f.getCalls < len(f.instanceResponses) {
		instances = f.instanceResponses[f.getCalls]
	}
	f.getCalls++
	return &model.InstancesResponse{Instances: copyInstances(instances)}, nil
}

func (f *fakePolarisConsumer) WatchService(request *api.WatchServiceRequest) (*model.WatchServiceResponse, error) {
	f.watchCalls++
	f.watchRequest = request
	if f.watchErr != nil {
		return nil, f.watchErr
	}
	events := make(chan model.SubScribeEvent, len(f.watchEvents))
	for _, event := range f.watchEvents {
		events <- event
	}
	close(events)
	return &model.WatchServiceResponse{
		EventChannel: events,
		GetAllInstancesResp: &model.InstancesResponse{
			Instances: copyInstances(f.watchInstances),
		},
	}, nil
}

type recordedPolarisEvent struct {
	action   remoting.EventType
	host     string
	port     string
	id       string
	revision string
}

type recordingPolarisNotifyListener struct {
	events []recordedPolarisEvent
}

type unsupportedPolarisNotifyListener []recordedPolarisEvent

type unsupportedMapPolarisNotifyListener map[string]int

type unsupportedFuncPolarisNotifyListener func()

type unsupportedChanPolarisNotifyListener chan struct{}

type unsupportedStructPolarisNotifyListener struct {
	_ []recordedPolarisEvent
}

type runtimeUnhashablePolarisNotifyListener struct {
	state any
}

type comparableValuePolarisNotifyListener struct {
	id       int
	recorder *recordingPolarisNotifyListener
}

type nilablePolarisNotifyListener struct{}

func (unsupportedPolarisNotifyListener) Notify(*registry.ServiceEvent) {}

func (unsupportedPolarisNotifyListener) NotifyAll([]*registry.ServiceEvent, func()) {}

func (unsupportedMapPolarisNotifyListener) Notify(*registry.ServiceEvent) {}

func (unsupportedMapPolarisNotifyListener) NotifyAll([]*registry.ServiceEvent, func()) {}

func (unsupportedFuncPolarisNotifyListener) Notify(*registry.ServiceEvent) {}

func (unsupportedFuncPolarisNotifyListener) NotifyAll([]*registry.ServiceEvent, func()) {}

func (unsupportedChanPolarisNotifyListener) Notify(*registry.ServiceEvent) {}

func (unsupportedChanPolarisNotifyListener) NotifyAll([]*registry.ServiceEvent, func()) {}

func (unsupportedStructPolarisNotifyListener) Notify(*registry.ServiceEvent) {}

func (unsupportedStructPolarisNotifyListener) NotifyAll([]*registry.ServiceEvent, func()) {}

func (runtimeUnhashablePolarisNotifyListener) Notify(*registry.ServiceEvent) {}

func (runtimeUnhashablePolarisNotifyListener) NotifyAll([]*registry.ServiceEvent, func()) {}

func (l comparableValuePolarisNotifyListener) Notify(event *registry.ServiceEvent) {
	l.recorder.Notify(event)
}

func (l comparableValuePolarisNotifyListener) NotifyAll(events []*registry.ServiceEvent, callback func()) {
	l.recorder.NotifyAll(events, callback)
}

func (*nilablePolarisNotifyListener) Notify(*registry.ServiceEvent) {}

func (*nilablePolarisNotifyListener) NotifyAll([]*registry.ServiceEvent, func()) {}

func (r *recordingPolarisNotifyListener) Notify(event *registry.ServiceEvent) {
	r.record(
		event.Action,
		event.Service.Ip,
		event.Service.Port,
		event.Service.GetParam(constant.PolarisInstanceID, ""),
		event.Service.GetParam(testPolarisRevisionKey, ""),
	)
}

func (r *recordingPolarisNotifyListener) NotifyAll(events []*registry.ServiceEvent, callback func()) {
	for _, event := range events {
		r.Notify(event)
	}
	if callback != nil {
		callback()
	}
}

func (r *recordingPolarisNotifyListener) recordInstances(action remoting.EventType, instances []model.Instance) {
	for _, instance := range instances {
		r.events = append(r.events, recordedPolarisEventFromInstance(action, instance))
	}
}

func (r *recordingPolarisNotifyListener) record(
	action remoting.EventType,
	host string,
	port string,
	id string,
	revision string,
) {
	r.events = append(r.events, recordedPolarisEvent{
		action: action, host: host, port: port, id: id, revision: revision,
	})
}

func TestPolarisInitialSnapshotReconciliation(t *testing.T) {
	serviceName := "com.test.InitialSnapshotService"
	instanceA := newPolarisTestInstance("instance-a", "10.0.0.1", 20001, serviceName, true)
	sameKeyA := newPolarisTestInstance("new-polaris-id", "10.0.0.1", 20001, serviceName, true)
	instanceA.GetMetadata()[testPolarisRevisionKey] = "before"
	sameKeyA.GetMetadata()[testPolarisRevisionKey] = "after"
	instanceB := newPolarisTestInstance("instance-b", "10.0.0.2", 20002, serviceName, true)
	instanceC := newPolarisTestInstance("instance-c", "10.0.0.3", 20003, serviceName, true)
	invalid := newPolarisTestInstance("invalid", "10.0.0.9", 20009, serviceName, false)

	tests := []struct {
		name           string
		loaded         []model.Instance
		current        []model.Instance
		watchEvents    []model.SubScribeEvent
		wantLoad       []recordedPolarisEvent
		wantListener   []recordedPolarisEvent
		wantInstance   model.Instance
		wantWatchCalls int
	}{
		{
			name:    "A replaced by B and incremental event is wired",
			loaded:  []model.Instance{instanceA, invalid},
			current: []model.Instance{instanceB},
			watchEvents: []model.SubScribeEvent{&model.InstanceEvent{
				AddEvent: &model.InstanceAddEvent{Instances: []model.Instance{instanceC}},
			}},
			wantLoad: []recordedPolarisEvent{
				{action: remoting.EventTypeAdd, host: "10.0.0.1", port: "20001"},
			},
			wantListener: []recordedPolarisEvent{
				{action: remoting.EventTypeDel, host: "10.0.0.1", port: "20001"},
				{action: remoting.EventTypeAdd, host: "10.0.0.2", port: "20002"},
				{action: remoting.EventTypeAdd, host: "10.0.0.3", port: "20003"},
			},
			wantWatchCalls: 1,
		},
		{
			name:     "first snapshot empty",
			loaded:   []model.Instance{instanceA},
			wantLoad: []recordedPolarisEvent{{action: remoting.EventTypeAdd, host: "10.0.0.1", port: "20001"}},
			wantListener: []recordedPolarisEvent{
				{action: remoting.EventTypeDel, host: "10.0.0.1", port: "20001"},
			},
			wantWatchCalls: 1,
		},
		{
			name:     "same InstanceKey with different Polaris ID",
			loaded:   []model.Instance{instanceA},
			current:  []model.Instance{sameKeyA},
			wantLoad: []recordedPolarisEvent{{action: remoting.EventTypeAdd, host: "10.0.0.1", port: "20001"}},
			wantListener: []recordedPolarisEvent{
				{action: remoting.EventTypeAdd, host: "10.0.0.1", port: "20001"},
			},
			wantInstance:   sameKeyA,
			wantWatchCalls: 1,
		},
		{
			name:           "invalid instance is excluded from baseline",
			loaded:         []model.Instance{invalid},
			wantWatchCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			consumer := &fakePolarisConsumer{
				instanceResponses: [][]model.Instance{tt.loaded},
				watchInstances:    tt.current,
				watchEvents:       tt.watchEvents,
			}
			pr := newTestPolarisRegistry(consumer)
			notify := &recordingPolarisNotifyListener{}
			url := newPolarisConsumerURL(serviceName)

			if err := pr.LoadSubscribeInstances(url, notify); err != nil {
				t.Fatalf("LoadSubscribeInstances() error = %v", err)
			}
			watcher := mustStoppedRegistryWatcher(t, pr, serviceName)
			listener, err := pr.createPolarisListener(serviceName, notify)
			if err != nil {
				t.Fatalf("createPolarisListener() error = %v", err)
			}
			defer closeTestPolarisListener(listener)
			if watchErr := watcher.watchOnce(); watchErr != nil {
				t.Fatalf("watchOnce() error = %v", watchErr)
			}

			assertPolarisEventsAddress(t, notify.events, tt.wantLoad)
			listenerEvents := assertPolarisListenerEvents(t, listener, tt.wantListener)
			if tt.wantInstance != nil {
				assertPolarisEventInstance(t, listenerEvents[0], remoting.EventTypeAdd, tt.wantInstance)
			}
			assertPolarisWatchRequest(t, consumer.watchRequest, watcher.subscribeParam, serviceName)
			if consumer.watchCalls != tt.wantWatchCalls {
				t.Fatalf("WatchService() calls = %d, want %d", consumer.watchCalls, tt.wantWatchCalls)
			}
		})
	}

	t.Run("WatchService error is returned", func(t *testing.T) {
		wantErr := errors.New("watch failed")
		watcher := newStoppedPolarisWatcher(t, &fakePolarisConsumer{watchErr: wantErr})
		if err := watcher.watchOnce(); !errors.Is(err, wantErr) {
			t.Fatalf("watchOnce() error = %v, want %v", err, wantErr)
		}
		if watcher.snapshotReady {
			t.Fatal("watcher snapshot is ready after WatchService error")
		}
	})
}

func TestPolarisReusedWatcherReconcilesSecondSubscriberAfterDelete(t *testing.T) {
	serviceName := "com.test.ReusedWatcherService"
	instanceA := newPolarisTestInstance("instance-a", "10.0.0.1", 20001, serviceName, true)
	consumer := &fakePolarisConsumer{
		instanceResponses: [][]model.Instance{{instanceA}},
		watchInstances:    []model.Instance{instanceA},
		watchEvents: []model.SubScribeEvent{&model.InstanceEvent{
			DeleteEvent: &model.InstanceDeleteEvent{Instances: []model.Instance{instanceA}},
		}},
	}
	pr := newTestPolarisRegistry(consumer)
	watcher := mustStoppedRegistryWatcher(t, pr, serviceName)

	firstNotify := &recordingPolarisNotifyListener{}
	firstListener, err := pr.createPolarisListener(serviceName, firstNotify)
	if err != nil {
		t.Fatalf("createPolarisListener(first) error = %v", err)
	}
	defer closeTestPolarisListener(firstListener)
	watcher.handleWatchSnapshot([]model.Instance{instanceA})
	assertPolarisListenerEvent(t, firstListener, remoting.EventTypeAdd, instanceA)

	secondNotify := &recordingPolarisNotifyListener{}
	if loadErr := pr.LoadSubscribeInstances(newPolarisConsumerURL(serviceName), secondNotify); loadErr != nil {
		t.Fatalf("LoadSubscribeInstances() error = %v", loadErr)
	}
	if watchErr := watcher.watchOnce(); watchErr != nil {
		t.Fatalf("watchOnce() error = %v", watchErr)
	}
	assertPolarisListenerEvent(t, firstListener, remoting.EventTypeAdd, instanceA)
	assertPolarisListenerEvent(t, firstListener, remoting.EventTypeDel, instanceA)

	secondListener, err := pr.createPolarisListener(serviceName, secondNotify)
	if err != nil {
		t.Fatalf("createPolarisListener(second) error = %v", err)
	}
	defer closeTestPolarisListener(secondListener)
	assertPolarisListenerEvent(t, secondListener, remoting.EventTypeDel, instanceA)
	assertNoPolarisListenerEvent(t, firstListener)

	watcher.handleWatchSnapshot(nil)
	assertNoPolarisListenerEvent(t, secondListener)
	if len(watcher.subscribers) != 2 || !watcher.subscribers[1].reconciled || watcher.subscribers[1].initialSnapshot != nil {
		t.Fatalf("second subscriber state = %#v, want reconciled with cleared baseline", watcher.subscribers[1])
	}
}

func TestPolarisInitialSubscribeInstancesAreListenerScoped(t *testing.T) {
	serviceName := "com.test.ListenerScopedBaselineService"
	instanceA := newPolarisTestInstance("instance-a", "10.0.0.1", 20001, serviceName, true)
	instanceB := newPolarisTestInstance("instance-b", "10.0.0.2", 20002, serviceName, true)
	consumer := &fakePolarisConsumer{instanceResponses: [][]model.Instance{{instanceA}, {instanceB}, nil}}
	pr := newTestPolarisRegistry(consumer)
	firstNotify := &recordingPolarisNotifyListener{}
	secondNotify := &recordingPolarisNotifyListener{}
	url := newPolarisConsumerURL(serviceName)

	if err := pr.LoadSubscribeInstances(url, firstNotify); err != nil {
		t.Fatalf("LoadSubscribeInstances(first) error = %v", err)
	}
	if err := pr.LoadSubscribeInstances(url, secondNotify); err != nil {
		t.Fatalf("LoadSubscribeInstances(second) error = %v", err)
	}
	if err := pr.LoadSubscribeInstances(url, firstNotify); err != nil {
		t.Fatalf("LoadSubscribeInstances(first empty) error = %v", err)
	}

	firstKey := mustInitialSubscribeInstancesKey(t, serviceName, firstNotify)
	if entry, instances := pr.loadInitialSubscribeInstances(firstKey); entry != nil || len(instances) != 0 {
		t.Fatalf("first pending entry = (%p, %v), want empty", entry, instances)
	}
	secondKey := mustInitialSubscribeInstancesKey(t, serviceName, secondNotify)
	secondEntry, instances := pr.loadInitialSubscribeInstances(secondKey)
	if secondEntry == nil || len(instances) != 1 || instances[0].GetInstanceKey() != instanceB.GetInstanceKey() {
		t.Fatalf("second pending entry = (%p, %v), want instance B", secondEntry, instances)
	}

	watcher := mustStoppedRegistryWatcher(t, pr, serviceName)
	watcher.handleWatchSnapshot(nil)
	listener, err := pr.createPolarisListener(serviceName, secondNotify)
	if err != nil {
		t.Fatalf("createPolarisListener() error = %v", err)
	}
	defer closeTestPolarisListener(listener)
	assertPolarisListenerEvent(t, listener, remoting.EventTypeDel, instanceB)
	if entry, _ := pr.loadInitialSubscribeInstances(secondKey); entry != nil {
		t.Fatalf("pending entry after listener success = %p, want nil", entry)
	}

	pr.storeInitialSubscribeInstances(secondKey, []model.Instance{instanceA})
	oldEntry, _ := pr.loadInitialSubscribeInstances(secondKey)
	pr.storeInitialSubscribeInstances(secondKey, []model.Instance{instanceB})
	newEntry, _ := pr.loadInitialSubscribeInstances(secondKey)
	pr.completeInitialSubscribeInstances(secondKey, oldEntry)
	remaining, remainingInstances := pr.loadInitialSubscribeInstances(secondKey)
	if remaining != newEntry || len(remainingInstances) != 1 || remainingInstances[0].GetInstanceKey() != instanceB.GetInstanceKey() {
		t.Fatalf("newer pending entry = (%p, %v), want preserved instance B", remaining, remainingInstances)
	}
}

func TestPolarisComparableValueNotifyListenersKeepIndependentBaselines(t *testing.T) {
	serviceName := "com.test.ComparableValueListenerService"
	instanceA := newPolarisTestInstance("instance-a", "10.0.0.1", 20001, serviceName, true)
	instanceB := newPolarisTestInstance("instance-b", "10.0.0.2", 20002, serviceName, true)
	consumer := &fakePolarisConsumer{instanceResponses: [][]model.Instance{{instanceA}, {instanceB}}}
	pr := newTestPolarisRegistry(consumer)
	recorder1 := &recordingPolarisNotifyListener{}
	recorder2 := &recordingPolarisNotifyListener{}
	listener1 := comparableValuePolarisNotifyListener{id: 1, recorder: recorder1}
	listener2 := comparableValuePolarisNotifyListener{id: 2, recorder: recorder2}
	url := newPolarisConsumerURL(serviceName)

	if err := pr.LoadSubscribeInstances(url, listener1); err != nil {
		t.Fatalf("LoadSubscribeInstances(listener1) error = %v", err)
	}
	if err := pr.LoadSubscribeInstances(url, listener2); err != nil {
		t.Fatalf("LoadSubscribeInstances(listener2) error = %v", err)
	}
	if len(pr.initialSubscribeInstances) != 2 {
		t.Errorf("pending entry count = %d, want 2", len(pr.initialSubscribeInstances))
	}

	key1 := mustInitialSubscribeInstancesKey(t, serviceName, listener1)
	key2 := mustInitialSubscribeInstancesKey(t, serviceName, listener2)
	entry1, pending1 := pr.loadInitialSubscribeInstances(key1)
	entry2, pending2 := pr.loadInitialSubscribeInstances(key2)
	if entry1 == nil || len(pending1) != 1 || pending1[0].GetInstanceKey() != instanceA.GetInstanceKey() {
		t.Errorf("listener1 pending entry = (%p, %v), want instance A", entry1, pending1)
	}
	if entry2 == nil || len(pending2) != 1 || pending2[0].GetInstanceKey() != instanceB.GetInstanceKey() {
		t.Errorf("listener2 pending entry = (%p, %v), want instance B", entry2, pending2)
	}

	watcher := mustStoppedRegistryWatcher(t, pr, serviceName)
	watcher.handleWatchSnapshot([]model.Instance{instanceB})
	polarisListener1, err := pr.createPolarisListener(serviceName, listener1)
	if err != nil {
		t.Fatalf("createPolarisListener(listener1) error = %v", err)
	}
	defer closeTestPolarisListener(polarisListener1)
	assertPolarisListenerEvents(t, polarisListener1, []recordedPolarisEvent{
		{action: remoting.EventTypeDel, host: "10.0.0.1", port: "20001"},
		{action: remoting.EventTypeAdd, host: "10.0.0.2", port: "20002"},
	})
	if pending, _ := pr.loadInitialSubscribeInstances(key1); pending != nil {
		t.Errorf("listener1 pending entry after listener success = %p, want nil", pending)
	}
	if pending, instances := pr.loadInitialSubscribeInstances(key2); pending != entry2 || len(instances) != 1 || instances[0].GetInstanceKey() != instanceB.GetInstanceKey() {
		t.Errorf("listener2 pending entry = (%p, %v), want preserved instance B", pending, instances)
	}

	polarisListener2, err := pr.createPolarisListener(serviceName, listener2)
	if err != nil {
		t.Fatalf("createPolarisListener(listener2) error = %v", err)
	}
	defer closeTestPolarisListener(polarisListener2)
	assertPolarisListenerEvent(t, polarisListener2, remoting.EventTypeAdd, instanceB)
	if pending, _ := pr.loadInitialSubscribeInstances(key2); pending != nil {
		t.Errorf("listener2 pending entry after listener success = %p, want nil", pending)
	}
}

func TestPolarisWatcherTracksCurrentState(t *testing.T) {
	serviceName := "com.test.CurrentStateService"
	instanceA := newPolarisTestInstance("instance-a", "10.0.0.1", 20001, serviceName, true)
	updatedA := newPolarisTestInstance("updated-a", "10.0.0.1", 20001, serviceName, true)
	instanceA.GetMetadata()[testPolarisRevisionKey] = "before"
	updatedA.GetMetadata()[testPolarisRevisionKey] = "after"
	instanceB := newPolarisTestInstance("instance-b", "10.0.0.2", 20002, serviceName, true)

	tests := []struct {
		name         string
		initial      []model.Instance
		event        *model.InstanceEvent
		wantIDs      []string
		wantEvents   []recordedPolarisEvent
		wantInstance model.Instance
	}{
		{
			name:    "ADD",
			initial: []model.Instance{instanceA},
			event: &model.InstanceEvent{AddEvent: &model.InstanceAddEvent{
				Instances: []model.Instance{instanceB},
			}},
			wantIDs:    []string{"instance-a", "instance-b"},
			wantEvents: []recordedPolarisEvent{{action: remoting.EventTypeAdd, host: "10.0.0.2", port: "20002"}},
		},
		{
			name:    "UPDATE same key",
			initial: []model.Instance{instanceA},
			event: &model.InstanceEvent{UpdateEvent: &model.InstanceUpdateEvent{
				UpdateList: []model.OneInstanceUpdate{{Before: instanceA, After: updatedA}},
			}},
			wantIDs:      []string{"updated-a"},
			wantEvents:   []recordedPolarisEvent{{action: remoting.EventTypeUpdate, host: "10.0.0.1", port: "20001"}},
			wantInstance: updatedA,
		},
		{
			name:    "UPDATE changed key replaces internally and emits only UPDATE",
			initial: []model.Instance{instanceA},
			event: &model.InstanceEvent{UpdateEvent: &model.InstanceUpdateEvent{
				UpdateList: []model.OneInstanceUpdate{{Before: instanceA, After: instanceB}},
			}},
			wantIDs:    []string{"instance-b"},
			wantEvents: []recordedPolarisEvent{{action: remoting.EventTypeUpdate, host: "10.0.0.2", port: "20002"}},
		},
		{
			name:    "DELETE",
			initial: []model.Instance{instanceA, instanceB},
			event: &model.InstanceEvent{DeleteEvent: &model.InstanceDeleteEvent{
				Instances: []model.Instance{instanceA},
			}},
			wantIDs:    []string{"instance-b"},
			wantEvents: []recordedPolarisEvent{{action: remoting.EventTypeDel, host: "10.0.0.1", port: "20001"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			watcher := newStoppedPolarisWatcher(t, nil)
			watcher.handleWatchSnapshot(tt.initial)
			notify := &recordingPolarisNotifyListener{}
			watcher.AddSubscriber(notify.recordInstances)
			watcher.handleInstanceEvent(tt.event)

			actualIDs := make([]string, 0, len(watcher.currentInstances))
			for _, instance := range watcher.currentInstances {
				actualIDs = append(actualIDs, instance.GetId())
			}
			if !reflect.DeepEqual(actualIDs, tt.wantIDs) {
				t.Fatalf("current instance IDs = %v, want %v", actualIDs, tt.wantIDs)
			}
			assertPolarisEventsAddress(t, notify.events, tt.wantEvents)
			if tt.wantInstance != nil {
				assertPolarisEventInstance(t, notify.events[0], remoting.EventTypeUpdate, tt.wantInstance)
				assertPolarisInstanceIdentity(t, watcher.currentInstances[0], tt.wantInstance)
			}
		})
	}
}

func TestPolarisWatcherClearsRemovedInstanceReferences(t *testing.T) {
	serviceName := "com.test.ClearRemovedReferencesService"
	instanceA := newPolarisTestInstance("instance-a", "10.0.0.1", 20001, serviceName, true)
	instanceB := newPolarisTestInstance("instance-b", "10.0.0.2", 20002, serviceName, true)
	instanceC := newPolarisTestInstance("instance-c", "10.0.0.3", 20003, serviceName, true)
	missing := newPolarisTestInstance("missing", "10.0.0.9", 20009, serviceName, true)

	t.Run("partial delete clears tail and preserves notification", func(t *testing.T) {
		watcher := newStoppedPolarisWatcher(t, nil)
		watcher.handleWatchSnapshot([]model.Instance{instanceA, instanceB, instanceC})
		backing := watcher.currentInstances
		notify := &recordingPolarisNotifyListener{}
		watcher.AddSubscriber(notify.recordInstances)

		watcher.handleInstanceEvent(&model.InstanceEvent{
			DeleteEvent: &model.InstanceDeleteEvent{Instances: []model.Instance{instanceB, instanceC}},
		})

		if len(watcher.currentInstances) != 1 {
			t.Fatalf("current instance count = %d, want 1", len(watcher.currentInstances))
		}
		assertPolarisInstanceIdentity(t, watcher.currentInstances[0], instanceA)
		if &watcher.currentInstances[0] != &backing[0] {
			t.Fatal("current instances did not reuse the original backing array")
		}
		for i, instance := range backing[len(watcher.currentInstances):] {
			if instance != nil {
				t.Errorf("removed backing slot %d = %v, want nil", i+len(watcher.currentInstances), instance)
			}
		}
		assertPolarisEventsAddress(t, notify.events, []recordedPolarisEvent{
			{action: remoting.EventTypeDel, host: "10.0.0.2", port: "20002"},
			{action: remoting.EventTypeDel, host: "10.0.0.3", port: "20003"},
		})
	})

	t.Run("delete all clears every slot", func(t *testing.T) {
		watcher := newStoppedPolarisWatcher(t, nil)
		watcher.handleWatchSnapshot([]model.Instance{instanceA, instanceB, instanceC})
		backing := watcher.currentInstances

		watcher.removeCurrentInstancesLocked([]model.Instance{instanceA, instanceB, instanceC})

		if len(watcher.currentInstances) != 0 {
			t.Fatalf("current instance count = %d, want 0", len(watcher.currentInstances))
		}
		for i, instance := range backing {
			if instance != nil {
				t.Errorf("removed backing slot %d = %v, want nil", i, instance)
			}
		}
	})

	t.Run("delete missing instance preserves order and storage", func(t *testing.T) {
		watcher := newStoppedPolarisWatcher(t, nil)
		watcher.handleWatchSnapshot([]model.Instance{instanceA, instanceB, instanceC})
		backing := watcher.currentInstances

		watcher.removeCurrentInstancesLocked([]model.Instance{missing})

		if len(watcher.currentInstances) != 3 {
			t.Fatalf("current instance count = %d, want 3", len(watcher.currentInstances))
		}
		for i, want := range []model.Instance{instanceA, instanceB, instanceC} {
			assertPolarisInstanceIdentity(t, watcher.currentInstances[i], want)
		}
		if &watcher.currentInstances[0] != &backing[0] {
			t.Fatal("current instances did not preserve the original backing array")
		}
	})

	t.Run("empty removal preserves current instances", func(t *testing.T) {
		watcher := newStoppedPolarisWatcher(t, nil)
		watcher.handleWatchSnapshot([]model.Instance{instanceA, instanceB})
		backing := watcher.currentInstances

		watcher.removeCurrentInstancesLocked(nil)

		if len(watcher.currentInstances) != 2 {
			t.Fatalf("current instance count = %d, want 2", len(watcher.currentInstances))
		}
		assertPolarisInstanceIdentity(t, watcher.currentInstances[0], instanceA)
		assertPolarisInstanceIdentity(t, watcher.currentInstances[1], instanceB)
		if &watcher.currentInstances[0] != &backing[0] {
			t.Fatal("current instances did not preserve the original backing array")
		}
	})

	t.Run("empty current instances is safe", func(t *testing.T) {
		watcher := newStoppedPolarisWatcher(t, nil)
		watcher.removeCurrentInstancesLocked([]model.Instance{instanceA})
		watcher.removeCurrentInstancesLocked(nil)
		if len(watcher.currentInstances) != 0 {
			t.Fatalf("current instance count = %d, want 0", len(watcher.currentInstances))
		}
	})
}

func TestPolarisApplicationSubscriberCompatibility(t *testing.T) {
	serviceName := "com.test.ApplicationSubscriberService"
	instanceA := newPolarisTestInstance("instance-a", "10.0.0.1", 20001, serviceName, true)
	instanceB := newPolarisTestInstance("instance-b", "10.0.0.2", 20002, serviceName, true)

	t.Run("public signature and before-ready snapshot", func(t *testing.T) {
		watcher := newStoppedPolarisWatcher(t, nil)
		notify := &recordingPolarisNotifyListener{}
		assertPolarisAddSubscriberSignature(watcher.AddSubscriber)
		watcher.AddSubscriber(notify.recordInstances)
		watcher.handleWatchSnapshot([]model.Instance{instanceA})
		want := []recordedPolarisEvent{{action: remoting.EventTypeAdd, host: "10.0.0.1", port: "20001"}}
		assertPolarisEventsAddress(t, notify.events, want)
	})

	t.Run("after-ready has no replay and receives increments", func(t *testing.T) {
		watcher := newStoppedPolarisWatcher(t, nil)
		watcher.handleWatchSnapshot([]model.Instance{instanceA})
		notify := &recordingPolarisNotifyListener{}
		watcher.AddSubscriber(notify.recordInstances)
		watcher.handleInstanceEvent(&model.InstanceEvent{
			AddEvent: &model.InstanceAddEvent{Instances: []model.Instance{instanceB}},
		})
		want := []recordedPolarisEvent{{action: remoting.EventTypeAdd, host: "10.0.0.2", port: "20002"}}
		assertPolarisEventsAddress(t, notify.events, want)
	})

	t.Run("NewPolarisListener wrapper", func(t *testing.T) {
		watcher := newStoppedPolarisWatcher(t, nil)
		watcher.handleWatchSnapshot([]model.Instance{instanceA})
		listener, err := NewPolarisListener(watcher)
		if err != nil {
			t.Fatalf("NewPolarisListener() error = %v", err)
		}
		defer closeTestPolarisListener(listener)
		assertNoPolarisListenerEvent(t, listener)
		watcher.handleInstanceEvent(&model.InstanceEvent{
			AddEvent: &model.InstanceAddEvent{Instances: []model.Instance{instanceB}},
		})
		assertPolarisListenerEvent(t, listener, remoting.EventTypeAdd, instanceB)
	})
}

func TestPolarisSnapshotsUseDefensiveCopies(t *testing.T) {
	serviceName := "com.test.DefensiveCopyService"
	instanceA := newPolarisTestInstance("instance-a", "10.0.0.1", 20001, serviceName, true)
	instanceB := newPolarisTestInstance("instance-b", "10.0.0.2", 20002, serviceName, true)

	t.Run("subscriber initial snapshot", func(t *testing.T) {
		watcher := newStoppedPolarisWatcher(t, nil)
		notify := &recordingPolarisNotifyListener{}
		initial := []model.Instance{instanceA}
		watcher.addSubscriberWithInitialSnapshot(initial, notify.recordInstances)
		initial[0] = instanceB
		watcher.handleWatchSnapshot(nil)
		want := []recordedPolarisEvent{{action: remoting.EventTypeDel, host: "10.0.0.1", port: "20001"}}
		assertPolarisEventsAddress(t, notify.events, want)
	})

	t.Run("watcher current state and callback slices", func(t *testing.T) {
		watcher := newStoppedPolarisWatcher(t, nil)
		watcher.addSubscriberWithInitialSnapshot(nil, func(_ remoting.EventType, instances []model.Instance) {
			if len(instances) > 0 {
				instances[0] = instanceB
			}
		})
		current := []model.Instance{instanceA}
		watcher.handleWatchSnapshot(current)
		current[0] = instanceB

		notify := &recordingPolarisNotifyListener{}
		watcher.addSubscriberWithInitialSnapshot(nil, notify.recordInstances)
		want := []recordedPolarisEvent{{action: remoting.EventTypeAdd, host: "10.0.0.1", port: "20001"}}
		assertPolarisEventsAddress(t, notify.events, want)
	})

	t.Run("pending registry entry", func(t *testing.T) {
		pr := newTestPolarisRegistry(nil)
		key := mustInitialSubscribeInstancesKey(t, serviceName, &recordingPolarisNotifyListener{})
		input := []model.Instance{instanceA}
		pr.storeInitialSubscribeInstances(key, input)
		input[0] = instanceB

		entry, pending := pr.loadInitialSubscribeInstances(key)
		if entry == nil || len(pending) != 1 {
			t.Fatalf("pending entry = (%p, %v), want one instance", entry, pending)
		}
		assertPolarisInstanceIdentity(t, pending[0], instanceA)
	})
}

func TestPolarisRejectsInvalidNotifyListenerIdentities(t *testing.T) {
	serviceName := "com.test.InvalidNotifyListenerService"
	instanceA := newPolarisTestInstance("instance-a", "10.0.0.1", 20001, serviceName, true)
	var typedNil *nilablePolarisNotifyListener
	var nilSlice unsupportedPolarisNotifyListener
	var nilMap unsupportedMapPolarisNotifyListener
	var nilFunc unsupportedFuncPolarisNotifyListener
	var nilChan unsupportedChanPolarisNotifyListener
	tests := []struct {
		name          string
		notify        registry.NotifyListener
		withInstances bool
	}{
		{name: "nil interface", notify: nil},
		{name: "typed nil pointer", notify: typedNil, withInstances: true},
		{name: "typed nil slice", notify: nilSlice, withInstances: true},
		{name: "typed nil map", notify: nilMap, withInstances: true},
		{name: "typed nil func", notify: nilFunc, withInstances: true},
		{name: "typed nil chan", notify: nilChan, withInstances: true},
		{name: "non-comparable slice", notify: unsupportedPolarisNotifyListener{}, withInstances: true},
		{name: "non-comparable map", notify: unsupportedMapPolarisNotifyListener{}, withInstances: true},
		{name: "non-comparable func", notify: unsupportedFuncPolarisNotifyListener(func() {}), withInstances: true},
		{name: "non-comparable struct", notify: unsupportedStructPolarisNotifyListener{}, withInstances: true},
		{
			name:          "runtime-unhashable interface field",
			notify:        runtimeUnhashablePolarisNotifyListener{state: []int{1}},
			withInstances: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+" load", func(t *testing.T) {
			var responses [][]model.Instance
			if tt.withInstances {
				responses = [][]model.Instance{{instanceA}}
			}
			consumer := &fakePolarisConsumer{instanceResponses: responses}
			pr := newTestPolarisRegistry(consumer)
			err := callPolarisRegistryWithoutPanic(t, func() error {
				return pr.LoadSubscribeInstances(newPolarisConsumerURL(serviceName), tt.notify)
			})
			if err == nil {
				t.Errorf("LoadSubscribeInstances() error = nil, want invalid listener error")
			} else if listenerType := fmt.Sprintf("%T", tt.notify); !strings.Contains(err.Error(), listenerType) {
				t.Errorf("LoadSubscribeInstances() error = %q, want listener type %q", err, listenerType)
			}
			if consumer.getCalls != 0 {
				t.Errorf("GetInstances() calls = %d, want 0", consumer.getCalls)
			}
			if len(pr.initialSubscribeInstances) != 0 {
				t.Errorf("pending entry count = %d, want 0", len(pr.initialSubscribeInstances))
			}
		})

		t.Run(tt.name+" create listener", func(t *testing.T) {
			pr := newTestPolarisRegistry(nil)
			mustStoppedRegistryWatcher(t, pr, serviceName)
			var listener *polarisListener
			err := callPolarisRegistryWithoutPanic(t, func() error {
				var createErr error
				listener, createErr = pr.createPolarisListener(serviceName, tt.notify)
				return createErr
			})
			if listener != nil {
				closeTestPolarisListener(listener)
			}
			if err == nil {
				t.Errorf("createPolarisListener() error = nil, want invalid listener error")
			} else if listenerType := fmt.Sprintf("%T", tt.notify); !strings.Contains(err.Error(), listenerType) {
				t.Errorf("createPolarisListener() error = %q, want listener type %q", err, listenerType)
			}
			if len(pr.initialSubscribeInstances) != 0 {
				t.Errorf("pending entry count = %d, want 0", len(pr.initialSubscribeInstances))
			}
		})
	}
}

func TestPolarisPointerNotifyListenerKeepsBaseline(t *testing.T) {
	serviceName := "com.test.PointerNotifyListenerService"
	instanceA := newPolarisTestInstance("instance-a", "10.0.0.1", 20001, serviceName, true)
	consumer := &fakePolarisConsumer{instanceResponses: [][]model.Instance{{instanceA}}}
	pr := newTestPolarisRegistry(consumer)
	notify := &recordingPolarisNotifyListener{}

	if err := pr.LoadSubscribeInstances(newPolarisConsumerURL(serviceName), notify); err != nil {
		t.Fatalf("LoadSubscribeInstances() error = %v", err)
	}
	key := mustInitialSubscribeInstancesKey(t, serviceName, notify)
	entry, instances := pr.loadInitialSubscribeInstances(key)
	if entry == nil || len(instances) != 1 || instances[0].GetInstanceKey() != instanceA.GetInstanceKey() {
		t.Fatalf("pointer listener pending entry = (%p, %v), want instance A", entry, instances)
	}

	watcher := mustStoppedRegistryWatcher(t, pr, serviceName)
	watcher.handleWatchSnapshot(nil)
	listener, err := pr.createPolarisListener(serviceName, notify)
	if err != nil {
		t.Fatalf("createPolarisListener() error = %v", err)
	}
	defer closeTestPolarisListener(listener)
	assertPolarisListenerEvent(t, listener, remoting.EventTypeDel, instanceA)
	if pending, _ := pr.loadInitialSubscribeInstances(key); pending != nil {
		t.Fatalf("pointer listener pending entry after listener success = %p, want nil", pending)
	}
}

func newTestPolarisRegistry(consumer api.ConsumerAPI) *polarisRegistry {
	return &polarisRegistry{
		namespace: testPolarisNamespace,
		consumer:  consumer,
		watchers:  make(map[string]*PolarisServiceWatcher),
	}
}

func callPolarisRegistryWithoutPanic(t *testing.T, call func() error) (err error) {
	t.Helper()
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Errorf("registry call panicked: %v", recovered)
		}
	}()
	return call()
}

func mustInitialSubscribeInstancesKey(
	t *testing.T,
	serviceName string,
	notify registry.NotifyListener,
) initialSubscribeInstancesKey {
	t.Helper()
	key, err := newInitialSubscribeInstancesKey(serviceName, notify)
	if err != nil {
		t.Fatalf("newInitialSubscribeInstancesKey() error = %v", err)
	}
	return key
}

func assertPolarisAddSubscriberSignature(_ func(func(remoting.EventType, []model.Instance))) {}

func closeTestPolarisListener(listener *polarisListener) {
	listener.Close()
	close(listener.events.In())
	for range listener.events.Out() {
	}
}

func mustStoppedRegistryWatcher(t *testing.T, pr *polarisRegistry, serviceName string) *PolarisServiceWatcher {
	t.Helper()
	watcher, err := pr.createPolarisWatcher(serviceName)
	if err != nil {
		t.Fatalf("createPolarisWatcher() error = %v", err)
	}
	watcher.execOnce.Do(func() {})
	return watcher
}

func newStoppedPolarisWatcher(t *testing.T, consumer api.ConsumerAPI) *PolarisServiceWatcher {
	t.Helper()
	watcher, err := newPolarisWatcher(nil, consumer)
	if err != nil {
		t.Fatalf("newPolarisWatcher() error = %v", err)
	}
	watcher.execOnce.Do(func() {})
	return watcher
}

func assertPolarisListenerEvents(
	t *testing.T,
	listener *polarisListener,
	events []recordedPolarisEvent,
) []recordedPolarisEvent {
	t.Helper()
	actualEvents := make([]recordedPolarisEvent, 0, len(events))
	for _, expected := range events {
		select {
		case value := <-listener.events.Out():
			event, ok := value.(*config_center.ConfigChangeEvent)
			if !ok {
				t.Fatalf("listener event type = %T, want *config_center.ConfigChangeEvent", value)
			}
			instance, ok := event.Value.(model.Instance)
			if !ok {
				t.Fatalf("listener event value type = %T, want model.Instance", event.Value)
			}
			actual := recordedPolarisEventFromInstance(event.ConfigType, instance)
			assertPolarisEventAddress(t, actual, expected)
			actualEvents = append(actualEvents, actual)
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for listener event %#v", expected)
		}
	}
	assertNoPolarisListenerEvent(t, listener)
	return actualEvents
}

func recordedPolarisEventFromInstance(action remoting.EventType, instance model.Instance) recordedPolarisEvent {
	return recordedPolarisEvent{
		action:   action,
		host:     instance.GetHost(),
		port:     strconv.Itoa(int(instance.GetPort())),
		id:       instance.GetId(),
		revision: instance.GetMetadata()[testPolarisRevisionKey],
	}
}

func assertPolarisEventsAddress(t *testing.T, actual, expected []recordedPolarisEvent) {
	t.Helper()
	if len(actual) != len(expected) {
		t.Fatalf("event count = %d, want %d; events = %#v", len(actual), len(expected), actual)
	}
	for i := range expected {
		assertPolarisEventAddress(t, actual[i], expected[i])
	}
}

func assertPolarisEventAddress(t *testing.T, actual, expected recordedPolarisEvent) {
	t.Helper()
	if actual.action != expected.action || actual.host != expected.host || actual.port != expected.port {
		t.Fatalf(
			"event address = (action=%v host=%s port=%s), want (action=%v host=%s port=%s)",
			actual.action,
			actual.host,
			actual.port,
			expected.action,
			expected.host,
			expected.port,
		)
	}
}

func assertPolarisEventInstance(
	t *testing.T,
	actual recordedPolarisEvent,
	action remoting.EventType,
	expected model.Instance,
) {
	t.Helper()
	want := recordedPolarisEventFromInstance(action, expected)
	if actual != want {
		t.Fatalf("event instance = %#v, want %#v", actual, want)
	}
}

func assertPolarisInstanceIdentity(t *testing.T, actual, expected model.Instance) {
	t.Helper()
	if actual.GetInstanceKey() != expected.GetInstanceKey() ||
		actual.GetId() != expected.GetId() ||
		actual.GetMetadata()[testPolarisRevisionKey] != expected.GetMetadata()[testPolarisRevisionKey] {
		t.Fatalf(
			"instance = (key=%v id=%s revision=%s), want (key=%v id=%s revision=%s)",
			actual.GetInstanceKey(),
			actual.GetId(),
			actual.GetMetadata()[testPolarisRevisionKey],
			expected.GetInstanceKey(),
			expected.GetId(),
			expected.GetMetadata()[testPolarisRevisionKey],
		)
	}
}

func assertPolarisWatchRequest(
	t *testing.T,
	actual *api.WatchServiceRequest,
	expected *api.WatchServiceRequest,
	serviceName string,
) {
	t.Helper()
	if actual == nil {
		t.Fatal("WatchService() request is nil")
	}
	if actual.Key.Namespace != testPolarisNamespace || actual.Key.Service != serviceName {
		t.Fatalf(
			"WatchService() request key = (%s, %s), want (%s, %s)",
			actual.Key.Namespace,
			actual.Key.Service,
			testPolarisNamespace,
			serviceName,
		)
	}
	if actual != expected {
		t.Fatalf("WatchService() request = %p, want watcher subscribeParam %p", actual, expected)
	}
}

func assertPolarisListenerEvent(t *testing.T, listener *polarisListener, action remoting.EventType, instance model.Instance) {
	t.Helper()
	select {
	case value := <-listener.events.Out():
		event, ok := value.(*config_center.ConfigChangeEvent)
		if !ok {
			t.Fatalf("listener event type = %T, want *config_center.ConfigChangeEvent", value)
		}
		actual, ok := event.Value.(model.Instance)
		if !ok || event.ConfigType != action || actual.GetInstanceKey() != instance.GetInstanceKey() {
			t.Fatalf("listener event = %#v, want action=%v instance=%v", event, action, instance)
		}
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for action=%v instance=%v", action, instance)
	}
}

func assertNoPolarisListenerEvent(t *testing.T, listener *polarisListener) {
	t.Helper()
	select {
	case event := <-listener.events.Out():
		t.Fatalf("unexpected listener event: %#v", event)
	case <-time.After(20 * time.Millisecond):
	}
}

func newPolarisConsumerURL(serviceName string) *common.URL {
	return common.NewURLWithOptions(
		common.WithProtocol("consumer"),
		common.WithInterface(serviceName),
		common.WithParamsValue(constant.RegistryRoleKey, strconv.Itoa(common.CONSUMER)),
	)
}

func newPolarisTestInstance(id string, host string, port uint32, serviceName string, valid bool) model.Instance {
	metadata := map[string]string{"interface": serviceName, "path": "/" + serviceName}
	if !valid {
		metadata = nil
	}
	instance := &namingpb.Instance{
		Id:       &wrappers.StringValue{Value: id},
		Host:     &wrappers.StringValue{Value: host},
		Port:     &wrappers.UInt32Value{Value: port},
		Protocol: &wrappers.StringValue{Value: "tri"},
		Healthy:  &wrappers.BoolValue{Value: true},
		Isolate:  &wrappers.BoolValue{Value: false},
		Metadata: metadata,
	}
	return pb.NewInstanceInProto(
		instance,
		&model.ServiceKey{Namespace: testPolarisNamespace, Service: serviceName},
		local.NewInstanceLocalValue(),
	)
}
