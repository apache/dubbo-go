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
	"reflect"
	"strconv"
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

const testPolarisNamespace = "test"

type fakePolarisConsumer struct {
	api.ConsumerAPI
	instances         []model.Instance
	instanceResponses [][]model.Instance
	getCalls          int
	watchInstances    []model.Instance
	watchCalls        int
}

func (f *fakePolarisConsumer) GetInstances(_ *api.GetInstancesRequest) (*model.InstancesResponse, error) {
	instances := f.instances
	if f.getCalls < len(f.instanceResponses) {
		instances = f.instanceResponses[f.getCalls]
	}
	f.getCalls++
	instances = append([]model.Instance(nil), instances...)
	return &model.InstancesResponse{Instances: instances}, nil
}

func (f *fakePolarisConsumer) WatchService(_ *api.WatchServiceRequest) (*model.WatchServiceResponse, error) {
	f.watchCalls++
	eventChannel := make(chan model.SubScribeEvent)
	close(eventChannel)
	return &model.WatchServiceResponse{
		EventChannel: eventChannel,
		GetAllInstancesResp: &model.InstancesResponse{
			Instances: append([]model.Instance(nil), f.watchInstances...),
		},
	}, nil
}

type recordedPolarisEvent struct {
	action remoting.EventType
	host   string
	port   string
}

type recordingPolarisNotifyListener struct {
	events  []recordedPolarisEvent
	eventCh chan recordedPolarisEvent
}

type nonComparablePolarisNotifyListener []recordedPolarisEvent

func (nonComparablePolarisNotifyListener) Notify(*registry.ServiceEvent) {}

func (nonComparablePolarisNotifyListener) NotifyAll([]*registry.ServiceEvent, func()) {}

func (r *recordingPolarisNotifyListener) Notify(event *registry.ServiceEvent) {
	r.record(recordedPolarisEvent{
		action: event.Action,
		host:   event.Service.Ip,
		port:   event.Service.Port,
	})
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
		r.record(recordedPolarisEvent{
			action: action,
			host:   instance.GetHost(),
			port:   strconv.Itoa(int(instance.GetPort())),
		})
	}
}

func (r *recordingPolarisNotifyListener) record(event recordedPolarisEvent) {
	r.events = append(r.events, event)
	if r.eventCh != nil {
		r.eventCh <- event
	}
}

func TestPolarisSubscribeInitialSnapshotTransfer(t *testing.T) {
	serviceName := "com.test.SubscribeSnapshotTransferService"
	instanceA := newPolarisTestInstance("instance-a", "10.0.0.1", 20001, serviceName, true)
	instanceB := newPolarisTestInstance("instance-b", "10.0.0.2", 20002, serviceName, true)
	consumer := &fakePolarisConsumer{
		instances:      []model.Instance{instanceA},
		watchInstances: []model.Instance{instanceB},
	}
	notify := &recordingPolarisNotifyListener{eventCh: make(chan recordedPolarisEvent, 3)}
	polarisRegistry := &polarisRegistry{
		namespace:                 testPolarisNamespace,
		consumer:                  consumer,
		watchers:                  make(map[string]*PolarisServiceWatcher),
		initialSubscribeInstances: make(map[initialSubscribeInstancesKey][]model.Instance),
	}

	if err := polarisRegistry.LoadSubscribeInstances(newPolarisConsumerURL(serviceName), notify); err != nil {
		t.Fatalf("LoadSubscribeInstances() error = %v", err)
	}

	_, err := polarisRegistry.createPolarisListener(serviceName, notify, func(watcher *PolarisServiceWatcher, initialSnapshot []model.Instance) (*polarisListener, error) {
		resp, watchErr := watcher.consumer.WatchService(watcher.subscribeParam)
		if watchErr != nil {
			return nil, watchErr
		}
		watcher.execOnce.Do(func() {})
		watcher.addSubscriberWithInitialSnapshot(initialSnapshot, notify.recordInstances)
		watcher.handleWatchSnapshot(resp.GetAllInstancesResp.Instances)
		return &polarisListener{}, nil
	})
	if err != nil {
		t.Fatalf("createPolarisListener() error = %v", err)
	}
	if consumer.watchCalls != 1 {
		t.Fatalf("WatchService() calls = %d, want 1", consumer.watchCalls)
	}

	expected := []recordedPolarisEvent{
		{action: remoting.EventTypeAdd, host: "10.0.0.1", port: "20001"},
		{action: remoting.EventTypeDel, host: "10.0.0.1", port: "20001"},
		{action: remoting.EventTypeAdd, host: "10.0.0.2", port: "20002"},
	}
	actual := make([]recordedPolarisEvent, 0, len(expected))
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for range expected {
		select {
		case event := <-notify.eventCh:
			actual = append(actual, event)
		case <-timer.C:
			t.Fatalf("timed out waiting for events, got %#v", actual)
		}
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("events = %#v, want %#v", actual, expected)
	}
}

func TestPolarisInitialSnapshotReconciliation(t *testing.T) {
	serviceName := "com.test.InitialSnapshotService"
	instanceA := newPolarisTestInstance(
		"instance-a",
		"10.0.0.1",
		20001,
		serviceName,
		true,
	)
	invalidInstance := newPolarisTestInstance(
		"invalid",
		"10.0.0.9",
		20009,
		serviceName,
		false,
	)
	instanceB := newPolarisTestInstance(
		"instance-b",
		"10.0.0.2",
		20002,
		serviceName,
		true,
	)

	t.Run("A is replaced by B", func(t *testing.T) {
		testPolarisInitialSnapshotReconciliation(
			t,
			serviceName,
			instanceA,
			invalidInstance,
			[]model.Instance{instanceB},
			[]recordedPolarisEvent{
				{
					action: remoting.EventTypeAdd,
					host:   "10.0.0.1",
					port:   "20001",
				},
				{
					action: remoting.EventTypeDel,
					host:   "10.0.0.1",
					port:   "20001",
				},
				{
					action: remoting.EventTypeAdd,
					host:   "10.0.0.2",
					port:   "20002",
				},
			},
		)
	})

	t.Run("first watcher snapshot is empty", func(t *testing.T) {
		testPolarisInitialSnapshotReconciliation(
			t,
			serviceName,
			instanceA,
			invalidInstance,
			nil,
			[]recordedPolarisEvent{
				{
					action: remoting.EventTypeAdd,
					host:   "10.0.0.1",
					port:   "20001",
				},
				{
					action: remoting.EventTypeDel,
					host:   "10.0.0.1",
					port:   "20001",
				},
			},
		)
	})
}

func testPolarisInitialSnapshotReconciliation(t *testing.T, serviceName string, instanceA model.Instance, invalidInstance model.Instance, current []model.Instance, expected []recordedPolarisEvent) {
	t.Helper()

	notify := &recordingPolarisNotifyListener{}
	consumer := &fakePolarisConsumer{
		instances: []model.Instance{instanceA, invalidInstance},
	}

	polarisRegistry := &polarisRegistry{
		namespace:                 testPolarisNamespace,
		consumer:                  consumer,
		initialSubscribeInstances: make(map[initialSubscribeInstancesKey][]model.Instance),
	}

	if err := polarisRegistry.LoadSubscribeInstances(newPolarisConsumerURL(serviceName), notify); err != nil {
		t.Fatalf("LoadSubscribeInstances() error = %v", err)
	}

	initial, ok := polarisRegistry.takeInitialSubscribeInstances(serviceName, notify)
	if !ok {
		t.Fatal("initial snapshot was not saved")
	}

	if len(initial) != 1 ||
		initial[0].GetInstanceKey() != instanceA.GetInstanceKey() {
		t.Fatalf("initial snapshot = %v, want only instance A", initial)
	}

	watcher := newStoppedPolarisWatcher(t, consumer)
	watcher.addSubscriberWithInitialSnapshot(initial, notify.recordInstances)
	watcher.handleWatchSnapshot(current)

	if !reflect.DeepEqual(notify.events, expected) {
		t.Fatalf("events = %#v, want %#v", notify.events, expected)
	}
}

func TestMissingInitialInstancesSameInstance(t *testing.T) {
	serviceName := "com.test.SameInstanceService"
	initial := newPolarisTestInstance("old-id", "10.0.0.1", 20001, serviceName, true)
	current := newPolarisTestInstance("new-id", "10.0.0.1", 20001, serviceName, true)

	missing := missingInitialInstances([]model.Instance{initial}, []model.Instance{current})
	if len(missing) != 0 {
		t.Fatalf("missingInitialInstances() = %v, want empty", missing)
	}
}

func TestMissingInitialInstancesPreservesInitialOrder(t *testing.T) {
	serviceName := "com.test.InstanceOrderService"
	instanceA := newPolarisTestInstance("instance-a", "10.0.0.1", 20001, serviceName, true)
	instanceB := newPolarisTestInstance("instance-b", "10.0.0.2", 20002, serviceName, true)
	instanceC := newPolarisTestInstance("instance-c", "10.0.0.3", 20003, serviceName, true)

	missing := missingInitialInstances(
		[]model.Instance{instanceA, instanceB, instanceC},
		[]model.Instance{instanceB},
	)
	if len(missing) != 2 {
		t.Fatalf("missingInitialInstances() length = %d, want 2", len(missing))
	}
	if missing[0].GetInstanceKey() != instanceA.GetInstanceKey() ||
		missing[1].GetInstanceKey() != instanceC.GetInstanceKey() {
		t.Fatalf("missingInitialInstances() = %v, want [instance A, instance C]", missing)
	}
}

func TestPolarisSubscriberInitialSnapshotReconciledOnce(t *testing.T) {
	serviceName := "com.test.OneShotService"
	instanceA := newPolarisTestInstance("instance-a", "10.0.0.1", 20001, serviceName, true)
	instanceB := newPolarisTestInstance("instance-b", "10.0.0.2", 20002, serviceName, true)
	watcher := newStoppedPolarisWatcher(t, nil)
	notify := &recordingPolarisNotifyListener{}

	snapshot := []model.Instance{instanceA}
	watcher.addSubscriberWithInitialSnapshot(snapshot, notify.recordInstances)
	snapshot[0] = instanceB
	watcher.handleWatchSnapshot([]model.Instance{instanceB})
	watcher.handleWatchSnapshot([]model.Instance{instanceB})

	expected := []recordedPolarisEvent{
		{action: remoting.EventTypeDel, host: "10.0.0.1", port: "20001"},
		{action: remoting.EventTypeAdd, host: "10.0.0.2", port: "20002"},
		{action: remoting.EventTypeAdd, host: "10.0.0.2", port: "20002"},
	}
	if !reflect.DeepEqual(notify.events, expected) {
		t.Fatalf("events = %#v, want %#v", notify.events, expected)
	}
	if len(watcher.subscribers) != 1 || !watcher.subscribers[0].reconciled || watcher.subscribers[0].initialSnapshot != nil {
		t.Fatalf("subscriber state = %#v, want reconciled state with cleared baseline", watcher.subscribers)
	}
}

func TestPolarisReusedWatcherReconcilesSecondSubscriberAfterDelete(t *testing.T) {
	serviceName := "com.test.ReusedWatcherService"
	instanceA := newPolarisTestInstance("instance-a", "10.0.0.1", 20001, serviceName, true)
	consumer := &fakePolarisConsumer{instances: []model.Instance{instanceA}}
	polarisRegistry := &polarisRegistry{
		namespace:                 testPolarisNamespace,
		consumer:                  consumer,
		watchers:                  make(map[string]*PolarisServiceWatcher),
		initialSubscribeInstances: make(map[initialSubscribeInstancesKey][]model.Instance),
	}

	watcher, err := polarisRegistry.createPolarisWatcher(serviceName)
	if err != nil {
		t.Fatalf("createPolarisWatcher() error = %v", err)
	}
	watcher.execOnce.Do(func() {})

	firstNotify := &recordingPolarisNotifyListener{}
	firstListener, err := polarisRegistry.createPolarisListener(serviceName, firstNotify, newPolarisListener)
	if err != nil {
		t.Fatalf("createPolarisListener(first) error = %v", err)
	}
	defer firstListener.Close()
	if firstListener.watcher != watcher {
		t.Fatal("first listener did not use the shared watcher")
	}
	watcher.handleWatchSnapshot([]model.Instance{instanceA})
	assertPolarisListenerEvent(t, firstListener, remoting.EventTypeAdd, instanceA)

	second := &recordingPolarisNotifyListener{}
	if err := polarisRegistry.LoadSubscribeInstances(newPolarisConsumerURL(serviceName), second); err != nil {
		t.Fatalf("LoadSubscribeInstances() error = %v", err)
	}
	watcher.handleInstanceEvent(&model.InstanceEvent{
		DeleteEvent: &model.InstanceDeleteEvent{Instances: []model.Instance{instanceA}},
	})
	assertPolarisListenerEvent(t, firstListener, remoting.EventTypeDel, instanceA)

	if expected := []recordedPolarisEvent{
		{action: remoting.EventTypeAdd, host: "10.0.0.1", port: "20001"},
	}; !reflect.DeepEqual(second.events, expected) {
		t.Fatalf("second subscriber events before joining = %#v, want %#v", second.events, expected)
	}

	secondListener, err := polarisRegistry.createPolarisListener(serviceName, second, newPolarisListener)
	if err != nil {
		t.Fatalf("createPolarisListener(second) error = %v", err)
	}
	defer secondListener.Close()
	if secondListener.watcher != watcher {
		t.Fatal("second listener did not reuse the shared watcher")
	}
	assertPolarisListenerEvent(t, secondListener, remoting.EventTypeDel, instanceA)

	assertNoPolarisListenerEvent(t, firstListener)

	watcher.handleWatchSnapshot(nil)
	assertNoPolarisListenerEvent(t, secondListener)
}

func TestPolarisInitialSubscribeInstancesAreScopedToNotifyListener(t *testing.T) {
	serviceName := "com.test.ListenerScopedBaselineService"
	instanceA := newPolarisTestInstance("instance-a", "10.0.0.1", 20001, serviceName, true)
	instanceB := newPolarisTestInstance("instance-b", "10.0.0.2", 20002, serviceName, true)
	consumer := &fakePolarisConsumer{instanceResponses: [][]model.Instance{{instanceA}, {instanceB}}}
	polarisRegistry := &polarisRegistry{
		namespace: testPolarisNamespace,
		consumer:  consumer,
		watchers:  make(map[string]*PolarisServiceWatcher),
	}
	firstNotify := &recordingPolarisNotifyListener{}
	secondNotify := &recordingPolarisNotifyListener{}
	url := newPolarisConsumerURL(serviceName)

	if err := polarisRegistry.LoadSubscribeInstances(url, firstNotify); err != nil {
		t.Fatalf("LoadSubscribeInstances(first) error = %v", err)
	}
	if err := polarisRegistry.LoadSubscribeInstances(url, secondNotify); err != nil {
		t.Fatalf("LoadSubscribeInstances(second) error = %v", err)
	}

	watcher, err := polarisRegistry.createPolarisWatcher(serviceName)
	if err != nil {
		t.Fatalf("createPolarisWatcher() error = %v", err)
	}
	watcher.execOnce.Do(func() {})
	watcher.handleWatchSnapshot(nil)

	firstListener, err := polarisRegistry.createPolarisListener(serviceName, firstNotify, newPolarisListener)
	if err != nil {
		t.Fatalf("createPolarisListener(first) error = %v", err)
	}
	defer firstListener.Close()
	assertPolarisListenerEvent(t, firstListener, remoting.EventTypeDel, instanceA)

	secondListener, err := polarisRegistry.createPolarisListener(serviceName, secondNotify, newPolarisListener)
	if err != nil {
		t.Fatalf("createPolarisListener(second) error = %v", err)
	}
	defer secondListener.Close()
	assertPolarisListenerEvent(t, secondListener, remoting.EventTypeDel, instanceB)
}

func TestPolarisEmptyLoadDoesNotClearAnotherListenerBaseline(t *testing.T) {
	serviceName := "com.test.ListenerScopedEmptyBaselineService"
	instanceA := newPolarisTestInstance("instance-a", "10.0.0.1", 20001, serviceName, true)
	consumer := &fakePolarisConsumer{instanceResponses: [][]model.Instance{{instanceA}, nil}}
	polarisRegistry := &polarisRegistry{
		namespace: testPolarisNamespace,
		consumer:  consumer,
		watchers:  make(map[string]*PolarisServiceWatcher),
	}
	firstNotify := &recordingPolarisNotifyListener{}
	secondNotify := &recordingPolarisNotifyListener{}
	url := newPolarisConsumerURL(serviceName)

	if err := polarisRegistry.LoadSubscribeInstances(url, firstNotify); err != nil {
		t.Fatalf("LoadSubscribeInstances(first) error = %v", err)
	}
	if err := polarisRegistry.LoadSubscribeInstances(url, secondNotify); err != nil {
		t.Fatalf("LoadSubscribeInstances(second) error = %v", err)
	}

	watcher, err := polarisRegistry.createPolarisWatcher(serviceName)
	if err != nil {
		t.Fatalf("createPolarisWatcher() error = %v", err)
	}
	watcher.execOnce.Do(func() {})
	watcher.handleWatchSnapshot(nil)

	firstListener, err := polarisRegistry.createPolarisListener(serviceName, firstNotify, newPolarisListener)
	if err != nil {
		t.Fatalf("createPolarisListener(first) error = %v", err)
	}
	defer firstListener.Close()
	assertPolarisListenerEvent(t, firstListener, remoting.EventTypeDel, instanceA)

	secondListener, err := polarisRegistry.createPolarisListener(serviceName, secondNotify, newPolarisListener)
	if err != nil {
		t.Fatalf("createPolarisListener(second) error = %v", err)
	}
	defer secondListener.Close()
	assertNoPolarisListenerEvent(t, secondListener)
}

func TestPolarisLoadSupportsNonComparableNotifyListener(t *testing.T) {
	serviceName := "com.test.NonComparableNotifyListenerService"
	instanceA := newPolarisTestInstance("instance-a", "10.0.0.1", 20001, serviceName, true)
	consumer := &fakePolarisConsumer{instances: []model.Instance{instanceA}}
	polarisRegistry := &polarisRegistry{namespace: testPolarisNamespace, consumer: consumer}
	notify := nonComparablePolarisNotifyListener{}

	if err := polarisRegistry.LoadSubscribeInstances(newPolarisConsumerURL(serviceName), notify); err != nil {
		t.Fatalf("LoadSubscribeInstances() error = %v", err)
	}
	if consumer.getCalls != 1 {
		t.Fatalf("GetInstances() calls = %d, want 1", consumer.getCalls)
	}
	stored, ok := polarisRegistry.takeInitialSubscribeInstances(serviceName, notify)
	if !ok || len(stored) != 1 || stored[0].GetInstanceKey() != instanceA.GetInstanceKey() {
		t.Fatalf("initial subscribe instances = (%v, %v), want instance A", stored, ok)
	}
}

func TestPolarisInitialSubscribeInstancesStateClearedAfterListenerCreation(t *testing.T) {
	serviceName := "com.test.InitialSubscribeInstancesCleanupService"
	instanceA := newPolarisTestInstance("instance-a", "10.0.0.1", 20001, serviceName, true)
	notify := &recordingPolarisNotifyListener{}
	polarisRegistry := &polarisRegistry{
		namespace: testPolarisNamespace,
		consumer:  &fakePolarisConsumer{instances: []model.Instance{instanceA}},
		watchers:  make(map[string]*PolarisServiceWatcher),
	}
	if err := polarisRegistry.LoadSubscribeInstances(newPolarisConsumerURL(serviceName), notify); err != nil {
		t.Fatalf("LoadSubscribeInstances() error = %v", err)
	}
	watcher, err := polarisRegistry.createPolarisWatcher(serviceName)
	if err != nil {
		t.Fatalf("createPolarisWatcher() error = %v", err)
	}
	watcher.execOnce.Do(func() {})

	listener, err := polarisRegistry.createPolarisListener(serviceName, notify, newPolarisListener)
	if err != nil {
		t.Fatalf("createPolarisListener() error = %v", err)
	}
	defer listener.Close()
	if len(polarisRegistry.initialSubscribeInstances) != 0 || len(polarisRegistry.initialSubscribeVersions) != 0 {
		t.Fatalf("initial subscribe instances state = (%d snapshots, %d versions), want empty", len(polarisRegistry.initialSubscribeInstances), len(polarisRegistry.initialSubscribeVersions))
	}
}

func TestPolarisWatcherTracksIncrementalCurrentInstances(t *testing.T) {
	serviceName := "com.test.IncrementalStateService"
	instanceA := newPolarisTestInstance("instance-a", "10.0.0.1", 20001, serviceName, true)
	instanceB := newPolarisTestInstance("instance-b", "10.0.0.2", 20002, serviceName, true)
	instanceC := newPolarisTestInstance("instance-c", "10.0.0.3", 20003, serviceName, true)
	replacementB := newPolarisTestInstance("replacement-b", "10.0.0.2", 20002, serviceName, true)
	watcher := newStoppedPolarisWatcher(t, nil)
	notify := &recordingPolarisNotifyListener{}
	watcher.addSubscriberWithInitialSnapshot(nil, notify.recordInstances)
	watcher.handleWatchSnapshot([]model.Instance{instanceA})

	watcher.handleInstanceEvent(&model.InstanceEvent{
		AddEvent: &model.InstanceAddEvent{Instances: []model.Instance{instanceB}},
	})
	watcher.handleInstanceEvent(&model.InstanceEvent{
		UpdateEvent: &model.InstanceUpdateEvent{UpdateList: []model.OneInstanceUpdate{
			{Before: instanceA, After: instanceC},
			{Before: instanceB, After: replacementB},
		}},
	})
	watcher.handleInstanceEvent(&model.InstanceEvent{
		DeleteEvent: &model.InstanceDeleteEvent{Instances: []model.Instance{instanceC}},
	})

	if len(watcher.currentInstances) != 1 || watcher.currentInstances[0].GetId() != replacementB.GetId() {
		t.Fatalf("currentInstances = %v, want only replacement B", watcher.currentInstances)
	}
	expectedEvents := []recordedPolarisEvent{
		{action: remoting.EventTypeAdd, host: "10.0.0.1", port: "20001"},
		{action: remoting.EventTypeAdd, host: "10.0.0.2", port: "20002"},
		{action: remoting.EventTypeUpdate, host: "10.0.0.3", port: "20003"},
		{action: remoting.EventTypeUpdate, host: "10.0.0.2", port: "20002"},
		{action: remoting.EventTypeDel, host: "10.0.0.3", port: "20003"},
	}
	if !reflect.DeepEqual(notify.events, expectedEvents) {
		t.Fatalf("incremental events = %#v, want %#v", notify.events, expectedEvents)
	}

}

func TestPolarisAddSubscriberBeforeInitialSnapshotReceivesCurrentInstances(t *testing.T) {
	serviceName := "com.test.AddSubscriberBeforeSnapshotService"
	instanceA := newPolarisTestInstance("instance-a", "10.0.0.1", 20001, serviceName, true)
	watcher := newStoppedPolarisWatcher(t, nil)
	notify := &recordingPolarisNotifyListener{}

	watcher.AddSubscriber(notify.recordInstances)
	watcher.handleWatchSnapshot([]model.Instance{instanceA})

	expected := []recordedPolarisEvent{{action: remoting.EventTypeAdd, host: "10.0.0.1", port: "20001"}}
	if !reflect.DeepEqual(notify.events, expected) {
		t.Fatalf("events = %#v, want %#v", notify.events, expected)
	}
}

func TestPolarisAddSubscriberAfterInitialSnapshotDoesNotReplayCurrentInstances(t *testing.T) {
	serviceName := "com.test.AddSubscriberAfterSnapshotService"
	instanceA := newPolarisTestInstance("instance-a", "10.0.0.1", 20001, serviceName, true)
	instanceB := newPolarisTestInstance("instance-b", "10.0.0.2", 20002, serviceName, true)
	watcher := newStoppedPolarisWatcher(t, nil)
	watcher.handleWatchSnapshot([]model.Instance{instanceA})
	notify := &recordingPolarisNotifyListener{}

	watcher.AddSubscriber(notify.recordInstances)
	if len(notify.events) != 0 {
		t.Fatalf("events after AddSubscriber() = %#v, want no replay", notify.events)
	}

	watcher.handleInstanceEvent(&model.InstanceEvent{
		AddEvent: &model.InstanceAddEvent{Instances: []model.Instance{instanceB}},
	})
	expected := []recordedPolarisEvent{{action: remoting.EventTypeAdd, host: "10.0.0.2", port: "20002"}}
	if !reflect.DeepEqual(notify.events, expected) {
		t.Fatalf("events after incremental update = %#v, want %#v", notify.events, expected)
	}
}

func TestNewPolarisListenerAfterInitialSnapshotDoesNotReplayCurrentInstances(t *testing.T) {
	serviceName := "com.test.NewListenerAfterSnapshotService"
	instanceA := newPolarisTestInstance("instance-a", "10.0.0.1", 20001, serviceName, true)
	instanceB := newPolarisTestInstance("instance-b", "10.0.0.2", 20002, serviceName, true)
	watcher := newStoppedPolarisWatcher(t, nil)
	watcher.handleWatchSnapshot([]model.Instance{instanceA})

	listener, err := NewPolarisListener(watcher)
	if err != nil {
		t.Fatalf("NewPolarisListener() error = %v", err)
	}
	defer listener.Close()
	assertNoPolarisListenerEvent(t, listener)

	watcher.handleInstanceEvent(&model.InstanceEvent{
		AddEvent: &model.InstanceAddEvent{Instances: []model.Instance{instanceB}},
	})
	assertPolarisListenerEvent(t, listener, remoting.EventTypeAdd, instanceB)
}

func TestPolarisWatcherReplaysDefensiveCurrentSnapshot(t *testing.T) {
	serviceName := "com.test.DefensiveCurrentSnapshotService"
	instanceA := newPolarisTestInstance("instance-a", "10.0.0.1", 20001, serviceName, true)
	instanceB := newPolarisTestInstance("instance-b", "10.0.0.2", 20002, serviceName, true)
	watcher := newStoppedPolarisWatcher(t, nil)
	watcher.addSubscriberWithInitialSnapshot(nil, func(_ remoting.EventType, instances []model.Instance) {
		if len(instances) > 0 {
			instances[0] = instanceB
		}
	})

	current := []model.Instance{instanceA}
	watcher.handleWatchSnapshot(current)
	current[0] = instanceB

	replayed := &recordingPolarisNotifyListener{}
	watcher.addSubscriberWithInitialSnapshot(nil, replayed.recordInstances)
	expected := []recordedPolarisEvent{{action: remoting.EventTypeAdd, host: "10.0.0.1", port: "20001"}}
	if !reflect.DeepEqual(replayed.events, expected) {
		t.Fatalf("replayed events = %#v, want %#v", replayed.events, expected)
	}
}

func TestCreatePolarisListenerRestoresInitialSubscribeInstancesOnFailure(t *testing.T) {
	serviceName := "com.test.ListenerFailureService"
	instanceA := newPolarisTestInstance("instance-a", "10.0.0.1", 20001, serviceName, true)
	polarisRegistry := &polarisRegistry{
		watchers:                  make(map[string]*PolarisServiceWatcher),
		initialSubscribeInstances: make(map[initialSubscribeInstancesKey][]model.Instance),
	}
	notify := &recordingPolarisNotifyListener{}
	key := newInitialSubscribeInstancesKey(serviceName, notify)
	polarisRegistry.storeInitialSubscribeInstances(key, []model.Instance{instanceA})

	_, err := polarisRegistry.createPolarisListener(serviceName, notify, func(*PolarisServiceWatcher, []model.Instance) (*polarisListener, error) {
		return nil, errors.New("listener failed")
	})
	if err == nil {
		t.Fatal("createPolarisListener() error = nil, want failure")
	}
	restored, ok := polarisRegistry.takeInitialSubscribeInstances(serviceName, notify)
	if !ok || len(restored) != 1 || restored[0].GetInstanceKey() != instanceA.GetInstanceKey() {
		t.Fatalf("restored baseline = (%v, %v), want instance A", restored, ok)
	}
}

func TestCreatePolarisListenerDoesNotRestoreOverNewerEmptyInitialSubscribeInstances(t *testing.T) {
	serviceName := "com.test.ListenerFailureNewerEmptyService"
	instanceA := newPolarisTestInstance("instance-a", "10.0.0.1", 20001, serviceName, true)
	polarisRegistry := &polarisRegistry{
		watchers:                  make(map[string]*PolarisServiceWatcher),
		initialSubscribeInstances: make(map[initialSubscribeInstancesKey][]model.Instance),
	}
	notify := &recordingPolarisNotifyListener{}
	key := newInitialSubscribeInstancesKey(serviceName, notify)
	polarisRegistry.storeInitialSubscribeInstances(key, []model.Instance{instanceA})

	_, err := polarisRegistry.createPolarisListener(serviceName, notify, func(*PolarisServiceWatcher, []model.Instance) (*polarisListener, error) {
		polarisRegistry.storeInitialSubscribeInstances(key, nil)
		return nil, errors.New("listener failed after newer empty load")
	})
	if err == nil {
		t.Fatal("createPolarisListener() error = nil, want failure")
	}
	restored, ok := polarisRegistry.takeInitialSubscribeInstances(serviceName, notify)
	if ok || len(restored) != 0 {
		t.Fatalf("initial subscribe instances = (%v, %v), want newer empty load to win", restored, ok)
	}
}

func TestLoadSubscribeInstancesClearsStaleEmptySnapshot(t *testing.T) {
	serviceName := "com.test.EmptyInitialSnapshotService"
	instanceA := newPolarisTestInstance("instance-a", "10.0.0.1", 20001, serviceName, true)
	polarisRegistry := &polarisRegistry{
		namespace:                 testPolarisNamespace,
		consumer:                  &fakePolarisConsumer{},
		initialSubscribeInstances: make(map[initialSubscribeInstancesKey][]model.Instance),
	}
	notify := &recordingPolarisNotifyListener{}
	key := newInitialSubscribeInstancesKey(serviceName, notify)
	polarisRegistry.storeInitialSubscribeInstances(key, []model.Instance{instanceA})

	if err := polarisRegistry.LoadSubscribeInstances(newPolarisConsumerURL(serviceName), notify); err != nil {
		t.Fatalf("LoadSubscribeInstances() error = %v", err)
	}
	snapshot, ok := polarisRegistry.takeInitialSubscribeInstances(serviceName, notify)
	if ok || len(snapshot) != 0 {
		t.Fatalf("takeInitialSubscribeInstances() = (%v, %v), want (empty, false)", snapshot, ok)
	}
}

func TestPolarisRegistryInitialSnapshotUsesDefensiveCopy(t *testing.T) {
	serviceName := "com.test.DefensiveCopyService"
	instanceA := newPolarisTestInstance("instance-a", "10.0.0.1", 20001, serviceName, true)
	instanceB := newPolarisTestInstance("instance-b", "10.0.0.2", 20002, serviceName, true)
	polarisRegistry := &polarisRegistry{initialSubscribeInstances: make(map[initialSubscribeInstancesKey][]model.Instance)}
	notify := &recordingPolarisNotifyListener{}
	key := newInitialSubscribeInstancesKey(serviceName, notify)

	snapshot := []model.Instance{instanceA}
	polarisRegistry.storeInitialSubscribeInstances(key, snapshot)
	snapshot[0] = instanceB

	stored, ok := polarisRegistry.takeInitialSubscribeInstances(serviceName, notify)
	if !ok {
		t.Fatal("takeInitialSubscribeInstances() ok = false, want true")
	}
	if len(stored) != 1 || stored[0].GetInstanceKey() != instanceA.GetInstanceKey() {
		t.Fatalf("takeInitialSubscribeInstances() = %v, want instance A", stored)
	}
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

func assertPolarisListenerEvent(
	t *testing.T,
	listener *polarisListener,
	action remoting.EventType,
	instance model.Instance,
) {
	t.Helper()
	select {
	case value := <-listener.events.Out():
		event, ok := value.(*config_center.ConfigChangeEvent)
		if !ok {
			t.Fatalf("listener event type = %T, want *config_center.ConfigChangeEvent", value)
		}
		actualInstance, ok := event.Value.(model.Instance)
		if !ok {
			t.Fatalf("listener event value type = %T, want model.Instance", event.Value)
		}
		if event.ConfigType != action || actualInstance.GetInstanceKey() != instance.GetInstanceKey() {
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
	case <-time.After(50 * time.Millisecond):
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
	metadata := map[string]string{
		"interface": serviceName,
		"path":      "/" + serviceName,
	}
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
