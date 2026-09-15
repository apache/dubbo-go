/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License. You may obtain a copy of the License at
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

type testConfig struct {
	prefix      string
	timeout     int
	initialized Scope
}

func (c *testConfig) Prefix() string {
	return c.prefix
}

func (c *testConfig) New() Config {
	return &testConfig{prefix: c.prefix, timeout: 1000}
}

func (c *testConfig) Init(scope Scope) error {
	if !scope.valid() {
		return errors.New("invalid scope")
	}
	c.initialized = scope
	return nil
}

func (c *testConfig) FilterNames(scope Scope) []string {
	switch scope {
	case ClientScope:
		return []string{"test-consumer"}
	case ServerScope:
		return []string{"test-provider"}
	default:
		return nil
	}
}

type testOption struct {
	prefix  string
	timeout int
}

func (o testOption) Prefix() string {
	return o.prefix
}

func (o testOption) Apply(config Config) error {
	config.(*testConfig).timeout = o.timeout
	return nil
}

func TestScopeValidity(t *testing.T) {
	for _, scope := range []Scope{InstanceScope, ClientScope, ServerScope} {
		assert.True(t, scope.valid())
	}
	for _, scope := range []Scope{0, Scope(4), Scope(255)} {
		assert.False(t, scope.valid())
	}
}

func TestConfigContract(t *testing.T) {
	const prefix = "config-contract-test"
	UnregisterConfig(prefix)
	t.Cleanup(func() { UnregisterConfig(prefix) })

	require.NoError(t, RegisterConfig(&testConfig{prefix: prefix}))
	prototype, ok := LookupConfig(prefix)
	require.True(t, ok)

	clientConfig := prototype.New()
	serverConfig := prototype.New()
	assert.NotSame(t, clientConfig, serverConfig)
	assert.Equal(t, 1000, clientConfig.(*testConfig).timeout)
	assert.Equal(t, prefix, clientConfig.Prefix())

	options := []Option{
		testOption{prefix: prefix, timeout: 2000},
		testOption{prefix: prefix, timeout: 3000},
	}
	for _, option := range options {
		require.NoError(t, option.Apply(clientConfig))
	}
	assert.Equal(t, 3000, clientConfig.(*testConfig).timeout)
	assert.Equal(t, 1000, serverConfig.(*testConfig).timeout)

	require.NoError(t, clientConfig.Init(ClientScope))
	assert.Equal(t, ClientScope, clientConfig.(*testConfig).initialized)
	assert.Equal(t, []string{"test-consumer"}, clientConfig.FilterNames(ClientScope))
	assert.Equal(t, []string{"test-provider"}, serverConfig.FilterNames(ServerScope))
	require.Error(t, clientConfig.Init(0))

	require.Error(t, RegisterConfig(&testConfig{prefix: prefix}))
	UnregisterConfig(prefix)
	_, ok = LookupConfig(prefix)
	assert.False(t, ok)
}

func TestRegisterConfigValidation(t *testing.T) {
	var typedNil *testConfig
	tests := []struct {
		name   string
		config Config
	}{
		{name: "nil"},
		{name: "typed nil", config: typedNil},
		{name: "empty prefix", config: &testConfig{}},
		{name: "whitespace prefix", config: &testConfig{prefix: " test "}},
		{name: "nil new config", config: nilNewConfig{}},
		{name: "mismatched new prefix", config: mismatchedNewConfig{}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Error(t, RegisterConfig(test.config))
		})
	}
}

func TestMustRegisterConfig(t *testing.T) {
	assert.Panics(t, func() { MustRegisterConfig(nil) })
}

type nilNewConfig struct{}

func (nilNewConfig) Prefix() string             { return "nil-new" }
func (nilNewConfig) New() Config                { return nil }
func (nilNewConfig) Init(Scope) error           { return nil }
func (nilNewConfig) FilterNames(Scope) []string { return nil }

type mismatchedNewConfig struct{}

func (mismatchedNewConfig) Prefix() string             { return "original" }
func (mismatchedNewConfig) New() Config                { return &testConfig{prefix: "different"} }
func (mismatchedNewConfig) Init(Scope) error           { return nil }
func (mismatchedNewConfig) FilterNames(Scope) []string { return nil }
