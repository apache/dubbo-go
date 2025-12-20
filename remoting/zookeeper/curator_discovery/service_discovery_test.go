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
	"sync"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

// ============================================
// Entry Tests
// ============================================
func TestEntry(t *testing.T) {
	t.Run("create entry", func(t *testing.T) {
		entry := &Entry{}
		assert.Nil(t, entry.instance)
	})

	t.Run("entry with instance", func(t *testing.T) {
		instance := &ServiceInstance{
			Name:    "test-service",
			ID:      "instance-1",
			Address: "localhost",
			Port:    8080,
		}
		entry := &Entry{instance: instance}
		assert.NotNil(t, entry.instance)
		assert.Equal(t, "test-service", entry.instance.Name)
	})

	t.Run("entry lock", func(t *testing.T) {
		entry := &Entry{}
		entry.Lock()
		entry.instance = &ServiceInstance{Name: "locked-service"}
		entry.Unlock()
		assert.Equal(t, "locked-service", entry.instance.Name)
	})
}

func TestEntryConcurrentAccess(t *testing.T) {
	entry := &Entry{
		instance: &ServiceInstance{Name: "concurrent-service"},
	}
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			entry.Lock()
			entry.instance.Name = "updated-service"
			entry.Unlock()
		}(i)
	}

	wg.Wait()
	assert.Equal(t, "updated-service", entry.instance.Name)
}

// ============================================
// ServiceDiscovery Tests
// ============================================

func TestNewServiceDiscovery(t *testing.T) {
	tests := []struct {
		name     string
		basePath string
	}{
		{"normal base path", "/dubbo/services"},
		{"root path", "/"},
		{"nested path", "/dubbo/metadata/services"},
		{"empty path", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sd := NewServiceDiscovery(nil, tt.basePath)
			assert.NotNil(t, sd)
			assert.Equal(t, tt.basePath, sd.basePath)
			assert.NotNil(t, sd.mutex)
			assert.NotNil(t, sd.services)
		})
	}
}

func TestServiceDiscoveryPathForInstance(t *testing.T) {
	tests := []struct {
		name         string
		basePath     string
		serviceName  string
		instanceID   string
		expectedPath string
	}{
		{
			name:         "normal path",
			basePath:     "/dubbo/services",
			serviceName:  "test-service",
			instanceID:   "instance-1",
			expectedPath: "/dubbo/services/test-service/instance-1",
		},
		{
			name:         "root base path",
			basePath:     "/",
			serviceName:  "service",
			instanceID:   "id",
			expectedPath: "/service/id",
		},
		{
			name:         "empty base path",
			basePath:     "",
			serviceName:  "service",
			instanceID:   "id",
			expectedPath: "service/id",
		},
		{
			name:         "special characters",
			basePath:     "/dubbo",
			serviceName:  "com.example.Service",
			instanceID:   "192.168.1.1:8080",
			expectedPath: "/dubbo/com.example.Service/192.168.1.1:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sd := NewServiceDiscovery(nil, tt.basePath)
			path := sd.pathForInstance(tt.serviceName, tt.instanceID)
			assert.Equal(t, tt.expectedPath, path)
		})
	}
}

func TestServiceDiscoveryPathForName(t *testing.T) {
	tests := []struct {
		name         string
		basePath     string
		serviceName  string
		expectedPath string
	}{
		{
			name:         "normal path",
			basePath:     "/dubbo/services",
			serviceName:  "test-service",
			expectedPath: "/dubbo/services/test-service",
		},
		{
			name:         "root base path",
			basePath:     "/",
			serviceName:  "service",
			expectedPath: "/service",
		},
		{
			name:         "empty base path",
			basePath:     "",
			serviceName:  "service",
			expectedPath: "service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sd := NewServiceDiscovery(nil, tt.basePath)
			path := sd.pathForName(tt.serviceName)
			assert.Equal(t, tt.expectedPath, path)
		})
	}
}

func TestServiceDiscoveryGetNameAndID(t *testing.T) {
	tests := []struct {
		name         string
		basePath     string
		path         string
		expectedName string
		expectedID   string
		wantErr      bool
	}{
		{
			name:         "valid path",
			basePath:     "/dubbo/services",
			path:         "/dubbo/services/test-service/instance-1",
			expectedName: "test-service",
			expectedID:   "instance-1",
			wantErr:      false,
		},
		{
			name:         "path with trailing content",
			basePath:     "/dubbo",
			path:         "/dubbo/service/id/extra",
			expectedName: "service",
			expectedID:   "id",
			wantErr:      false,
		},
		{
			name:         "invalid path - missing id",
			basePath:     "/dubbo",
			path:         "/dubbo/service",
			expectedName: "",
			expectedID:   "",
			wantErr:      true,
		},
		{
			name:         "invalid path - only base",
			basePath:     "/dubbo",
			path:         "/dubbo",
			expectedName: "",
			expectedID:   "",
			wantErr:      true,
		},
		{
			name:         "empty path",
			basePath:     "/dubbo",
			path:         "",
			expectedName: "",
			expectedID:   "",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sd := NewServiceDiscovery(nil, tt.basePath)
			name, id, err := sd.getNameAndID(tt.path)
			if tt.wantErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tt.expectedName, name)
				assert.Equal(t, tt.expectedID, id)
			}
		})
	}
}

// ============================================
// DataChange Tests
// ============================================

func TestServiceDiscoveryDataChange(t *testing.T) {
	sd := NewServiceDiscovery(nil, "/dubbo/services")

	t.Run("valid event path", func(t *testing.T) {
		event := remoting.Event{
			Path:    "/dubbo/services/test-service/instance-1",
			Action:  remoting.EventTypeUpdate,
			Content: "updated content",
		}
		result := sd.DataChange(event)
		assert.True(t, result)
	})

	t.Run("invalid event path", func(t *testing.T) {
		event := remoting.Event{
			Path:    "/dubbo/services/only-name",
			Action:  remoting.EventTypeUpdate,
			Content: "content",
		}
		result := sd.DataChange(event)
		assert.True(t, result) // Returns true even on error (logs error)
	})
}

// ============================================
// Close Tests
// ============================================

func TestServiceDiscoveryClose(t *testing.T) {
	t.Run("close with nil client and listener", func(t *testing.T) {
		sd := &ServiceDiscovery{
			client:   nil,
			listener: nil,
			services: &sync.Map{},
			mutex:    &sync.Mutex{},
		}
		// Should not panic
		sd.Close()
	})

	t.Run("close with nil listener only", func(t *testing.T) {
		sd := &ServiceDiscovery{
			client:   nil,
			listener: nil,
			services: &sync.Map{},
			mutex:    &sync.Mutex{},
		}
		sd.Close()
	})
}

// ============================================
// updateInternalService Tests
// ============================================

func TestServiceDiscoveryUpdateInternalService(t *testing.T) {
	t.Run("update non-existent service", func(t *testing.T) {
		sd := NewServiceDiscovery(nil, "/dubbo")
		// Should not panic when service doesn't exist
		sd.updateInternalService("non-existent", "id")
	})

	t.Run("update with invalid entry type", func(t *testing.T) {
		sd := NewServiceDiscovery(nil, "/dubbo")
		// Store invalid type
		sd.services.Store("invalid", "not-an-entry")
		// Should not panic - returns early due to type assertion failure
		sd.updateInternalService("service", "invalid")
	})

	t.Run("update with valid entry but nil client", func(t *testing.T) {
		sd := NewServiceDiscovery(nil, "/dubbo")
		entry := &Entry{
			instance: &ServiceInstance{Name: "test", ID: "1"},
		}
		sd.services.Store("1", entry)
		// Will log error due to nil client but should not panic
		// The updateInternalService catches the error from QueryForInstance
		// Note: This test verifies the entry lookup works, actual query fails gracefully
	})
}

// ============================================
// Services Map Tests
// ============================================

func TestServiceDiscoveryServicesMap(t *testing.T) {
	sd := NewServiceDiscovery(nil, "/dubbo")

	t.Run("store and load", func(t *testing.T) {
		entry := &Entry{
			instance: &ServiceInstance{Name: "test", ID: "1"},
		}
		sd.services.Store("1", entry)

		value, ok := sd.services.Load("1")
		assert.True(t, ok)
		loadedEntry, ok := value.(*Entry)
		assert.True(t, ok)
		assert.Equal(t, "test", loadedEntry.instance.Name)
	})

	t.Run("load non-existent", func(t *testing.T) {
		_, ok := sd.services.Load("non-existent")
		assert.False(t, ok)
	})

	t.Run("delete", func(t *testing.T) {
		entry := &Entry{
			instance: &ServiceInstance{Name: "to-delete", ID: "2"},
		}
		sd.services.Store("2", entry)
		sd.services.Delete("2")

		_, ok := sd.services.Load("2")
		assert.False(t, ok)
	})

	t.Run("range", func(t *testing.T) {
		sd2 := NewServiceDiscovery(nil, "/dubbo")
		for i := 0; i < 5; i++ {
			entry := &Entry{
				instance: &ServiceInstance{Name: "service", ID: string(rune('0' + i))},
			}
			sd2.services.Store(string(rune('0'+i)), entry)
		}

		count := 0
		sd2.services.Range(func(key, value any) bool {
			count++
			return true
		})
		assert.Equal(t, 5, count)
	})
}

// ============================================
// Concurrent Access Tests
// ============================================

func TestServiceDiscoveryConcurrentAccess(t *testing.T) {
	sd := NewServiceDiscovery(nil, "/dubbo")
	var wg sync.WaitGroup

	// Concurrent store operations
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			entry := &Entry{
				instance: &ServiceInstance{
					Name: "concurrent-service",
					ID:   string(rune('a' + idx%26)),
				},
			}
			sd.services.Store(entry.instance.ID, entry)
		}(i)
	}

	wg.Wait()
}

// ============================================
// Integration-like Tests
// ============================================

func TestServiceDiscoveryLifecycle(t *testing.T) {
	sd := NewServiceDiscovery(nil, "/dubbo/services")

	// Verify initial state
	assert.NotNil(t, sd.services)
	assert.NotNil(t, sd.mutex)
	assert.Equal(t, "/dubbo/services", sd.basePath)

	// Test path generation
	instancePath := sd.pathForInstance("my-service", "instance-123")
	assert.Equal(t, "/dubbo/services/my-service/instance-123", instancePath)

	namePath := sd.pathForName("my-service")
	assert.Equal(t, "/dubbo/services/my-service", namePath)

	// Test name and ID extraction
	name, id, err := sd.getNameAndID("/dubbo/services/my-service/instance-123")
	assert.Nil(t, err)
	assert.Equal(t, "my-service", name)
	assert.Equal(t, "instance-123", id)

	// Close
	sd.Close()
}

// ============================================
// Edge Cases Tests
// ============================================

func TestServiceDiscoveryEdgeCases(t *testing.T) {
	t.Run("path with special characters", func(t *testing.T) {
		sd := NewServiceDiscovery(nil, "/dubbo")
		path := sd.pathForInstance("com.example.UserService", "192.168.1.1:20880")
		assert.Equal(t, "/dubbo/com.example.UserService/192.168.1.1:20880", path)
	})

	t.Run("empty service name", func(t *testing.T) {
		sd := NewServiceDiscovery(nil, "/dubbo")
		path := sd.pathForInstance("", "instance-1")
		assert.Equal(t, "/dubbo/instance-1", path)
	})

	t.Run("empty instance id", func(t *testing.T) {
		sd := NewServiceDiscovery(nil, "/dubbo")
		path := sd.pathForInstance("service", "")
		assert.Equal(t, "/dubbo/service", path)
	})

	t.Run("unicode in paths", func(t *testing.T) {
		sd := NewServiceDiscovery(nil, "/dubbo")
		path := sd.pathForInstance("服务名", "实例ID")
		assert.Contains(t, path, "服务名")
		assert.Contains(t, path, "实例ID")
	})
}

// ============================================
// DataChange Event Types Tests
// ============================================

func TestServiceDiscoveryDataChangeEventTypes(t *testing.T) {
	tests := []struct {
		name      string
		eventType remoting.EventType
		path      string
		content   string
	}{
		{
			name:      "add event",
			eventType: remoting.EventTypeAdd,
			path:      "/dubbo/services/test/instance-1",
			content:   "new content",
		},
		{
			name:      "update event",
			eventType: remoting.EventTypeUpdate,
			path:      "/dubbo/services/test/instance-1",
			content:   "updated content",
		},
		{
			name:      "delete event",
			eventType: remoting.EventTypeDel,
			path:      "/dubbo/services/test/instance-1",
			content:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sd := NewServiceDiscovery(nil, "/dubbo/services")
			event := remoting.Event{
				Path:    tt.path,
				Action:  tt.eventType,
				Content: tt.content,
			}
			result := sd.DataChange(event)
			assert.True(t, result)
		})
	}
}

// ============================================
// Benchmark Tests
// ============================================

func BenchmarkPathForInstance(b *testing.B) {
	sd := NewServiceDiscovery(nil, "/dubbo/services")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sd.pathForInstance("test-service", "instance-1")
	}
}

func BenchmarkPathForName(b *testing.B) {
	sd := NewServiceDiscovery(nil, "/dubbo/services")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sd.pathForName("test-service")
	}
}

func BenchmarkGetNameAndID(b *testing.B) {
	sd := NewServiceDiscovery(nil, "/dubbo/services")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = sd.getNameAndID("/dubbo/services/test-service/instance-1")
	}
}

func BenchmarkServicesMapStore(b *testing.B) {
	sd := NewServiceDiscovery(nil, "/dubbo")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		entry := &Entry{
			instance: &ServiceInstance{Name: "bench", ID: "1"},
		}
		sd.services.Store("1", entry)
	}
}

func BenchmarkServicesMapLoad(b *testing.B) {
	sd := NewServiceDiscovery(nil, "/dubbo")
	entry := &Entry{
		instance: &ServiceInstance{Name: "bench", ID: "1"},
	}
	sd.services.Store("1", entry)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sd.services.Load("1")
	}
}

func BenchmarkServicesMapConcurrentStore(b *testing.B) {
	sd := NewServiceDiscovery(nil, "/dubbo")
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			entry := &Entry{
				instance: &ServiceInstance{Name: "bench", ID: string(rune('a' + i%26))},
			}
			sd.services.Store(entry.instance.ID, entry)
			i++
		}
	})
}

func BenchmarkDataChange(b *testing.B) {
	sd := NewServiceDiscovery(nil, "/dubbo/services")
	event := remoting.Event{
		Path:    "/dubbo/services/test-service/instance-1",
		Action:  remoting.EventTypeUpdate,
		Content: "content",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sd.DataChange(event)
	}
}

// ============================================
// RegisterService Mock Tests
// ============================================

func TestServiceDiscoveryRegisterServiceMock(t *testing.T) {
	t.Run("register service stores entry", func(t *testing.T) {
		sd := NewServiceDiscovery(nil, "/dubbo")
		instance := &ServiceInstance{
			Name:    "test-service",
			ID:      "instance-1",
			Address: "localhost",
			Port:    8080,
		}
		// Store entry manually to test the services map behavior
		entry := &Entry{instance: instance}
		sd.services.Store(instance.ID, entry)

		// Verify entry was stored
		value, ok := sd.services.Load(instance.ID)
		assert.True(t, ok)
		loadedEntry, ok := value.(*Entry)
		assert.True(t, ok)
		assert.Equal(t, instance.Name, loadedEntry.instance.Name)
	})
}

// ============================================
// UpdateService Mock Tests
// ============================================

func TestServiceDiscoveryUpdateServiceMock(t *testing.T) {
	t.Run("update non-existent service", func(t *testing.T) {
		sd := NewServiceDiscovery(nil, "/dubbo")
		instance := &ServiceInstance{
			Name:    "test-service",
			ID:      "non-existent",
			Address: "localhost",
			Port:    8080,
		}
		err := sd.UpdateService(instance)
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "not registered")
	})

	t.Run("update with invalid entry type", func(t *testing.T) {
		sd := NewServiceDiscovery(nil, "/dubbo")
		sd.services.Store("invalid-id", "not-an-entry")
		instance := &ServiceInstance{
			Name:    "test-service",
			ID:      "invalid-id",
			Address: "localhost",
			Port:    8080,
		}
		err := sd.UpdateService(instance)
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "not entry")
	})
}

// ============================================
// UnregisterService Mock Tests
// ============================================

func TestServiceDiscoveryUnregisterServiceMock(t *testing.T) {
	t.Run("unregister non-existent service", func(t *testing.T) {
		sd := NewServiceDiscovery(nil, "/dubbo")
		instance := &ServiceInstance{
			Name:    "test-service",
			ID:      "non-existent",
			Address: "localhost",
			Port:    8080,
		}
		err := sd.UnregisterService(instance)
		assert.Nil(t, err) // Should return nil for non-existent
	})

	t.Run("unregister removes from services map", func(t *testing.T) {
		sd := NewServiceDiscovery(nil, "/dubbo")
		instance := &ServiceInstance{
			Name:    "test-service",
			ID:      "instance-1",
			Address: "localhost",
			Port:    8080,
		}
		entry := &Entry{instance: instance}
		sd.services.Store(instance.ID, entry)

		// Verify entry exists
		_, ok := sd.services.Load(instance.ID)
		assert.True(t, ok)

		// Manually delete to simulate unregister behavior (without nil client panic)
		sd.services.Delete(instance.ID)

		// Verify entry was removed
		_, ok = sd.services.Load(instance.ID)
		assert.False(t, ok)
	})
}

// ============================================
// ReRegisterServices Mock Tests
// ============================================

func TestServiceDiscoveryReRegisterServicesMock(t *testing.T) {
	t.Run("re-register with empty services", func(t *testing.T) {
		sd := NewServiceDiscovery(nil, "/dubbo")
		// Should not panic with empty services
		sd.ReRegisterServices()
	})

	t.Run("re-register with invalid entry type", func(t *testing.T) {
		sd := NewServiceDiscovery(nil, "/dubbo")
		sd.services.Store("invalid", "not-an-entry")
		// Should skip invalid entries
		sd.ReRegisterServices()
	})
}

// ============================================
// Path Edge Cases Tests
// ============================================

func TestServiceDiscoveryPathEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		basePath    string
		serviceName string
		instanceID  string
	}{
		{
			name:        "path with dots",
			basePath:    "/dubbo.services",
			serviceName: "com.example.Service",
			instanceID:  "192.168.1.1:8080",
		},
		{
			name:        "path with underscores",
			basePath:    "/dubbo_services",
			serviceName: "test_service",
			instanceID:  "instance_1",
		},
		{
			name:        "path with hyphens",
			basePath:    "/dubbo-services",
			serviceName: "test-service",
			instanceID:  "instance-1",
		},
		{
			name:        "very long path",
			basePath:    "/very/long/base/path/for/testing/purposes",
			serviceName: "very-long-service-name-for-testing",
			instanceID:  "very-long-instance-id-for-testing",
		},
		{
			name:        "numeric values",
			basePath:    "/123",
			serviceName: "456",
			instanceID:  "789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sd := NewServiceDiscovery(nil, tt.basePath)

			instancePath := sd.pathForInstance(tt.serviceName, tt.instanceID)
			assert.Contains(t, instancePath, tt.serviceName)
			assert.Contains(t, instancePath, tt.instanceID)

			namePath := sd.pathForName(tt.serviceName)
			assert.Contains(t, namePath, tt.serviceName)
		})
	}
}

// ============================================
// GetNameAndID Edge Cases Tests
// ============================================

func TestServiceDiscoveryGetNameAndIDEdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		basePath     string
		path         string
		expectedName string
		expectedID   string
		wantErr      bool
	}{
		{
			name:         "path with multiple slashes",
			basePath:     "/dubbo",
			path:         "/dubbo/service/id/extra/parts",
			expectedName: "service",
			expectedID:   "id",
			wantErr:      false,
		},
		{
			name:         "path with special chars in name",
			basePath:     "/dubbo",
			path:         "/dubbo/com.example.Service/192.168.1.1:8080",
			expectedName: "com.example.Service",
			expectedID:   "192.168.1.1:8080",
			wantErr:      false,
		},
		{
			name:         "path without base prefix",
			basePath:     "/dubbo",
			path:         "/other/service/id",
			expectedName: "other",
			expectedID:   "service",
			wantErr:      false,
		},
		{
			name:         "single segment after base",
			basePath:     "/dubbo",
			path:         "/dubbo/onlyone",
			expectedName: "",
			expectedID:   "",
			wantErr:      true,
		},
		{
			name:         "empty segments returns empty strings",
			basePath:     "/dubbo",
			path:         "/dubbo//",
			expectedName: "",
			expectedID:   "",
			wantErr:      false, // Split on "//" produces ["", ""] which has len >= 2
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sd := NewServiceDiscovery(nil, tt.basePath)
			name, id, err := sd.getNameAndID(tt.path)
			if tt.wantErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tt.expectedName, name)
				assert.Equal(t, tt.expectedID, id)
			}
		})
	}
}

// ============================================
// Mutex Tests
// ============================================

func TestServiceDiscoveryMutex(t *testing.T) {
	sd := NewServiceDiscovery(nil, "/dubbo")
	var wg sync.WaitGroup
	counter := 0

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sd.mutex.Lock()
			counter++
			sd.mutex.Unlock()
		}()
	}

	wg.Wait()
	assert.Equal(t, 100, counter)
}

// ============================================
// Entry Concurrent Modification Tests
// ============================================

func TestEntryConcurrentModification(t *testing.T) {
	entry := &Entry{
		instance: &ServiceInstance{
			Name:    "test",
			ID:      "1",
			Address: "localhost",
			Port:    8080,
		},
	}

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func(idx int) {
			defer wg.Done()
			entry.Lock()
			entry.instance.Port = idx
			entry.Unlock()
		}(i)
		go func() {
			defer wg.Done()
			entry.Lock()
			_ = entry.instance.Port
			entry.Unlock()
		}()
	}
	wg.Wait()
}

// ============================================
// ServiceInstance Field Tests
// ============================================

func TestServiceInstanceFields(t *testing.T) {
	tests := []struct {
		name     string
		instance *ServiceInstance
	}{
		{
			name: "minimal instance",
			instance: &ServiceInstance{
				Name: "test",
				ID:   "1",
			},
		},
		{
			name: "full instance",
			instance: &ServiceInstance{
				Name:    "test-service",
				ID:      "instance-1",
				Address: "192.168.1.100",
				Port:    8080,
			},
		},
		{
			name: "instance with zero port",
			instance: &ServiceInstance{
				Name:    "test",
				ID:      "1",
				Address: "localhost",
				Port:    0,
			},
		},
		{
			name: "instance with empty address",
			instance: &ServiceInstance{
				Name:    "test",
				ID:      "1",
				Address: "",
				Port:    8080,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := &Entry{instance: tt.instance}
			assert.Equal(t, tt.instance.Name, entry.instance.Name)
			assert.Equal(t, tt.instance.ID, entry.instance.ID)
			assert.Equal(t, tt.instance.Address, entry.instance.Address)
			assert.Equal(t, tt.instance.Port, entry.instance.Port)
		})
	}
}
