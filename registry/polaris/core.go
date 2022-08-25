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
	"github.com/polarismesh/polaris-go/api"
	"github.com/polarismesh/polaris-go/pkg/model"
)

import (
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

type item func(remoting.EventType, []model.Instance)

type PolarisServiceWatcher struct {
	consumer       api.ConsumerAPI
	subscribeParam *api.WatchServiceRequest
	lock           *sync.RWMutex
	subscribers    []item
	execOnce       *sync.Once
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

// startWatch start run work to watch target service by polaris
func (watcher *PolarisServiceWatcher) startWatch() {
	for {
		resp, err := watcher.consumer.WatchService(watcher.subscribeParam)
		if err != nil {
			time.Sleep(time.Duration(500 * time.Millisecond))
			continue
		}
		watcher.notifyAllSubscriber(&config_center.ConfigChangeEvent{
			Value:      resp.GetAllInstancesResp.Instances,
			ConfigType: remoting.EventTypeAdd,
		})

		for {
			select {
			case event := <-resp.EventChannel:
				eType := event.GetSubScribeEventType()
				if eType == api.EventInstance {
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
