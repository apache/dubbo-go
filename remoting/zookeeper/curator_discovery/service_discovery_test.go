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
	"fmt"
	"sync"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

const (
	testBasePath    = "/dubbo/services"
	testServiceName = "test-service"
	testInstanceID  = "instance-1"
)

func TestEntry(t *testing.T) {
	t.Run("basic operations", func(t *testing.T) {
		entry := &Entry{}
		assert.Nil(t, entry.instance)

		entry.Lock()
		entry.instance = &ServiceInstance{Name: testServiceName}
		entry.Unlock()
		assert.Equal(t, testServiceName, entry.instance.Name)
	})

	t.Run("concurrent access", func(t *testing.T) {
		entry := &Entry{instance: &ServiceInstance{Name: testServiceName}}
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
	for _, basePath := range []string{testBasePath, "/", ""} {
		sd := NewServiceDiscovery(nil, basePath)
		assert.NotNil(t, sd)
		assert.Equal(t, basePath, sd.basePath)
		assert.NotNil(t, sd.mutex)
		assert.NotNil(t, sd.services)
	}
}

func TestServiceDiscoveryPathForInstance(t *testing.T) {
	tests := []struct {
		basePath, serviceName, instanceID, expected string
	}{
		{testBasePath, testServiceName, testInstanceID, testBasePath + "/" + testServiceName + "/" + testInstanceID},
		{"/", "service", "id", "/service/id"},
		{"", "service", "id", "service/id"},
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
		{testBasePath, testServiceName, testBasePath + "/" + testServiceName},
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
		{"valid path", testBasePath, testBasePath + "/" + testServiceName + "/" + testInstanceID, testServiceName, testInstanceID, false},
		{"missing id", "/dubbo", "/dubbo/service", "", "", true},
		{"empty path", "/dubbo", "", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sd := NewServiceDiscovery(nil, tt.basePath)
			name, id, err := sd.getNameAndID(tt.path)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedName, name)
				assert.Equal(t, tt.expectedID, id)
			}
		})
	}
}

func TestServiceDiscoveryDataChange(t *testing.T) {
	sd := NewServiceDiscovery(nil, testBasePath)
	testPath := testBasePath + "/test/" + testInstanceID

	for _, eventType := range []remoting.EventType{remoting.EventTypeAdd, remoting.EventTypeUpdate, remoting.EventTypeDel} {
		event := remoting.Event{Path: testPath, Action: eventType, Content: "content"}
		assert.True(t, sd.DataChange(event))
	}

	// Invalid path
	assert.True(t, sd.DataChange(remoting.Event{Path: testBasePath + "/only-name"}))
}

func TestServiceDiscoveryClose(t *testing.T) {
	sd := &ServiceDiscovery{client: nil, listener: nil, services: &sync.Map{}, mutex: &sync.Mutex{}}
	sd.Close() // Should not panic
}

func TestServiceDiscoveryServicesMap(t *testing.T) {
	sd := NewServiceDiscovery(nil, testBasePath)

	entry := &Entry{instance: &ServiceInstance{Name: testServiceName, ID: testInstanceID}}
	sd.services.Store(testInstanceID, entry)

	value, ok := sd.services.Load(testInstanceID)
	assert.True(t, ok)
	assert.Equal(t, testServiceName, value.(*Entry).instance.Name)

	sd.services.Delete(testInstanceID)
	_, ok = sd.services.Load(testInstanceID)
	assert.False(t, ok)
}

func TestServiceDiscoveryUpdateService(t *testing.T) {
	sd := NewServiceDiscovery(nil, testBasePath)

	// Update non-existent
	err := sd.UpdateService(&ServiceInstance{Name: testServiceName, ID: "non-existent"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not registered")

	// Update with invalid entry type
	sd.services.Store("invalid-id", "not-an-entry")
	err = sd.UpdateService(&ServiceInstance{Name: testServiceName, ID: "invalid-id"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not entry")
}

func TestServiceDiscoveryUnregisterService(t *testing.T) {
	sd := NewServiceDiscovery(nil, testBasePath)

	err := sd.UnregisterService(&ServiceInstance{Name: testServiceName, ID: "non-existent"})
	assert.NoError(t, err)
}

func TestServiceDiscoveryConcurrentAccess(t *testing.T) {
	sd := NewServiceDiscovery(nil, testBasePath)
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			//Confirm ID is equal
			entry := &Entry{instance: &ServiceInstance{Name: testServiceName, ID: fmt.Sprintf("id-%d", idx)}}
			sd.services.Store(entry.instance.ID, entry)
		}(i)
	}
	wg.Wait()

	// test whether 100 datas is inserted
	count := 0
	sd.services.Range(func(_, _ any) bool {
		count++
		return true
	})
	assert.Equal(t, 100, count)
}
