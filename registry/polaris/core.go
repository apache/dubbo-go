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

//@Author: chuntaojun <liaochuntao@live.com>
//@Description:
//@Time: 2021/11/26 00:28

package polaris

import (
	"sync"
	"sync/atomic"
	"time"

	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/remoting"
	gxchan "github.com/dubbogo/gost/container/chan"
	gxset "github.com/dubbogo/gost/container/set"
	perrors "github.com/pkg/errors"
	"github.com/polarismesh/polaris-go/api"
	"github.com/polarismesh/polaris-go/pkg/model"
)

// PolarisServiceWatcher
type PolarisServiceWatcher struct {
	consumer       api.ConsumerAPI
	subscribeParam *api.WatchServiceRequest
	events         *gxchan.UnboundedChan
	lock           *sync.RWMutex
	subscribers    []func(remoting.EventType, []model.Instance)
	isRun          int32
}

// NewPolarisWatcher
func newPolarisWatcher(param *api.WatchServiceRequest, consumer api.ConsumerAPI) (*PolarisServiceWatcher, error) {
	watcher := &PolarisServiceWatcher{
		subscribeParam: param,
		consumer:       consumer,
		events:         gxchan.NewUnboundedChan(1024),
		lock:           &sync.RWMutex{},
		subscribers:    make([]func(remoting.EventType, []model.Instance), 0),
	}
	return watcher, nil
}

// AddSubscriber
func (watcher *PolarisServiceWatcher) AddSubscriber(subscriber func(remoting.EventType, []model.Instance)) {

	watcher.lazyRun()

	watcher.lock.Lock()
	defer watcher.lock.Unlock()

	watcher.subscribers = append(watcher.subscribers, subscriber)
}

// Run
func (watcher *PolarisServiceWatcher) lazyRun() {
	if atomic.CompareAndSwapInt32(&watcher.isRun, 0, 1) {
		go watcher.startWatcher()
		go watcher.startDispatcher()
	}
}

func (watcher *PolarisServiceWatcher) startDispatcher() {
	for {
		select {
		case val := <-watcher.events.Out():
			event := val.(*config_center.ConfigChangeEvent)

			func(event *config_center.ConfigChangeEvent) {

				watcher.lock.RLock()
				defer watcher.lock.RUnlock()

				for i := 0; i < len(watcher.subscribers); i++ {
					subscriber := watcher.subscribers[i]
					subscriber(event.ConfigType, event.Value.([]model.Instance))
				}

			}(event)
		}
	}
}

func (watcher *PolarisServiceWatcher) startWatcher() {

	for {
		resp, err := watcher.consumer.WatchService(watcher.subscribeParam)
		if err != nil {
			time.Sleep(time.Duration(500 * time.Millisecond))
			continue
		}

		watcher.events.In() <- &config_center.ConfigChangeEvent{
			Value:      resp.GetAllInstancesResp.Instances,
			ConfigType: remoting.EventTypeAdd,
		}

		select {
		case event := <-resp.EventChannel:
			eType := event.GetSubScribeEventType()
			if eType == api.EventInstance {
				insEvent := event.(*model.InstanceEvent)
				if insEvent.AddEvent != nil {
					watcher.events.In() <- &config_center.ConfigChangeEvent{
						Value:      insEvent.AddEvent.Instances,
						ConfigType: remoting.EventTypeAdd,
					}
				}
				if insEvent.UpdateEvent != nil {
					instances := make([]model.Instance, len(insEvent.UpdateEvent.UpdateList))
					for i := range insEvent.UpdateEvent.UpdateList {
						instances[i] = insEvent.UpdateEvent.UpdateList[i].After
					}

					watcher.events.In() <- &config_center.ConfigChangeEvent{
						Value:      instances,
						ConfigType: remoting.EventTypeUpdate,
					}
				}
				if insEvent.DeleteEvent != nil {
					watcher.events.In() <- &config_center.ConfigChangeEvent{
						Value:      insEvent.DeleteEvent.Instances,
						ConfigType: remoting.EventTypeDel,
					}
				}
			}
		}
	}
}

type SubscribeRunMode int

const (
	ModeForNotify                      SubscribeRunMode = 1
	ModeForWatchServiceInstancesChange SubscribeRunMode = 2
)

type SubscribeWorker struct {
	mode            SubscribeRunMode
	listener        *polarisListener
	lock            *sync.RWMutex
	exec            int32
	notifyListeners *gxset.HashSet
}

func newSubscribeWorker(mode SubscribeRunMode, listener *polarisListener) *SubscribeWorker {
	return &SubscribeWorker{
		mode:            mode,
		listener:        listener,
		lock:            &sync.RWMutex{},
		notifyListeners: gxset.NewSet(nil),
	}
}

func (holder *SubscribeWorker) addNotifyListener(notifyListener registry.NotifyListener) *ListenerWrapper {
	holder.lock.Lock()
	defer holder.lock.Unlock()

	wrapper := &ListenerWrapper{
		errCh:          make(chan error),
		notifyListener: notifyListener,
	}

	holder.notifyListeners.Add(wrapper)

	if atomic.CompareAndSwapInt32(&holder.exec, 0, 1) {
		go holder.runInTargetMode()
	}

	return wrapper
}

func (holder *SubscribeWorker) removeNotifyListener(notifyListener registry.NotifyListener) {
	holder.lock.Lock()
	defer holder.lock.Unlock()

	holder.notifyListeners.Remove(notifyListener)
}

func (holder *SubscribeWorker) runInTargetMode() {
	if holder.mode == ModeForNotify {
		holder.runInNotifyListenerMode()
	} else {

	}
}

func (holder *SubscribeWorker) runInNotifyListenerMode() {
	for {
		func() {
			serviceEvent, err := holder.listener.Next()
			if err != nil {
				logger.Warnf("Selector.watch() = error{%v}", perrors.WithStack(err))
				holder.listener.Close()
				for _, item := range holder.notifyListeners.Values() {
					wrapper := item.(*ListenerWrapper)
					wrapper.errCh <- err
				}
			}
			logger.Infof("update begin, service event: %v", serviceEvent.String())
			for _, item := range holder.notifyListeners.Values() {
				wrapper := item.(*ListenerWrapper)
				wrapper.notifyListener.Notify(serviceEvent)
			}
		}()
	}
}

type ListenerWrapper struct {
	notifyListener  registry.NotifyListener
	changedListener registry.ServiceInstancesChangedListener
	errCh           chan error
}
