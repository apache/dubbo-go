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
	gxchan "github.com/dubbogo/gost/container/chan"
	gxzookeeper "github.com/dubbogo/gost/database/kv/zk"
	"github.com/dubbogo/gost/log/logger"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

// RegistryDataListener contains all URL information subscribed by zookeeper registry
type RegistryDataListener struct {
	subscribed map[string]config_center.ConfigurationListener
	mutex      sync.Mutex
	closed     bool
}

// NewRegistryDataListener constructs a new RegistryDataListener
func NewRegistryDataListener() *RegistryDataListener {
	return &RegistryDataListener{
		subscribed: make(map[string]config_center.ConfigurationListener),
	}
}

// SubscribeURL is used to set a watch listener for url
func (l *RegistryDataListener) SubscribeURL(url *common.URL, listener config_center.ConfigurationListener) {
	if l.closed {
		return
	}
	l.subscribed[url.ServiceKey()] = listener
}

// UnSubscribeURL is used to unset a watch listener for url
func (l *RegistryDataListener) UnSubscribeURL(url *common.URL) config_center.ConfigurationListener {
	if l.closed {
		return nil
	}
	listener := l.subscribed[url.ServiceKey()]
	listener.(*RegistryConfigurationListener).Close()
	delete(l.subscribed, url.ServiceKey())
	return listener
}

// DataChange accepts all events sent from the zookeeper server and trigger the corresponding listener for processing
func (l *RegistryDataListener) DataChange(event remoting.Event) bool {
	providersPath := constant.PathSeparator + constant.ProviderCategory + constant.PathSeparator
	// Intercept the last bit
	index := strings.Index(event.Path, providersPath)
	if index == -1 {
		logger.Warnf("[RegistryDataListener][DataChange]Listen error zk node path {%s}, "+
			"this listener is used to listen services which under the directory of providers/", event.Path)
		return false
	}
	url := event.Path[index+len(providersPath):]
	serviceURL, err := common.NewURL(url)
	if err != nil {
		logger.Errorf("[RegistryDataListener][DataChange]Listen NewURL({%s}) = error{%+v} event.Path={%s}", url, err, event.Path)
		return false
	}
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if l.closed {
		return false
	}
	for serviceKey, listener := range l.subscribed {
		if serviceURL.ServiceKey() == serviceKey {
			listener.Process(
				&config_center.ConfigChangeEvent{
					Key:        event.Path,
					Value:      serviceURL,
					ConfigType: event.Action,
				},
			)
			return true
		}
	}
	return false
}

// Close all RegistryConfigurationListener in subscribed
func (l *RegistryDataListener) Close() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.closed = true
	for _, listener := range l.subscribed {
		listener.(*RegistryConfigurationListener).Close()
	}
}

// RegistryConfigurationListener represent the processor of zookeeper watcher
type RegistryConfigurationListener struct {
	client       *gxzookeeper.ZookeeperClient
	registry     *zkRegistry
	events       *gxchan.UnboundedChan
	isClosed     bool
	close        chan struct{}
	closeOnce    sync.Once
	subscribeURL *common.URL
}

// NewRegistryConfigurationListener for listening the event of zk.
func NewRegistryConfigurationListener(client *gxzookeeper.ZookeeperClient, reg *zkRegistry, conf *common.URL) *RegistryConfigurationListener {
	reg.WaitGroup().Add(1)
	return &RegistryConfigurationListener{
		client:       client,
		registry:     reg,
		events:       gxchan.NewUnboundedChan(32),
		isClosed:     false,
		close:        make(chan struct{}, 1),
		subscribeURL: conf,
	}
}

// Process submit the ConfigChangeEvent to the event chan to notify all observer
func (l *RegistryConfigurationListener) Process(configType *config_center.ConfigChangeEvent) {
	l.events.In() <- configType
}

// Next will observe the registry state and events chan
func (l *RegistryConfigurationListener) Next() (*registry.ServiceEvent, error) {
	for {
		select {
		case <-l.close:
			return nil, perrors.New("listener have been closed")
		case <-l.registry.Done():
			logger.Warnf("zk consumer register has quit, so zk event listener exit now. (registry url {%v}", l.registry.BaseRegistry.URL)
			return nil, perrors.New("zookeeper registry, (registry url{%v}) stopped")
		case val := <-l.events.Out():
			e, _ := val.(*config_center.ConfigChangeEvent)
			logger.Debugf("got zk event %s", e)
			if e.ConfigType == remoting.EventTypeDel && !l.valid() {
				logger.Warnf("update @result{%s}. But its connection to registry is invalid", e.Value)
				continue
			}
			return &registry.ServiceEvent{Action: e.ConfigType, Service: e.Value.(*common.URL)}, nil
		}
	}
}

// Close RegistryConfigurationListener only once
func (l *RegistryConfigurationListener) Close() {
	// ensure that the listener will be closed at most once.
	l.closeOnce.Do(func() {
		l.isClosed = true
		l.close <- struct{}{}
		l.registry.WaitGroup().Done()
	})
}

// valid return the true if the client conn isn't nil
func (l *RegistryConfigurationListener) valid() bool {
	return l.client.ZkConnValid()
}
