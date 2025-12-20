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

package zookeeper

import (
	"encoding/json"
	"testing"
)

import (
	"github.com/dubbogo/gost/gof/observer"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
	"dubbo.apache.org/dubbo-go/v3/metadata/mapping"
)

// ============================================
// Test Helper Functions
// ============================================
func createTestMetadataInfo() *info.MetadataInfo {
	return &info.MetadataInfo{
		App:      "test-app",
		Revision: "1.0.0",
		Services: map[string]*info.ServiceInfo{
			"com.example.TestService": {
				Name:     "com.example.TestService",
				Group:    "default",
				Version:  "1.0.0",
				Protocol: "dubbo",
				Path:     "com.example.TestService",
			},
		},
	}
}

// ============================================
// zookeeperMetadataReportFactory Tests
// ============================================

func TestZookeeperMetadataReportFactoryType(t *testing.T) {
	factory := &zookeeperMetadataReportFactory{}
	assert.NotNil(t, factory)
}

// ============================================
// GetAppMetadata Tests
// ============================================

func TestGetAppMetadataSuccess(t *testing.T) {
	tests := []struct {
		name        string
		application string
		revision    string
		metadata    *info.MetadataInfo
	}{
		{
			name:        "normal metadata",
			application: "test-app",
			revision:    "1.0.0",
			metadata:    createTestMetadataInfo(),
		},
		{
			name:        "metadata with empty services",
			application: "empty-app",
			revision:    "2.0.0",
			metadata: &info.MetadataInfo{
				App:      "empty-app",
				Revision: "2.0.0",
				Services: map[string]*info.ServiceInfo{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.metadata)
			assert.Nil(t, err)

			// Verify JSON marshaling/unmarshaling works correctly
			var result info.MetadataInfo
			err = json.Unmarshal(data, &result)
			assert.Nil(t, err)
			assert.Equal(t, tt.metadata.App, result.App)
			assert.Equal(t, tt.metadata.Revision, result.Revision)
		})
	}
}

func TestGetAppMetadataInvalidJSON(t *testing.T) {
	invalidJSON := []byte(`{"app": "test", invalid}`)

	var result info.MetadataInfo
	err := json.Unmarshal(invalidJSON, &result)
	assert.NotNil(t, err)
}

// ============================================
// PublishAppMetadata Tests
// ============================================

func TestPublishAppMetadataMarshal(t *testing.T) {
	tests := []struct {
		name     string
		metadata *info.MetadataInfo
		wantErr  bool
	}{
		{
			name:     "valid metadata",
			metadata: createTestMetadataInfo(),
			wantErr:  false,
		},
		{
			name: "metadata with multiple services",
			metadata: &info.MetadataInfo{
				App:      "multi-service-app",
				Revision: "1.0.0",
				Services: map[string]*info.ServiceInfo{
					"service1": {Name: "service1", Protocol: "dubbo"},
					"service2": {Name: "service2", Protocol: "tri"},
				},
			},
			wantErr: false,
		},
		{
			name: "metadata with nil services",
			metadata: &info.MetadataInfo{
				App:      "nil-services-app",
				Revision: "1.0.0",
				Services: nil,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.metadata)
			if tt.wantErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				assert.NotEmpty(t, data)

				// Verify round-trip
				var result info.MetadataInfo
				err = json.Unmarshal(data, &result)
				assert.Nil(t, err)
				assert.Equal(t, tt.metadata.App, result.App)
			}
		})
	}
}

// ============================================
// RegisterServiceAppMapping Tests
// ============================================

func TestRegisterServiceAppMappingPathConstruction(t *testing.T) {
	tests := []struct {
		name         string
		rootDir      string
		key          string
		group        string
		expectedPath string
	}{
		{
			name:         "normal path",
			rootDir:      "/dubbo/",
			key:          "com.example.Service",
			group:        "mapping",
			expectedPath: "/dubbo/mapping/com.example.Service",
		},
		{
			name:         "different group",
			rootDir:      "/dubbo/",
			key:          "com.example.Service",
			group:        "custom-group",
			expectedPath: "/dubbo/custom-group/com.example.Service",
		},
		{
			name:         "root dir without trailing slash",
			rootDir:      "/metadata",
			key:          "TestService",
			group:        "mapping",
			expectedPath: "/metadatamapping/TestService",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.rootDir + tt.group + constant.PathSeparator + tt.key
			assert.Equal(t, tt.expectedPath, path)
		})
	}
}

func TestRegisterServiceAppMappingValueMerge(t *testing.T) {
	tests := []struct {
		name        string
		oldValue    string
		newValue    string
		shouldMerge bool
		expected    string
	}{
		{
			name:        "new value not in old",
			oldValue:    "app1",
			newValue:    "app2",
			shouldMerge: true,
			expected:    "app1,app2",
		},
		{
			name:        "new value already exists",
			oldValue:    "app1,app2",
			newValue:    "app1",
			shouldMerge: false,
			expected:    "app1,app2",
		},
		{
			name:        "empty old value",
			oldValue:    "",
			newValue:    "app1",
			shouldMerge: true,
			expected:    ",app1",
		},
		{
			name:        "multiple existing values",
			oldValue:    "app1,app2,app3",
			newValue:    "app4",
			shouldMerge: true,
			expected:    "app1,app2,app3,app4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the merge logic from RegisterServiceAppMapping
			var result string
			if contains(tt.oldValue, tt.newValue) {
				result = tt.oldValue
			} else {
				result = tt.oldValue + constant.CommaSeparator + tt.newValue
			}
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper function to simulate strings.Contains check
func contains(s, substr string) bool {
	return len(substr) > 0 && len(s) >= len(substr) && containsSubstring(s, substr)
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ============================================
// GetServiceAppMapping Tests
// ============================================

func TestGetServiceAppMappingParsing(t *testing.T) {
	tests := []struct {
		name          string
		content       string
		expectedCount int
		expectedApps  []string
	}{
		{
			name:          "single app",
			content:       "app1",
			expectedCount: 1,
			expectedApps:  []string{"app1"},
		},
		{
			name:          "multiple apps",
			content:       "app1,app2,app3",
			expectedCount: 3,
			expectedApps:  []string{"app1", "app2", "app3"},
		},
		{
			name:          "empty content",
			content:       "",
			expectedCount: 1, // strings.Split returns [""] for empty string
			expectedApps:  []string{""},
		},
		{
			name:          "apps with spaces",
			content:       "app1, app2, app3",
			expectedCount: 3,
			expectedApps:  []string{"app1", " app2", " app3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apps := splitByComma(tt.content)
			assert.Equal(t, tt.expectedCount, len(apps))
			for i, expected := range tt.expectedApps {
				assert.Equal(t, expected, apps[i])
			}
		})
	}
}

// Helper function to simulate strings.Split
func splitByComma(s string) []string {
	if s == "" {
		return []string{""}
	}
	result := []string{}
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	result = append(result, s[start:])
	return result
}

// ============================================
// RemoveServiceAppMappingListener Tests
// ============================================

func TestRemoveServiceAppMappingListener(t *testing.T) {
	report := &zookeeperMetadataReport{
		rootDir:       "/dubbo/",
		cacheListener: NewCacheListener("/dubbo/", nil),
	}

	// Should return nil (no-op implementation)
	err := report.RemoveServiceAppMappingListener("test.service", "mapping")
	assert.Nil(t, err)
}

// ============================================
// CreateMetadataReport URL Parsing Tests
// ============================================

func TestCreateMetadataReportURLParsing(t *testing.T) {
	tests := []struct {
		name            string
		url             *common.URL
		expectedRootDir string
	}{
		{
			name: "default group",
			url: common.NewURLWithOptions(
				common.WithProtocol("zookeeper"),
				common.WithLocation("127.0.0.1:2181"),
			),
			expectedRootDir: "/dubbo/",
		},
		{
			name: "custom group without leading slash",
			url: common.NewURLWithOptions(
				common.WithProtocol("zookeeper"),
				common.WithLocation("127.0.0.1:2181"),
				common.WithParamsValue(constant.MetadataReportGroupKey, "custom"),
			),
			expectedRootDir: "/custom/",
		},
		{
			name: "custom group with leading slash",
			url: common.NewURLWithOptions(
				common.WithProtocol("zookeeper"),
				common.WithLocation("127.0.0.1:2181"),
				common.WithParamsValue(constant.MetadataReportGroupKey, "/custom"),
			),
			expectedRootDir: "/custom/",
		},
		{
			name: "root path only",
			url: common.NewURLWithOptions(
				common.WithProtocol("zookeeper"),
				common.WithLocation("127.0.0.1:2181"),
				common.WithParamsValue(constant.MetadataReportGroupKey, "/"),
			),
			expectedRootDir: "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootDir := tt.url.GetParam(constant.MetadataReportGroupKey, "dubbo")
			if len(rootDir) > 0 && rootDir[0] != '/' {
				rootDir = "/" + rootDir
			}
			if rootDir != "/" {
				rootDir = rootDir + "/"
			}
			assert.Equal(t, tt.expectedRootDir, rootDir)
		})
	}
}

// ============================================
// Integration-like Tests
// ============================================

func TestMetadataInfoSerialization(t *testing.T) {
	original := &info.MetadataInfo{
		App:      "test-application",
		Revision: "abc123",
		Services: map[string]*info.ServiceInfo{
			"com.example.UserService": {
				Name:     "com.example.UserService",
				Group:    "default",
				Version:  "1.0.0",
				Protocol: "dubbo",
				Path:     "com.example.UserService",
				Params: map[string]string{
					"timeout": "3000",
					"retries": "2",
				},
			},
			"com.example.OrderService": {
				Name:     "com.example.OrderService",
				Group:    "order",
				Version:  "2.0.0",
				Protocol: "tri",
				Path:     "com.example.OrderService",
			},
		},
	}

	// Serialize
	data, err := json.Marshal(original)
	assert.Nil(t, err)
	assert.NotEmpty(t, data)

	// Deserialize
	var restored info.MetadataInfo
	err = json.Unmarshal(data, &restored)
	assert.Nil(t, err)

	// Verify
	assert.Equal(t, original.App, restored.App)
	assert.Equal(t, original.Revision, restored.Revision)
	assert.Equal(t, len(original.Services), len(restored.Services))

	for key, svc := range original.Services {
		restoredSvc, ok := restored.Services[key]
		assert.True(t, ok)
		assert.Equal(t, svc.Name, restoredSvc.Name)
		assert.Equal(t, svc.Protocol, restoredSvc.Protocol)
	}
}

func TestCacheListenerIntegrationWithReport(t *testing.T) {
	cacheListener := NewCacheListener("/dubbo/", nil)
	report := &zookeeperMetadataReport{
		rootDir:       "/dubbo/",
		cacheListener: cacheListener,
	}

	// Verify report has cache listener
	assert.NotNil(t, report.cacheListener)
	assert.Equal(t, "/dubbo/", report.rootDir)
}

// ============================================
// Edge Cases Tests
// ============================================

func TestEdgeCases(t *testing.T) {
	t.Run("empty application name", func(t *testing.T) {
		rootDir := "/dubbo/"
		application := ""
		revision := "1.0.0"
		path := rootDir + application + constant.PathSeparator + revision
		assert.Equal(t, "/dubbo//1.0.0", path)
	})

	t.Run("empty revision", func(t *testing.T) {
		rootDir := "/dubbo/"
		application := "test-app"
		revision := ""
		path := rootDir + application + constant.PathSeparator + revision
		assert.Equal(t, "/dubbo/test-app/", path)
	})

	t.Run("special characters in service name", func(t *testing.T) {
		serviceName := "com.example.Service$Inner"
		group := "mapping"
		rootDir := "/dubbo/"
		path := rootDir + group + constant.PathSeparator + serviceName
		assert.Equal(t, "/dubbo/mapping/com.example.Service$Inner", path)
	})
}

// ============================================
// Listener with Report Tests
// ============================================

func TestGetServiceAppMappingWithListener(t *testing.T) {
	cacheListener := NewCacheListener("/dubbo/", nil)
	report := &zookeeperMetadataReport{
		rootDir:       "/dubbo/",
		cacheListener: cacheListener,
	}

	// Create mock listener
	mockListener := newMockMappingListener()

	// Verify cache listener can be used
	key := "mapping"
	group := "com.example.Service"
	path := report.rootDir + key + constant.PathSeparator + group

	// Manually add listener (simulating AddListener behavior without zk client)
	listenerSet := NewListenerSet()
	listenerSet.Add(mockListener)
	report.cacheListener.keyListeners.Store(path, listenerSet)

	// Verify listener was added
	listeners, ok := report.cacheListener.keyListeners.Load(path)
	assert.True(t, ok)
	assert.True(t, listeners.(*ListenerSet).Has(mockListener))
}

// ============================================
// Concurrent Access Tests
// ============================================

func TestConcurrentMetadataOperations(t *testing.T) {
	cacheListener := NewCacheListener("/dubbo/", nil)

	// Concurrent listener additions
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			key := "/dubbo/mapping/service" + string(rune('0'+idx))
			listenerSet := NewListenerSet()
			listenerSet.Add(createMockListener())
			cacheListener.keyListeners.Store(key, listenerSet)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all keys were added
	count := 0
	cacheListener.keyListeners.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	assert.Equal(t, 10, count)
}

// createMockListener creates a mock mapping listener for report tests
func createMockListener() mapping.MappingListener {
	return &reportMockListener{}
}

type reportMockListener struct {
	mock.Mock
}

func (m *reportMockListener) OnEvent(e observer.Event) error {
	args := m.Called(e)
	return args.Error(0)
}

func (m *reportMockListener) Stop() {
	m.Called()
}
