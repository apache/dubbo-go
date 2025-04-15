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
	"context"
	"sync"
)

import (
	constant2 "github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/logger"
	"dubbo.apache.org/dubbo-go/v3/metrics"
	metricsConfigCenter "dubbo.apache.org/dubbo-go/v3/metrics/config_center"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

func callback(listenersMap *sync.Map, _, group, dataId, data string) {
	listenersMap.Range(func(key, value any) bool {
		key.(config_center.ConfigurationListener).Process(&config_center.ConfigChangeEvent{Key: dataId, Value: data, ConfigType: remoting.EventTypeUpdate})
		metrics.Publish(metricsConfigCenter.NewIncMetricEvent(dataId, group, remoting.EventTypeUpdate, metricsConfigCenter.Nacos))
		return true
	})
}

func (n *nacosDynamicConfiguration) addListener(key string, listener config_center.ConfigurationListener) {
	rawListenersMap, loaded := n.keyListeners.Load(key)
	if !loaded {
		_, cancel := context.WithCancel(context.Background())
		listenersMap := &sync.Map{}
		listenersMap.Store(listener, cancel)

		// double load for invalid race
		rawListenersMap, loaded = n.keyListeners.LoadOrStore(key, listenersMap)
		if !loaded {
			err := n.client.Client().ListenConfig(vo.ConfigParam{
				DataId: key,
				Group:  n.resolvedGroup(n.url.GetParam(constant.NacosGroupKey, constant2.DEFAULT_GROUP)),
				OnChange: func(namespace, group, dataId, data string) {
					go callback(listenersMap, namespace, group, dataId, data)
				},
			})
			if err != nil {
				n.keyListeners.Delete(key)
				logger.Errorf("nacos : listen config fail, error:%v ", err)
				return
			}
			return
		}
	}
	_, cancel := context.WithCancel(context.Background())
	listenersMap := rawListenersMap.(*sync.Map)
	listenersMap.Store(listener, cancel)
}

func (n *nacosDynamicConfiguration) removeListener(key string, listener config_center.ConfigurationListener) {
	rawListenersMap, loaded := n.keyListeners.Load(key)
	if !loaded {
		logger.Errorf("nacos : key:%s is not be listened", key)
	} else {
		listenersMap := rawListenersMap.(*sync.Map)
		listenersMap.Delete(listener)
	}
}
