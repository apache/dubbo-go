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
	"github.com/zouyx/agollo/v3"
	"github.com/zouyx/agollo/v3/storage"
	"gopkg.in/yaml.v2"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

type apolloListener struct {
	listeners map[config_center.ConfigurationListener]struct{}
}

// nolint
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
		logger.Errorf("apollo onNewestChange err %+v",
			err)
		return
	}
	content := string(b)
	for listener := range a.listeners {
		listener.Process(&config_center.ConfigChangeEvent{
			ConfigType: remoting.EventTypeUpdate,
			Key:        changeEvent.Namespace,
			Value:      content,
		})
	}
}

// AddListener adds a listener for apollo
func (a *apolloListener) AddListener(l config_center.ConfigurationListener) {
	if _, ok := a.listeners[l]; !ok {
		a.listeners[l] = struct{}{}
		agollo.AddChangeListener(a)
	}
}

// RemoveListener removes listeners of apollo
func (a *apolloListener) RemoveListener(l config_center.ConfigurationListener) {
	delete(a.listeners, l)
}
