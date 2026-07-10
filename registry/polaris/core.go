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
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

type item func(remoting.EventType, []model.Instance)

type PolarisServiceWatcher struct {
	consumer                api.ConsumerAPI
	subscribeParam          *api.WatchServiceRequest
	lock                    *sync.RWMutex
	subscribers             []item
	execOnce                *sync.Once
	initialSnapshotLock     sync.Mutex
	initialSnapshot         []model.Instance
	hasInitialSnapshot      bool
	initialSnapshotConsumed bool
}

// newPolarisWatcher create PolarisServiceWatcher to do watch service action
func newPolarisWatcher(param *api.WatchServiceRequest, consumer api.ConsumerAPI) (*PolarisServiceWatcher, error) {
	watcher := &PolarisServiceWatcher{
		subscribeParam: param,
		consumer:       consumer,
		lock:           &sync.RWMutex{},
		subscribers:    make([]item, 0),
		execOnce:       &sync.Once{},
	}
	return watcher, nil
}

// AddSubscriber add subscriber into watcher's subscribers
func (watcher *PolarisServiceWatcher) AddSubscriber(subscriber func(remoting.EventType, []model.Instance)) {

	watcher.lock.Lock()
	watcher.lazyRun()
	defer watcher.lock.Unlock()

	watcher.subscribers = append(watcher.subscribers, subscriber)
}

// lazyRun Delayed execution, only triggered when AddSubscriber is called, and will only be executed once
func (watcher *PolarisServiceWatcher) lazyRun() {
	watcher.execOnce.Do(func() {
		go watcher.startWatch()
	})
}

// setInitialSnapshot stores synchronously loaded instances as the baseline for
// reconciling the first successful watcher full snapshot.
func (watcher *PolarisServiceWatcher) setInitialSnapshot(instances []model.Instance) {
	watcher.initialSnapshotLock.Lock()
	defer watcher.initialSnapshotLock.Unlock()

	// A watcher is reused per service. Never re-arm reconciliation after its first successful snapshot.
	if watcher.initialSnapshotConsumed {
		return
	}
	watcher.initialSnapshot = append([]model.Instance(nil), instances...)
	watcher.hasInitialSnapshot = true
}

// takeInitialSnapshot returns and consumes the baseline for the first successful watcher full snapshot.
// Failed WatchService attempts do not call it, and a consumed watcher cannot be armed again.
func (watcher *PolarisServiceWatcher) takeInitialSnapshot() ([]model.Instance, bool) {
	watcher.initialSnapshotLock.Lock()
	defer watcher.initialSnapshotLock.Unlock()

	if watcher.initialSnapshotConsumed {
		return nil, false
	}
	watcher.initialSnapshotConsumed = true
	if !watcher.hasInitialSnapshot {
		return nil, false
	}

	instances := append([]model.Instance(nil), watcher.initialSnapshot...)
	watcher.initialSnapshot = nil
	watcher.hasInitialSnapshot = false
	return instances, true
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

// handleInitialWatchSnapshot reconciles the first successful watcher full snapshot
// with the synchronous baseline before publishing the current instances.
func (watcher *PolarisServiceWatcher) handleInitialWatchSnapshot(current []model.Instance) {
	if initial, ok := watcher.takeInitialSnapshot(); ok {
		missing := missingInitialInstances(initial, current)
		if len(missing) > 0 {
			watcher.notifyAllSubscriber(&config_center.ConfigChangeEvent{
				Value:      missing,
				ConfigType: remoting.EventTypeDel,
			})
		}
	}

	watcher.notifyAllSubscriber(&config_center.ConfigChangeEvent{
		Value:      current,
		ConfigType: remoting.EventTypeAdd,
	})
}

// startWatch start run work to watch target service by polaris
func (watcher *PolarisServiceWatcher) startWatch() {
	for {
		resp, err := watcher.consumer.WatchService(watcher.subscribeParam)
		if err != nil {
			time.Sleep(time.Duration(500 * time.Millisecond))
			continue
		}
		watcher.handleInitialWatchSnapshot(resp.GetAllInstancesResp.Instances)

		for event := range resp.EventChannel {
			eType := event.GetSubScribeEventType()
			if eType == internalapi.EventInstance {
				insEvent := event.(*model.InstanceEvent)

				if insEvent.AddEvent != nil {
					watcher.notifyAllSubscriber(&config_center.ConfigChangeEvent{
						Value:      insEvent.AddEvent.Instances,
						ConfigType: remoting.EventTypeAdd,
					})
				}
				if insEvent.UpdateEvent != nil {
					instances := make([]model.Instance, len(insEvent.UpdateEvent.UpdateList))
					for i := range insEvent.UpdateEvent.UpdateList {
						instances[i] = insEvent.UpdateEvent.UpdateList[i].After
					}
					watcher.notifyAllSubscriber(&config_center.ConfigChangeEvent{
						Value:      instances,
						ConfigType: remoting.EventTypeUpdate,
					})
				}
				if insEvent.DeleteEvent != nil {
					watcher.notifyAllSubscriber(&config_center.ConfigChangeEvent{
						Value:      insEvent.DeleteEvent.Instances,
						ConfigType: remoting.EventTypeDel,
					})
				}
			}
		}

	}
}

// notifyAllSubscriber notify config_center.ConfigChangeEvent to all subscriber
func (watcher *PolarisServiceWatcher) notifyAllSubscriber(event *config_center.ConfigChangeEvent) {
	watcher.lock.RLock()
	defer watcher.lock.RUnlock()

	for i := 0; i < len(watcher.subscribers); i++ {
		subscriber := watcher.subscribers[i]
		subscriber(event.ConfigType, event.Value.([]model.Instance))
	}

}
