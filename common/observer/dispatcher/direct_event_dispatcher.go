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

package dispatcher

import (
	"reflect"
)

import (
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/common/observer"
)

func init() {
	observer.SetEventDispatcher("direct", NewDirectEventDispatcher)
}

// DirectEventDispatcher is align with DirectEventDispatcher interface in Java.
// it's the top abstraction
// Align with 2.7.5
// Dispatcher event to listener direct
type DirectEventDispatcher struct {
	observer.BaseListenable
}

func NewDirectEventDispatcher() observer.EventDispatcher {
	return &DirectEventDispatcher{}
}

func (ded *DirectEventDispatcher) Dispatch(event observer.Event) {
	eventType := reflect.TypeOf(event).Elem()
	value, loaded := ded.ListenersCache.Load(eventType)
	if !loaded {
		return
	}
	listenersSlice := value.([]observer.EventListener)
	for _, listener := range listenersSlice {
		if err := listener.OnEvent(event); err != nil {
			logger.Warnf("[DirectEventDispatcher] dispatch event error:%v", err)
		}
	}
}
