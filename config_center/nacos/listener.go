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
)

import (
	"github.com/nacos-group/nacos-sdk-go/vo"
)

import (
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/config_center"
	"github.com/apache/dubbo-go/remoting"
)

func callback(listener config_center.ConfigurationListener, _, _, dataId, data string) {
	listener.Process(&config_center.ConfigChangeEvent{Key: dataId, Value: data, ConfigType: remoting.EventTypeUpdate})
}

func (n *nacosDynamicConfiguration) addListener(key string, listener config_center.ConfigurationListener) {
	_, loaded := n.keyListeners.Load(key)
	if !loaded {
		_, cancel := context.WithCancel(context.Background())
		err := (*n.client.Client()).ListenConfig(vo.ConfigParam{
			DataId: key,
			Group:  "dubbo",
			OnChange: func(namespace, group, dataId, data string) {
				go callback(listener, namespace, group, dataId, data)
			},
		})
		if err != nil {
			logger.Errorf("nacos : listen config fail, error:%v ", err)
			return
		}
		newListener := make(map[config_center.ConfigurationListener]context.CancelFunc)
		newListener[listener] = cancel
		n.keyListeners.Store(key, newListener)
	} else {
		// TODO check goroutine alive, but this version of go_nacos_sdk is not support.
		logger.Infof("profile:%s. this profile is already listening", key)
	}
}

func (n *nacosDynamicConfiguration) removeListener(key string, listener config_center.ConfigurationListener) {
	// TODO: not supported in current go_nacos_sdk version
	logger.Warn("not supported in current go_nacos_sdk version")
}
