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

func TestEntry(t *testing.T) {
	t.Run("basic operations", func(t *testing.T) {
		entry := &Entry{}
		assert.Nil(t, entry.instance)

		entry.Lock()
		entry.instance = &ServiceInstance{Name: "test"}
		entry.Unlock()
		assert.Equal(t, "test", entry.instance.Name)
	})

	t.Run("concurrent access", func(t *testing.T) {
		entry := &Entry{instance: &ServiceInstance{Name: "test"}}
		var wg sync.WaitGroup

		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				entry.Lock()
				entry.instance.Name = "updated"
				entry.Unlock()
			}()
		}
		wg.Wait()
		assert.Equal(t, "updated", entry.instance.Name)
	})
}

func TestNewServiceDiscovery(t *testing.T) {
	tests := []struct {
		basePath string
	}{
		{"/dubbo/services"},
		{"/"},
		{""},
	}

	for _, tt := range tests {
		sd := NewServiceDiscovery(nil, tt.basePath)
		assert.NotNil(t, sd)
		assert.Equal(t, tt.basePath, sd.basePath)
		assert.NotNil(t, sd.mutex)
		assert.NotNil(t, sd.services)
	}
}

func TestServiceDiscoveryPathForInstance(t *testing.T) {
	tests := []struct {
		basePath, serviceName, instanceID, expected string
	}{
		{"/dubbo/services", "test-service", "instance-1", "/dubbo/services/test-service/instance-1"},
		{"/", "service", "id", "/service/id"},
		{"", "service", "id", "service/id"},
		{"/dubbo", "com.example.Service", "192.168.1.1:8080", "/dubbo/com.example.Service/192.168.1.1:8080"},
	}

	for _, tt := range tests {
		sd := NewServiceDiscovery(nil, tt.basePath)
		assert.Equal(t, tt.expected, sd.pathForInstance(tt.serviceName, tt.instanceID))
	}
}

func TestServiceDiscoveryPathForName(t *testing.T) {
	tests := []struct {
		basePath, serviceName, expected string
	}{
		{"/dubbo/services", "test-service", "/dubbo/services/test-service"},
		{"/", "service", "/service"},
		{"", "service", "service"},
	}

	for _, tt := range tests {
		sd := NewServiceDiscovery(nil, tt.basePath)
		assert.Equal(t, tt.expected, sd.pathForName(tt.serviceName))
	}
}

func TestServiceDiscoveryGetNameAndID(t *testing.T) {
	tests := []struct {
		name, basePath, path, expectedName, expectedID string
		wantErr                                        bool
	}{
		{"valid path", "/dubbo/services", "/dubbo/services/test-service/instance-1", "test-service", "instance-1", false},
		{"extra parts", "/dubbo", "/dubbo/service/id/extra", "service", "id", false},
		{"missing id", "/dubbo", "/dubbo/service", "", "", true},
		{"only base", "/dubbo", "/dubbo", "", "", true},
		{"empty path", "/dubbo", "", "", "", true},
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

func TestServiceDiscoveryDataChange(t *testing.T) {
	sd := NewServiceDiscovery(nil, "/dubbo/services")

	tests := []struct {
		eventType remoting.EventType
		path      string
	}{
		{remoting.EventTypeAdd, "/dubbo/services/test/instance-1"},
		{remoting.EventTypeUpdate, "/dubbo/services/test/instance-1"},
		{remoting.EventTypeDel, "/dubbo/services/test/instance-1"},
	}

	for _, tt := range tests {
		event := remoting.Event{Path: tt.path, Action: tt.eventType, Content: "content"}
		assert.True(t, sd.DataChange(event))
	}

	// Invalid path
	assert.True(t, sd.DataChange(remoting.Event{Path: "/dubbo/services/only-name"}))
}

func TestServiceDiscoveryClose(t *testing.T) {
	sd := &ServiceDiscovery{
		client: nil, listener: nil, services: &sync.Map{}, mutex: &sync.Mutex{},
	}
	sd.Close() // Should not panic
}

func TestServiceDiscoveryServicesMap(t *testing.T) {
	sd := NewServiceDiscovery(nil, "/dubbo")

	// Store and load
	entry := &Entry{instance: &ServiceInstance{Name: "test", ID: "1"}}
	sd.services.Store("1", entry)

	value, ok := sd.services.Load("1")
	assert.True(t, ok)
	assert.Equal(t, "test", value.(*Entry).instance.Name)

	// Load non-existent
	_, ok = sd.services.Load("non-existent")
	assert.False(t, ok)

	// Delete
	sd.services.Delete("1")
	_, ok = sd.services.Load("1")
	assert.False(t, ok)
}

func TestServiceDiscoveryConcurrentAccess(t *testing.T) {
	sd := NewServiceDiscovery(nil, "/dubbo")
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			entry := &Entry{instance: &ServiceInstance{Name: "service", ID: string(rune('a' + idx%26))}}
			sd.services.Store(entry.instance.ID, entry)
		}(i)
	}
	wg.Wait()
}

func TestServiceDiscoveryUpdateService(t *testing.T) {
	sd := NewServiceDiscovery(nil, "/dubbo")

	// Update non-existent
	err := sd.UpdateService(&ServiceInstance{Name: "test", ID: "non-existent"})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "not registered")

	// Update with invalid entry type
	sd.services.Store("invalid-id", "not-an-entry")
	err = sd.UpdateService(&ServiceInstance{Name: "test", ID: "invalid-id"})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "not entry")
}

func TestServiceDiscoveryUnregisterService(t *testing.T) {
	sd := NewServiceDiscovery(nil, "/dubbo")

	// Unregister non-existent
	err := sd.UnregisterService(&ServiceInstance{Name: "test", ID: "non-existent"})
	assert.Nil(t, err)

	// Store and delete
	entry := &Entry{instance: &ServiceInstance{Name: "test", ID: "1"}}
	sd.services.Store("1", entry)
	sd.services.Delete("1")
	_, ok := sd.services.Load("1")
	assert.False(t, ok)
}

func TestServiceDiscoveryReRegisterServices(t *testing.T) {
	sd := NewServiceDiscovery(nil, "/dubbo")
	sd.ReRegisterServices() // Empty services, should not panic

	sd.services.Store("invalid", "not-an-entry")
	sd.ReRegisterServices() // Invalid entry, should skip
}

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
