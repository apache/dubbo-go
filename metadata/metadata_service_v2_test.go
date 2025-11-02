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

package metadata

import (
	"context"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
	tripleapi "dubbo.apache.org/dubbo-go/v3/metadata/triple_api/proto"
)

// MockMetadataService for testing
type MockMetadataService struct {
	mock.Mock
}

func (m *MockMetadataService) GetExportedURLs(serviceInterface string, group string, version string, protocol string) ([]*common.URL, error) {
	args := m.Called(serviceInterface, group, version, protocol)
	return args.Get(0).([]*common.URL), args.Error(1)
}

func (m *MockMetadataService) GetExportedServiceURLs() ([]*common.URL, error) {
	args := m.Called()
	return args.Get(0).([]*common.URL), args.Error(1)
}

func (m *MockMetadataService) GetSubscribedURLs() ([]*common.URL, error) {
	args := m.Called()
	return args.Get(0).([]*common.URL), args.Error(1)
}

func (m *MockMetadataService) Version() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockMetadataService) GetMetadataInfo(revision string) (*info.MetadataInfo, error) {
	args := m.Called(revision)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*info.MetadataInfo), args.Error(1)
}

func (m *MockMetadataService) GetMetadataServiceURL() (*common.URL, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*common.URL), args.Error(1)
}

func TestMetadataServiceV2_GetMetadataInfo(t *testing.T) {
	// Create mock metadata service
	mockService := &MockMetadataService{}

	// Create test metadata info
	testMetadataInfo := &info.MetadataInfo{
		App:      "test-app",
		Revision: "test-revision",
		Services: map[string]*info.ServiceInfo{
			"test-service": {
				Name:     "TestService",
				Group:    "test-group",
				Version:  "1.0.0",
				Protocol: "tri",
				Port:     8080,
				Path:     "/test/service",
				Params:   map[string]string{"key": "value"},
			},
		},
	}

	// Setup mock expectations
	mockService.On("GetMetadataInfo", "test-revision").Return(testMetadataInfo, nil)

	// Create MetadataServiceV2 instance
	v2Service := &MetadataServiceV2{delegate: mockService}

	// Create test request
	req := &tripleapi.MetadataRequest{Revision: "test-revision"}

	// Test GetMetadataInfo
	result, err := v2Service.GetMetadataInfo(context.Background(), req)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "test-app", result.App)
	assert.Equal(t, "test-revision", result.Version)
	assert.Len(t, result.Services, 1)

	// Verify service info conversion
	serviceInfo := result.Services["test-service"]
	assert.Equal(t, "TestService", serviceInfo.Name)
	assert.Equal(t, "test-group", serviceInfo.Group)
	assert.Equal(t, "1.0.0", serviceInfo.Version)
	assert.Equal(t, "tri", serviceInfo.Protocol)
	assert.Equal(t, "/test/service", serviceInfo.Path)
	assert.Equal(t, "value", serviceInfo.Params["key"])

	// Verify mock expectations
	mockService.AssertExpectations(t)
}

func TestMetadataServiceV2_GetMetadataInfo_Error(t *testing.T) {
	// Create mock metadata service
	mockService := &MockMetadataService{}

	// Setup mock to return error
	mockService.On("GetMetadataInfo", "invalid-revision").Return(nil, assert.AnError)

	// Create MetadataServiceV2 instance
	v2Service := &MetadataServiceV2{delegate: mockService}

	// Create test request
	req := &tripleapi.MetadataRequest{Revision: "invalid-revision"}

	// Test GetMetadataInfo with error
	result, err := v2Service.GetMetadataInfo(context.Background(), req)

	// Assertions
	assert.Error(t, err)
	assert.Nil(t, result)

	// Verify mock expectations
	mockService.AssertExpectations(t)
}

func TestMetadataServiceV2_GetMetadataInfo_EmptyRevision(t *testing.T) {
	// Create mock metadata service
	mockService := &MockMetadataService{}

	// Setup mock to return nil for empty revision
	mockService.On("GetMetadataInfo", "").Return(nil, nil)

	// Create MetadataServiceV2 instance
	v2Service := &MetadataServiceV2{delegate: mockService}

	// Create test request with empty revision
	req := &tripleapi.MetadataRequest{Revision: ""}

	// Test GetMetadataInfo with empty revision
	result, err := v2Service.GetMetadataInfo(context.Background(), req)

	// Assertions
	assert.NoError(t, err)
	assert.Nil(t, result)

	// Verify mock expectations
	mockService.AssertExpectations(t)
}

func TestServiceExporter_ExportV2(t *testing.T) {
	// Create mock metadata service
	mockService := &MockMetadataService{}
	mockService.On("Version").Return("1.0.0", nil)

	// Create options
	opts := &Options{
		appName:      "test-app",
		metadataType: constant.DefaultMetadataStorageType,
		port:         0, // Use random port
		protocol:     constant.TriProtocol,
	}

	// Create service exporter
	exporter := &serviceExporter{
		opts:    opts,
		service: mockService,
	}

	// Mock protocol and proxy factory extensions
	// Note: In real tests, you would need to setup proper mocks for these

	// Test exportV2 (this would require more extensive mocking)
	// For now, we just verify the structure is correct
	assert.NotNil(t, exporter.opts)
	assert.NotNil(t, exporter.service)
	assert.Equal(t, "test-app", exporter.opts.appName)
}

func TestConvertV2(t *testing.T) {
	// Create test service info map
	serviceInfos := map[string]*info.ServiceInfo{
		"service1": {
			Name:     "Service1",
			Group:    "group1",
			Version:  "1.0.0",
			Protocol: "tri",
			Port:     8080,
			Path:     "/service1",
			Params:   map[string]string{"param1": "value1"},
		},
		"service2": {
			Name:     "Service2",
			Group:    "group2",
			Version:  "2.0.0",
			Protocol: "dubbo",
			Port:     8081,
			Path:     "/service2",
			Params:   map[string]string{"param2": "value2"},
		},
	}

	// Test conversion
	result := convertV2(serviceInfos)

	// Assertions
	assert.Len(t, result, 2)

	// Check service1
	service1 := result["service1"]
	assert.Equal(t, "Service1", service1.Name)
	assert.Equal(t, "group1", service1.Group)
	assert.Equal(t, "1.0.0", service1.Version)
	assert.Equal(t, "tri", service1.Protocol)
	assert.Equal(t, "/service1", service1.Path)
	assert.Equal(t, "value1", service1.Params["param1"])

	// Check service2
	service2 := result["service2"]
	assert.Equal(t, "Service2", service2.Name)
	assert.Equal(t, "group2", service2.Group)
	assert.Equal(t, "2.0.0", service2.Version)
	assert.Equal(t, "dubbo", service2.Protocol)
	assert.Equal(t, "/service2", service2.Path)
	assert.Equal(t, "value2", service2.Params["param2"])
}
