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
	"github.com/zouyx/agollo"
)

import (
	"github.com/apache/dubbo-go/config_center"
)

type apolloListener struct {
	listeners map[config_center.ConfigurationListener]struct{}
}

// NewApolloListener creates a new apolloListener
func NewApolloListener() *apolloListener {
	return &apolloListener{
		listeners: make(map[config_center.ConfigurationListener]struct{}, 0),
	}
}

// OnChange ...
func (a *apolloListener) OnChange(changeEvent *agollo.ChangeEvent) {
	for key, change := range changeEvent.Changes {
		for listener := range a.listeners {
			listener.Process(&config_center.ConfigChangeEvent{
				ConfigType: getChangeType(change.ChangeType),
				Key:        key,
				Value:      change.NewValue,
			})
		}
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
