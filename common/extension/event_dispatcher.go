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

package extension

import (
	"sync"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/common/observer"
)

var (
	globalEventDispatcher observer.EventDispatcher
	initEventListeners    []func() observer.EventListener
	initEventOnce         sync.Once
)

var dispatchers = make(map[string]func() observer.EventDispatcher, 8)

// SetEventDispatcher, actually, it doesn't really init the global dispatcher
func SetEventDispatcher(name string, v func() observer.EventDispatcher) {
	dispatchers[name] = v
}

// SetAndInitGlobalDispatcher will actually init the global dispatcher
// if there is already a global dispatcher,
// it will be override
// if the dispatcher with the name not found, it will panic
func SetAndInitGlobalDispatcher(name string) {
	if len(name) == 0 {
		name = "direct"
	}
	if globalEventDispatcher != nil {
		logger.Warnf("EventDispatcher has been initialized. It will be replaced")
	}

	if dp, ok := dispatchers[name]; !ok || dp == nil {
		panic("EventDispatcher for " + name + " is not found, make sure you have import the package, " +
			"like import _ dubbo.apache.org/dubbo-go/v3/common/observer/dispatcher.")
	}
	globalEventDispatcher = dispatchers[name]()
}

// GetGlobalDispatcher will init all listener and then return dispatcher
func GetGlobalDispatcher() observer.EventDispatcher {
	initEventOnce.Do(func() {
		// we should delay to add the listeners to avoid some listeners left
		for _, l := range initEventListeners {
			globalEventDispatcher.AddEventListener(l())
		}
	})
	return globalEventDispatcher
}

// AddEventListener it will be added in global event dispatcher
func AddEventListener(creator func() observer.EventListener) {
	initEventListeners = append(initEventListeners, creator)
}
