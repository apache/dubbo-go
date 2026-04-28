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
	"context"
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"
)

import (
	gostlogger "github.com/dubbogo/gost/log/logger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/graceful_shutdown"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

//go:linkname extensionProtocols dubbo.apache.org/dubbo-go/v3/common/extension.protocols
var extensionProtocols *extension.Registry[func() base.Protocol]

//go:linkname gracefulShutdownInitOnce dubbo.apache.org/dubbo-go/v3/graceful_shutdown.initOnce
var gracefulShutdownInitOnce sync.Once

//go:linkname gracefulShutdownProtocols dubbo.apache.org/dubbo-go/v3/graceful_shutdown.protocols
var gracefulShutdownProtocols map[string]struct{}

//go:linkname gracefulShutdownProMu dubbo.apache.org/dubbo-go/v3/graceful_shutdown.proMu
var gracefulShutdownProMu sync.Mutex

//go:linkname gracefulShutdownConfigMu dubbo.apache.org/dubbo-go/v3/graceful_shutdown.shutdownConfigMu
var gracefulShutdownConfigMu sync.RWMutex

//go:linkname gracefulShutdownConfig dubbo.apache.org/dubbo-go/v3/graceful_shutdown.shutdownConfig
var gracefulShutdownConfig *global.ShutdownConfig

//go:linkname gracefulShutdownOnce dubbo.apache.org/dubbo-go/v3/graceful_shutdown.shutdownOnce
var gracefulShutdownOnce sync.Once

//go:linkname gracefulShutdownStarted dubbo.apache.org/dubbo-go/v3/graceful_shutdown.shutdownStarted
var gracefulShutdownStarted atomic.Bool

//go:linkname gracefulShutdownDone dubbo.apache.org/dubbo-go/v3/graceful_shutdown.shutdownDone
var gracefulShutdownDone chan struct{}

//go:linkname gracefulShutdownResult dubbo.apache.org/dubbo-go/v3/graceful_shutdown.shutdownResult
var gracefulShutdownResult error

//go:linkname gracefulShutdownSignalNotify dubbo.apache.org/dubbo-go/v3/graceful_shutdown.signalNotify
var gracefulShutdownSignalNotify func(chan<- os.Signal, ...os.Signal)

func resetInternalProviderServicesForTest(t *testing.T) {
	t.Helper()

	internalProLock.Lock()
	originalServices := internalProServices
	internalProServices = nil
	internalProLock.Unlock()

	t.Cleanup(func() {
		internalProLock.Lock()
		defer internalProLock.Unlock()
		internalProServices = originalServices
	})
}

type mockServeProtocol struct {
	base.BaseProtocol
}

type mockServeRegistryFactoryProtocol struct {
	base.BaseProtocol
}

type mockServeRegistry struct{}

func (p *mockServeRegistryFactoryProtocol) GetRegistries() []registry.Registry {
	return []registry.Registry{&mockServeRegistry{}}
}

func (r *mockServeRegistry) GetURL() *common.URL {
	return &common.URL{}
}

func (r *mockServeRegistry) IsAvailable() bool {
	return true
}

func (r *mockServeRegistry) Destroy() {}

func (r *mockServeRegistry) Register(*common.URL) error {
	return nil
}

func (r *mockServeRegistry) UnRegister(*common.URL) error {
	return nil
}

func (r *mockServeRegistry) Subscribe(*common.URL, registry.NotifyListener) error {
	return nil
}

func (r *mockServeRegistry) UnSubscribe(*common.URL, registry.NotifyListener) error {
	return nil
}

func (r *mockServeRegistry) LoadSubscribeInstances(*common.URL, registry.NotifyListener) error {
	return nil
}

func (r *mockServeRegistry) RegisterService() error {
	return nil
}

func (r *mockServeRegistry) UnRegisterService() error {
	return nil
}

type countingServeExporter struct {
	invoker       base.Invoker
	unexportCount *atomic.Int32
}

func (e *countingServeExporter) GetInvoker() base.Invoker {
	return e.invoker
}

func (e *countingServeExporter) UnExport() {
	if e.unexportCount != nil {
		e.unexportCount.Add(1)
	}
	if e.invoker != nil {
		e.invoker.Destroy()
	}
}

type countingServeProtocol struct {
	base.BaseProtocol
	exportCount   *atomic.Int32
	unexportCount *atomic.Int32
}

func (p *countingServeProtocol) Export(invoker base.Invoker) base.Exporter {
	if p.exportCount != nil {
		p.exportCount.Add(1)
	}
	return &countingServeExporter{
		invoker:       invoker,
		unexportCount: p.unexportCount,
	}
}

type countingServeRegistry struct {
	registerCount   *atomic.Int32
	unregisterCount *atomic.Int32
	registerBlock   <-chan struct{}
}

func (r *countingServeRegistry) GetURL() *common.URL {
	return &common.URL{}
}

func (r *countingServeRegistry) IsAvailable() bool {
	return true
}

func (r *countingServeRegistry) Destroy() {}

func (r *countingServeRegistry) Register(*common.URL) error {
	return nil
}

func (r *countingServeRegistry) UnRegister(*common.URL) error {
	return nil
}

func (r *countingServeRegistry) Subscribe(*common.URL, registry.NotifyListener) error {
	return nil
}

func (r *countingServeRegistry) UnSubscribe(*common.URL, registry.NotifyListener) error {
	return nil
}

func (r *countingServeRegistry) LoadSubscribeInstances(*common.URL, registry.NotifyListener) error {
	return nil
}

func (r *countingServeRegistry) RegisterService() error {
	if r.registerCount != nil {
		r.registerCount.Add(1)
	}
	if r.registerBlock != nil {
		<-r.registerBlock
	}
	return nil
}

func (r *countingServeRegistry) UnRegisterService() error {
	if r.unregisterCount != nil {
		r.unregisterCount.Add(1)
	}
	return nil
}

type countingServeRegistryFactoryProtocol struct {
	base.BaseProtocol
	registry registry.Registry
}

func (p *countingServeRegistryFactoryProtocol) GetRegistries() []registry.Registry {
	return []registry.Registry{p.registry}
}

func registerServeTestProtocols(t *testing.T) {
	t.Helper()

	originalProtocols := extensionProtocols.Snapshot()
	extension.SetProtocol("dubbo", func() base.Protocol {
		return &mockServeProtocol{BaseProtocol: base.NewBaseProtocol()}
	})
	extension.SetProtocol(constant.RegistryKey, func() base.Protocol {
		return &mockServeRegistryFactoryProtocol{BaseProtocol: base.NewBaseProtocol()}
	})
	t.Cleanup(func() {
		for name, factory := range originalProtocols {
			extension.SetProtocol(name, factory)
		}
		if _, ok := originalProtocols["dubbo"]; !ok {
			extension.UnregisterProtocol("dubbo")
		}
		if _, ok := originalProtocols[constant.RegistryKey]; !ok {
			extension.UnregisterProtocol(constant.RegistryKey)
		}
	})
}

func registerCountingServeTestProtocols(
	t *testing.T,
	exportCount, unexportCount, registerCount, unregisterCount *atomic.Int32,
	registerBlock <-chan struct{},
) {
	t.Helper()

	originalProtocols := extensionProtocols.Snapshot()
	extension.SetProtocol(constant.TriProtocol, func() base.Protocol {
		return &countingServeProtocol{
			BaseProtocol:  base.NewBaseProtocol(),
			exportCount:   exportCount,
			unexportCount: unexportCount,
		}
	})
	extension.SetProtocol(constant.RegistryKey, func() base.Protocol {
		return &countingServeRegistryFactoryProtocol{
			BaseProtocol: base.NewBaseProtocol(),
			registry: &countingServeRegistry{
				registerCount:   registerCount,
				unregisterCount: unregisterCount,
				registerBlock:   registerBlock,
			},
		}
	})
	t.Cleanup(func() {
		for name, factory := range originalProtocols {
			extension.SetProtocol(name, factory)
		}
		if _, ok := originalProtocols[constant.TriProtocol]; !ok {
			extension.UnregisterProtocol(constant.TriProtocol)
		}
		if _, ok := originalProtocols[constant.RegistryKey]; !ok {
			extension.UnregisterProtocol(constant.RegistryKey)
		}
	})
}

func resetGracefulShutdownStateForTest(t *testing.T) {
	t.Helper()

	gracefulShutdownInitOnce = sync.Once{}
	gracefulShutdownProtocols = nil
	gracefulShutdownProMu = sync.Mutex{}
	gracefulShutdownConfigMu = sync.RWMutex{}
	gracefulShutdownConfig = nil
	gracefulShutdownOnce = sync.Once{}
	gracefulShutdownStarted = atomic.Bool{}
	gracefulShutdownDone = make(chan struct{})
	gracefulShutdownResult = nil
	gracefulShutdownSignalNotify = signal.Notify
}

func TestServeContextReturnsAfterContextCancellation(t *testing.T) {
	resetGracefulShutdownStateForTest(t)
	t.Cleanup(func() {
		resetGracefulShutdownStateForTest(t)
	})
	resetInternalProviderServicesForTest(t)
	registerServeTestProtocols(t)

	internalSignal := false
	shutdownCfg := global.DefaultShutdownConfig()
	shutdownCfg.InternalSignal = &internalSignal
	shutdownCfg.ConsumerUpdateWaitTime = "0s"
	shutdownCfg.StepTimeout = "0s"
	shutdownCfg.NotifyTimeout = "10ms"
	shutdownCfg.OfflineRequestWindowTimeout = "0s"

	srv, err := NewServer(SetServerShutdown(shutdownCfg))
	require.NoError(t, err)
	require.NoError(t, srv.Register(&MockServerRPCService{}, nil))

	ctx, cancel := context.WithCancel(context.Background())
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- srv.ServeContext(ctx)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-serveDone:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("ServeContext did not return after context cancellation")
	}

	select {
	case <-graceful_shutdown.Done():
	case <-time.After(time.Second):
		t.Fatal("process-level graceful shutdown did not finish after context cancellation")
	}
}

func TestServeContextDoesNotStartWhenContextAlreadyCanceled(t *testing.T) {
	resetGracefulShutdownStateForTest(t)
	t.Cleanup(func() {
		resetGracefulShutdownStateForTest(t)
	})
	resetInternalProviderServicesForTest(t)

	var exportCount, unexportCount, registerCount, unregisterCount atomic.Int32
	registerCountingServeTestProtocols(t, &exportCount, &unexportCount, &registerCount, &unregisterCount, nil)

	internalSignal := false
	shutdownCfg := global.DefaultShutdownConfig()
	shutdownCfg.InternalSignal = &internalSignal
	protocols := map[string]*global.ProtocolConfig{
		constant.TriProtocol: global.DefaultProtocolConfig(),
	}

	srv, err := NewServer(SetServerShutdown(shutdownCfg), SetServerProtocols(protocols))
	require.NoError(t, err)
	require.NoError(t, srv.Register(&MockServerRPCService{}, nil))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = srv.ServeContext(ctx)
	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, int32(0), exportCount.Load())
	assert.Equal(t, int32(0), unexportCount.Load())
	assert.Equal(t, int32(0), registerCount.Load())
	assert.Equal(t, int32(0), unregisterCount.Load())
}

func TestServeContextRollsBackWhenCanceledDuringStartup(t *testing.T) {
	resetGracefulShutdownStateForTest(t)
	t.Cleanup(func() {
		resetGracefulShutdownStateForTest(t)
	})
	resetInternalProviderServicesForTest(t)

	var exportCount, unexportCount, registerCount, unregisterCount atomic.Int32
	registerBlock := make(chan struct{})
	registerCountingServeTestProtocols(t, &exportCount, &unexportCount, &registerCount, &unregisterCount, registerBlock)

	internalSignal := false
	shutdownCfg := global.DefaultShutdownConfig()
	shutdownCfg.InternalSignal = &internalSignal
	protocols := map[string]*global.ProtocolConfig{
		constant.TriProtocol: global.DefaultProtocolConfig(),
	}

	srv, err := NewServer(SetServerShutdown(shutdownCfg), SetServerProtocols(protocols))
	require.NoError(t, err)
	require.NoError(t, srv.Register(&MockServerRPCService{}, nil))

	ctx, cancel := context.WithCancel(context.Background())
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- srv.ServeContext(ctx)
	}()

	require.Eventually(t, func() bool {
		return registerCount.Load() == 1
	}, time.Second, 10*time.Millisecond)

	cancel()
	close(registerBlock)

	select {
	case err := <-serveDone:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("ServeContext did not return after startup cancellation")
	}

	assert.Equal(t, int32(1), exportCount.Load())
	assert.Equal(t, int32(1), unexportCount.Load())
	assert.Equal(t, int32(1), registerCount.Load())
	assert.Equal(t, int32(1), unregisterCount.Load())
}

func TestServeContextDoesNotRestartAfterGracefulShutdownCompletes(t *testing.T) {
	resetGracefulShutdownStateForTest(t)
	t.Cleanup(func() {
		resetGracefulShutdownStateForTest(t)
	})
	resetInternalProviderServicesForTest(t)

	var exportCount, unexportCount, registerCount, unregisterCount atomic.Int32
	registerCountingServeTestProtocols(t, &exportCount, &unexportCount, &registerCount, &unregisterCount, nil)

	internalSignal := false
	shutdownCfg := global.DefaultShutdownConfig()
	shutdownCfg.InternalSignal = &internalSignal
	shutdownCfg.ConsumerUpdateWaitTime = "0s"
	shutdownCfg.StepTimeout = "0s"
	shutdownCfg.NotifyTimeout = "0s"
	shutdownCfg.OfflineRequestWindowTimeout = "0s"

	srv, err := NewServer(SetServerShutdown(shutdownCfg))
	require.NoError(t, err)
	require.NoError(t, srv.Register(&MockServerRPCService{}, nil))

	ctx, cancel := context.WithCancel(context.Background())
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- srv.ServeContext(ctx)
	}()

	require.Eventually(t, func() bool {
		return registerCount.Load() == 1
	}, time.Second, 10*time.Millisecond)

	cancel()
	select {
	case err := <-serveDone:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("ServeContext did not return after context cancellation")
	}
	select {
	case <-graceful_shutdown.Done():
	case <-time.After(time.Second):
		t.Fatal("process-level graceful shutdown did not finish after context cancellation")
	}

	exportCountAfterShutdown := exportCount.Load()
	unexportCountAfterShutdown := unexportCount.Load()
	registerCountAfterShutdown := registerCount.Load()
	unregisterCountAfterShutdown := unregisterCount.Load()

	err = srv.ServeContext(context.Background())
	require.Error(t, err)
	assert.Equal(t, int32(1), registerCountAfterShutdown)
	assert.Equal(t, exportCountAfterShutdown, exportCount.Load())
	assert.Equal(t, unexportCountAfterShutdown, unexportCount.Load())
	assert.Equal(t, registerCountAfterShutdown, registerCount.Load())
	assert.Equal(t, unregisterCountAfterShutdown, unregisterCount.Load())
}

// Test NewServer creates a server successfully
func TestNewServer(t *testing.T) {
	srv, err := NewServer()
	require.NoError(t, err)
	assert.NotNil(t, srv)

	// Verify server is properly initialized by using public API
	// Try to register and retrieve a service to verify internal maps work
	svcOpts := defaultServiceOptions()
	svcOpts.Id = "init-test-service"
	srv.registerServiceOptions(svcOpts)

	retrieved := srv.GetServiceOptions("init-test-service")
	assert.NotNil(t, retrieved, "Server should have properly initialized service maps")
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
	require.NoError(t, err)
	assert.NotNil(t, srv)

	// Verify configuration by registering a service and checking its options
	handler := &mockServerRPCService{}
	err = srv.Register(handler, nil)
	require.NoError(t, err)

	svcOpts := srv.GetServiceOptions(handler.Reference())
	assert.NotNil(t, svcOpts)
	assert.Equal(t, "test-group", svcOpts.Service.Group)
}

// Test GetServiceOptions
func TestGetServiceOptions(t *testing.T) {
	srv, err := NewServer()
	require.NoError(t, err)

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
	require.NoError(t, err)

	retrieved := srv.GetServiceOptions("non-existent")
	assert.Nil(t, retrieved)
}

// Test GetServiceInfo
func TestGetServiceInfo(t *testing.T) {
	srv, err := NewServer()
	require.NoError(t, err)

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
	require.NoError(t, err)

	info := srv.GetServiceInfo("non-existent")
	assert.Nil(t, info)
}

// Test GetRPCService
func TestGetRPCService(t *testing.T) {
	srv, err := NewServer()
	require.NoError(t, err)

	svcOpts := defaultServiceOptions()
	svcOpts.Id = "test-service"
	mockService := &mockServerRPCService{}
	svcOpts.rpcService = mockService
	srv.registerServiceOptions(svcOpts)

	rpcService := srv.GetRPCService("test-service")
	assert.NotNil(t, rpcService)
	assert.Equal(t, mockService, rpcService)
}

// Test GetRPCService returns nil for non-existent service
func TestGetRPCServiceNotFound(t *testing.T) {
	srv, err := NewServer()
	require.NoError(t, err)

	rpcService := srv.GetRPCService("non-existent")
	assert.Nil(t, rpcService)
}

// Test GetServiceOptionsByInterfaceName
func TestGetServiceOptionsByInterfaceName(t *testing.T) {
	srv, err := NewServer()
	require.NoError(t, err)

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
	require.NoError(t, err)

	retrieved := srv.GetServiceOptionsByInterfaceName("non.existent.Service")
	assert.Nil(t, retrieved)
}

// Test registerServiceOptions
func TestRegisterServiceOptions(t *testing.T) {
	srv, err := NewServer()
	require.NoError(t, err)

	svcOpts := defaultServiceOptions()
	svcOpts.Id = "test-service"
	svcOpts.Service.Interface = "com.example.TestService"

	srv.registerServiceOptions(svcOpts)

	// Verify via public API methods
	retrieved := srv.GetServiceOptions("test-service")
	assert.NotNil(t, retrieved)
	assert.Equal(t, svcOpts, retrieved)

	retrievedByInterface := srv.GetServiceOptionsByInterfaceName("com.example.TestService")
	assert.NotNil(t, retrievedByInterface)
	assert.Equal(t, svcOpts, retrievedByInterface)
}

// Test registerServiceOptions with empty interface
func TestRegisterServiceOptionsEmptyInterface(t *testing.T) {
	srv, err := NewServer()
	require.NoError(t, err)

	svcOpts := defaultServiceOptions()
	svcOpts.Id = "test-service"
	svcOpts.Service.Interface = ""

	srv.registerServiceOptions(svcOpts)

	// Should be retrievable by service ID but not by interface name
	retrieved := srv.GetServiceOptions("test-service")
	assert.NotNil(t, retrieved)
	assert.Equal(t, svcOpts, retrieved)

	// Empty interface should not be retrievable by interface name
	retrievedByInterface := srv.GetServiceOptionsByInterfaceName("")
	assert.Nil(t, retrievedByInterface)
}

// Test SetProviderServices
func TestSetProviderServices(t *testing.T) {
	// Lock and backup original state
	internalProLock.Lock()
	originalServices := internalProServices
	internalProServices = make([]*InternalService, 0, 16)
	internalProLock.Unlock()

	// Register cleanup to restore original state
	t.Cleanup(func() {
		internalProLock.Lock()
		defer internalProLock.Unlock()
		internalProServices = originalServices
	})

	internalService := &InternalService{
		Name:     "test-internal-service",
		Priority: 0,
		Init: func(options *ServiceOptions) (*ServiceDefinition, bool) {
			return &ServiceDefinition{
				Handler: &mockServerRPCService{},
				Info:    nil,
				Opts:    []ServiceOption{},
			}, true
		},
	}

	// This function will acquire the lock internally
	SetProviderServices(internalService)

	// Access the global variable safely with lock
	internalProLock.Lock()
	defer internalProLock.Unlock()
	assert.Len(t, internalProServices, 1)
	assert.Equal(t, "test-internal-service", internalProServices[0].Name)
}

// Test SetProviderServices with empty name
func TestSetProviderServicesEmptyName(t *testing.T) {
	// Lock and backup original state
	internalProLock.Lock()
	originalServices := internalProServices
	internalProServices = make([]*InternalService, 0, 16)

	// Unlock before calling SetProviderServices (which needs the lock)
	internalProLock.Unlock()

	// Register cleanup to restore original state
	t.Cleanup(func() {
		internalProLock.Lock()
		defer internalProLock.Unlock()
		internalProServices = originalServices
	})

	internalService := &InternalService{
		Name: "",
	}

	// This function will acquire the lock internally
	SetProviderServices(internalService)

	// Access the global variable safely with lock
	internalProLock.Lock()
	defer internalProLock.Unlock()
	assert.Empty(t, internalProServices)
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
	// ServiceInfo.Methods must only contain original method names.
	// Swapped-case aliases are registered at the transport layer, not here.
	assert.Len(t, result.Methods, 1)
	assert.Equal(t, "sayHello", result.Methods[0].Name)
}

// greetServiceForTest is a minimal service used to verify MethodFunc backfill.
type greetServiceForTest struct{}

func (g *greetServiceForTest) Greet(ctx context.Context, req string) (string, error) {
	return req, nil
}

func (g *greetServiceForTest) Reference() string { return "greetServiceForTest" }

type variadicReflectionServiceForTest struct{}

func (s *variadicReflectionServiceForTest) HelloVariadic(ctx context.Context, prefix string, names ...string) (string, error) {
	return prefix + ":" + strconv.Itoa(len(names)), nil
}

// TestEnhanceServiceInfoMethodFuncBackfillExactName verifies that
// enhanceServiceInfo fills in MethodFunc when the ServiceInfo method name
// matches the Go exported method name exactly (PascalCase).
func TestEnhanceServiceInfoMethodFuncBackfillExactName(t *testing.T) {
	svc := &greetServiceForTest{}
	info := &common.ServiceInfo{
		ServiceType: svc,
		Methods: []common.MethodInfo{
			{Name: "Greet"}, // exact Go name — must be found
		},
	}

	result := enhanceServiceInfo(info)
	assert.NotNil(t, result)
	assert.Len(t, result.Methods, 1)
	assert.NotNil(t, result.Methods[0].MethodFunc,
		"MethodFunc must be filled in for exact-name match to avoid nil-func panic")
}

// TestEnhanceServiceInfoMethodFuncBackfillJavaStyleName verifies that
// enhanceServiceInfo still fills in MethodFunc for lowercase-first method names
// so reflection-based invocation can reach the exported Go method.
func TestEnhanceServiceInfoMethodFuncBackfillJavaStyleName(t *testing.T) {
	svc := &greetServiceForTest{}
	info := &common.ServiceInfo{
		ServiceType: svc,
		Methods: []common.MethodInfo{
			{Name: "greet"}, // Java/Dubbo-style lowercase-first name
		},
	}

	result := enhanceServiceInfo(info)
	assert.NotNil(t, result)
	assert.Len(t, result.Methods, 1)
	assert.NotNil(t, result.Methods[0].MethodFunc,
		"MethodFunc must be found via swapped-case lookup to avoid nil-func panic on lowercase-first method names")
}

func TestCallMethodByReflectionVariadic(t *testing.T) {
	svc := &variadicReflectionServiceForTest{}
	method, ok := reflect.TypeOf(svc).MethodByName("HelloVariadic")
	require.True(t, ok)

	t.Run("generic packed variadic tail uses call slice", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), constant.DubboCtxKey(constant.GenericVariadicCallSliceKey), true)
		res, err := CallMethodByReflection(ctx, method, svc, []any{"hello", []string{"alice", "bob"}})
		require.NoError(t, err)
		assert.Equal(t, "hello:2", res)
	})

	t.Run("ordinary discrete variadic call remains unchanged", func(t *testing.T) {
		res, err := CallMethodByReflection(context.Background(), method, svc, []any{"hello", "alice", "bob"})
		require.NoError(t, err)
		assert.Equal(t, "hello:2", res)
	})
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

// mockServerRPCService is a mock implementation for testing server_test.go only
type mockServerRPCService struct{}

func (m *mockServerRPCService) Invoke(methodName string, params []any, results []any) error {
	return nil
}

func (m *mockServerRPCService) Reference() string {
	return "com.example.MockService"
}

// MockServerRPCService is an exported version used by protocol paths that reflect on handler types.
type MockServerRPCService struct{}

func (m *MockServerRPCService) Invoke(methodName string, params []any, results []any) error {
	return nil
}

func (m *MockServerRPCService) Reference() string {
	return "com.example.MockService"
}

// variadicServerRPCService exposes a variadic RPC method for warning tests.
type variadicServerRPCService struct{}

func (m *variadicServerRPCService) Broadcast(ctx context.Context, names ...string) error {
	return nil
}

func (m *variadicServerRPCService) Reference() string {
	return "com.example.VariadicService"
}

// NoReferenceVariadicServerRPCService relies on the default reference fallback
// for warning-path tests.
type NoReferenceVariadicServerRPCService struct{}

func (m *NoReferenceVariadicServerRPCService) Broadcast(ctx context.Context, names ...string) error {
	return nil
}

// captureWarnLogger records warning logs for assertions.
type captureWarnLogger struct {
	gostlogger.Logger
	warns []string
}

func (l *captureWarnLogger) Warnf(template string, args ...any) {
	l.warns = append(l.warns, fmt.Sprintf(template, args...))
}

// Test concurrency: multiple goroutines registering services
func TestConcurrentServiceRegistration(t *testing.T) {
	srv, err := NewServer()
	require.NoError(t, err)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			svcOpts := defaultServiceOptions()
			svcOpts.Id = "service-" + strconv.Itoa(idx)
			srv.registerServiceOptions(svcOpts)
		}(i)
	}

	wg.Wait()

	// Verify all services were registered using public API
	for i := 0; i < 10; i++ {
		svcID := "service-" + strconv.Itoa(i)
		retrieved := srv.GetServiceOptions(svcID)
		assert.NotNil(t, retrieved, "Service %s should be registered", svcID)
		assert.Equal(t, svcID, retrieved.Id)
	}
}

// Test Register with ServiceInfo
func TestRegisterWithServiceInfo(t *testing.T) {
	srv, err := NewServer()
	require.NoError(t, err)

	handler := &mockServerRPCService{}
	info := &common.ServiceInfo{
		InterfaceName: "com.example.Service",
		Methods: []common.MethodInfo{
			{Name: "Method1"},
		},
	}

	err = srv.Register(handler, info)
	require.NoError(t, err)

	// Service should be registered
	svcOpts := srv.GetServiceOptions(handler.Reference())
	assert.NotNil(t, svcOpts)
}

// Test RegisterService without ServiceInfo
func TestRegisterService(t *testing.T) {
	srv, err := NewServer()
	require.NoError(t, err)

	handler := &mockServerRPCService{}
	err = srv.RegisterService(handler)
	require.NoError(t, err)

	// Service should be registered with handler reference name
	svcOpts := srv.GetServiceOptions(handler.Reference())
	assert.NotNil(t, svcOpts)
}

// Test RegisterService warns on variadic RPC methods
func TestRegisterServiceWarnsOnVariadicRPCMethods(t *testing.T) {
	srv, err := NewServer()
	require.NoError(t, err)

	prev := gostlogger.GetLogger()
	capture := &captureWarnLogger{Logger: prev}
	gostlogger.SetLogger(capture)
	t.Cleanup(func() {
		gostlogger.SetLogger(prev)
	})

	handler := &variadicServerRPCService{}
	err = srv.RegisterService(handler)
	require.NoError(t, err)

	svcOpts := srv.GetServiceOptions(handler.Reference())
	assert.NotNil(t, svcOpts)
	require.Len(t, capture.warns, 1)
	assert.Contains(t, capture.warns[0], handler.Reference())
	assert.Contains(t, capture.warns[0], "Broadcast")
	assert.Contains(t, capture.warns[0], "[]T")
}

// Test RegisterService warns on variadic RPC methods even when the handler
// uses the default reference name.
func TestRegisterServiceWarnsOnVariadicRPCMethodsWithoutReference(t *testing.T) {
	srv, err := NewServer()
	require.NoError(t, err)

	prev := gostlogger.GetLogger()
	capture := &captureWarnLogger{Logger: prev}
	gostlogger.SetLogger(capture)
	t.Cleanup(func() {
		gostlogger.SetLogger(prev)
	})

	handler := &NoReferenceVariadicServerRPCService{}
	err = srv.RegisterService(handler)
	require.NoError(t, err)

	interfaceName := common.GetReference(handler)
	svcOpts := srv.GetServiceOptions(interfaceName)
	assert.NotNil(t, svcOpts)
	require.Len(t, capture.warns, 1)
	assert.Contains(t, capture.warns[0], interfaceName)
	assert.Contains(t, capture.warns[0], "Broadcast")
}

// Test RegisterService does not warn on non-variadic RPC methods
func TestRegisterServiceDoesNotWarnOnNonVariadicRPCMethods(t *testing.T) {
	srv, err := NewServer()
	require.NoError(t, err)

	prev := gostlogger.GetLogger()
	capture := &captureWarnLogger{Logger: prev}
	gostlogger.SetLogger(capture)
	t.Cleanup(func() {
		gostlogger.SetLogger(prev)
	})

	handler := &mockServerRPCService{}
	err = srv.RegisterService(handler)
	require.NoError(t, err)
	assert.Empty(t, capture.warns)
}

// Test Register with handler that has method config
func TestRegisterWithMethodConfig(t *testing.T) {
	srv, err := NewServer()
	require.NoError(t, err)

	handler := &mockServerRPCService{}
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
	require.NoError(t, err)

	svcOpts := srv.GetServiceOptions(handler.Reference())
	assert.NotNil(t, svcOpts)
	assert.Equal(t, "test-group", svcOpts.Service.Group)
	assert.Equal(t, "1.0.0", svcOpts.Service.Version)
}

// Test genSvcOpts with missing server config
func TestGenSvcOptsWithMissingServerConfig(t *testing.T) {
	srv := &Server{
		cfg: nil,
	}

	handler := &mockServerRPCService{}
	_, err := srv.genSvcOpts(handler, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "has not been initialized")
}

// Test exportServices with empty service map
func TestExportServicesEmpty(t *testing.T) {
	srv, err := NewServer()
	require.NoError(t, err)

	ctx := context.Background()
	err = srv.exportServices(ctx)
	assert.NoError(t, err)
}

// Test NewServer with custom group option
func TestNewServerWithCustomGroup(t *testing.T) {
	srv, err := NewServer(WithServerGroup("test"))
	require.NoError(t, err)
	assert.NotNil(t, srv)

	// Verify the group option by registering a service and checking its configuration
	handler := &mockServerRPCService{}
	err = srv.Register(handler, nil)
	require.NoError(t, err)

	svcOpts := srv.GetServiceOptions(handler.Reference())
	assert.NotNil(t, svcOpts)
	assert.Equal(t, "test", svcOpts.Service.Group)
}

// Test isNillable checks if a reflect.Value's kind supports nil checking
func TestIsNillable(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected bool
	}{
		// nillable types
		{"chan", make(chan int), true},
		{"func", func() {}, true},
		{"interface", (*error)(nil), true},
		{"map", map[string]int{}, true},
		{"pointer", new(int), true},
		{"slice", []int{}, true},
		{"unsafe.Pointer", unsafe.Pointer(nil), true},
		{"nil chan", (chan int)(nil), true},
		{"nil map", (map[string]int)(nil), true},
		{"nil slice", ([]int)(nil), true},

		// non-nillable types
		{"int", 42, false},
		{"string", "hello", false},
		{"bool", true, false},
		{"float64", 3.14, false},
		{"struct", struct{}{}, false},
		{"array", [3]int{1, 2, 3}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := reflect.ValueOf(tt.value)
			result := isNillable(v)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test isReflectNil safely checks if a reflect.Value is nil
func TestIsReflectNil(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected bool
	}{
		// nil nillable types
		{"nil chan", (chan int)(nil), true},
		{"nil map", (map[string]int)(nil), true},
		{"nil slice", ([]int)(nil), true},
		{"nil func", (func())(nil), true},
		{"nil pointer", (*int)(nil), true},

		// non-nil nillable types
		{"non-nil chan", make(chan int), false},
		{"non-nil map", map[string]int{"a": 1}, false},
		{"non-nil slice", []int{1, 2, 3}, false},
		{"non-nil func", func() {}, false},
		{"non-nil pointer", new(int), false},

		// non-nillable types (should return false, not panic)
		{"int", 42, false},
		{"string", "hello", false},
		{"bool", true, false},
		{"float64", 3.14, false},
		{"struct", struct{ Name string }{"test"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := reflect.ValueOf(tt.value)
			// should not panic
			result := isReflectNil(v)
			assert.Equal(t, tt.expected, result)
		})
	}
}
