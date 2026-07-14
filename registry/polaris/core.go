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
	"sync"
	"time"
)

import (
	api "github.com/polarismesh/polaris-go"
	internalapi "github.com/polarismesh/polaris-go/api"
	"github.com/polarismesh/polaris-go/pkg/model"
)

import (
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

type item func(remoting.EventType, []model.Instance)

type subscriberState struct {
	notify          item
	initialSnapshot []model.Instance
	reconciled      bool
}

type PolarisServiceWatcher struct {
	consumer         api.ConsumerAPI
	subscribeParam   *api.WatchServiceRequest
	lock             *sync.Mutex
	subscribers      []*subscriberState
	execOnce         *sync.Once
	currentInstances []model.Instance
	snapshotReady    bool
}

// newPolarisWatcher create PolarisServiceWatcher to do watch service action
func newPolarisWatcher(param *api.WatchServiceRequest, consumer api.ConsumerAPI) (*PolarisServiceWatcher, error) {
	watcher := &PolarisServiceWatcher{
		subscribeParam: param,
		consumer:       consumer,
		lock:           &sync.Mutex{},
		subscribers:    make([]*subscriberState, 0),
		execOnce:       &sync.Once{},
	}
	return watcher, nil
}

// AddSubscriber add subscriber into watcher's subscribers
func (watcher *PolarisServiceWatcher) AddSubscriber(
	subscriber func(remoting.EventType, []model.Instance),
) {
	state := &subscriberState{
		notify:     item(subscriber),
		reconciled: true,
	}

	func() {
		watcher.lock.Lock()
		defer watcher.lock.Unlock()

		watcher.subscribers = append(watcher.subscribers, state)
	}()

	watcher.lazyRun()
}

// lazyRun Delayed execution, only triggered when AddSubscriber is called, and will only be executed once
func (watcher *PolarisServiceWatcher) lazyRun() {
	watcher.execOnce.Do(func() {
		go watcher.startWatch()
	})
}

func (watcher *PolarisServiceWatcher) addSubscriberWithInitialSnapshot(
	initialSnapshot []model.Instance,
	subscriber item,
) {
	state := &subscriberState{
		notify:          subscriber,
		initialSnapshot: copyInstances(initialSnapshot),
	}

	func() {
		watcher.lock.Lock()
		defer watcher.lock.Unlock()

		watcher.subscribers = append(watcher.subscribers, state)
		if watcher.snapshotReady {
			watcher.reconcileSubscriberLocked(state)
		}
	}()

	// Start only after the subscriber is registered and any available current
	// snapshot has been replayed.
	watcher.lazyRun()
}

// missingInitialInstances returns initial - current by model.InstanceKey while
// preserving the order of the initial snapshot.
func missingInitialInstances(initial []model.Instance, current []model.Instance) []model.Instance {
	currentInstances := make(map[model.InstanceKey]struct{}, len(current))
	for _, instance := range current {
		currentInstances[instance.GetInstanceKey()] = struct{}{}
	}

	missing := make([]model.Instance, 0, len(initial))
	for _, instance := range initial {
		if _, ok := currentInstances[instance.GetInstanceKey()]; !ok {
			missing = append(missing, instance)
		}
	}
	return missing
}

// handleWatchSnapshot replaces the watcher's current state and reconciles each
// subscriber's own synchronous-load baseline exactly once.
func (watcher *PolarisServiceWatcher) handleWatchSnapshot(current []model.Instance) {
	watcher.lock.Lock()
	defer watcher.lock.Unlock()

	watcher.currentInstances = copyInstances(current)
	watcher.snapshotReady = true
	for _, subscriber := range watcher.subscribers {
		if subscriber.reconciled {
			watcher.notifySubscriberLocked(subscriber, remoting.EventTypeAdd, watcher.currentInstances)
			continue
		}
		watcher.reconcileSubscriberLocked(subscriber)
	}
}

func (watcher *PolarisServiceWatcher) reconcileSubscriberLocked(subscriber *subscriberState) {
	missing := missingInitialInstances(subscriber.initialSnapshot, watcher.currentInstances)
	if len(missing) > 0 {
		watcher.notifySubscriberLocked(subscriber, remoting.EventTypeDel, missing)
	}
	watcher.notifySubscriberLocked(subscriber, remoting.EventTypeAdd, watcher.currentInstances)
	subscriber.reconciled = true
	subscriber.initialSnapshot = nil
}

// startWatch start run work to watch target service by polaris
func (watcher *PolarisServiceWatcher) startWatch() {
	for {
		if err := watcher.watchOnce(); err != nil {
			time.Sleep(time.Duration(500 * time.Millisecond))
		}
	}
}

func (watcher *PolarisServiceWatcher) watchOnce() error {
	resp, err := watcher.consumer.WatchService(watcher.subscribeParam)
	if err != nil {
		return err
	}
	watcher.handleWatchSnapshot(resp.GetAllInstancesResp.Instances)
	for event := range resp.EventChannel {
		if event.GetSubScribeEventType() == internalapi.EventInstance {
			watcher.handleInstanceEvent(event.(*model.InstanceEvent))
		}
	}
	return nil
}

func (watcher *PolarisServiceWatcher) handleInstanceEvent(event *model.InstanceEvent) {
	if event == nil {
		return
	}

	watcher.lock.Lock()
	defer watcher.lock.Unlock()

	if event.AddEvent != nil {
		instances := copyInstances(event.AddEvent.Instances)
		for _, instance := range instances {
			watcher.upsertCurrentInstanceLocked(instance)
		}
		watcher.notifyReconciledSubscribersLocked(remoting.EventTypeAdd, instances)
	}
	if event.UpdateEvent != nil {
		instances := make([]model.Instance, 0, len(event.UpdateEvent.UpdateList))
		for _, update := range event.UpdateEvent.UpdateList {
			if update.Before.GetInstanceKey() != update.After.GetInstanceKey() {
				watcher.removeCurrentInstancesLocked([]model.Instance{update.Before})
			}
			watcher.upsertCurrentInstanceLocked(update.After)
			instances = append(instances, update.After)
		}
		watcher.notifyReconciledSubscribersLocked(remoting.EventTypeUpdate, instances)
	}
	if event.DeleteEvent != nil {
		instances := copyInstances(event.DeleteEvent.Instances)
		watcher.removeCurrentInstancesLocked(instances)
		watcher.notifyReconciledSubscribersLocked(remoting.EventTypeDel, instances)
	}
}

func (watcher *PolarisServiceWatcher) upsertCurrentInstanceLocked(instance model.Instance) {
	key := instance.GetInstanceKey()
	for i := range watcher.currentInstances {
		if watcher.currentInstances[i].GetInstanceKey() == key {
			watcher.currentInstances[i] = instance
			return
		}
	}
	watcher.currentInstances = append(watcher.currentInstances, instance)
}

func (watcher *PolarisServiceWatcher) removeCurrentInstancesLocked(instances []model.Instance) {
	keys := make(map[model.InstanceKey]struct{}, len(instances))
	for _, instance := range instances {
		keys[instance.GetInstanceKey()] = struct{}{}
	}

	old := watcher.currentInstances
	current := old[:0]
	for _, instance := range old {
		if _, remove := keys[instance.GetInstanceKey()]; !remove {
			current = append(current, instance)
		}
	}
	clear(old[len(current):])
	watcher.currentInstances = current
}

func (watcher *PolarisServiceWatcher) notifyReconciledSubscribersLocked(eventType remoting.EventType, instances []model.Instance) {
	for _, subscriber := range watcher.subscribers {
		if subscriber.reconciled {
			watcher.notifySubscriberLocked(subscriber, eventType, instances)
		}
	}
}

func (watcher *PolarisServiceWatcher) notifySubscriberLocked(
	subscriber *subscriberState,
	eventType remoting.EventType,
	instances []model.Instance,
) {
	subscriber.notify(eventType, copyInstances(instances))
}

func copyInstances(instances []model.Instance) []model.Instance {
	return append([]model.Instance(nil), instances...)
}
