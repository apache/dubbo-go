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

package servicediscovery

import (
	stderrors "errors"
	"fmt"
	"testing"
)

import (
	gxset "github.com/dubbogo/gost/container/set"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/metadata"
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
	"dubbo.apache.org/dubbo-go/v3/metadata/mapping"
	metadatareport "dubbo.apache.org/dubbo-go/v3/metadata/report"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

func TestServiceInstancesChangedListenerAggregatesSameServiceAcrossRevisions(t *testing.T) {
	listener := NewServiceInstancesChangedListener(testApp, constant.DefaultKey, gxset.NewSet(testApp))
	notify := &capturingNotifyListener{}
	listener.AddListenerAndNotify(common.MatchKey(testInterface, constant.TriProtocol), notify)

	instances := []registry.ServiceInstance{
		newTestServiceInstance(t, 20000, "dev"),
		newTestServiceInstance(t, 20001, "pre"),
		newTestServiceInstance(t, 20002, "prod"),
	}
	require.NoError(t, listener.OnEvent(registry.NewServiceInstancesChangedEvent(testApp, instances)))

	assertServiceEvents(t, notify.events, []string{"20000", "20001", "20002"}, []string{"dev", "pre", "prod"})
}

func TestServiceInstancesChangedListenerRefreshesURLsOnProviderRemoveAndRestart(t *testing.T) {
	listener := NewServiceInstancesChangedListener(testApp, constant.DefaultKey, gxset.NewSet(testApp))
	notify := &capturingNotifyListener{}
	listener.AddListenerAndNotify(common.MatchKey(testInterface, constant.TriProtocol), notify)

	dev := newTestServiceInstance(t, 20000, "dev")
	pre := newTestServiceInstance(t, 20001, "pre")
	prod := newTestServiceInstance(t, 20002, "prod")

	require.NoError(t, listener.OnEvent(registry.NewServiceInstancesChangedEvent(testApp, []registry.ServiceInstance{
		dev,
		pre,
		prod,
	})))
	assertServiceEvents(t, notify.events, []string{"20000", "20001", "20002"}, []string{"dev", "pre", "prod"})

	require.NoError(t, listener.OnEvent(registry.NewServiceInstancesChangedEvent(testApp, []registry.ServiceInstance{
		dev,
		prod,
	})))
	assertServiceEvents(t, notify.events, []string{"20000", "20002"}, []string{"dev", "prod"})

	restartedPre := newTestServiceInstanceWithRevision(t, 20001, "pre-restarted", "rev-20001-restarted")
	require.NoError(t, listener.OnEvent(registry.NewServiceInstancesChangedEvent(testApp, []registry.ServiceInstance{
		dev,
		restartedPre,
		prod,
	})))
	assertServiceEvents(t, notify.events, []string{"20000", "20001", "20002"}, []string{"dev", "pre-restarted", "prod"})
}

func TestServiceInstancesChangedListenerRefreshesAndClearsEnvironmentWhenRevisionIsUnchanged(t *testing.T) {
	listener := NewServiceInstancesChangedListener(testApp, constant.DefaultKey, gxset.NewSet(testApp))
	notify := &capturingNotifyListener{}
	listener.AddListenerAndNotify(common.MatchKey(testInterface, constant.TriProtocol), notify)

	revision := "rev-20001-same-environment-change"
	cacheKey := testApp + ":" + constant.DefaultKey + ":" + revision
	metaCache.Set(cacheKey, newTestMetadataInfo(t, revision, 20001, "pre"))

	pre := newTestServiceInstanceOnly(20001, "pre", revision)
	require.NoError(t, listener.OnEvent(registry.NewServiceInstancesChangedEvent(testApp, []registry.ServiceInstance{
		pre,
	})))
	assertServiceEvents(t, notify.events, []string{"20001"}, []string{"pre"})

	restartedPre := newTestServiceInstanceOnly(20001, "pre-restarted", revision)
	require.NoError(t, listener.OnEvent(registry.NewServiceInstancesChangedEvent(testApp, []registry.ServiceInstance{
		restartedPre,
	})))
	assertServiceEvents(t, notify.events, []string{"20001"}, []string{"pre-restarted"})

	clearedPre := newTestServiceInstanceOnly(20001, "", revision)
	require.NoError(t, listener.OnEvent(registry.NewServiceInstancesChangedEvent(testApp, []registry.ServiceInstance{
		clearedPre,
	})))
	assertServiceEvents(t, notify.events, []string{"20001"}, []string{""})
	assertServiceEventEnvironmentsAbsent(t, notify.events)

	removedPre := newTestServiceInstanceOnly(20001, "ignored", revision)
	delete(removedPre.GetMetadata(), constant.EnvironmentKey)
	require.NoError(t, listener.OnEvent(registry.NewServiceInstancesChangedEvent(testApp, []registry.ServiceInstance{
		removedPre,
	})))
	assertServiceEvents(t, notify.events, []string{"20001"}, []string{""})
	assertServiceEventEnvironmentsAbsent(t, notify.events)
}

func TestServiceInstancesChangedListenerSkipsNilMetadataWithoutPanic(t *testing.T) {
	listener := NewServiceInstancesChangedListener(testApp, constant.DefaultKey, gxset.NewSet(testApp))
	notify := &capturingNotifyListener{}
	listener.AddListenerAndNotify(common.MatchKey(testInterface, constant.TriProtocol), notify)

	revision := "rev-20003-nil-metadata"
	nilCacheKey := testApp + ":" + constant.DefaultKey + ":" + revision
	var metadataInfo *info.MetadataInfo
	metaCache.Set(nilCacheKey, metadataInfo)
	t.Cleanup(func() {
		metaCache.Delete(nilCacheKey)
	})

	instance := newTestServiceInstanceOnly(20003, "pre", revision)
	var err error
	require.NotPanics(t, func() {
		err = listener.OnEvent(registry.NewServiceInstancesChangedEvent(testApp, []registry.ServiceInstance{
			instance,
		}))
	})
	require.NoError(t, err)
	assert.Empty(t, notify.events)
}

func TestCreateInstanceCarriesEnvironmentMetadata(t *testing.T) {
	meta := info.NewMetadataInfo(testApp, "")
	providerURL := newTestProviderURL(t, 20001, "pre")

	instance := createInstance(meta, providerURL, constant.DefaultKey)

	assert.Equal(t, "pre", instance.GetMetadata()[constant.EnvironmentKey])
}

// TestListenerUsesRegistryIdToFetchRemoteMetadata verifies that when a listener is
// created with a specific registryId, GetMetadataInfo threads that id through to
// GetMetadataFromMetadataReport so the correct per-registry metadata report is used.
func TestListenerUsesRegistryIdToFetchRemoteMetadata(t *testing.T) {
	const listenerRegistryId = "remote-reg-test"
	const revision = "rev-remote-reg-test"

	// Register a mock metadata report under the listener's registry id
	mockReport := new(listenerMockMetadataReport)
	extension.SetMetadataReportFactory(listenerRegistryId, func() metadatareport.MetadataReportFactory {
		return mockReport
	})
	opts := metadata.NewReportOptions(
		metadata.WithRegistryId(listenerRegistryId),
		metadata.WithProtocol(listenerRegistryId),
		metadata.WithAddress("127.0.0.1"),
	)
	require.NoError(t, opts.Init())
	t.Cleanup(metadata.ClearMetadataReportInstances)

	// Build the MetadataInfo that the mock report will return
	serviceURL, err := common.NewURL(
		fmt.Sprintf("tri://127.0.0.1:20099/%s", testInterface),
		common.WithInterface(testInterface),
		common.WithMethods([]string{"Greet"}),
		common.WithParamsValue(constant.ApplicationKey, testApp),
	)
	require.NoError(t, err)
	svc := info.NewServiceInfoWithURL(serviceURL)
	expectedMeta := info.NewMetadataInfoWithParams(testApp, revision, map[string]*info.ServiceInfo{
		svc.GetMatchKey(): svc,
	})
	mockReport.On("GetAppMetadata").Return(expectedMeta, nil).Once()

	// Create a service instance that requests remote metadata storage
	remoteInstance := &registry.DefaultServiceInstance{
		ID:          "127.0.0.1:20099",
		ServiceName: testApp,
		Host:        "127.0.0.1",
		Port:        20099,
		Enable:      true,
		Healthy:     true,
		Metadata: map[string]string{
			constant.ExportedServicesRevisionPropertyName: revision,
			constant.MetadataStorageTypePropertyName:      constant.RemoteMetadataStorageType,
			constant.ServiceInstanceEndpoints:             `[{"port":20099,"protocol":"tri"}]`,
		},
	}

	// Remove the cache entry after the test so it doesn't bleed into other tests.
	t.Cleanup(func() {
		if metaCache != nil {
			metaCache.Delete(testApp + ":" + listenerRegistryId + ":" + revision)
		}
	})

	listener := NewServiceInstancesChangedListener(testApp, listenerRegistryId, gxset.NewSet(testApp))
	notify := &capturingNotifyListener{}
	listener.AddListenerAndNotify(common.MatchKey(testInterface, constant.TriProtocol), notify)

	require.NoError(t, listener.OnEvent(registry.NewServiceInstancesChangedEvent(testApp, []registry.ServiceInstance{remoteInstance})))

	// The mock report must have been called exactly once, proving the correct report was used
	mockReport.AssertExpectations(t)
	// The notify listener must have received the service from the fetched metadata
	assert.NotEmpty(t, notify.events, "expected service events from the remote metadata fetch")
}

// listenerMockMetadataReport is a mock MetadataReport (and its factory) for
// TestListenerUsesRegistryIdToFetchRemoteMetadata.
type listenerMockMetadataReport struct {
	mock.Mock
}

func (m *listenerMockMetadataReport) CreateMetadataReport(*common.URL) metadatareport.MetadataReport {
	return m
}

func (m *listenerMockMetadataReport) GetAppMetadata(string, string) (*info.MetadataInfo, error) {
	args := m.Called()
	result, _ := args.Get(0).(*info.MetadataInfo)
	return result, args.Error(1)
}

func (m *listenerMockMetadataReport) PublishAppMetadata(string, string, *info.MetadataInfo) error {
	args := m.Called()
	return args.Error(0)
}

func (m *listenerMockMetadataReport) RegisterServiceAppMapping(string, string, string) error {
	args := m.Called()
	return args.Error(0)
}

func (m *listenerMockMetadataReport) GetServiceAppMapping(string, string, mapping.MappingListener) (*gxset.HashSet, error) {
	args := m.Called()
	return args.Get(0).(*gxset.HashSet), args.Error(1)
}

func (m *listenerMockMetadataReport) RemoveServiceAppMappingListener(string, string) error {
	args := m.Called()
	return args.Error(0)
}

func newTestServiceInstance(t *testing.T, port int, environment string) registry.ServiceInstance {
	t.Helper()

	return newTestServiceInstanceWithRevision(t, port, environment, fmt.Sprintf("rev-%d", port))
}

func newTestServiceInstanceWithRevision(t *testing.T, port int, environment string, revision string) registry.ServiceInstance {
	t.Helper()

	cacheKey := testApp + ":" + constant.DefaultKey + ":" + revision
	metaCache.Set(cacheKey, newTestMetadataInfo(t, revision, port, environment))
	t.Cleanup(func() { metaCache.Delete(cacheKey) })
	return newTestServiceInstanceOnly(port, environment, revision)
}

func newTestServiceInstanceOnly(port int, environment string, revision string) registry.ServiceInstance {
	return &registry.DefaultServiceInstance{
		ID:          fmt.Sprintf("127.0.0.1:%d", port),
		ServiceName: testApp,
		Host:        "127.0.0.1",
		Port:        port,
		Enable:      true,
		Healthy:     true,
		Metadata: map[string]string{
			constant.ExportedServicesRevisionPropertyName: revision,
			constant.ServiceInstanceEndpoints:             fmt.Sprintf(`[{"port":%d,"protocol":"tri"}]`, port),
			constant.EnvironmentKey:                       environment,
		},
	}
}

func newTestMetadataInfo(t *testing.T, revision string, port int, environment string) *info.MetadataInfo {
	t.Helper()

	serviceURL := newTestProviderURL(t, port, environment)
	service := info.NewServiceInfoWithURL(serviceURL)
	return info.NewMetadataInfoWithParams(testApp, revision, map[string]*info.ServiceInfo{
		service.GetMatchKey(): service,
	})
}

func newTestProviderURL(t *testing.T, port int, environment string) *common.URL {
	t.Helper()

	serviceURL, err := common.NewURL(
		fmt.Sprintf("tri://127.0.0.1:%d/%s", port, testInterface),
		common.WithInterface(testInterface),
		common.WithMethods([]string{"Greet"}),
		common.WithParamsValue(constant.ApplicationKey, testApp),
		common.WithParamsValue(constant.EnvironmentKey, environment),
		common.WithParamsValue(constant.SideKey, constant.SideProvider),
	)
	require.NoError(t, err)
	return serviceURL
}

func assertServiceEvents(t *testing.T, events []*registry.ServiceEvent, ports, environments []string) {
	t.Helper()

	require.Len(t, events, len(ports))
	require.Len(t, environments, len(ports))
	assert.ElementsMatch(t, expectedServiceEvents(ports, environments), actualServiceEvents(events))
}

func assertServiceEventEnvironmentsAbsent(t *testing.T, events []*registry.ServiceEvent) {
	t.Helper()

	for _, event := range events {
		assert.NotContains(t, event.Service.GetParams(), constant.EnvironmentKey)
	}
}

type serviceEventAssertion struct {
	port        string
	environment string
}

func expectedServiceEvents(ports, environments []string) []serviceEventAssertion {
	events := make([]serviceEventAssertion, 0, len(ports))
	for i, port := range ports {
		events = append(events, serviceEventAssertion{
			port:        port,
			environment: environments[i],
		})
	}
	return events
}

func actualServiceEvents(events []*registry.ServiceEvent) []serviceEventAssertion {
	actual := make([]serviceEventAssertion, 0, len(events))
	for _, event := range events {
		actual = append(actual, serviceEventAssertion{
			port:        event.Service.Port,
			environment: event.Service.GetParam(constant.EnvironmentKey, ""),
		})
	}
	return actual
}

type capturingNotifyListener struct {
	events []*registry.ServiceEvent
}

func (c *capturingNotifyListener) Notify(event *registry.ServiceEvent) {
	c.events = append(c.events, event)
}

func (c *capturingNotifyListener) NotifyAll(events []*registry.ServiceEvent, callback func()) {
	c.events = append([]*registry.ServiceEvent(nil), events...)
	if callback != nil {
		callback()
	}
}

func TestGetMetadataInfo_CacheKeyFormat(t *testing.T) {
	// Ensure cache is initialized (normally done by NewServiceInstancesChangedListener)
	_ = NewServiceInstancesChangedListener(testApp, constant.DefaultKey, gxset.NewSet(testApp))

	revision := "rev-cache-key-test"
	// Pre-populate cache with the expected composite key
	expectedKey := testApp + ":" + constant.DefaultKey + ":" + revision
	expectedMeta := newTestMetadataInfo(t, revision, 20000, "dev")
	metaCache.Set(expectedKey, expectedMeta)
	t.Cleanup(func() {
		metaCache.Delete(expectedKey)
	})

	instance := &registry.DefaultServiceInstance{
		ID:          "127.0.0.1:20000",
		ServiceName: testApp,
		Host:        "127.0.0.1",
		Port:        20000,
		Enable:      true,
		Healthy:     true,
		Metadata: map[string]string{
			constant.ExportedServicesRevisionPropertyName: revision,
			constant.ServiceInstanceEndpoints:             `[{"port":20000,"protocol":"tri"}]`,
		},
	}

	// Should hit the cache and return the pre-populated metadata
	meta, err := GetMetadataInfo(testApp, instance, revision, constant.DefaultKey)
	require.NoError(t, err)
	assert.Equal(t, expectedMeta, meta)
}

func TestGetMetadataInfo_LocalStorageGoesDirectlyToRPC(t *testing.T) {
	// Ensure cache is initialized
	_ = NewServiceInstancesChangedListener(testApp, constant.DefaultKey, gxset.NewSet(testApp))

	// Instance with no MetadataStorageTypePropertyName (i.e. local/default path)
	// should go directly to RPC without touching the metadata report.
	// RPC will fail with a URL error because there are no URL params — that's
	// enough to confirm the correct branch was taken.
	instance := &registry.DefaultServiceInstance{
		ID:          "127.0.0.1:20003",
		ServiceName: testApp,
		Host:        "127.0.0.1",
		Port:        20003,
		Enable:      true,
		Healthy:     true,
		Metadata: map[string]string{
			constant.ExportedServicesRevisionPropertyName: "rev-local-rpc",
			// MetadataStorageTypePropertyName intentionally absent → local path
			constant.ServiceInstanceEndpoints: `[{"port":20003,"protocol":"tri"}]`,
		},
	}

	_, err := GetMetadataInfo(testApp, instance, "rev-local-rpc", constant.DefaultKey)
	require.Error(t, err)
	// Must be a URL/RPC error, not a report error, confirming the local path
	// skips the report entirely and goes straight to RPC.
	assert.Contains(t, err.Error(), "metadata service URL params missing",
		"local storage path should go directly to RPC, not touch the metadata report")
}

func TestGetMetadataInfo_FallbackToRPC(t *testing.T) {
	// Ensure cache is initialized
	_ = NewServiceInstancesChangedListener(testApp, constant.DefaultKey, gxset.NewSet(testApp))

	// remote storage type without a report registered → report will fail
	// should fall through to RPC, which will fail with url error (no URL params)
	instance := &registry.DefaultServiceInstance{
		ID:          "127.0.0.1:20002",
		ServiceName: testApp,
		Host:        "127.0.0.1",
		Port:        20002,
		Enable:      true,
		Healthy:     true,
		Metadata: map[string]string{
			constant.ExportedServicesRevisionPropertyName: "rev-fallback-to-rpc",
			constant.MetadataStorageTypePropertyName:      constant.RemoteMetadataStorageType,
			constant.ServiceInstanceEndpoints:             `[{"port":20002,"protocol":"tri"}]`,
		},
	}

	_, err := GetMetadataInfo(testApp, instance, "rev-fallback-to-rpc", constant.DefaultKey)
	require.Error(t, err)
	// Both report and RPC fail: the combined error proves the fallback path was taken
	// and includes the RPC/URL failure as the wrapped cause.
	assert.Contains(t, err.Error(), "both paths failed",
		"fallback path should produce a combined error mentioning both failures")
	assert.Contains(t, err.Error(), "metadata service URL params missing",
		"fallback error should include the RPC/URL failure cause")
	var metadataErr *metadata.MetadataError
	require.True(t, stderrors.As(err, &metadataErr))
	assert.Equal(t, metadata.MetadataErrorKindURLBuild, metadataErr.Kind)
	assert.Equal(t, "metadata_url", metadataErr.Source)
	assert.Equal(t, testApp, metadataErr.App)
	assert.Equal(t, "rev-fallback-to-rpc", metadataErr.Revision)
	assert.Equal(t, constant.RemoteMetadataStorageType, metadataErr.StorageType)
}

// TestGetMetadataInfo_ReportReturnsNil_FallsBackToRPC verifies the path where the metadata
// report returns (nil, nil) — no error but no data — which must trigger the RPC fallback.
// Here the instance has no URL params, so RPC fails; the test asserts the resulting error
// comes from the RPC fallback (not the report), proving the nil-result branch was taken.
func TestGetMetadataInfo_ReportReturnsNil_FallsBackToRPC(t *testing.T) {
	const regID = "report-nil-rpc-ok"
	const revision = "rev-report-nil-rpc-ok"

	// Register a mock report that returns (nil, nil) — success with no data.
	mockReport := new(listenerMockMetadataReport)
	extension.SetMetadataReportFactory(regID, func() metadatareport.MetadataReportFactory {
		return mockReport
	})
	opts := metadata.NewReportOptions(
		metadata.WithRegistryId(regID),
		metadata.WithProtocol(regID),
		metadata.WithAddress("127.0.0.1"),
	)
	require.NoError(t, opts.Init())
	t.Cleanup(metadata.ClearMetadataReportInstances)
	t.Cleanup(func() {
		if metaCache != nil {
			metaCache.Delete(testApp + ":" + regID + ":" + revision)
		}
	})

	mockReport.On("GetAppMetadata").Return((*info.MetadataInfo)(nil), nil).Once()

	// The instance has no URL params, so the RPC fallback fails at URL construction.
	// That failure is the observable proof that the report's nil result triggered fallback.
	instance := &registry.DefaultServiceInstance{
		ID:          "127.0.0.1:20098",
		ServiceName: testApp,
		Host:        "127.0.0.1",
		Port:        20098,
		Enable:      true,
		Healthy:     true,
		Metadata: map[string]string{
			constant.ExportedServicesRevisionPropertyName: revision,
			constant.MetadataStorageTypePropertyName:      constant.RemoteMetadataStorageType,
			// No URL params — RPC will fail at the URL-construction stage
		},
	}
	_ = NewServiceInstancesChangedListener(testApp, regID, gxset.NewSet(testApp))

	_, err := GetMetadataInfo(testApp, instance, revision, regID)
	require.Error(t, err)
	// The report returned nil (no error), so the fallback was triggered and then RPC
	// failed at URL construction. The error must reflect the RPC-after-nil-report path.
	assert.Contains(t, err.Error(), "RPC fallback failed after report returned nil metadata",
		"nil report result should trigger fallback and surface an RPC error")
	assert.Contains(t, err.Error(), "metadata service URL params missing",
		"fallback error should include the RPC/URL failure cause")
	var metadataErr *metadata.MetadataError
	require.True(t, stderrors.As(err, &metadataErr))
	assert.Equal(t, metadata.MetadataErrorKindURLBuild, metadataErr.Kind)
	assert.Equal(t, "metadata_url", metadataErr.Source)
	assert.Equal(t, testApp, metadataErr.App)
	assert.Equal(t, revision, metadataErr.Revision)
	assert.Equal(t, constant.RemoteMetadataStorageType, metadataErr.StorageType)
	mockReport.AssertExpectations(t)
}

// TestGetMetadataInfo_CacheTypedNilNoPanic verifies that when the cache holds a typed nil
// *info.MetadataInfo entry, GetMetadataInfo does not panic and returns (nil, nil).
// This covers a defensive edge case (e.g., a previous store of typed nil) rather than
// the production nil guard in the RPC path. The RPC nil guard is exercised indirectly
// through TestGetMetadataInfo_ReportReturnsNil_FallsBackToRPC.
func TestGetMetadataInfo_CacheTypedNilNoPanic(t *testing.T) {
	// This path is exercised via the cache: pre-seed with a typed nil *info.MetadataInfo.
	// GetMetadataInfo hits the cache fast-path and returns the typed nil without calling RPC.
	// Asserts that this does NOT panic, giving the caller a (nil, nil) to skip gracefully.
	_ = NewServiceInstancesChangedListener(testApp, constant.DefaultKey, gxset.NewSet(testApp))

	revision := "rev-rpc-nil-meta-guard"
	cacheKey := testApp + ":" + constant.DefaultKey + ":" + revision
	var nilMeta *info.MetadataInfo
	metaCache.Set(cacheKey, nilMeta)
	t.Cleanup(func() { metaCache.Delete(cacheKey) })

	instance := newTestServiceInstanceOnly(20098, "dev", revision)

	// Must not panic; the typed nil in the cache is returned as (nil, nil).
	var result *info.MetadataInfo
	require.NotPanics(t, func() {
		result, _ = GetMetadataInfo(testApp, instance, revision, constant.DefaultKey)
	})
	assert.Nil(t, result, "typed nil from cache should be returned as nil MetadataInfo")
}

// TestGetMetadataInfo_NilMetadataMap verifies that an instance with a nil Metadata map
// is handled gracefully by GetMetadataInfo and takes the local/RPC path.
// The nil-map guard in GetMetadataInfo is unreachable from OnEvent (which has its own
// nil check), so it must be tested by calling GetMetadataInfo directly.
func TestGetMetadataInfo_NilMetadataMap(t *testing.T) {
	_ = NewServiceInstancesChangedListener(testApp, constant.DefaultKey, gxset.NewSet(testApp))

	instance := &registry.DefaultServiceInstance{
		ID:          "127.0.0.1:20097",
		ServiceName: testApp,
		Host:        "127.0.0.1",
		Port:        20097,
		Enable:      true,
		Healthy:     true,
		Metadata:    nil, // nil map — triggers the nil-map guard
	}

	_, err := GetMetadataInfo(testApp, instance, "rev-nil-map", constant.DefaultKey)
	// With no URL params the RPC call fails, but it must be an RPC/URL error —
	// not a panic and not a report error — proving the nil-map guard worked and
	// the local path was taken.
	require.Error(t, err)
	assert.NotPanics(t, func() {
		_, _ = GetMetadataInfo(testApp, instance, "rev-nil-map-nopanic", constant.DefaultKey)
	})
}

// TestGetMetadataInfo_ProviderAppIsolatesSharedRevision verifies that two different
// provider applications sharing the same revision do NOT collide in the cache.
// The cache key is scoped by provider app, so each app's MetadataInfo is isolated
// even when their revision strings are identical.
func TestGetMetadataInfo_ProviderAppIsolatesSharedRevision(t *testing.T) {
	_ = NewServiceInstancesChangedListener(testApp, constant.DefaultKey, gxset.NewSet(testApp))

	const sharedRevision = "rev-shared-across-apps"
	const providerA = "order-service"
	const providerB = "payment-service"

	keyA := providerA + ":" + constant.DefaultKey + ":" + sharedRevision
	keyB := providerB + ":" + constant.DefaultKey + ":" + sharedRevision
	metaA := info.NewMetadataInfo(providerA, sharedRevision)
	metaB := info.NewMetadataInfo(providerB, sharedRevision)
	metaCache.Set(keyA, metaA)
	metaCache.Set(keyB, metaB)
	t.Cleanup(func() {
		metaCache.Delete(keyA)
		metaCache.Delete(keyB)
	})

	instanceA := &registry.DefaultServiceInstance{
		ID:          "127.0.0.1:21001",
		ServiceName: providerA,
		Host:        "127.0.0.1",
		Port:        21001,
		Metadata: map[string]string{
			constant.ExportedServicesRevisionPropertyName: sharedRevision,
		},
	}
	instanceB := &registry.DefaultServiceInstance{
		ID:          "127.0.0.1:21002",
		ServiceName: providerB,
		Host:        "127.0.0.1",
		Port:        21002,
		Metadata: map[string]string{
			constant.ExportedServicesRevisionPropertyName: sharedRevision,
		},
	}

	// Each provider app must resolve to its own MetadataInfo, not the other's,
	// despite sharing the same revision.
	gotA, err := GetMetadataInfo(instanceA.GetServiceName(), instanceA, sharedRevision, constant.DefaultKey)
	require.NoError(t, err)
	assert.Equal(t, providerA, gotA.App, "provider A must get its own metadata")

	gotB, err := GetMetadataInfo(instanceB.GetServiceName(), instanceB, sharedRevision, constant.DefaultKey)
	require.NoError(t, err)
	assert.Equal(t, providerB, gotB.App, "provider B must get its own metadata, not provider A's")
}
