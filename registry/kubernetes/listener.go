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

package kubernetes

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

// NewRegistryDataListener creates a data listener for kubernetes
func NewRegistryDataListener(listener config_center.ConfigurationListener) *dataListener {
	return &dataListener{listener: listener}
}

// AddInterestedURL adds the @url of registry center to the listener
func (l *dataListener) AddInterestedURL(url *common.URL) {
	l.interestedURL = append(l.interestedURL, url)
}

// DataChange
// notify listen, when interest event
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
	registry *kubernetesRegistry
	events   chan *config_center.ConfigChangeEvent
}

// NewConfigurationListener for listening the event of kubernetes.
func NewConfigurationListener(reg *kubernetesRegistry) *configurationListener {
	// add a new waiter
	reg.WaitGroup().Add(1)
	return &configurationListener{registry: reg, events: make(chan *config_center.ConfigChangeEvent, 32)}
}

// Process processes the data change event from config center of kubernetes
func (l *configurationListener) Process(configType *config_center.ConfigChangeEvent) {
	l.events <- configType
}

// Next returns next service event once received
func (l *configurationListener) Next() (*registry.ServiceEvent, error) {
	for {
		select {
		case <-l.registry.Done():
			logger.Warnf("listener's kubernetes client connection is broken, so kubernetes event listener exits now.")
			return nil, perrors.New("listener stopped")

		case e := <-l.events:
			logger.Debugf("got kubernetes event %#v", e)
			if e.ConfigType == remoting.EventTypeDel && !l.registry.client.Valid() {
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

// Close kubernetes registry center
func (l *configurationListener) Close() {
	l.registry.WaitGroup().Done()
}
