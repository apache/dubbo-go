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

package apollo

import (
	"sync"
)

import (
	"github.com/apolloconfig/agollo/v4/storage"

	"github.com/dubbogo/gost/log/logger"

	"gopkg.in/yaml.v3"
)

import (
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

type apolloListener struct {
	mu        sync.RWMutex
	listeners map[config_center.ConfigurationListener]struct{}
}

// newApolloListener constructs an apolloListener with empty listener set.
func newApolloListener() *apolloListener {
	return &apolloListener{
		listeners: make(map[config_center.ConfigurationListener]struct{}),
	}
}

// OnChange process each listener
func (a *apolloListener) OnChange(changeEvent *storage.ChangeEvent) {
}

// OnNewestChange process each listener by all changes
func (a *apolloListener) OnNewestChange(changeEvent *storage.FullChangeEvent) {
	b, err := yaml.Marshal(changeEvent.Changes)
	if err != nil {
		logger.Errorf("[ConfigCenter][Apollo] onNewestChange err=%v", err)
		return
	}
	content := string(b)
	for _, listener := range a.snapshotListeners() {
		listener.Process(&config_center.ConfigChangeEvent{
			ConfigType: remoting.EventTypeUpdate,
			Key:        changeEvent.Namespace,
			Value:      content,
		})
	}
}

// AddListener adds a listener for apollo
func (a *apolloListener) AddListener(l config_center.ConfigurationListener) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.listeners[l] = struct{}{}
}

// RemoveListener removes listeners of apollo
func (a *apolloListener) RemoveListener(l config_center.ConfigurationListener) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.listeners, l)
	return len(a.listeners) == 0
}

// IsEmpty Check if listeners is empty
func (a *apolloListener) IsEmpty() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return len(a.listeners) == 0
}

func (a *apolloListener) snapshotListeners() []config_center.ConfigurationListener {
	a.mu.RLock()
	defer a.mu.RUnlock()

	listeners := make([]config_center.ConfigurationListener, 0, len(a.listeners))
	for listener := range a.listeners {
		listeners = append(listeners, listener)
	}
	return listeners
}
