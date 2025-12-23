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

package config

import (
	"sync"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestGetEnvInstance(t *testing.T) {
	GetEnvInstance()
	assert.NotNil(t, instance)
}

func TestEnvironmentUpdateExternalConfigMap(t *testing.T) {
	GetEnvInstance().UpdateExternalConfigMap(map[string]string{"1": "2"})
	v, ok := GetEnvInstance().externalConfigMap.Load("1")
	assert.True(t, ok)
	assert.Equal(t, "2", v)

	GetEnvInstance().UpdateExternalConfigMap(map[string]string{"a": "b"})
	v, ok = GetEnvInstance().externalConfigMap.Load("a")
	assert.True(t, ok)
	assert.Equal(t, "b", v)
	v, ok = GetEnvInstance().externalConfigMap.Load("1")
	assert.False(t, ok)
	assert.Nil(t, v)
}

func TestEnvironmentUpdateAppExternalConfigMap(t *testing.T) {
	GetEnvInstance().UpdateAppExternalConfigMap(map[string]string{"1": "2"})
	v, ok := GetEnvInstance().appExternalConfigMap.Load("1")
	assert.True(t, ok)
	assert.Equal(t, "2", v)

	GetEnvInstance().UpdateAppExternalConfigMap(map[string]string{"a": "b"})
	v, ok = GetEnvInstance().appExternalConfigMap.Load("a")
	assert.True(t, ok)
	assert.Equal(t, "b", v)
	v, ok = GetEnvInstance().appExternalConfigMap.Load("1")
	assert.False(t, ok)
	assert.Nil(t, v)
}

func TestEnvironmentConfigurationAndGetProperty(t *testing.T) {
	GetEnvInstance().UpdateExternalConfigMap(map[string]string{"1": "2"})
	list := GetEnvInstance().Configuration()
	ok, v := list.Back().Value.(*InmemoryConfiguration).GetProperty("1")
	assert.True(t, ok)
	assert.Equal(t, "2", v)
}

func TestInmemoryConfigurationGetSubProperty(t *testing.T) {
	GetEnvInstance().UpdateExternalConfigMap(map[string]string{"123": "2"})
	list := GetEnvInstance().Configuration()
	m := list.Front().Value.(*InmemoryConfiguration).GetSubProperty("1")

	assert.Equal(t, struct{}{}, m["123"])
}

func TestNewEnvInstance(t *testing.T) {
	// Reset instance
	oldInstance := instance
	defer func() { instance = oldInstance }()

	NewEnvInstance()
	assert.NotNil(t, instance)
	assert.True(t, instance.configCenterFirst)
}

func TestSetAndGetDynamicConfiguration(t *testing.T) {
	env := GetEnvInstance()

	// Initially nil
	assert.Nil(t, env.GetDynamicConfiguration())

	// Set and get
	// Using nil as mock since we just test the setter/getter
	env.SetDynamicConfiguration(nil)
	assert.Nil(t, env.GetDynamicConfiguration())
}

func TestInmemoryConfigurationGetPropertyNilStore(t *testing.T) {
	conf := &InmemoryConfiguration{store: nil}

	ok, v := conf.GetProperty("key")
	assert.False(t, ok)
	assert.Equal(t, "", v)
}

func TestInmemoryConfigurationGetSubPropertyNilStore(t *testing.T) {
	conf := &InmemoryConfiguration{store: nil}

	result := conf.GetSubProperty("key")
	assert.Nil(t, result)
}

func TestInmemoryConfigurationGetSubPropertyWithDot(t *testing.T) {
	store := &sync.Map{}
	store.Store("dubbo.protocol.name", "triple")
	store.Store("dubbo.protocol.port", "20000")
	store.Store("dubbo.registry.address", "nacos://127.0.0.1:8848")

	conf := &InmemoryConfiguration{store: store}

	// Get sub properties with prefix "dubbo."
	result := conf.GetSubProperty("dubbo.")
	assert.NotNil(t, result)
	assert.Contains(t, result, "protocol")
	assert.Contains(t, result, "registry")
}

func TestInmemoryConfigurationGetSubPropertyNoMatch(t *testing.T) {
	store := &sync.Map{}
	store.Store("other.key", "value")

	conf := &InmemoryConfiguration{store: store}

	result := conf.GetSubProperty("dubbo.")
	assert.NotNil(t, result)
	assert.Equal(t, 0, len(result))
}
