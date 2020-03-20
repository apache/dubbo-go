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

package zookeeper

import (
	"strings"
	"sync"
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
	zk "github.com/apache/dubbo-go/remoting/zookeeper"
)

// RegistryDataListener ...
type RegistryDataListener struct {
	subscribed map[*common.URL]config_center.ConfigurationListener
	listener   config_center.ConfigurationListener
}

// NewRegistryDataListener ...
func NewRegistryDataListener() *RegistryDataListener {
	return &RegistryDataListener{
		subscribed: make(map[*common.URL]config_center.ConfigurationListener)}
}

// SubscribeURL is used to set a watch listener for url
func (l *RegistryDataListener) SubscribeURL(url *common.URL, listener config_center.ConfigurationListener) {
	l.subscribed[url] = listener
}

// DataChange ...
func (l *RegistryDataListener) DataChange(eventType remoting.Event) bool {
	// Intercept the last bit
	index := strings.Index(eventType.Path, "/providers/")
	if index == -1 {
		logger.Warnf("Listen with no url, event.path={%v}", eventType.Path)
		return false
	}
	url := eventType.Path[index+len("/providers/"):]
	serviceURL, err := common.NewURL(url)
	if err != nil {
		logger.Errorf("Listen NewURL(r{%s}) = error{%v} eventType.Path={%v}", url, err, eventType.Path)
		return false
	}
	for url, listener := range l.subscribed {
		if serviceURL.URLEqual(*url) {
			listener.Process(
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

func (l *RegistryDataListener) Close() {
	for _, listener := range l.subscribed {
		listener.(*RegistryConfigurationListener).Close()
	}
}

// RegistryConfigurationListener ...
type RegistryConfigurationListener struct {
	client    *zk.ZookeeperClient
	registry  *zkRegistry
	events    chan *config_center.ConfigChangeEvent
	isClosed  bool
	closeOnce sync.Once
}

// NewRegistryConfigurationListener for listening the event of zk.
func NewRegistryConfigurationListener(client *zk.ZookeeperClient, reg *zkRegistry) *RegistryConfigurationListener {
	reg.WaitGroup().Add(1)
	return &RegistryConfigurationListener{client: client, registry: reg, events: make(chan *config_center.ConfigChangeEvent, 32), isClosed: false}
}

// Process ...
func (l *RegistryConfigurationListener) Process(configType *config_center.ConfigChangeEvent) {
	l.events <- configType
}

// Next ...
func (l *RegistryConfigurationListener) Next() (*registry.ServiceEvent, error) {
	for {
		select {
		case <-l.client.Done():
			logger.Warnf("listener's zk client connection is broken, so zk event listener exit now.")
			return nil, perrors.New("listener stopped")

		case <-l.registry.Done():
			logger.Warnf("zk consumer register has quit, so zk event listener exit now.")
			return nil, perrors.New("listener stopped")

		case e := <-l.events:
			logger.Debugf("got zk event %s", e)
			if e.ConfigType == remoting.EventTypeDel && !l.valid() {
				logger.Warnf("update @result{%s}. But its connection to registry is invalid", e.Value)
				continue
			}
			//r.update(e.res)
			//write to invoker
			//r.outerEventCh <- e.res
			return &registry.ServiceEvent{Action: e.ConfigType, Service: e.Value.(common.URL)}, nil
		}
	}
}

// Close ...
func (l *RegistryConfigurationListener) Close() {
	// ensure that the listener will be closed at most once.
	l.closeOnce.Do(func() {
		l.isClosed = true
		l.registry.WaitGroup().Done()
	})
}

func (l *RegistryConfigurationListener) valid() bool {
	return l.client.ZkConnValid()
}
