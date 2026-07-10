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
	"reflect"
	"strconv"
	"testing"
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

const testPolarisNamespace = "test"

type fakePolarisConsumer struct {
	api.ConsumerAPI
	instances []model.Instance
}

func (f *fakePolarisConsumer) GetInstances(_ *api.GetInstancesRequest) (*model.InstancesResponse, error) {
	instances := append([]model.Instance(nil), f.instances...)
	return &model.InstancesResponse{Instances: instances}, nil
}

type recordedPolarisEvent struct {
	action remoting.EventType
	host   string
	port   string
}

type recordingPolarisNotifyListener struct {
	events []recordedPolarisEvent
}

func (r *recordingPolarisNotifyListener) Notify(event *registry.ServiceEvent) {
	r.events = append(r.events, recordedPolarisEvent{
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
		r.events = append(r.events, recordedPolarisEvent{
			action: action,
			host:   instance.GetHost(),
			port:   strconv.Itoa(int(instance.GetPort())),
		})
	}
}

func TestPolarisInitialSnapshotReconciliation(t *testing.T) {
	serviceName := "com.test.InitialSnapshotService"
	instanceA := newPolarisTestInstance("instance-a", "10.0.0.1", 20001, serviceName, true)
	invalidInstance := newPolarisTestInstance("invalid", "10.0.0.9", 20009, serviceName, false)
	instanceB := newPolarisTestInstance("instance-b", "10.0.0.2", 20002, serviceName, true)

	tests := []struct {
		name     string
		current  []model.Instance
		expected []recordedPolarisEvent
	}{
		{
			name:    "A is replaced by B",
			current: []model.Instance{instanceB},
			expected: []recordedPolarisEvent{
				{action: remoting.EventTypeAdd, host: "10.0.0.1", port: "20001"},
				{action: remoting.EventTypeDel, host: "10.0.0.1", port: "20001"},
				{action: remoting.EventTypeAdd, host: "10.0.0.2", port: "20002"},
			},
		},
		{
			name:    "first watcher snapshot is empty",
			current: []model.Instance{},
			expected: []recordedPolarisEvent{
				{action: remoting.EventTypeAdd, host: "10.0.0.1", port: "20001"},
				{action: remoting.EventTypeDel, host: "10.0.0.1", port: "20001"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notify := &recordingPolarisNotifyListener{}
			consumer := &fakePolarisConsumer{instances: []model.Instance{instanceA, invalidInstance}}
			polarisRegistry := &polarisRegistry{
				namespace:        testPolarisNamespace,
				consumer:         consumer,
				initialSnapshots: make(map[string][]model.Instance),
			}

			if err := polarisRegistry.LoadSubscribeInstances(newPolarisConsumerURL(serviceName), notify); err != nil {
				t.Fatalf("LoadSubscribeInstances() error = %v", err)
			}

			initial, ok := polarisRegistry.takeInitialSnapshot(serviceName)
			if !ok {
				t.Fatal("initial snapshot was not saved")
			}
			if len(initial) != 1 || initial[0].GetInstanceKey() != instanceA.GetInstanceKey() {
				t.Fatalf("initial snapshot = %v, want only instance A", initial)
			}

			watcher, err := newPolarisWatcher(nil, consumer)
			if err != nil {
				t.Fatalf("newPolarisWatcher() error = %v", err)
			}
			watcher.setInitialSnapshot(initial)
			watcher.subscribers = append(watcher.subscribers, notify.recordInstances)
			watcher.handleInitialWatchSnapshot(tt.current)

			if !reflect.DeepEqual(notify.events, tt.expected) {
				t.Fatalf("events = %#v, want %#v", notify.events, tt.expected)
			}
		})
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

func TestPolarisWatcherInitialSnapshotConsumedOnce(t *testing.T) {
	serviceName := "com.test.OneShotService"
	instanceA := newPolarisTestInstance("instance-a", "10.0.0.1", 20001, serviceName, true)
	instanceB := newPolarisTestInstance("instance-b", "10.0.0.2", 20002, serviceName, true)
	watcher, err := newPolarisWatcher(nil, nil)
	if err != nil {
		t.Fatalf("newPolarisWatcher() error = %v", err)
	}

	snapshot := []model.Instance{instanceA}
	watcher.setInitialSnapshot(snapshot)
	snapshot[0] = instanceB

	first, ok := watcher.takeInitialSnapshot()
	if !ok {
		t.Fatal("first takeInitialSnapshot() ok = false, want true")
	}
	if len(first) != 1 || first[0].GetInstanceKey() != instanceA.GetInstanceKey() {
		t.Fatalf("first takeInitialSnapshot() = %v, want instance A", first)
	}

	watcher.setInitialSnapshot([]model.Instance{instanceB})
	if second, ok := watcher.takeInitialSnapshot(); ok || len(second) != 0 {
		t.Fatalf("second takeInitialSnapshot() = (%v, %v), want (empty, false)", second, ok)
	}
}

func TestLoadSubscribeInstancesClearsStaleEmptySnapshot(t *testing.T) {
	serviceName := "com.test.EmptyInitialSnapshotService"
	instanceA := newPolarisTestInstance("instance-a", "10.0.0.1", 20001, serviceName, true)
	polarisRegistry := &polarisRegistry{
		namespace:        testPolarisNamespace,
		consumer:         &fakePolarisConsumer{},
		initialSnapshots: make(map[string][]model.Instance),
	}
	polarisRegistry.storeInitialSnapshot(serviceName, []model.Instance{instanceA})

	if err := polarisRegistry.LoadSubscribeInstances(newPolarisConsumerURL(serviceName), &recordingPolarisNotifyListener{}); err != nil {
		t.Fatalf("LoadSubscribeInstances() error = %v", err)
	}
	if snapshot, ok := polarisRegistry.takeInitialSnapshot(serviceName); ok || len(snapshot) != 0 {
		t.Fatalf("takeInitialSnapshot() = (%v, %v), want (empty, false)", snapshot, ok)
	}
}

func TestPolarisRegistryInitialSnapshotUsesDefensiveCopy(t *testing.T) {
	serviceName := "com.test.DefensiveCopyService"
	instanceA := newPolarisTestInstance("instance-a", "10.0.0.1", 20001, serviceName, true)
	instanceB := newPolarisTestInstance("instance-b", "10.0.0.2", 20002, serviceName, true)
	polarisRegistry := &polarisRegistry{initialSnapshots: make(map[string][]model.Instance)}

	snapshot := []model.Instance{instanceA}
	polarisRegistry.storeInitialSnapshot(serviceName, snapshot)
	snapshot[0] = instanceB

	stored, ok := polarisRegistry.takeInitialSnapshot(serviceName)
	if !ok {
		t.Fatal("takeInitialSnapshot() ok = false, want true")
	}
	if len(stored) != 1 || stored[0].GetInstanceKey() != instanceA.GetInstanceKey() {
		t.Fatalf("takeInitialSnapshot() = %v, want instance A", stored)
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
