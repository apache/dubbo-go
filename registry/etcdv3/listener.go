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

package etcdv3

import (
	"strings"
	"sync"
)

import (
	gxchan "github.com/dubbogo/gost/container/chan"
	"github.com/dubbogo/gost/log/logger"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

type dataListener struct {
	subscribed map[string]config_center.ConfigurationListener
	mutex      sync.Mutex
	closed     bool
}

// NewRegistryDataListener creates a data listener for etcd
func NewRegistryDataListener() *dataListener {
	return &dataListener{
		subscribed: make(map[string]config_center.ConfigurationListener),
	}
}

// SubscribeURL is used to set a watch listener for url
func (l *dataListener) SubscribeURL(url *common.URL, listener config_center.ConfigurationListener) {
	if l.closed {
		return
	}
	l.subscribed[url.ServiceKey()] = listener
}

// UnSubscribeURL is used to unset a watch listener for url
func (l *dataListener) UnSubscribeURL(url *common.URL) config_center.ConfigurationListener {
	if l.closed {
		return nil
	}
	listener := l.subscribed[url.ServiceKey()]
	if listener == nil {
		return nil
	}
	listener.(*configurationListener).Close()
	delete(l.subscribed, url.ServiceKey())
	return listener
}

// DataChange processes the data change event from registry center of etcd
func (l *dataListener) DataChange(eventType remoting.Event) bool {
	index := strings.Index(eventType.Path, "/providers/")
	if index == -1 {
		logger.Warnf("[Registry][Etcdv3] listen with no url, event.path=%v", eventType.Path)
		return false
	}
	url := eventType.Path[index+len("/providers/"):]
	serviceURL, err := common.NewURL(url)
	if err != nil {
		logger.Warnf("[Registry][Etcdv3] listen NewURL, path=%s err=%v", eventType.Path, err)
		return false
	}
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if l.closed {
		return false
	}
	match := false
	for serviceKey, listener := range l.subscribed {
		intf, group, version := common.ParseServiceKey(serviceKey)
		if serviceURL.ServiceKey() == serviceKey || common.IsAnyCondition(intf, group, version, serviceURL) {
			listener.Process(
				&config_center.ConfigChangeEvent{
					Key:        eventType.Path,
					Value:      serviceURL.Clone(),
					ConfigType: eventType.Action,
				},
			)
			match = true
		}
	}
	return match
}

// Close all subscribed listeners
func (l *dataListener) Close() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.closed = true
	for _, listener := range l.subscribed {
		listener.(*configurationListener).Close()
	}
}

type configurationListener struct {
	registry  *etcdV3Registry
	events    *gxchan.UnboundedChan
	isClosed  bool
	close     chan struct{}
	closeOnce sync.Once
}

// NewConfigurationListener for listening the event of etcdv3.
func NewConfigurationListener(reg *etcdV3Registry) *configurationListener {
	// add a new waiter
	reg.WaitGroup().Add(1)
	return &configurationListener{registry: reg, events: gxchan.NewUnboundedChan(32), close: make(chan struct{}, 1)}
}

// Process data change event from config center of etcd
func (l *configurationListener) Process(configType *config_center.ConfigChangeEvent) {
	l.events.In() <- configType
}

// Next returns next service event once received
func (l *configurationListener) Next() (*registry.ServiceEvent, error) {
	for {
		select {
		case <-l.close:
			logger.Warn("[Registry][Etcdv3] listener has been closed")
			return nil, perrors.New("listener has been closed")
		case <-l.registry.Done():
			logger.Warn("[Registry][Etcdv3] listener's etcd client connection is broken, so etcd event listener exit now")
			return nil, perrors.New("listener stopped")

		case val := <-l.events.Out():
			e, _ := val.(*config_center.ConfigChangeEvent)
			logger.Infof("[Registry][Etcdv3] got etcd event %#v", e)
			if e.ConfigType == remoting.EventTypeDel && l.registry.client.Valid() {
				select {
				case <-l.registry.Done():
					logger.Warnf("[Registry][Etcdv3] update @result{%s}. But its connection to registry is invalid", e.Value)
				default:
				}
				continue
			}
			return &registry.ServiceEvent{Action: e.ConfigType, Service: e.Value.(*common.URL)}, nil
		}
	}
}

// Close closes the listener only once
func (l *configurationListener) Close() {
	// ensure that the listener will be closed at most once.
	l.closeOnce.Do(func() {
		l.isClosed = true
		l.close <- struct{}{}
		l.registry.WaitGroup().Done()
	})
}
