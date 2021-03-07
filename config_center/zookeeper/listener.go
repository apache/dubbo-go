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
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/config_center"
	"github.com/apache/dubbo-go/remoting"
)

// CacheListener defines keyListeners and rootPath
type CacheListener struct {
	keyListeners sync.Map
	rootPath     string
}

// NewCacheListener creates a new CacheListener
func NewCacheListener(rootPath string) *CacheListener {
	return &CacheListener{rootPath: rootPath}
}

// AddListener will add a listener if loaded
func (l *CacheListener) AddListener(key string, listener config_center.ConfigurationListener) {

	// reference from https://stackoverflow.com/questions/34018908/golang-why-dont-we-have-a-set-datastructure
	// make a map[your type]struct{} like set in java
	listeners, loaded := l.keyListeners.LoadOrStore(key, map[config_center.ConfigurationListener]struct{}{listener: {}})
	if loaded {
		listeners.(map[config_center.ConfigurationListener]struct{})[listener] = struct{}{}
		l.keyListeners.Store(key, listeners)
	}
}

// RemoveListener will delete a listener if loaded
func (l *CacheListener) RemoveListener(key string, listener config_center.ConfigurationListener) {
	listeners, loaded := l.keyListeners.Load(key)
	if loaded {
		delete(listeners.(map[config_center.ConfigurationListener]struct{}), listener)
	}
}

// DataChange changes all listeners' event
func (l *CacheListener) DataChange(event remoting.Event) bool {
	if event.Content == "" {
		//meanings new node
		return true
	}
	key := l.pathToKey(event.Path)
	if key != "" {
		if listeners, ok := l.keyListeners.Load(key); ok {
			for listener := range listeners.(map[config_center.ConfigurationListener]struct{}) {
				listener.Process(&config_center.ConfigChangeEvent{Key: key, Value: event.Content, ConfigType: event.Action})
			}
			return true
		}
	}
	return false
}

func (l *CacheListener) pathToKey(path string) string {
	key := strings.Replace(strings.Replace(path, l.rootPath+"/", "", -1), "/", ".", -1)
	if strings.HasSuffix(key, constant.CONFIGURATORS_SUFFIX) ||
		strings.HasSuffix(key, constant.TagRouterRuleSuffix) ||
		strings.HasSuffix(key, constant.ConditionRouterRuleSuffix) {
		//governance config, so we remove the "dubbo." prefix
		return key[strings.Index(key, ".")+1:]
	}
	return key
}
