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
)

import (
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/config_center"
	"github.com/apache/dubbo-go/registry"
	"github.com/apache/dubbo-go/remoting"
)

type dataListener struct {
	interestedURL []*common.URL
	listener      config_center.ConfigurationListener
}

// NewRegistryDataListener ...
func NewRegistryDataListener(listener config_center.ConfigurationListener) *dataListener {
	return &dataListener{listener: listener, interestedURL: []*common.URL{}}
}

func (l *dataListener) AddInterestedURL(url *common.URL) {
	l.interestedURL = append(l.interestedURL, url)
}

func (l *dataListener) DataChange(eventType remoting.Event) bool {

	url := eventType.Path[strings.Index(eventType.Path, "/providers/")+len("/providers/"):]
	serviceURL, err := common.NewURL(url)
	if err != nil {
		logger.Warnf("Listen NewURL(r{%s}) = error{%v}", eventType.Path, err)
		return false
	}

	for _, v := range l.interestedURL {
		if serviceURL.URLEqual(*v) {
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
	registry *etcdV3Registry
	events   chan *config_center.ConfigChangeEvent
}

// NewConfigurationListener for listening the event of etcdv3.
func NewConfigurationListener(reg *etcdV3Registry) *configurationListener {
	// add a new waiter
	reg.WaitGroup().Add(1)
	return &configurationListener{registry: reg, events: make(chan *config_center.ConfigChangeEvent, 32)}
}
func (l *configurationListener) Process(configType *config_center.ConfigChangeEvent) {
	l.events <- configType
}

func (l *configurationListener) Next() (*registry.ServiceEvent, error) {
	for {
		select {
		case <-l.registry.Done():
			logger.Warnf("listener's etcd client connection is broken, so etcd event listener exit now.")
			return nil, perrors.New("listener stopped")

		case e := <-l.events:
			logger.Infof("got etcd event %#v", e)
			if e.ConfigType == remoting.EventTypeDel {
				select {
				case <-l.registry.Done():
					logger.Warnf("update @result{%s}. But its connection to registry is invalid", e.Value)
				default:
				}
				continue
			}
			return &registry.ServiceEvent{Action: e.ConfigType, Service: e.Value.(common.URL)}, nil
		}
	}
}
func (l *configurationListener) Close() {
	l.registry.WaitGroup().Done()
}
