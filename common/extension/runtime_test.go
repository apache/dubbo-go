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
	"errors"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScopeValidity(t *testing.T) {
	for _, scope := range []Scope{InstanceScope, ClientScope, ServerScope} {
		assert.True(t, scope.valid())
	}
	for _, scope := range []Scope{0, ClientScope | ServerScope, Scope(8)} {
		assert.False(t, scope.valid())
	}
}

func TestConfigRegistrationCreatesIndependentInstances(t *testing.T) {
	const prefix = "runtime-test"
	UnregisterConfig(prefix)
	t.Cleanup(func() { UnregisterConfig(prefix) })

	type config struct{ Timeout int }
	var initialized []Scope
	descriptor := configDescriptor{
		prefix: prefix,
		scopes: ClientScope | ServerScope,
		new:    func() any { return &config{Timeout: 1000} },
		init: func(scope Scope, value any) error {
			if _, ok := value.(*config); !ok {
				return errors.New("unexpected config type")
			}
			initialized = append(initialized, scope)
			return nil
		},
	}
	SetConfig(descriptor)

	registered, ok := GetConfig(prefix)
	require.True(t, ok)
	assert.True(t, registered.Scopes().Supports(ClientScope))
	assert.True(t, registered.Scopes().Supports(ServerScope))
	assert.False(t, registered.Scopes().Supports(InstanceScope))
	clientConfig := registered.New()
	serverConfig := registered.New()
	require.NotSame(t, clientConfig, serverConfig)
	require.NoError(t, registered.Init(ClientScope, clientConfig))
	require.NoError(t, registered.Init(ServerScope, serverConfig))
	assert.Equal(t, []Scope{ClientScope, ServerScope}, initialized)
}

type configDescriptor struct {
	prefix string
	scopes Scope
	new    func() any
	init   func(Scope, any) error
}

func (d configDescriptor) Prefix() string { return d.prefix }
func (d configDescriptor) Scopes() Scope  { return d.scopes }
func (d configDescriptor) New() any       { return d.new() }
func (d configDescriptor) Init(scope Scope, config any) error {
	return d.init(scope, config)
}
