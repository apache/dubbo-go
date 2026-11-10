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
	"context"
	"errors"
	"os"
	"os/exec"
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

type gatedPolarisWatchResult struct {
	response *model.WatchServiceResponse
	err      error
}

type gatedPolarisConsumer struct {
	*fakePolarisConsumer
	watchRequests chan *api.WatchServiceRequest
	watchResults  chan gatedPolarisWatchResult
}

func newGatedPolarisConsumer(instanceResponses ...[]model.Instance) *gatedPolarisConsumer {
	return &gatedPolarisConsumer{
		fakePolarisConsumer: &fakePolarisConsumer{instanceResponses: instanceResponses},
		watchRequests:       make(chan *api.WatchServiceRequest),
		watchResults:        make(chan gatedPolarisWatchResult),
	}
}

func (f *gatedPolarisConsumer) WatchService(request *api.WatchServiceRequest) (*model.WatchServiceResponse, error) {
	f.watchRequests <- request
	result := <-f.watchResults
	return result.response, result.err
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

type recordedNotification struct {
	kind        polarisNotificationKind
	events      []recordedPolarisEvent
	callbackNil bool
}

type recordingPolarisNotifyListener struct {
	notifications []recordedNotification
	onNotify      func(*registry.ServiceEvent)
}

type nonComparablePolarisNotifyListener []*recordingPolarisNotifyListener

type nonComparableMapPolarisNotifyListener map[string]*recordingPolarisNotifyListener

type nonComparableFuncPolarisNotifyListener func(*registry.ServiceEvent, []*registry.ServiceEvent, func())

type nonComparableStructPolarisNotifyListener struct {
	recorder *recordingPolarisNotifyListener
	_        []recordedPolarisEvent
}

type runtimeUnhashablePolarisNotifyListener struct {
	state    any
	recorder *recordingPolarisNotifyListener
}

type comparableValuePolarisNotifyListener struct {
	id       int
	recorder *recordingPolarisNotifyListener
}

type nilablePolarisNotifyListener struct{}

type nilableChannelPolarisNotifyListener chan struct{}

type stopSubscribePump struct{}

func (l nonComparablePolarisNotifyListener) Notify(event *registry.ServiceEvent) {
	if len(l) > 0 && l[0] != nil {
		l[0].Notify(event)
	}
}

func (l nonComparablePolarisNotifyListener) NotifyAll(events []*registry.ServiceEvent, callback func()) {
	if len(l) > 0 && l[0] != nil {
		l[0].NotifyAll(events, callback)
	}
}

func (l nonComparableMapPolarisNotifyListener) Notify(event *registry.ServiceEvent) {
	if recorder := l["recorder"]; recorder != nil {
		recorder.Notify(event)
	}
}

func (l nonComparableMapPolarisNotifyListener) NotifyAll(events []*registry.ServiceEvent, callback func()) {
	if recorder := l["recorder"]; recorder != nil {
		recorder.NotifyAll(events, callback)
	}
}

func (l nonComparableFuncPolarisNotifyListener) Notify(event *registry.ServiceEvent) {
	if l != nil {
		l(event, nil, nil)
	}
}

func (l nonComparableFuncPolarisNotifyListener) NotifyAll(events []*registry.ServiceEvent, callback func()) {
	if l != nil {
		l(nil, events, callback)
	}
}

func (l nonComparableStructPolarisNotifyListener) Notify(event *registry.ServiceEvent) {
	if l.recorder != nil {
		l.recorder.Notify(event)
	}
}

func (l nonComparableStructPolarisNotifyListener) NotifyAll(events []*registry.ServiceEvent, callback func()) {
	if l.recorder != nil {
		l.recorder.NotifyAll(events, callback)
	}
}

func (l runtimeUnhashablePolarisNotifyListener) Notify(event *registry.ServiceEvent) {
	if l.recorder != nil {
		l.recorder.Notify(event)
	}
}

func (l runtimeUnhashablePolarisNotifyListener) NotifyAll(events []*registry.ServiceEvent, callback func()) {
	if l.recorder != nil {
		l.recorder.NotifyAll(events, callback)
	}
}

func (l comparableValuePolarisNotifyListener) Notify(event *registry.ServiceEvent) {
	l.recorder.Notify(event)
}

func (l comparableValuePolarisNotifyListener) NotifyAll(events []*registry.ServiceEvent, callback func()) {
	l.recorder.NotifyAll(events, callback)
}

func (*nilablePolarisNotifyListener) Notify(*registry.ServiceEvent) {}

func (*nilablePolarisNotifyListener) NotifyAll([]*registry.ServiceEvent, func()) {}

func (nilableChannelPolarisNotifyListener) Notify(*registry.ServiceEvent) {}

func (nilableChannelPolarisNotifyListener) NotifyAll([]*registry.ServiceEvent, func()) {}

func (r *recordingPolarisNotifyListener) Notify(event *registry.ServiceEvent) {
	r.notifications = append(r.notifications, recordedNotification{
		kind:   incrementalNotification,
		events: []recordedPolarisEvent{recordedPolarisEventFromServiceEvent(event)},
	})
	if r.onNotify != nil {
		r.onNotify(event)
	}
}

func (r *recordingPolarisNotifyListener) NotifyAll(events []*registry.ServiceEvent, callback func()) {
	r.notifications = append(r.notifications, recordedNotification{
		kind:        fullSnapshotNotification,
		events:      recordedPolarisEventsFromServiceEvents(events),
		callbackNil: callback == nil,
	})
	if callback != nil {
		callback()
	}
}

func (r *recordingPolarisNotifyListener) recordInstances(action remoting.EventType, instances []model.Instance) {
	if len(instances) == 0 {
		return
	}
	events := make([]recordedPolarisEvent, 0, len(instances))
	for _, instance := range instances {
		events = append(events, recordedPolarisEventFromInstance(action, instance))
	}
	r.notifications = append(r.notifications, recordedNotification{kind: incrementalNotification, events: events})
}

func (r *recordingPolarisNotifyListener) recordedEvents() []recordedPolarisEvent {
	var events []recordedPolarisEvent
	for _, notification := range r.notifications {
		events = append(events, notification.events...)
	}
	return events
}

func TestPolarisInitialSnapshotReconciliation(t *testing.T) {
	serviceName := "com.test.InitialSnapshotService"
	instanceA := newPolarisTestInstance("instance-a", "10.0.0.1", 20001, serviceName, true)
	sameKeyA := newPolarisTestInstance("new-polaris-id", "10.0.0.1", 20001, serviceName, true)
	instanceA.GetMetadata()[testPolarisRevisionKey] = "before"
	sameKeyA.GetMetadata()[testPolarisRevisionKey] = "after"
	invalid := newPolarisTestInstance("invalid", "10.0.0.9", 20009, serviceName, false)
	consumer := &fakePolarisConsumer{
		instanceResponses: [][]model.Instance{{instanceA, invalid}},
		watchInstances:    []model.Instance{sameKeyA},
	}
	pr := newTestPolarisRegistry(consumer)
	notify := &recordingPolarisNotifyListener{}
	if err := pr.LoadSubscribeInstances(newPolarisConsumerURL(serviceName), notify); err != nil {
		t.Fatalf("LoadSubscribeInstances() error = %v", err)
	}
	watcher := mustStoppedRegistryWatcher(t, pr, serviceName)
	listener, err := pr.createPolarisListener(serviceName, notify)
	if err != nil {
		t.Fatalf("createPolarisListener() error = %v", err)
	}
	defer closeTestPolarisListener(listener)
	if err := watcher.watchOnce(); err != nil {
		t.Fatalf("watchOnce() error = %v", err)
	}

	assertPolarisEventsAddress(t, notify.recordedEvents(), []recordedPolarisEvent{{
		action: remoting.EventTypeAdd, host: "10.0.0.1", port: "20001",
	}})
	events := assertPolarisListenerEvents(t, listener, []recordedPolarisEvent{{
		action: remoting.EventTypeAdd, host: "10.0.0.1", port: "20001",
	}})
	assertPolarisEventInstance(t, events[0], remoting.EventTypeAdd, sameKeyA)
	assertPolarisWatchRequest(t, consumer.watchRequest, watcher.subscribeParam, serviceName)

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

func TestPolarisWatchServiceReconnectLifecycle(t *testing.T) {
	serviceName := "com.test.ReconnectSnapshotService"
	instanceA := newPolarisTestInstance("instance-a", "10.0.0.1", 20001, serviceName, true)
	instanceB := newPolarisTestInstance("instance-b", "10.0.0.2", 20002, serviceName, true)
	consumer := newGatedPolarisConsumer()
	pr := newTestPolarisRegistry(consumer)
	watcher := mustStoppedRegistryWatcher(t, pr, serviceName)
	listener, err := newPolarisListener(watcher, nil, reconcileWithBaseline)
	if err != nil {
		t.Fatalf("newPolarisListener() error = %v", err)
	}
	defer closeTestPolarisListener(listener)

	firstRequest := runGatedPolarisWatchOnce(t, watcher, consumer, []model.Instance{instanceA, instanceB})
	assertPolarisListenerEvents(t, listener, []recordedPolarisEvent{
		{action: remoting.EventTypeAdd, host: "10.0.0.1", port: "20001"},
		{action: remoting.EventTypeAdd, host: "10.0.0.2", port: "20002"},
	})
	secondRequest := runGatedPolarisWatchOnce(t, watcher, consumer, []model.Instance{instanceB})
	assertPolarisListenerEvents(t, listener, []recordedPolarisEvent{
		{action: remoting.EventTypeDel, host: "10.0.0.1", port: "20001"},
		{action: remoting.EventTypeAdd, host: "10.0.0.2", port: "20002"},
	})
	assertPolarisWatchRequest(t, firstRequest, watcher.subscribeParam, serviceName)
	assertPolarisWatchRequest(t, secondRequest, watcher.subscribeParam, serviceName)

	t.Run("single instance to empty snapshot", func(t *testing.T) {
		pr := newTestPolarisRegistry(nil)
		watcher := mustStoppedRegistryWatcher(t, pr, serviceName)
		listener, err := newPolarisListener(watcher, nil, reconcileWithBaseline)
		if err != nil {
			t.Fatalf("newPolarisListener() error = %v", err)
		}
		defer closeTestPolarisListener(listener)

		watcher.handleWatchSnapshot([]model.Instance{instanceA})
		assertPolarisListenerEvents(t, listener, []recordedPolarisEvent{{
			action: remoting.EventTypeAdd, host: "10.0.0.1", port: "20001",
		}})

		watcher.handleWatchSnapshot(nil)
		assertPolarisListenerEvents(t, listener, []recordedPolarisEvent{{
			action: remoting.EventTypeDel, host: "10.0.0.1", port: "20001",
		}})
		if len(watcher.currentInstances) != 0 {
			t.Fatalf("current instance count = %d, want 0", len(watcher.currentInstances))
		}
	})
}

func TestPolarisWatchSnapshotIsolatedFromInputMutation(t *testing.T) {
	serviceName := "com.test.InputMutationSnapshotService"
	instanceA := newPolarisTestInstance("instance-a", "10.0.0.1", 20001, serviceName, true)
	wantKey := instanceA.GetInstanceKey()
	wantHost := instanceA.GetHost()
	wantPort := instanceA.GetPort()
	wantInterface := instanceA.GetMetadata()["interface"]
	wantPath := instanceA.GetMetadata()["path"]
	watcher := newStoppedPolarisWatcher(t, nil)
	listener, err := newPolarisListener(watcher, nil, reconcileWithBaseline)
	if err != nil {
		t.Fatalf("newPolarisListener() error = %v", err)
	}
	defer closeTestPolarisListener(listener)

	watcher.handleWatchSnapshot([]model.Instance{instanceA})
	assertPolarisListenerEvent(t, listener, remoting.EventTypeAdd, instanceA)
	if len(watcher.currentInstances) != 1 {
		t.Fatalf("current instance count = %d, want 1", len(watcher.currentInstances))
	}
	current := watcher.currentInstances[0]
	if current == instanceA {
		t.Fatal("watcher current instance shares the input instance")
	}

	delete(instanceA.GetMetadata(), "interface")
	delete(instanceA.GetMetadata(), "path")
	if current.GetMetadata()["interface"] != wantInterface || current.GetMetadata()["path"] != wantPath {
		t.Fatalf("watcher metadata = %v, want interface=%q path=%q", current.GetMetadata(), wantInterface, wantPath)
	}

	watcher.handleWatchSnapshot(nil)
	select {
	case value := <-listener.events.Out():
		notification, ok := value.(*polarisNotification)
		if !ok || notification == nil {
			t.Fatalf("listener notification type = %T, want *polarisNotification", value)
		}
		if notification.eventType != remoting.EventTypeDel || len(notification.instances) != 1 {
			t.Fatalf("listener notification = %#v, want one DEL instance", notification)
		}
		deleted := notification.instances[0]
		if deleted.GetInstanceKey() != wantKey || deleted.GetHost() != wantHost || deleted.GetPort() != wantPort {
			t.Fatalf(
				"deleted instance = (key=%v host=%s port=%d), want (key=%v host=%s port=%d)",
				deleted.GetInstanceKey(),
				deleted.GetHost(),
				deleted.GetPort(),
				wantKey,
				wantHost,
				wantPort,
			)
		}
		if deleted.GetMetadata()["interface"] != wantInterface || deleted.GetMetadata()["path"] != wantPath {
			t.Fatalf("deleted instance metadata = %v, want interface=%q path=%q", deleted.GetMetadata(), wantInterface, wantPath)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for DEL after input mutation")
	}
	assertNoPolarisListenerEvent(t, listener)
	if len(watcher.currentInstances) != 0 {
		t.Fatalf("current instance count = %d, want 0", len(watcher.currentInstances))
	}
}

func TestPolarisSubscriberMutationDoesNotAffectWatcherSnapshot(t *testing.T) {
	serviceName := "com.test.SubscriberMutationSnapshotService"
	instanceA := newPolarisTestInstance("instance-a", "10.0.0.1", 20001, serviceName, true)
	wantInterface := instanceA.GetMetadata()["interface"]
	wantPath := instanceA.GetMetadata()["path"]
	watcher := newStoppedPolarisWatcher(t, nil)
	listener, err := newPolarisListener(watcher, nil, reconcileWithBaseline)
	if err != nil {
		t.Fatalf("newPolarisListener() error = %v", err)
	}
	defer closeTestPolarisListener(listener)

	watcher.AddSubscriber(func(_ remoting.EventType, instances []model.Instance) {
		if len(instances) == 0 {
			return
		}
		delete(instances[0].GetMetadata(), "interface")
		delete(instances[0].GetMetadata(), "path")
	})
	var secondInterface string
	var secondPath string
	watcher.AddSubscriber(func(_ remoting.EventType, instances []model.Instance) {
		if len(instances) == 0 {
			return
		}
		secondInterface = instances[0].GetMetadata()["interface"]
		secondPath = instances[0].GetMetadata()["path"]
	})

	watcher.handleWatchSnapshot([]model.Instance{instanceA})
	assertPolarisListenerEvent(t, listener, remoting.EventTypeAdd, instanceA)
	if secondInterface != wantInterface || secondPath != wantPath {
		t.Fatalf("second subscriber metadata = (interface=%q path=%q), want (interface=%q path=%q)", secondInterface, secondPath, wantInterface, wantPath)
	}
	if len(watcher.currentInstances) != 1 {
		t.Fatalf("current instance count = %d, want 1", len(watcher.currentInstances))
	}
	currentMetadata := watcher.currentInstances[0].GetMetadata()
	if currentMetadata["interface"] != wantInterface || currentMetadata["path"] != wantPath {
		t.Fatalf("watcher metadata = %v, want interface=%q path=%q", currentMetadata, wantInterface, wantPath)
	}

	watcher.handleWatchSnapshot(nil)
	assertPolarisListenerEvent(t, listener, remoting.EventTypeDel, instanceA)
}

func TestPolarisComparableSubscriberDeletesInstanceThatBecomesInvalid(t *testing.T) {
	serviceName := "com.test.ValidToInvalidSnapshotService"
	validA := newPolarisTestInstance("valid-a", "10.0.0.1", 20001, serviceName, true)
	invalidA := newPolarisTestInstance("invalid-a", "10.0.0.1", 20001, serviceName, false)

	t.Run("first reconciliation compares baseline with notifiable current set", func(t *testing.T) {
		pr := newTestPolarisRegistry(&fakePolarisConsumer{
			instanceResponses: [][]model.Instance{{validA}},
		})
		notify := &recordingPolarisNotifyListener{}
		if err := pr.LoadSubscribeInstances(newPolarisConsumerURL(serviceName), notify); err != nil {
			t.Fatalf("LoadSubscribeInstances() error = %v", err)
		}
		watcher := mustStoppedRegistryWatcher(t, pr, serviceName)
		listener, err := pr.createPolarisListener(serviceName, notify)
		if err != nil {
			t.Fatalf("createPolarisListener() error = %v", err)
		}
		defer closeTestPolarisListener(listener)

		watcher.handleWatchSnapshot([]model.Instance{invalidA})
		assertPolarisListenerEvents(t, listener, []recordedPolarisEvent{
			{action: remoting.EventTypeDel, host: "10.0.0.1", port: "20001"},
		})
	})

	t.Run("reconnect deletes previously valid instance before skipping invalid current", func(t *testing.T) {
		watcher := newStoppedPolarisWatcher(t, nil)
		listener, err := newPolarisListener(watcher, nil, reconcileWithBaseline)
		if err != nil {
			t.Fatalf("newPolarisListener() error = %v", err)
		}
		defer closeTestPolarisListener(listener)
		watcher.handleWatchSnapshot([]model.Instance{validA})
		assertPolarisListenerEvent(t, listener, remoting.EventTypeAdd, validA)

		watcher.handleWatchSnapshot([]model.Instance{invalidA})
		assertPolarisListenerEvents(t, listener, []recordedPolarisEvent{
			{action: remoting.EventTypeDel, host: "10.0.0.1", port: "20001"},
		})
	})
}

func TestPolarisRegistrySubscriberUpdateNotificationMatrix(t *testing.T) {
	serviceName := "com.test.UpdateNotificationMatrixService"
	validA := newPolarisTestInstance("valid-a", "10.0.0.1", 20001, serviceName, true)
	updatedA := newPolarisTestInstance("updated-a", "10.0.0.1", 20001, serviceName, true)
	validB := newPolarisTestInstance("valid-b", "10.0.0.2", 20002, serviceName, true)
	invalidA := newPolarisTestInstance("invalid-a", "10.0.0.1", 20001, serviceName, false)
	invalidB := newPolarisTestInstance("invalid-b", "10.0.0.2", 20002, serviceName, false)

	tests := []struct {
		name            string
		before          model.Instance
		after           model.Instance
		wantRegistry    []recordedPolarisEvent
		verifyReconnect bool
		verifyNoRepeat  bool
	}{
		{
			name:   "same key valid to valid updates after",
			before: validA,
			after:  updatedA,
			wantRegistry: []recordedPolarisEvent{
				{action: remoting.EventTypeUpdate, host: "10.0.0.1", port: "20001"},
			},
		},
		{
			name:   "changed key valid to valid deletes before updating",
			before: validA,
			after:  validB,
			wantRegistry: []recordedPolarisEvent{
				{action: remoting.EventTypeDel, host: "10.0.0.1", port: "20001"},
				{action: remoting.EventTypeUpdate, host: "10.0.0.2", port: "20002"},
			},
			verifyReconnect: true,
		},
		{
			name:   "same key valid to invalid deletes before",
			before: validA,
			after:  invalidA,
			wantRegistry: []recordedPolarisEvent{
				{action: remoting.EventTypeDel, host: "10.0.0.1", port: "20001"},
			},
			verifyNoRepeat: true,
		},
		{
			name:   "changed key valid to invalid deletes before",
			before: validA,
			after:  invalidB,
			wantRegistry: []recordedPolarisEvent{
				{action: remoting.EventTypeDel, host: "10.0.0.1", port: "20001"},
			},
		},
		{
			name:   "invalid to valid only updates after",
			before: invalidA,
			after:  validB,
			wantRegistry: []recordedPolarisEvent{
				{action: remoting.EventTypeUpdate, host: "10.0.0.2", port: "20002"},
			},
		},
		{
			name:   "invalid to invalid has no registry notification",
			before: invalidA,
			after:  invalidB,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			watcher := newStoppedPolarisWatcher(t, nil)
			listener, err := newPolarisListener(watcher, nil, reconcileWithBaseline)
			if err != nil {
				t.Fatalf("newPolarisListener() error = %v", err)
			}
			defer closeTestPolarisListener(listener)

			watcher.handleWatchSnapshot([]model.Instance{tt.before})
			if polarisInstanceURLValidationError(tt.before) == "" {
				assertPolarisListenerEvent(t, listener, remoting.EventTypeAdd, tt.before)
			} else {
				assertNoPolarisListenerEvent(t, listener)
			}
			application := &recordingPolarisNotifyListener{}
			watcher.AddSubscriber(application.recordInstances)

			watcher.handleInstanceEvent(&model.InstanceEvent{
				UpdateEvent: &model.InstanceUpdateEvent{UpdateList: []model.OneInstanceUpdate{{
					Before: tt.before,
					After:  tt.after,
				}}},
			})

			var registryEvents []recordedPolarisEvent
			if len(tt.wantRegistry) == 0 {
				assertNoPolarisListenerEvent(t, listener)
			} else {
				registryEvents = assertPolarisIncrementalListenerEvents(t, listener, tt.wantRegistry)
			}
			for _, event := range registryEvents {
				if event.action == remoting.EventTypeUpdate {
					assertPolarisEventInstance(t, event, remoting.EventTypeUpdate, tt.after)
				}
			}
			if len(application.recordedEvents()) != 1 {
				t.Fatalf("application event count = %d, want 1", len(application.recordedEvents()))
			}
			assertPolarisEventInstance(t, application.recordedEvents()[0], remoting.EventTypeUpdate, tt.after)
			if len(watcher.currentInstances) != 1 {
				t.Fatalf("current instance count = %d, want 1", len(watcher.currentInstances))
			}
			assertPolarisInstanceIdentity(t, watcher.currentInstances[0], tt.after)

			if tt.verifyReconnect {
				watcher.handleWatchSnapshot([]model.Instance{tt.after})
				assertPolarisListenerEvents(t, listener, []recordedPolarisEvent{{
					action: remoting.EventTypeAdd,
					host:   tt.after.GetHost(),
					port:   strconv.Itoa(int(tt.after.GetPort())),
				}})
			}
			if tt.verifyNoRepeat {
				watcher.handleWatchSnapshot([]model.Instance{tt.after})
				assertNoPolarisListenerEvent(t, listener)
			}
		})
	}
}

func TestPolarisChangedKeyUpdateBatchPreservesEveryAfterInstance(t *testing.T) {
	serviceName := "com.test.ChangedKeyUpdateBatchService"
	instanceA := newPolarisTestInstance("instance-a", "10.0.0.1", 20001, serviceName, true)
	instanceB := newPolarisTestInstance("instance-b", "10.0.0.2", 20002, serviceName, true)
	movedA := newPolarisTestInstance("moved-a", "10.0.0.2", 20002, serviceName, true)
	movedB := newPolarisTestInstance("moved-b", "10.0.0.3", 20003, serviceName, true)
	watcher := newStoppedPolarisWatcher(t, nil)
	listener, err := newPolarisListener(watcher, nil, reconcileWithBaseline)
	if err != nil {
		t.Fatalf("newPolarisListener() error = %v", err)
	}
	defer closeTestPolarisListener(listener)

	watcher.handleWatchSnapshot([]model.Instance{instanceA, instanceB})
	assertPolarisListenerEvents(t, listener, []recordedPolarisEvent{
		{action: remoting.EventTypeAdd, host: "10.0.0.1", port: "20001"},
		{action: remoting.EventTypeAdd, host: "10.0.0.2", port: "20002"},
	})
	application := &recordingPolarisNotifyListener{}
	watcher.AddSubscriber(application.recordInstances)

	watcher.handleInstanceEvent(&model.InstanceEvent{
		UpdateEvent: &model.InstanceUpdateEvent{UpdateList: []model.OneInstanceUpdate{
			{Before: instanceA, After: movedA},
			{Before: instanceB, After: movedB},
		}},
	})

	registryEvents := assertPolarisIncrementalListenerEvents(t, listener, []recordedPolarisEvent{
		{action: remoting.EventTypeDel, host: "10.0.0.1", port: "20001"},
		{action: remoting.EventTypeDel, host: "10.0.0.2", port: "20002"},
		{action: remoting.EventTypeUpdate, host: "10.0.0.2", port: "20002"},
		{action: remoting.EventTypeUpdate, host: "10.0.0.3", port: "20003"},
	})
	assertPolarisEventInstance(t, registryEvents[2], remoting.EventTypeUpdate, movedA)
	assertPolarisEventInstance(t, registryEvents[3], remoting.EventTypeUpdate, movedB)
	if len(application.recordedEvents()) != 2 {
		t.Fatalf("application event count = %d, want 2", len(application.recordedEvents()))
	}
	assertPolarisEventInstance(t, application.recordedEvents()[0], remoting.EventTypeUpdate, movedA)
	assertPolarisEventInstance(t, application.recordedEvents()[1], remoting.EventTypeUpdate, movedB)
	if len(watcher.currentInstances) != 2 {
		t.Fatalf("current instance count = %d, want 2", len(watcher.currentInstances))
	}
	assertPolarisInstanceIdentity(t, watcher.currentInstances[0], movedA)
	assertPolarisInstanceIdentity(t, watcher.currentInstances[1], movedB)

	watcher.handleWatchSnapshot([]model.Instance{movedB})
	assertPolarisListenerEvents(t, listener, []recordedPolarisEvent{
		{action: remoting.EventTypeDel, host: "10.0.0.2", port: "20002"},
		{action: remoting.EventTypeAdd, host: "10.0.0.3", port: "20003"},
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
	newerA := newPolarisTestInstance("newer-a", "10.0.0.1", 20001, serviceName, true)
	newerA.GetMetadata()[testPolarisRevisionKey] = "newer"

	for _, tt := range []struct {
		name        string
		responses   [][]model.Instance
		current     []model.Instance
		wantPending int
		want        []recordedPolarisEvent
	}{
		{
			name:        "Load A then Load B keeps A for first Watch B",
			responses:   [][]model.Instance{{instanceA}, {instanceB}},
			current:     []model.Instance{instanceB},
			wantPending: 2,
			want: []recordedPolarisEvent{
				{action: remoting.EventTypeDel, host: "10.0.0.1", port: "20001"},
				{action: remoting.EventTypeAdd, host: "10.0.0.2", port: "20002"},
			},
		},
		{
			name:        "empty Load preserves prior pending baseline",
			responses:   [][]model.Instance{{instanceA}, nil},
			wantPending: 1,
			want:        []recordedPolarisEvent{{action: remoting.EventTypeDel, host: "10.0.0.1", port: "20001"}},
		},
		{
			name:        "duplicate InstanceKey keeps latest instance once",
			responses:   [][]model.Instance{{instanceA}, {newerA}},
			wantPending: 1,
			want:        []recordedPolarisEvent{{action: remoting.EventTypeDel, host: "10.0.0.1", port: "20001"}},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			consumer := &fakePolarisConsumer{instanceResponses: tt.responses}
			pr := newTestPolarisRegistry(consumer)
			notify := &recordingPolarisNotifyListener{}
			url := newPolarisConsumerURL(serviceName)
			for range tt.responses {
				if err := pr.LoadSubscribeInstances(url, notify); err != nil {
					t.Fatalf("LoadSubscribeInstances() error = %v", err)
				}
			}

			key := mustInitialSubscribeInstancesKey(t, serviceName, notify)
			entry, pending := pr.loadInitialSubscribeInstances(key)
			if entry == nil || len(pending) != tt.wantPending {
				t.Fatalf("pending baseline = (%p, %v), want %d instances", entry, pending, tt.wantPending)
			}
			if tt.name == "duplicate InstanceKey keeps latest instance once" {
				assertPolarisInstanceIdentity(t, pending[0], newerA)
			}

			watcher := mustStoppedRegistryWatcher(t, pr, serviceName)
			watcher.handleWatchSnapshot(tt.current)
			listener, err := pr.createPolarisListener(serviceName, notify)
			if err != nil {
				t.Fatalf("createPolarisListener() error = %v", err)
			}
			defer closeTestPolarisListener(listener)
			events := assertPolarisListenerEvents(t, listener, tt.want)
			if tt.name == "duplicate InstanceKey keeps latest instance once" {
				assertPolarisEventInstance(t, events[0], remoting.EventTypeDel, newerA)
			}
			if pendingEntry, _ := pr.loadInitialSubscribeInstances(key); pendingEntry != nil {
				t.Fatalf("pending entry after listener success = %p, want nil", pendingEntry)
			}
		})
	}

	t.Run("baseline is stored before Notify", func(t *testing.T) {
		pr := newTestPolarisRegistry(&fakePolarisConsumer{instanceResponses: [][]model.Instance{{instanceA}}})
		baselineWasReady := false
		notify := &recordingPolarisNotifyListener{}
		notify.onNotify = func(*registry.ServiceEvent) {
			key := mustInitialSubscribeInstancesKey(t, serviceName, notify)
			entry, instances := pr.loadInitialSubscribeInstances(key)
			baselineWasReady = entry != nil && len(instances) == 1
		}
		if err := pr.LoadSubscribeInstances(newPolarisConsumerURL(serviceName), notify); err != nil {
			t.Fatalf("LoadSubscribeInstances() error = %v", err)
		}
		if !baselineWasReady {
			t.Fatal("pending baseline was not stored before Notify(ADD)")
		}
	})

	t.Run("stored baseline is isolated from input mutation", func(t *testing.T) {
		baselineA := newPolarisTestInstance("baseline-a", "10.0.0.1", 20001, serviceName, true)
		wantInterface := baselineA.GetMetadata()["interface"]
		wantPath := baselineA.GetMetadata()["path"]
		pr := newTestPolarisRegistry(nil)
		notify := &recordingPolarisNotifyListener{}
		key := mustInitialSubscribeInstancesKey(t, serviceName, notify)

		pr.storeInitialSubscribeInstances(key, []model.Instance{baselineA})
		delete(baselineA.GetMetadata(), "interface")
		delete(baselineA.GetMetadata(), "path")
		_, baseline := pr.loadInitialSubscribeInstances(key)
		if len(baseline) != 1 {
			t.Fatalf("pending baseline count = %d, want 1", len(baseline))
		}
		metadata := baseline[0].GetMetadata()
		if metadata["interface"] != wantInterface || metadata["path"] != wantPath {
			t.Fatalf("pending baseline metadata = %v, want interface=%q path=%q", metadata, wantInterface, wantPath)
		}
	})

	t.Run("Destroy clears pending baseline", func(t *testing.T) {
		pr := newTestPolarisRegistry(nil)
		notify := &recordingPolarisNotifyListener{}
		key := mustInitialSubscribeInstancesKey(t, serviceName, notify)
		pr.storeInitialSubscribeInstances(key, []model.Instance{instanceA})
		pr.Destroy()
		pr.Destroy()
		if len(pr.initialSubscribeInstances) != 0 {
			t.Fatalf("pending entry count after Destroy = %d, want 0", len(pr.initialSubscribeInstances))
		}
	})
}

func TestPolarisInitialSubscribeCompletionPreservesNewerEntry(t *testing.T) {
	serviceName := "com.test.InitialSubscribeCompletionService"
	instanceA := newPolarisTestInstance("instance-a", "10.0.0.1", 20001, serviceName, true)
	instanceB := newPolarisTestInstance("instance-b", "10.0.0.2", 20002, serviceName, true)
	pr := newTestPolarisRegistry(nil)
	notify := &recordingPolarisNotifyListener{}
	key := mustInitialSubscribeInstancesKey(t, serviceName, notify)

	pr.storeInitialSubscribeInstances(key, []model.Instance{instanceA})
	oldEntry, oldBaseline := pr.loadInitialSubscribeInstances(key)
	if oldEntry == nil || len(oldBaseline) != 1 {
		t.Fatalf("old pending entry = (%p, %v), want instance A", oldEntry, oldBaseline)
	}
	assertPolarisInstanceIdentity(t, oldBaseline[0], instanceA)

	pr.storeInitialSubscribeInstances(key, []model.Instance{instanceB})
	newEntry, newBaseline := pr.loadInitialSubscribeInstances(key)
	if newEntry == nil || newEntry == oldEntry {
		t.Fatalf("new pending entry = %p, want non-nil entry distinct from %p", newEntry, oldEntry)
	}
	if len(newBaseline) != 2 {
		t.Fatalf("new pending baseline = %v, want instances A and B", newBaseline)
	}
	assertPolarisInstanceIdentity(t, newBaseline[0], instanceA)
	assertPolarisInstanceIdentity(t, newBaseline[1], instanceB)

	pr.completeInitialSubscribeInstances(key, oldEntry)
	remainingEntry, remainingBaseline := pr.loadInitialSubscribeInstances(key)
	if remainingEntry != newEntry || len(remainingBaseline) != 2 {
		t.Fatalf("pending entry after old completion = (%p, %v), want newer entry %p with A and B", remainingEntry, remainingBaseline, newEntry)
	}
	assertPolarisInstanceIdentity(t, remainingBaseline[0], instanceA)
	assertPolarisInstanceIdentity(t, remainingBaseline[1], instanceB)

	pr.completeInitialSubscribeInstances(key, newEntry)
	entry, instances := pr.loadInitialSubscribeInstances(key)
	if entry != nil || len(instances) != 0 {
		t.Fatalf("pending entry after new completion = (%p, %v), want empty", entry, instances)
	}
}

func TestPolarisSubscribeConsumesTaggedNotifications(t *testing.T) {
	const childEnv = "DUBBO_GO_POLARIS_SUBSCRIBE_TEST_CHILD"
	if os.Getenv(childEnv) == "" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestPolarisSubscribeConsumesTaggedNotifications$", "-test.count=1")
		cmd.Env = append(os.Environ(), childEnv+"=1")
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("Subscribe child test failed: %v\n%s", err, output)
		}
		return
	}

	serviceName := "com.test.SubscribeNotificationPumpService"
	instanceB := newPolarisTestInstance("instance-b", "10.0.0.2", 20002, serviceName, true)
	instanceC := newPolarisTestInstance("instance-c", "10.0.0.3", 20003, serviceName, true)

	t.Run("nil listener is rejected before retry", func(t *testing.T) {
		errCh := make(chan error, 1)
		go func() {
			errCh <- newTestPolarisRegistry(nil).Subscribe(newPolarisConsumerURL(serviceName), nil)
		}()
		select {
		case err := <-errCh:
			if err == nil {
				t.Fatal("Subscribe() error = nil, want nil listener error")
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Subscribe() did not reject nil listener before entering retry loop")
		}
	})

	t.Run("comparable Load baseline reaches public Subscribe FIFO", func(t *testing.T) {
		instanceA := newPolarisTestInstance("instance-a", "10.0.0.1", 20001, serviceName, true)
		consumer := newGatedPolarisConsumer([]model.Instance{instanceA})
		pr := newTestPolarisRegistry(consumer)
		notify := &recordingPolarisNotifyListener{}
		if err := pr.LoadSubscribeInstances(newPolarisConsumerURL(serviceName), notify); err != nil {
			t.Fatalf("LoadSubscribeInstances() error = %v", err)
		}
		watcher := mustStoppedRegistryWatcher(t, pr, serviceName)
		delivered := 0
		notify.onNotify = func(*registry.ServiceEvent) {
			delivered++
			if delivered == 2 {
				panic(stopSubscribePump{})
			}
		}
		stopped := startTestPolarisSubscribe(pr, serviceName, notify)
		waitForPolarisSubscribers(t, watcher, 1)
		request := runGatedPolarisWatchOnce(t, watcher, consumer, []model.Instance{instanceB})
		assertSubscribePumpStopped(t, stopped)

		assertPolarisWatchRequest(t, request, watcher.subscribeParam, serviceName)
		assertPolarisEventsAddress(t, notify.recordedEvents(), []recordedPolarisEvent{
			{action: remoting.EventTypeAdd, host: "10.0.0.1", port: "20001"},
			{action: remoting.EventTypeDel, host: "10.0.0.1", port: "20001"},
			{action: remoting.EventTypeAdd, host: "10.0.0.2", port: "20002"},
		})
		if consumer.getCalls != 1 {
			t.Fatalf("GetInstances() calls = %d, want 1", consumer.getCalls)
		}
	})

	t.Run("full snapshot is consumed before following incremental event", func(t *testing.T) {
		pr := newTestPolarisRegistry(nil)
		watcher := mustStoppedRegistryWatcher(t, pr, serviceName)
		calls := make(chan recordedNotification, 2)
		notify := nonComparableFuncPolarisNotifyListener(func(
			event *registry.ServiceEvent,
			events []*registry.ServiceEvent,
			callback func(),
		) {
			if event != nil {
				calls <- recordedNotification{
					kind:   incrementalNotification,
					events: []recordedPolarisEvent{recordedPolarisEventFromServiceEvent(event)},
				}
				panic(stopSubscribePump{})
			}
			calls <- recordedNotification{
				kind:        fullSnapshotNotification,
				events:      recordedPolarisEventsFromServiceEvents(events),
				callbackNil: callback == nil,
			}
			if callback != nil {
				callback()
			}
		})
		stopped := startTestPolarisSubscribe(pr, serviceName, notify)
		waitForPolarisSubscribers(t, watcher, 1)

		watcher.handleWatchSnapshot([]model.Instance{instanceB})
		watcher.handleInstanceEvent(&model.InstanceEvent{
			AddEvent: &model.InstanceAddEvent{Instances: []model.Instance{instanceC}},
		})

		full := receiveRecordedNotification(t, calls)
		incremental := receiveRecordedNotification(t, calls)
		if full.kind != fullSnapshotNotification || full.callbackNil {
			t.Fatalf("first Subscribe call = %#v, want full with non-nil callback", full)
		}
		assertPolarisEventsAddress(t, full.events, []recordedPolarisEvent{
			{action: remoting.EventTypeUpdate, host: "10.0.0.2", port: "20002"},
		})
		if incremental.kind != incrementalNotification {
			t.Fatalf("second Subscribe call kind = %v, want incremental", incremental.kind)
		}
		assertPolarisEventsAddress(t, incremental.events, []recordedPolarisEvent{
			{action: remoting.EventTypeAdd, host: "10.0.0.3", port: "20003"},
		})
		assertSubscribePumpStopped(t, stopped)
	})

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
			name:    "UPDATE changed key preserves raw application UPDATE",
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
			assertPolarisEventsAddress(t, notify.recordedEvents(), tt.wantEvents)
			if tt.wantInstance != nil {
				assertPolarisEventInstance(t, notify.recordedEvents()[0], remoting.EventTypeUpdate, tt.wantInstance)
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
	watcher := newStoppedPolarisWatcher(t, nil)
	watcher.handleWatchSnapshot([]model.Instance{instanceA, instanceB, instanceC})
	backing := watcher.currentInstances
	watcher.handleInstanceEvent(&model.InstanceEvent{
		DeleteEvent: &model.InstanceDeleteEvent{Instances: []model.Instance{instanceA, instanceB, instanceC}},
	})

	if len(watcher.currentInstances) != 0 {
		t.Fatalf("current instance count = %d, want 0", len(watcher.currentInstances))
	}
	for i, instance := range backing {
		if instance != nil {
			t.Errorf("removed backing slot %d = %v, want nil", i, instance)
		}
	}
	watcher.currentInstances = append(watcher.currentInstances, instanceA)
	if &watcher.currentInstances[0] != &backing[0] {
		t.Fatal("append did not reuse the cleared backing array")
	}
	assertPolarisInstanceIdentity(t, watcher.currentInstances[0], instanceA)
	for i, instance := range backing[1:] {
		if instance != nil {
			t.Errorf("backing slot %d exposed stale instance %v", i+1, instance)
		}
	}
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
		assertPolarisEventsAddress(t, notify.recordedEvents(), want)
	})

	t.Run("reconnect keeps full add behavior without synthetic deletes", func(t *testing.T) {
		watcher := newStoppedPolarisWatcher(t, nil)
		notify := &recordingPolarisNotifyListener{}
		watcher.AddSubscriber(notify.recordInstances)

		watcher.handleWatchSnapshot([]model.Instance{instanceA, instanceB})
		watcher.handleWatchSnapshot([]model.Instance{instanceB})

		want := []recordedPolarisEvent{
			{action: remoting.EventTypeAdd, host: "10.0.0.1", port: "20001"},
			{action: remoting.EventTypeAdd, host: "10.0.0.2", port: "20002"},
			{action: remoting.EventTypeAdd, host: "10.0.0.2", port: "20002"},
		}
		assertPolarisEventsAddress(t, notify.recordedEvents(), want)
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

func TestPolarisListenerNextCompatibility(t *testing.T) {
	serviceName := "com.test.ListenerNextCompatibilityService"
	instanceA := newPolarisTestInstance("instance-a", "10.0.0.1", 20001, serviceName, true)
	instanceB := newPolarisTestInstance("instance-b", "10.0.0.2", 20002, serviceName, true)
	invalid := newPolarisTestInstance("invalid", "10.0.0.9", 20009, serviceName, false)

	t.Run("expands tagged notifications and skips invalid instances", func(t *testing.T) {
		listener := newPolarisListenerState(newStoppedPolarisWatcher(t, nil))
		defer closeTestPolarisListener(listener)
		listener.notify(remoting.EventTypeAdd, []model.Instance{instanceA, invalid, instanceB})
		for _, want := range []model.Instance{instanceA, instanceB} {
			event, err := listener.Next()
			if err != nil {
				t.Fatalf("Next() error = %v", err)
			}
			if event.Action != remoting.EventTypeAdd || event.Service.Ip != want.GetHost() {
				t.Fatalf("Next() event = %#v, want ADD %s", event, want.GetHost())
			}
		}

		listener.notifyFullSnapshot([]model.Instance{invalid, instanceB})
		event, err := listener.Next()
		if err != nil {
			t.Fatalf("Next() full snapshot error = %v", err)
		}
		if event.Action != remoting.EventTypeUpdate || event.Service.Ip != instanceB.GetHost() {
			t.Fatalf("Next() full snapshot event = %#v, want UPDATE %s", event, instanceB.GetHost())
		}

		listener.notifyFullSnapshot(nil)
		listener.notify(remoting.EventTypeUpdate, []model.Instance{instanceA})
		event, err = listener.Next()
		if err != nil {
			t.Fatalf("Next() after empty full snapshot error = %v", err)
		}
		if event.Action != remoting.EventTypeUpdate || event.Service.Ip != instanceA.GetHost() {
			t.Fatalf("Next() after empty full snapshot event = %#v, want UPDATE %s", event, instanceA.GetHost())
		}
	})

	t.Run("Close after partial batch prevents pending drain", func(t *testing.T) {
		listener := newPolarisListenerState(newStoppedPolarisWatcher(t, nil))
		listener.notify(remoting.EventTypeAdd, []model.Instance{instanceA, instanceB})
		event, err := listener.Next()
		if err != nil {
			t.Fatalf("first Next() error = %v", err)
		}
		if event == nil || event.Action != remoting.EventTypeAdd || event.Service.Ip != instanceA.GetHost() {
			t.Fatalf("first Next() event = %#v, want ADD %s", event, instanceA.GetHost())
		}

		listener.Close()
		event, err = listener.Next()
		if event != nil {
			t.Fatalf("Next() after Close event = %#v, want nil", event)
		}
		if err == nil || !strings.Contains(err.Error(), "listener stopped") {
			t.Fatalf("Next() after Close error = %v, want listener stopped", err)
		}
	})
}

func TestPolarisSnapshotsUseDefensiveCopies(t *testing.T) {
	serviceName := "com.test.DefensiveCopyService"
	instanceA := newPolarisTestInstance("instance-a", "10.0.0.1", 20001, serviceName, true)
	instanceB := newPolarisTestInstance("instance-b", "10.0.0.2", 20002, serviceName, true)

	t.Run("watcher current state and callback slices", func(t *testing.T) {
		watcher := newStoppedPolarisWatcher(t, nil)
		watcher.addRegistrySubscriber(nil, reconcileWithBaseline, func(_ remoting.EventType, instances []model.Instance) {
			if len(instances) > 0 {
				instances[0] = instanceB
			}
		}, func([]model.Instance) {})
		current := []model.Instance{instanceA}
		watcher.handleWatchSnapshot(current)
		current[0] = instanceB

		notify := &recordingPolarisNotifyListener{}
		watcher.addRegistrySubscriber(nil, reconcileWithBaseline, notify.recordInstances, func([]model.Instance) {})
		want := []recordedPolarisEvent{{action: remoting.EventTypeAdd, host: "10.0.0.1", port: "20001"}}
		assertPolarisEventsAddress(t, notify.recordedEvents(), want)
	})
}

func TestPolarisNonComparableListenerReconciliation(t *testing.T) {
	serviceName := "com.test.NonComparableListenerService"
	instanceA := newPolarisTestInstance("instance-a", "10.0.0.1", 20001, serviceName, true)
	invalid := newPolarisTestInstance("invalid", "10.0.0.9", 20009, serviceName, false)

	t.Run("nil listeners are rejected before querying instances", func(t *testing.T) {
		var pointer *nilablePolarisNotifyListener
		var slice nonComparablePolarisNotifyListener
		var mapped nonComparableMapPolarisNotifyListener
		var function nonComparableFuncPolarisNotifyListener
		var channel nilableChannelPolarisNotifyListener
		for _, tt := range []struct {
			name   string
			notify registry.NotifyListener
		}{
			{name: "nil interface"},
			{name: "pointer", notify: pointer},
			{name: "slice", notify: slice},
			{name: "map", notify: mapped},
			{name: "func", notify: function},
			{name: "channel", notify: channel},
		} {
			t.Run(tt.name, func(t *testing.T) {
				consumer := &fakePolarisConsumer{instanceResponses: [][]model.Instance{{instanceA}}}
				pr := newTestPolarisRegistry(consumer)
				err := callPolarisRegistryWithoutPanic(t, func() error {
					return pr.LoadSubscribeInstances(newPolarisConsumerURL(serviceName), tt.notify)
				})
				if err == nil {
					t.Fatal("LoadSubscribeInstances() error = nil, want nil listener error")
				}
				if consumer.getCalls != 0 {
					t.Fatalf("GetInstances() calls = %d, want 0", consumer.getCalls)
				}
			})
		}
	})

	t.Run("listener shapes load without pending baseline", func(t *testing.T) {
		for _, tt := range newNonComparablePolarisNotifyListenerCases() {
			t.Run(tt.name, func(t *testing.T) {
				consumer := &fakePolarisConsumer{instanceResponses: [][]model.Instance{{instanceA}}}
				pr := newTestPolarisRegistry(consumer)
				if err := pr.LoadSubscribeInstances(newPolarisConsumerURL(serviceName), tt.notify); err != nil {
					t.Fatalf("LoadSubscribeInstances() error = %v", err)
				}
				if consumer.getCalls != 1 {
					t.Fatalf("GetInstances() calls = %d, want 1", consumer.getCalls)
				}
				if len(pr.initialSubscribeInstances) != 0 {
					t.Fatalf("pending entry count = %d, want 0", len(pr.initialSubscribeInstances))
				}
				assertPolarisEventsAddress(t, tt.recorder.recordedEvents(), []recordedPolarisEvent{{
					action: remoting.EventTypeAdd, host: "10.0.0.1", port: "20001",
				}})
			})
		}
	})

	for _, tt := range []struct {
		name    string
		current []model.Instance
	}{
		{name: "empty snapshot calls NotifyAll", current: nil},
		{name: "all invalid snapshot calls NotifyAll empty", current: []model.Instance{invalid}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			listenerCase := newNonComparablePolarisNotifyListenerCases()[2]
			pr := newTestPolarisRegistry(nil)
			watcher := mustStoppedRegistryWatcher(t, pr, serviceName)
			listener, err := pr.createPolarisListener(serviceName, listenerCase.notify)
			if err != nil {
				t.Fatalf("createPolarisListener() error = %v", err)
			}
			defer closeTestPolarisListener(listener)
			watcher.handleWatchSnapshot(tt.current)
			deliverNextPolarisNotification(t, listener, listenerCase.notify)
			assertPolarisFullSnapshots(t, listenerCase.recorder, [][]recordedPolarisEvent{{}})
		})
	}

}

func TestPolarisNonComparableListenerReplacesSynchronousLoad(t *testing.T) {
	const serviceName = "com.test.NonComparableListenerReplacementService"

	run := func(t *testing.T, current []model.Instance, want []recordedPolarisEvent) {
		t.Helper()
		instanceA := newPolarisTestInstance("instance-a", "10.0.0.1", 20001, serviceName, true)
		consumer := &fakePolarisConsumer{instanceResponses: [][]model.Instance{{instanceA}}}
		pr := newTestPolarisRegistry(consumer)
		recorder := &recordingPolarisNotifyListener{}
		notify := nonComparablePolarisNotifyListener{recorder}

		if err := pr.LoadSubscribeInstances(newPolarisConsumerURL(serviceName), notify); err != nil {
			t.Fatalf("LoadSubscribeInstances() error = %v", err)
		}
		if len(recorder.notifications) != 1 || recorder.notifications[0].kind != incrementalNotification {
			t.Fatalf("notifications after LoadSubscribeInstances() = %#v, want one incremental notification", recorder.notifications)
		}
		assertPolarisEventsAddress(t, recorder.notifications[0].events, []recordedPolarisEvent{{
			action: remoting.EventTypeAdd, host: "10.0.0.1", port: "20001",
		}})
		if len(pr.initialSubscribeInstances) != 0 {
			t.Fatalf("pending entry count = %d, want 0", len(pr.initialSubscribeInstances))
		}

		watcher := mustStoppedRegistryWatcher(t, pr, serviceName)
		listener, err := pr.createPolarisListener(serviceName, notify)
		if err != nil {
			t.Fatalf("createPolarisListener() error = %v", err)
		}
		defer closeTestPolarisListener(listener)

		watcher.handleWatchSnapshot(current)
		deliverNextPolarisNotification(t, listener, notify)
		if len(recorder.notifications) != 2 || recorder.notifications[1].kind != fullSnapshotNotification {
			t.Fatalf("notifications after Watch snapshot = %#v, want incremental load followed by full snapshot", recorder.notifications)
		}
		assertPolarisFullSnapshots(t, recorder, [][]recordedPolarisEvent{want})
	}

	t.Run("A to B", func(t *testing.T) {
		instanceB := newPolarisTestInstance("instance-b", "10.0.0.2", 20002, serviceName, true)
		run(t, []model.Instance{instanceB}, []recordedPolarisEvent{{
			action: remoting.EventTypeUpdate, host: "10.0.0.2", port: "20002",
		}})
	})

	t.Run("A to empty", func(t *testing.T) {
		run(t, nil, []recordedPolarisEvent{})
	})
}

type nonComparablePolarisNotifyListenerCase struct {
	name     string
	notify   registry.NotifyListener
	recorder *recordingPolarisNotifyListener
}

func newNonComparablePolarisNotifyListenerCases() []nonComparablePolarisNotifyListenerCase {
	newFuncListener := func(recorder *recordingPolarisNotifyListener) nonComparableFuncPolarisNotifyListener {
		return func(event *registry.ServiceEvent, events []*registry.ServiceEvent, callback func()) {
			if event != nil {
				recorder.Notify(event)
				return
			}
			recorder.NotifyAll(events, callback)
		}
	}

	sliceRecorder := &recordingPolarisNotifyListener{}
	mapRecorder := &recordingPolarisNotifyListener{}
	funcRecorder := &recordingPolarisNotifyListener{}
	structRecorder := &recordingPolarisNotifyListener{}
	runtimeRecorder := &recordingPolarisNotifyListener{}
	return []nonComparablePolarisNotifyListenerCase{
		{name: "named slice", notify: nonComparablePolarisNotifyListener{sliceRecorder}, recorder: sliceRecorder},
		{name: "named map", notify: nonComparableMapPolarisNotifyListener{"recorder": mapRecorder}, recorder: mapRecorder},
		{name: "named func", notify: newFuncListener(funcRecorder), recorder: funcRecorder},
		{
			name:     "struct containing slice",
			notify:   nonComparableStructPolarisNotifyListener{recorder: structRecorder},
			recorder: structRecorder,
		},
		{
			name:     "runtime-unhashable interface field",
			notify:   runtimeUnhashablePolarisNotifyListener{state: []int{1}, recorder: runtimeRecorder},
			recorder: runtimeRecorder,
		},
	}
}

func TestPolarisNonComparableSubscribersAreIndependent(t *testing.T) {
	serviceName := "com.test.IndependentFullSnapshotService"
	instanceA := newPolarisTestInstance("instance-a", "10.0.0.1", 20001, serviceName, true)

	t.Run("two subscribers receive independent full refresh", func(t *testing.T) {
		pr := newTestPolarisRegistry(nil)
		watcher := mustStoppedRegistryWatcher(t, pr, serviceName)
		first := newNonComparablePolarisNotifyListenerCases()[0]
		second := newNonComparablePolarisNotifyListenerCases()[1]
		firstListener, err := pr.createPolarisListener(serviceName, first.notify)
		if err != nil {
			t.Fatalf("createPolarisListener(first) error = %v", err)
		}
		defer closeTestPolarisListener(firstListener)
		secondListener, err := pr.createPolarisListener(serviceName, second.notify)
		if err != nil {
			t.Fatalf("createPolarisListener(second) error = %v", err)
		}
		defer closeTestPolarisListener(secondListener)

		watcher.handleWatchSnapshot([]model.Instance{instanceA})
		deliverNextPolarisNotification(t, firstListener, first.notify)
		deliverNextPolarisNotification(t, secondListener, second.notify)
		want := [][]recordedPolarisEvent{{{
			action: remoting.EventTypeUpdate, host: "10.0.0.1", port: "20001",
		}}}
		assertPolarisFullSnapshots(t, first.recorder, want)
		assertPolarisFullSnapshots(t, second.recorder, want)
	})

	t.Run("subscriber added after watcher ready immediately receives current snapshot", func(t *testing.T) {
		pr := newTestPolarisRegistry(nil)
		watcher := mustStoppedRegistryWatcher(t, pr, serviceName)
		watcher.handleWatchSnapshot([]model.Instance{instanceA})
		tt := newNonComparablePolarisNotifyListenerCases()[0]
		listener, err := pr.createPolarisListener(serviceName, tt.notify)
		if err != nil {
			t.Fatalf("createPolarisListener() error = %v", err)
		}
		defer closeTestPolarisListener(listener)
		deliverNextPolarisNotification(t, listener, tt.notify)
		assertPolarisFullSnapshots(t, tt.recorder, [][]recordedPolarisEvent{{{
			action: remoting.EventTypeUpdate, host: "10.0.0.1", port: "20001",
		}}})
	})
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
	key, cacheable, err := initialSubscribeInstancesKeyFor(serviceName, notify)
	if err != nil {
		t.Fatalf("initialSubscribeInstancesKeyFor() error = %v", err)
	}
	if !cacheable {
		t.Fatalf("initialSubscribeInstancesKeyFor() cacheable = false, want true")
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

func deliverNextPolarisNotification(
	t *testing.T,
	listener *polarisListener,
	notify registry.NotifyListener,
) {
	t.Helper()
	notification, err := listener.nextNotification()
	if err != nil {
		t.Fatalf("nextNotification() error = %v", err)
	}
	notifyPolarisNotification(notification, notify)
}

func startTestPolarisSubscribe(
	pr *polarisRegistry,
	serviceName string,
	notify registry.NotifyListener,
) <-chan any {
	stopped := make(chan any, 1)
	go func() {
		defer func() {
			stopped <- recover()
		}()
		_ = pr.Subscribe(newPolarisConsumerURL(serviceName), notify)
	}()
	return stopped
}

func waitForPolarisSubscribers(t *testing.T, watcher *PolarisServiceWatcher, want int) {
	t.Helper()
	deadline := time.NewTimer(time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		watcher.lock.Lock()
		count := len(watcher.subscribers)
		watcher.lock.Unlock()
		if count == want {
			return
		}
		select {
		case <-deadline.C:
			t.Fatalf("subscriber count = %d, want %d", count, want)
		case <-ticker.C:
		}
	}
}

func runGatedPolarisWatchOnce(
	t *testing.T,
	watcher *PolarisServiceWatcher,
	consumer *gatedPolarisConsumer,
	instances []model.Instance,
) *api.WatchServiceRequest {
	t.Helper()
	done := make(chan error, 1)
	go func() {
		done <- watcher.watchOnce()
	}()

	var request *api.WatchServiceRequest
	select {
	case request = <-consumer.watchRequests:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for WatchService call")
	}
	events := make(chan model.SubScribeEvent)
	consumer.watchResults <- gatedPolarisWatchResult{response: &model.WatchServiceResponse{
		EventChannel: events,
		GetAllInstancesResp: &model.InstancesResponse{
			Instances: copyInstances(instances),
		},
	}}
	close(events)
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("watchOnce() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("watchOnce() did not stop after EventChannel closed")
	}
	return request
}

func receiveRecordedNotification(t *testing.T, calls <-chan recordedNotification) recordedNotification {
	t.Helper()
	select {
	case call := <-calls:
		return call
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for Subscribe notification")
		return recordedNotification{}
	}
}

func assertSubscribePumpStopped(t *testing.T, stopped <-chan any) {
	t.Helper()
	select {
	case recovered := <-stopped:
		if _, ok := recovered.(stopSubscribePump); !ok {
			t.Fatalf("Subscribe stopped with %#v, want stopSubscribePump", recovered)
		}
	case <-time.After(time.Second):
		t.Fatal("Subscribe notification pump did not stop")
	}
}

func assertPolarisFullSnapshots(
	t *testing.T,
	recorder *recordingPolarisNotifyListener,
	want [][]recordedPolarisEvent,
) {
	t.Helper()
	var full []recordedNotification
	for _, notification := range recorder.notifications {
		if notification.kind == fullSnapshotNotification {
			full = append(full, notification)
		}
	}
	if len(full) != len(want) {
		t.Fatalf("full snapshot count = %d, want %d", len(full), len(want))
	}
	for i := range want {
		assertPolarisEventsAddress(t, full[i].events, want[i])
		if full[i].callbackNil {
			t.Fatalf("NotifyAll() callback %d was nil", i)
		}
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
	for len(actualEvents) < len(events) {
		select {
		case value := <-listener.events.Out():
			notification, ok := value.(*polarisNotification)
			if !ok {
				t.Fatalf("listener event type = %T, want *polarisNotification", value)
			}
			action := notification.eventType
			if notification.kind == fullSnapshotNotification {
				action = remoting.EventTypeUpdate
			}
			for _, instance := range notification.instances {
				actualEvents = append(actualEvents, recordedPolarisEventFromInstance(action, instance))
			}
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for listener events %#v", events)
		}
	}
	assertPolarisEventsAddress(t, actualEvents, events)
	assertNoPolarisListenerEvent(t, listener)
	return actualEvents
}

func assertPolarisIncrementalListenerEvents(
	t *testing.T,
	listener *polarisListener,
	want []recordedPolarisEvent,
) []recordedPolarisEvent {
	t.Helper()

	actual := make([]recordedPolarisEvent, 0, len(want))
	for len(actual) < len(want) {
		select {
		case value := <-listener.events.Out():
			notification, ok := value.(*polarisNotification)
			if !ok || notification == nil {
				t.Fatalf("listener notification type = %T, want *polarisNotification", value)
			}
			if notification.kind != incrementalNotification {
				t.Fatalf("notification kind = %v, want incrementalNotification", notification.kind)
			}
			for _, instance := range notification.instances {
				actual = append(
					actual,
					recordedPolarisEventFromInstance(notification.eventType, instance),
				)
			}
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for incremental listener events %#v", want)
		}
	}

	assertPolarisEventsAddress(t, actual, want)
	assertNoPolarisListenerEvent(t, listener)
	return actual
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

func recordedPolarisEventFromServiceEvent(event *registry.ServiceEvent) recordedPolarisEvent {
	return recordedPolarisEvent{
		action:   event.Action,
		host:     event.Service.Ip,
		port:     event.Service.Port,
		id:       event.Service.GetParam(constant.PolarisInstanceID, ""),
		revision: event.Service.GetParam(testPolarisRevisionKey, ""),
	}
}

func recordedPolarisEventsFromServiceEvents(events []*registry.ServiceEvent) []recordedPolarisEvent {
	recorded := make([]recordedPolarisEvent, 0, len(events))
	for _, event := range events {
		recorded = append(recorded, recordedPolarisEventFromServiceEvent(event))
	}
	return recorded
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
		notification, ok := value.(*polarisNotification)
		if !ok {
			t.Fatalf("listener event type = %T, want *polarisNotification", value)
		}
		if len(notification.instances) != 1 || notification.eventType != action ||
			notification.instances[0].GetInstanceKey() != instance.GetInstanceKey() {
			t.Fatalf("listener event = %#v, want action=%v instance=%v", notification, action, instance)
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
