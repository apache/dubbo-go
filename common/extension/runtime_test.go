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
	assert.True(t, (ClientScope | ServerScope).Supports(ClientScope))
	assert.True(t, (ClientScope | ServerScope).Supports(ServerScope))
	assert.False(t, (ClientScope | ServerScope).Supports(InstanceScope))
	assert.False(t, (ClientScope | ServerScope).Supports(ClientScope|ServerScope))
}

func TestDefinitionRegistration(t *testing.T) {
	const prefix = "runtime-test"
	Unregister(prefix)
	t.Cleanup(func() { Unregister(prefix) })

	type config struct{ Timeout int }
	var decoded map[string]any
	var initialized int
	definition := Definition{
		Prefix: prefix,
		Scopes: ClientScope | ServerScope,
		New:    func() any { return &config{Timeout: 1000} },
		Decode: func(raw map[string]any, value any) error {
			decoded = raw
			return nil
		},
		Init: func(value any) error {
			initialized++
			return nil
		},
		ConsumerFilters: func(value any) []string { return []string{"test-consumer"} },
		ProviderFilters: func(value any) []string { return []string{"test-provider"} },
	}
	require.NoError(t, Register(definition))

	registered, ok := Lookup(prefix)
	require.True(t, ok)
	assert.True(t, registered.Supports(ClientScope))
	assert.True(t, registered.Supports(ServerScope))
	assert.False(t, registered.Supports(InstanceScope))
	clientConfig := registered.New()
	serverConfig := registered.New()
	assert.NotSame(t, clientConfig, serverConfig)
	raw := map[string]any{"greet.GreetService:::Greet": map[string]any{"timeout": 1000}}
	require.NoError(t, registered.Decode(raw, clientConfig))
	require.NoError(t, registered.Init(clientConfig))
	assert.Equal(t, raw, decoded)
	assert.Equal(t, 1, initialized)
	assert.Equal(t, []string{"test-consumer"}, registered.ConsumerFilters(clientConfig))
	assert.Equal(t, []string{"test-provider"}, registered.ProviderFilters(serverConfig))
	require.Error(t, Register(definition))
}

func TestDefinitionValidation(t *testing.T) {
	for _, definition := range []Definition{
		{Scopes: ClientScope, New: func() any { return nil }},
		{Prefix: "test", New: func() any { return nil }},
		{Prefix: "test", Scopes: Scope(8), New: func() any { return nil }},
		{Prefix: "test", Scopes: ClientScope},
	} {
		assert.Error(t, Register(definition))
	}
}
