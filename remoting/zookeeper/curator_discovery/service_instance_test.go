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

package curator_discovery

import (
	"encoding/json"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

// ============================================
// ServiceInstance Tests
// ============================================
func TestServiceInstanceCreation(t *testing.T) {
	tests := []struct {
		name     string
		instance *ServiceInstance
		validate func(*testing.T, *ServiceInstance)
	}{
		{
			name: "basic instance",
			instance: &ServiceInstance{
				Name:    "test-service",
				ID:      "instance-1",
				Address: "192.168.1.100",
				Port:    8080,
			},
			validate: func(t *testing.T, i *ServiceInstance) {
				assert.Equal(t, "test-service", i.Name)
				assert.Equal(t, "instance-1", i.ID)
				assert.Equal(t, "192.168.1.100", i.Address)
				assert.Equal(t, 8080, i.Port)
			},
		},
		{
			name: "instance with payload",
			instance: &ServiceInstance{
				Name:    "service-with-payload",
				ID:      "instance-2",
				Address: "10.0.0.1",
				Port:    9090,
				Payload: map[string]string{"key": "value"},
			},
			validate: func(t *testing.T, i *ServiceInstance) {
				assert.NotNil(t, i.Payload)
				payload, ok := i.Payload.(map[string]string)
				assert.True(t, ok)
				assert.Equal(t, "value", payload["key"])
			},
		},
		{
			name: "instance with registration time",
			instance: &ServiceInstance{
				Name:                "timed-service",
				ID:                  "instance-3",
				Address:             "localhost",
				Port:                3000,
				RegistrationTimeUTC: time.Now().UnixMilli(),
			},
			validate: func(t *testing.T, i *ServiceInstance) {
				assert.Greater(t, i.RegistrationTimeUTC, int64(0))
			},
		},
		{
			name: "instance with tag",
			instance: &ServiceInstance{
				Name:    "tagged-service",
				ID:      "instance-4",
				Address: "127.0.0.1",
				Port:    5000,
				Tag:     "production",
			},
			validate: func(t *testing.T, i *ServiceInstance) {
				assert.Equal(t, "production", i.Tag)
			},
		},
		{
			name:     "empty instance",
			instance: &ServiceInstance{},
			validate: func(t *testing.T, i *ServiceInstance) {
				assert.Empty(t, i.Name)
				assert.Empty(t, i.ID)
				assert.Empty(t, i.Address)
				assert.Equal(t, 0, i.Port)
				assert.Nil(t, i.Payload)
				assert.Equal(t, int64(0), i.RegistrationTimeUTC)
				assert.Empty(t, i.Tag)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(t, tt.instance)
		})
	}
}

// ============================================
// JSON Serialization Tests
// ============================================

func TestServiceInstanceJSONSerialization(t *testing.T) {
	tests := []struct {
		name     string
		instance *ServiceInstance
	}{
		{
			name: "full instance",
			instance: &ServiceInstance{
				Name:                "json-service",
				ID:                  "json-instance-1",
				Address:             "192.168.1.1",
				Port:                8080,
				Payload:             map[string]any{"version": "1.0", "weight": 100},
				RegistrationTimeUTC: 1609459200000,
				Tag:                 "v1",
			},
		},
		{
			name: "minimal instance",
			instance: &ServiceInstance{
				Name: "minimal-service",
				ID:   "minimal-1",
			},
		},
		{
			name: "instance with nil payload",
			instance: &ServiceInstance{
				Name:    "nil-payload-service",
				ID:      "nil-payload-1",
				Address: "localhost",
				Port:    3000,
				Payload: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Serialize
			data, err := json.Marshal(tt.instance)
			assert.Nil(t, err)
			assert.NotEmpty(t, data)

			// Deserialize
			var restored ServiceInstance
			err = json.Unmarshal(data, &restored)
			assert.Nil(t, err)

			// Verify
			assert.Equal(t, tt.instance.Name, restored.Name)
			assert.Equal(t, tt.instance.ID, restored.ID)
			assert.Equal(t, tt.instance.Address, restored.Address)
			assert.Equal(t, tt.instance.Port, restored.Port)
			assert.Equal(t, tt.instance.RegistrationTimeUTC, restored.RegistrationTimeUTC)
			assert.Equal(t, tt.instance.Tag, restored.Tag)
		})
	}
}

func TestServiceInstanceJSONOmitEmpty(t *testing.T) {
	instance := &ServiceInstance{}
	data, err := json.Marshal(instance)
	assert.Nil(t, err)

	// Verify omitempty works
	jsonStr := string(data)
	assert.Equal(t, "{}", jsonStr)
}

func TestServiceInstanceJSONWithComplexPayload(t *testing.T) {
	tests := []struct {
		name    string
		payload any
	}{
		{
			name:    "string payload",
			payload: "simple string",
		},
		{
			name:    "int payload",
			payload: 12345,
		},
		{
			name:    "map payload",
			payload: map[string]any{"nested": map[string]int{"value": 1}},
		},
		{
			name:    "array payload",
			payload: []string{"a", "b", "c"},
		},
		{
			name:    "bool payload",
			payload: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := &ServiceInstance{
				Name:    "payload-test",
				ID:      "payload-1",
				Payload: tt.payload,
			}

			data, err := json.Marshal(instance)
			assert.Nil(t, err)

			var restored ServiceInstance
			err = json.Unmarshal(data, &restored)
			assert.Nil(t, err)
			assert.NotNil(t, restored.Payload)
		})
	}
}

// ============================================
// Edge Cases Tests
// ============================================

func TestServiceInstanceEdgeCases(t *testing.T) {
	t.Run("special characters in name", func(t *testing.T) {
		instance := &ServiceInstance{
			Name:    "service-name_v1.0",
			ID:      "instance-id_123",
			Address: "host.domain.com",
		}
		data, err := json.Marshal(instance)
		assert.Nil(t, err)

		var restored ServiceInstance
		err = json.Unmarshal(data, &restored)
		assert.Nil(t, err)
		assert.Equal(t, instance.Name, restored.Name)
	})

	t.Run("unicode in fields", func(t *testing.T) {
		instance := &ServiceInstance{
			Name:    "服务名称",
			ID:      "实例ID",
			Address: "地址",
			Tag:     "标签",
		}
		data, err := json.Marshal(instance)
		assert.Nil(t, err)

		var restored ServiceInstance
		err = json.Unmarshal(data, &restored)
		assert.Nil(t, err)
		assert.Equal(t, "服务名称", restored.Name)
	})

	t.Run("max port number", func(t *testing.T) {
		instance := &ServiceInstance{
			Name: "max-port-service",
			ID:   "max-port-1",
			Port: 65535,
		}
		assert.Equal(t, 65535, instance.Port)
	})

	t.Run("zero port", func(t *testing.T) {
		instance := &ServiceInstance{
			Name: "zero-port-service",
			ID:   "zero-port-1",
			Port: 0,
		}
		assert.Equal(t, 0, instance.Port)
	})

	t.Run("negative registration time", func(t *testing.T) {
		instance := &ServiceInstance{
			Name:                "negative-time-service",
			ID:                  "negative-time-1",
			RegistrationTimeUTC: -1,
		}
		assert.Equal(t, int64(-1), instance.RegistrationTimeUTC)
	})
}

// ============================================
// Comparison Tests
// ============================================

func TestServiceInstanceEquality(t *testing.T) {
	instance1 := &ServiceInstance{
		Name:    "service",
		ID:      "instance-1",
		Address: "localhost",
		Port:    8080,
	}

	instance2 := &ServiceInstance{
		Name:    "service",
		ID:      "instance-1",
		Address: "localhost",
		Port:    8080,
	}

	instance3 := &ServiceInstance{
		Name:    "service",
		ID:      "instance-2", // Different ID
		Address: "localhost",
		Port:    8080,
	}

	// Same values
	assert.Equal(t, instance1.Name, instance2.Name)
	assert.Equal(t, instance1.ID, instance2.ID)
	assert.Equal(t, instance1.Address, instance2.Address)
	assert.Equal(t, instance1.Port, instance2.Port)

	// Different ID
	assert.NotEqual(t, instance1.ID, instance3.ID)
}

// ============================================
// Benchmark Tests
// ============================================

func BenchmarkServiceInstanceMarshal(b *testing.B) {
	instance := &ServiceInstance{
		Name:                "benchmark-service",
		ID:                  "benchmark-instance",
		Address:             "192.168.1.100",
		Port:                8080,
		Payload:             map[string]string{"key": "value"},
		RegistrationTimeUTC: time.Now().UnixMilli(),
		Tag:                 "production",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(instance)
	}
}

func BenchmarkServiceInstanceUnmarshal(b *testing.B) {
	instance := &ServiceInstance{
		Name:                "benchmark-service",
		ID:                  "benchmark-instance",
		Address:             "192.168.1.100",
		Port:                8080,
		Payload:             map[string]string{"key": "value"},
		RegistrationTimeUTC: time.Now().UnixMilli(),
		Tag:                 "production",
	}
	data, _ := json.Marshal(instance)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var restored ServiceInstance
		_ = json.Unmarshal(data, &restored)
	}
}
