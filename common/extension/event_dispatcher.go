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
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/common/observer"
)

var globalEventDispatcher observer.EventDispatcher

var initEventListeners []observer.EventListener

var (
	dispatchers = make(map[string]func() observer.EventDispatcher, 8)
)

// SetEventDispatcher by name
func SetEventDispatcher(name string, v func() observer.EventDispatcher) {
	dispatchers[name] = v
}

// SetAndInitGlobalDispatcher
func SetAndInitGlobalDispatcher(name string) {
	if len(name) == 0 {
		name = "direct"
	}
	if globalEventDispatcher != nil {
		logger.Warnf("EventDispatcher already init. It will be replaced")
	}
	if dp, ok := dispatchers[name]; !ok || dp == nil {
		panic("EventDispatcher for " + name + " is not existing, make sure you have import the package.")
	}
	globalEventDispatcher = dispatchers[name]()
	globalEventDispatcher.AddEventListeners(initEventListeners)
}

// GetGlobalDispatcher
func GetGlobalDispatcher() observer.EventDispatcher {
	return globalEventDispatcher
}

// AddEventListener it will be added in global event dispatcher
func AddEventListener(listener observer.EventListener) {
	initEventListeners = append(initEventListeners, listener)
}
