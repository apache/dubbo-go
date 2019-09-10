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
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestGetEnvInstance(t *testing.T) {
	GetEnvInstance()
	assert.NotNil(t, instance)
}

func TestEnvironment_UpdateExternalConfigMap(t *testing.T) {
	GetEnvInstance().UpdateExternalConfigMap(map[string]string{"1": "2"})
	v, ok := GetEnvInstance().externalConfigMap.Load("1")
	assert.True(t, ok)
	assert.Equal(t, "2", v)
}

func TestEnvironment_ConfigurationAndGetProperty(t *testing.T) {
	GetEnvInstance().UpdateExternalConfigMap(map[string]string{"1": "2"})
	list := GetEnvInstance().Configuration()
	ok, v := list.Back().Value.(*InmemoryConfiguration).GetProperty("1")
	assert.True(t, ok)
	assert.Equal(t, "2", v)
}

func TestInmemoryConfiguration_GetSubProperty(t *testing.T) {
	GetEnvInstance().UpdateExternalConfigMap(map[string]string{"123": "2"})
	list := GetEnvInstance().Configuration()
	m := list.Front().Value.(*InmemoryConfiguration).GetSubProperty("1")

	assert.Equal(t, struct{}{}, m["123"])
}
