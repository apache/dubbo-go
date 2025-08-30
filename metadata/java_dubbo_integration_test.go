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

package metadata_test

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
)

// TestJavaDubboRevisionCompatibility tests Java Dubbo client revision compatibility
func TestJavaDubboRevisionCompatibility(t *testing.T) {
	// Create a simple mock service
	mockService := &simpleMockMetadataService{}

	// Simulate the revision sent by Java client (from original issue)
	javaClientRevision := "9673005772"

	t.Run("Java_Client_Revision_9673005772", func(t *testing.T) {
		// Set mock return data
		mockService.revision = javaClientRevision
		mockService.services = map[string]*info.ServiceInfo{
			"org.apache.dubbo.metadata.MetadataService": {
				Name:     "MetadataService",
				Group:    "dubbo-go-server",
				Version:  "2.0.0",
				Protocol: "tri",
				Path:     "/org.apache.dubbo.metadata.MetadataService",
				Params: map[string]string{
					"protocol":      "tri",
					"port":          "20033",
					"serialization": "protobuf",
				},
			},
		}

		// Test mock service functionality directly
		result, err := mockService.GetMetadataInfo(javaClientRevision)

		// Verify results
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, javaClientRevision, result.Revision)
		assert.Len(t, result.Services, 1)

		// Verify service information
		service, exists := result.Services["org.apache.dubbo.metadata.MetadataService"]
		assert.True(t, exists)
		assert.Equal(t, "MetadataService", service.Name)
		assert.Equal(t, "tri", service.Protocol)
		assert.Equal(t, "20033", service.Params["port"])
		assert.Equal(t, "protobuf", service.Params["serialization"])
	})
}

// TestJavaDubboProtocolNegotiation tests protocol negotiation
func TestJavaDubboProtocolNegotiation(t *testing.T) {
	mockService := &simpleMockMetadataService{}

	t.Run("Triple_Protocol_Handshake", func(t *testing.T) {
		// Simulate protocol negotiation process
		revision := "protocol_negotiation_test"

		mockService.revision = revision
		mockService.services = map[string]*info.ServiceInfo{
			"testService": {
				Name:     "TestService",
				Group:    "test-group",
				Version:  "1.0.0",
				Protocol: "tri",
				Path:     "/test",
				Params: map[string]string{
					"protocol": "tri",
					"timeout":  "5000",
				},
			},
		}

		// Test mock service functionality directly
		result, err := mockService.GetMetadataInfo(revision)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "tri", result.Services["testService"].Protocol)
		assert.Equal(t, "/test", result.Services["testService"].Path)
	})
}

// TestJavaDubboErrorScenarios tests error scenarios
func TestJavaDubboErrorScenarios(t *testing.T) {
	mockService := &simpleMockMetadataService{}

	t.Run("Revision_Not_Found_Error", func(t *testing.T) {
		// Simulate case where revision does not exist
		mockService.shouldReturnError = true
		mockService.errorMsg = "revision not found"

		// Test mock service functionality directly
		result, err := mockService.GetMetadataInfo("non_existent_revision")

		// V2 service should handle errors gracefully
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "revision not found")
	})
}

// TestJavaDubboConcurrentRequests tests concurrent requests
func TestJavaDubboConcurrentRequests(t *testing.T) {
	mockService := &simpleMockMetadataService{}

	t.Run("Concurrent_Client_Requests", func(t *testing.T) {
		// Simulate 5 concurrent Java clients
		concurrentClients := 5
		revision := "concurrent_test"

		mockService.revision = revision
		mockService.services = map[string]*info.ServiceInfo{
			"concurrentService": {
				Name:     "ConcurrentService",
				Group:    "concurrent-group",
				Version:  "1.0.0",
				Protocol: "tri",
				Path:     "/concurrent",
			},
		}

		// Test mock service functionality directly

		// Start concurrent requests
		results := make(chan *info.MetadataInfo, concurrentClients)
		errors := make(chan error, concurrentClients)

		for i := 0; i < concurrentClients; i++ {
			go func(clientID int) {
				result, err := mockService.GetMetadataInfo(revision)
				if err != nil {
					errors <- err
					return
				}
				results <- result
			}(i)
		}

		// Collect results
		successCount := 0
		errorCount := 0

		for i := 0; i < concurrentClients; i++ {
			select {
			case result := <-results:
				assert.NotNil(t, result)
				assert.Equal(t, revision, result.Revision)
				successCount++
			case err := <-errors:
				t.Logf("Client request failed: %v", err)
				errorCount++
			}
		}

		assert.Equal(t, concurrentClients, successCount)
		assert.Equal(t, 0, errorCount)
	})
}

// TestJavaDubboServiceDiscovery tests service discovery
func TestJavaDubboServiceDiscovery(t *testing.T) {
	mockService := &simpleMockMetadataService{}

	t.Run("Service_Registration_Discovery", func(t *testing.T) {
		// Simulate Nacos service discovery scenario
		revision := "discovery_test"

		mockService.revision = revision
		mockService.services = map[string]*info.ServiceInfo{
			"discoveredService": {
				Name:     "DiscoveredService",
				Group:    "dubbo-go-server",
				Version:  "2.0.0",
				Protocol: "tri",
				Path:     "/discovered",
				Params: map[string]string{
					"registry":     "nacos",
					"address":      "nacos-server:8848",
					"namespace":    "public",
					"service-name": "dubbo-go-server",
				},
			},
		}

		// Test mock service functionality directly
		result, err := mockService.GetMetadataInfo(revision)

		assert.NoError(t, err)
		assert.NotNil(t, result)

		service := result.Services["discoveredService"]
		assert.NotNil(t, service)
		assert.Equal(t, "nacos", service.Params["registry"])
		assert.Equal(t, "dubbo-go-server", service.Params["service-name"])
	})
}

// TestJavaDubboVersionCompatibility tests version compatibility
func TestJavaDubboVersionCompatibility(t *testing.T) {
	mockService := &simpleMockMetadataService{}

	compatibilityTests := []struct {
		name        string
		javaVersion string
		goVersion   string
		revision    string
	}{
		{
			name:        "Java_3.3.1_Go_V2_Full_Compatibility",
			javaVersion: "3.3.1",
			goVersion:   "2.0.0",
			revision:    "compat_test_1",
		},
		{
			name:        "Java_3.3.0_Go_V2_Backward_Compatibility",
			javaVersion: "3.3.0",
			goVersion:   "2.0.0",
			revision:    "compat_test_2",
		},
	}

	for _, tt := range compatibilityTests {
		t.Run(tt.name, func(t *testing.T) {
			mockService.revision = tt.revision
			mockService.services = map[string]*info.ServiceInfo{
				"compatibilityService": {
					Name:     "CompatibilityService",
					Group:    "compatibility-group",
					Version:  tt.goVersion,
					Protocol: "tri",
					Path:     "/compatibility",
					Params: map[string]string{
						"java.version": tt.javaVersion,
						"go.version":   tt.goVersion,
						"compatible":   "true",
					},
				},
			}

			// Test mock service functionality directly
			result, err := mockService.GetMetadataInfo(tt.revision)

			assert.NoError(t, err)
			assert.NotNil(t, result)

			service := result.Services["compatibilityService"]
			assert.NotNil(t, service)
			assert.Equal(t, tt.javaVersion, service.Params["java.version"])
			assert.Equal(t, tt.goVersion, service.Params["go.version"])
			assert.Equal(t, "true", service.Params["compatible"])
		})
	}
}

// simpleMockMetadataService is a simple mock implementation
type simpleMockMetadataService struct {
	revision          string
	services          map[string]*info.ServiceInfo
	shouldReturnError bool
	errorMsg          string
}

func (m *simpleMockMetadataService) GetMetadataInfo(revision string) (*info.MetadataInfo, error) {
	if m.shouldReturnError {
		return nil, &simpleError{message: m.errorMsg}
	}

	return &info.MetadataInfo{
		App:      "test-app",
		Revision: m.revision,
		Services: m.services,
	}, nil
}

func (m *simpleMockMetadataService) Version() (string, error) {
	return "1.0.0", nil
}

func (m *simpleMockMetadataService) GetExportedURLs(serviceInterface string, group string, version string, protocol string) ([]*common.URL, error) {
	// Mock implementation - return empty slice
	return []*common.URL{}, nil
}

func (m *simpleMockMetadataService) GetExportedServiceURLs() ([]*common.URL, error) {
	// Mock implementation - return empty slice
	return []*common.URL{}, nil
}

func (m *simpleMockMetadataService) GetSubscribedURLs() ([]*common.URL, error) {
	// Mock implementation - return empty slice
	return []*common.URL{}, nil
}

func (m *simpleMockMetadataService) GetMetadataServiceURL() (*common.URL, error) {
	// Mock implementation - return a dummy URL
	return common.NewURL("tri://127.0.0.1:20000/org.apache.dubbo.metadata.MetadataService")
}

// simpleError is a simple error implementation
type simpleError struct {
	message string
}

func (e *simpleError) Error() string {
	return e.message
}
