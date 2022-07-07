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
	interestedURL []*common.URL
	listener      config_center.ConfigurationListener
}

// NewRegistryDataListener creates a data listener for etcd
func NewRegistryDataListener(listener config_center.ConfigurationListener) *dataListener {
	return &dataListener{listener: listener}
}

// AddInterestedURL adds a registration @url to listen
func (l *dataListener) AddInterestedURL(url *common.URL) {
	l.interestedURL = append(l.interestedURL, url)
}

// DataChange processes the data change event from registry center of etcd
func (l *dataListener) DataChange(eventType remoting.Event) bool {
	index := strings.Index(eventType.Path, "/providers/")
	if index == -1 {
		logger.Warnf("Listen with no url, event.path={%v}", eventType.Path)
		return false
	}
	url := eventType.Path[index+len("/providers/"):]
	serviceURL, err := common.NewURL(url)
	if err != nil {
		logger.Warnf("Listen NewURL(r{%s}) = error{%v}", eventType.Path, err)
		return false
	}

	for _, v := range l.interestedURL {
		if serviceURL.URLEqual(v) {
			l.listener.Process(
				&config_center.ConfigChangeEvent{
					Key:        eventType.Path,
					Value:      serviceURL,
					ConfigType: eventType.Action,
				},
			)
			return true
		}
	}
	return false
}

type configurationListener struct {
	registry  *etcdV3Registry
	events    *gxchan.UnboundedChan
	closeOnce sync.Once
}

// NewConfigurationListener for listening the event of etcdv3.
func NewConfigurationListener(reg *etcdV3Registry) *configurationListener {
	// add a new waiter
	reg.WaitGroup().Add(1)
	return &configurationListener{registry: reg, events: gxchan.NewUnboundedChan(32)}
}

// Process data change event from config center of etcd
func (l *configurationListener) Process(configType *config_center.ConfigChangeEvent) {
	l.events.In() <- configType
}

// Next returns next service event once received
func (l *configurationListener) Next() (*registry.ServiceEvent, error) {
	for {
		select {
		case <-l.registry.Done():
			logger.Warnf("listener's etcd client connection is broken, so etcd event listener exit now.")
			return nil, perrors.New("listener stopped")

		case val := <-l.events.Out():
			e, _ := val.(*config_center.ConfigChangeEvent)
			logger.Infof("got etcd event %#v", e)
			if e.ConfigType == remoting.EventTypeDel && l.registry.client.Valid() {
				select {
				case <-l.registry.Done():
					logger.Warnf("update @result{%s}. But its connection to registry is invalid", e.Value)
				default:
				}
				continue
			}
			return &registry.ServiceEvent{Action: e.ConfigType, Service: e.Value.(*common.URL)}, nil
		}
	}
}

// Close etcd registry center
func (l *configurationListener) Close() {
	l.closeOnce.Do(func() {
		l.registry.WaitGroup().Done()
	})
}
