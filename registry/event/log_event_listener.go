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

package event

import (
	"reflect"
	"sync"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/common/observer"
)

func init() {
	extension.AddEventListener(GetLogEventListener)
}

// logEventListener is singleton
type logEventListener struct{}

func (l *logEventListener) GetPriority() int {
	return 0
}

func (l *logEventListener) OnEvent(e observer.Event) error {
	logger.Info("Event happen: " + e.String())
	return nil
}

func (l *logEventListener) GetEventType() reflect.Type {
	return reflect.TypeOf(&observer.BaseEvent{})
}

var (
	logEventListenerInstance *logEventListener
	logEventListenerOnce     sync.Once
)

func GetLogEventListener() observer.EventListener {
	logEventListenerOnce.Do(func() {
		logEventListenerInstance = &logEventListener{}
	})
	return logEventListenerInstance
}
