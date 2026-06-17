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

package nacos

import (
	"sync"
)

import (
	"github.com/dubbogo/gost/log/logger"

	constant2 "github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/metrics"
	metricsConfigCenter "dubbo.apache.org/dubbo-go/v3/metrics/config_center"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

// keyListenerSet holds the listeners for a single config key.
// The mutex protects the listeners map so that add/remove/check-empty
// operations are atomic, preventing races between concurrent addListener
// and removeListener calls for the same key.
type keyListenerSet struct {
	mu        sync.Mutex
	listeners map[config_center.ConfigurationListener]struct{}
	group     string // resolved group used to register with nacos, stored for consistent cancel
}

func newKeyListenerSet(group string) *keyListenerSet {
	return &keyListenerSet{
		listeners: make(map[config_center.ConfigurationListener]struct{}),
		group:     group,
	}
}

func (s *keyListenerSet) add(listener config_center.ConfigurationListener) {
	s.mu.Lock()
	s.listeners[listener] = struct{}{}
	s.mu.Unlock()
}

// remove removes a listener and reports whether the set is now empty.
// The caller must NOT rely on a non-empty result to skip CancelListenConfig,
// because a concurrent add could re-populate the set after this call returns.
func (s *keyListenerSet) remove(listener config_center.ConfigurationListener) bool {
	s.mu.Lock()
	delete(s.listeners, listener)
	empty := len(s.listeners) == 0
	s.mu.Unlock()
	return empty
}

// snapshot returns a snapshot of the current listeners for safe iteration.
func (s *keyListenerSet) snapshot() []config_center.ConfigurationListener {
	s.mu.Lock()
	snapshot := make([]config_center.ConfigurationListener, 0, len(s.listeners))
	for l := range s.listeners {
		snapshot = append(snapshot, l)
	}
	s.mu.Unlock()
	return snapshot
}

func callback(set *keyListenerSet, _, group, dataId, data string) {
	for _, l := range set.snapshot() {
		l.Process(&config_center.ConfigChangeEvent{Key: dataId, Value: data, ConfigType: remoting.EventTypeUpdate})
		metrics.Publish(metricsConfigCenter.NewIncMetricEvent(dataId, group, remoting.EventTypeUpdate, metricsConfigCenter.Nacos))
	}
}

func (n *nacosDynamicConfiguration) addListener(key string, listener config_center.ConfigurationListener) {
	group := n.resolvedGroup(n.url.GetParam(constant.NacosGroupKey, constant2.DEFAULT_GROUP))

	rawSet, loaded := n.keyListeners.Load(key)
	if !loaded {
		set := newKeyListenerSet(group)
		set.add(listener)

		// double load for invalid race
		var actual any
		actual, loaded = n.keyListeners.LoadOrStore(key, set)
		if !loaded {
			err := n.client.Client().ListenConfig(vo.ConfigParam{
				DataId: key,
				Group:  group,
				OnChange: func(namespace, group, dataId, data string) {
					go callback(set, namespace, group, dataId, data)
				},
			})
			if err != nil {
				n.keyListeners.Delete(key)
				logger.Errorf("[ConfigCenter][Nacos] listen config fail, err=%v", err)
				return
			}
			return
		}
		rawSet = actual
	}
	rawSet.(*keyListenerSet).add(listener)
}

func (n *nacosDynamicConfiguration) removeListener(key string, listener config_center.ConfigurationListener) {
	rawSet, loaded := n.keyListeners.Load(key)
	if !loaded {
		logger.Errorf("[ConfigCenter][Nacos] key is not be listened, key=%s", key)
		return
	}
	set := rawSet.(*keyListenerSet)
	isEmpty := set.remove(listener)

	if isEmpty {
		// Delete from keyListeners first to prevent new addListener from
		// finding a stale set after we cancel the nacos subscription.
		n.keyListeners.Delete(key)
		if n.client != nil {
			err := n.client.Client().CancelListenConfig(vo.ConfigParam{
				DataId: key,
				Group:  set.group,
			})
			if err != nil {
				logger.Errorf("[ConfigCenter][Nacos] cancel listen config fail, key=%s, err=%v", key, err)
			}
		}
	}
}
