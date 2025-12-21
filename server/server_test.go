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

package server

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/global"
)

// Test NewServer creates a server successfully
func TestNewServer(t *testing.T) {
	srv, err := NewServer()
	assert.NoError(t, err)
	assert.NotNil(t, srv)
	assert.NotNil(t, srv.cfg)
	assert.NotNil(t, srv.svcOptsMap)
	assert.NotNil(t, srv.interfaceNameServices)
}

// Test NewServer with options
func TestNewServerWithOptions(t *testing.T) {
	appCfg := &global.ApplicationConfig{
		Name: "test-app",
	}
	srv, err := NewServer(
		SetServerApplication(appCfg),
		WithServerGroup("test-group"),
	)
	assert.NoError(t, err)
	assert.NotNil(t, srv)
	assert.Equal(t, appCfg, srv.cfg.Application)
	assert.Equal(t, "test-group", srv.cfg.Provider.Group)
}

// Test GetServiceOptions
func TestGetServiceOptions(t *testing.T) {
	srv, err := NewServer()
	assert.NoError(t, err)

	svcOpts := defaultServiceOptions()
	svcOpts.Id = "test-service"
	srv.registerServiceOptions(svcOpts)

	retrieved := srv.GetServiceOptions("test-service")
	assert.NotNil(t, retrieved)
	assert.Equal(t, "test-service", retrieved.Id)
}

// Test GetServiceOptions returns nil for non-existent service
func TestGetServiceOptionsNotFound(t *testing.T) {
	srv, err := NewServer()
	assert.NoError(t, err)

	retrieved := srv.GetServiceOptions("non-existent")
	assert.Nil(t, retrieved)
}

// Test GetServiceInfo
func TestGetServiceInfo(t *testing.T) {
	srv, err := NewServer()
	assert.NoError(t, err)

	svcOpts := defaultServiceOptions()
	svcOpts.Id = "test-service"
	svcOpts.info = &common.ServiceInfo{
		InterfaceName: "com.example.TestService",
	}
	srv.registerServiceOptions(svcOpts)

	info := srv.GetServiceInfo("test-service")
	assert.NotNil(t, info)
	assert.Equal(t, "com.example.TestService", info.InterfaceName)
}

// Test GetServiceInfo returns nil for non-existent service
func TestGetServiceInfoNotFound(t *testing.T) {
	srv, err := NewServer()
	assert.NoError(t, err)

	info := srv.GetServiceInfo("non-existent")
	assert.Nil(t, info)
}

// Test GetRPCService
func TestGetRPCService(t *testing.T) {
	srv, err := NewServer()
	assert.NoError(t, err)

	svcOpts := defaultServiceOptions()
	svcOpts.Id = "test-service"
	mockService := &mockRPCService{}
	svcOpts.rpcService = mockService
	srv.registerServiceOptions(svcOpts)

	rpcService := srv.GetRPCService("test-service")
	assert.NotNil(t, rpcService)
	assert.Equal(t, mockService, rpcService)
}

// Test GetRPCService returns nil for non-existent service
func TestGetRPCServiceNotFound(t *testing.T) {
	srv, err := NewServer()
	assert.NoError(t, err)

	rpcService := srv.GetRPCService("non-existent")
	assert.Nil(t, rpcService)
}

// Test GetServiceOptionsByInterfaceName
func TestGetServiceOptionsByInterfaceName(t *testing.T) {
	srv, err := NewServer()
	assert.NoError(t, err)

	svcOpts := defaultServiceOptions()
	svcOpts.Id = "test-service"
	svcOpts.Service.Interface = "com.example.TestService"
	srv.registerServiceOptions(svcOpts)

	retrieved := srv.GetServiceOptionsByInterfaceName("com.example.TestService")
	assert.NotNil(t, retrieved)
	assert.Equal(t, "test-service", retrieved.Id)
}

// Test GetServiceOptionsByInterfaceName returns nil for non-existent interface
func TestGetServiceOptionsByInterfaceNameNotFound(t *testing.T) {
	srv, err := NewServer()
	assert.NoError(t, err)

	retrieved := srv.GetServiceOptionsByInterfaceName("non.existent.Service")
	assert.Nil(t, retrieved)
}

// Test registerServiceOptions
func TestRegisterServiceOptions(t *testing.T) {
	srv, err := NewServer()
	assert.NoError(t, err)

	svcOpts := defaultServiceOptions()
	svcOpts.Id = "test-service"
	svcOpts.Service.Interface = "com.example.TestService"

	srv.registerServiceOptions(svcOpts)

	// Verify via maps
	assert.Equal(t, svcOpts, srv.svcOptsMap["test-service"])
	assert.Equal(t, svcOpts, srv.interfaceNameServices["com.example.TestService"])
}

// Test registerServiceOptions with empty interface
func TestRegisterServiceOptionsEmptyInterface(t *testing.T) {
	srv, err := NewServer()
	assert.NoError(t, err)

	svcOpts := defaultServiceOptions()
	svcOpts.Id = "test-service"
	svcOpts.Service.Interface = ""

	srv.registerServiceOptions(svcOpts)

	// Should be in svcOptsMap but not in interfaceNameServices
	assert.Equal(t, svcOpts, srv.svcOptsMap["test-service"])
	assert.Nil(t, srv.interfaceNameServices[""])
}

// Test SetProviderServices
func TestSetProviderServices(t *testing.T) {
	// Reset the internal services slice
	internalProServices = make([]*InternalService, 0, 16)

	internalService := &InternalService{
		Name:     "test-internal-service",
		Priority: 0,
		Init: func(options *ServiceOptions) (*ServiceDefinition, bool) {
			return &ServiceDefinition{
				Handler: &mockRPCService{},
				Info:    nil,
				Opts:    []ServiceOption{},
			}, true
		},
	}

	SetProviderServices(internalService)

	assert.Len(t, internalProServices, 1)
	assert.Equal(t, "test-internal-service", internalProServices[0].Name)
}

// Test SetProviderServices with empty name
func TestSetProviderServicesEmptyName(t *testing.T) {
	// Reset the internal services slice
	internalProServices = make([]*InternalService, 0, 16)

	internalService := &InternalService{
		Name: "",
	}

	SetProviderServices(internalService)

	assert.Len(t, internalProServices, 0)
}

// Test enhanceServiceInfo with nil info
func TestEnhanceServiceInfoNil(t *testing.T) {
	info := enhanceServiceInfo(nil)
	assert.Nil(t, info)
}

// Test enhanceServiceInfo with methods
func TestEnhanceServiceInfo(t *testing.T) {
	info := &common.ServiceInfo{
		InterfaceName: "com.example.Service",
		Methods: []common.MethodInfo{
			{
				Name: "sayHello",
			},
		},
	}

	result := enhanceServiceInfo(info)
	assert.NotNil(t, result)
	// Should have doubled methods (original + case-swapped)
	assert.Equal(t, 2, len(result.Methods))
	assert.Equal(t, "sayHello", result.Methods[0].Name)
	// The swapped version should have capitalized first letter
	assert.NotEqual(t, "sayHello", result.Methods[1].Name)
}

// Test getMetadataPort with default protocol
func TestGetMetadataPortWithDefaultProtocol(t *testing.T) {
	opts := defaultServerOptions()
	opts.Application.MetadataServicePort = ""
	opts.Protocols = map[string]*global.ProtocolConfig{
		"dubbo": {
			Port: "20880",
		},
	}

	port := getMetadataPort(opts)
	assert.Equal(t, 20880, port)
}

// Test getMetadataPort with explicit port
func TestGetMetadataPortExplicit(t *testing.T) {
	opts := defaultServerOptions()
	opts.Application.MetadataServicePort = "30880"

	port := getMetadataPort(opts)
	assert.Equal(t, 30880, port)
}

// Test getMetadataPort with no port
func TestGetMetadataPortNoPort(t *testing.T) {
	opts := defaultServerOptions()
	opts.Application.MetadataServicePort = ""
	opts.Protocols = map[string]*global.ProtocolConfig{}

	port := getMetadataPort(opts)
	assert.Equal(t, 0, port)
}

// Test getMetadataPort with invalid port
func TestGetMetadataPortInvalid(t *testing.T) {
	opts := defaultServerOptions()
	opts.Application.MetadataServicePort = "invalid"

	port := getMetadataPort(opts)
	assert.Equal(t, 0, port)
}

// Mock RPCService for testing
type mockRPCService struct{}

func (m *mockRPCService) Invoke(methodName string, params []any, results []any) error {
	return nil
}

func (m *mockRPCService) Reference() string {
	return "com.example.MockService"
}

// Test concurrency: multiple goroutines registering services
func TestConcurrentServiceRegistration(t *testing.T) {
	srv, err := NewServer()
	assert.NoError(t, err)

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			svcOpts := defaultServiceOptions()
			svcOpts.Id = "service-" + string(rune(idx))
			srv.registerServiceOptions(svcOpts)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all services were registered
	assert.Equal(t, 10, len(srv.svcOptsMap))
}

// Test Register with ServiceInfo
func TestRegisterWithServiceInfo(t *testing.T) {
	srv, err := NewServer()
	assert.NoError(t, err)

	handler := &mockRPCService{}
	info := &common.ServiceInfo{
		InterfaceName: "com.example.Service",
		Methods: []common.MethodInfo{
			{Name: "Method1"},
		},
	}

	err = srv.Register(handler, info)
	assert.NoError(t, err)

	// Service should be registered
	svcOpts := srv.GetServiceOptions(handler.Reference())
	assert.NotNil(t, svcOpts)
}

// Test RegisterService without ServiceInfo
func TestRegisterService(t *testing.T) {
	srv, err := NewServer()
	assert.NoError(t, err)

	handler := &mockRPCService{}
	err = srv.RegisterService(handler)
	assert.NoError(t, err)

	// Service should be registered with handler reference name
	svcOpts := srv.GetServiceOptions(handler.Reference())
	assert.NotNil(t, svcOpts)
}

// Test Register with handler that has method config
func TestRegisterWithMethodConfig(t *testing.T) {
	srv, err := NewServer()
	assert.NoError(t, err)

	handler := &mockRPCService{}
	info := &common.ServiceInfo{
		InterfaceName: "com.example.TestService",
	}

	err = srv.Register(
		handler,
		info,
		WithInterface("com.example.TestService"),
		WithGroup("test-group"),
		WithVersion("1.0.0"),
	)
	assert.NoError(t, err)

	svcOpts := srv.GetServiceOptions(handler.Reference())
	assert.NotNil(t, svcOpts)
	if svcOpts != nil {
		assert.Equal(t, "test-group", svcOpts.Service.Group)
		assert.Equal(t, "1.0.0", svcOpts.Service.Version)
	}
}

// Test genSvcOpts with missing server config
func TestGenSvcOptsWithMissingServerConfig(t *testing.T) {
	srv := &Server{
		cfg: nil,
	}

	handler := &mockRPCService{}
	_, err := srv.genSvcOpts(handler, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "has not been initialized")
}

// Test exportServices with empty service map
func TestExportServicesEmpty(t *testing.T) {
	srv, err := NewServer()
	assert.NoError(t, err)

	err = srv.exportServices()
	assert.NoError(t, err)
}

// Test NewServer with invalid config
func TestNewServerWithInvalidConfig(t *testing.T) {
	// This should not error during creation, but config validation might fail on use
	srv, err := NewServer(WithServerGroup("test"))
	assert.NoError(t, err)
	assert.NotNil(t, srv)
	assert.Equal(t, "test", srv.cfg.Provider.Group)
}
