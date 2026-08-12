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
	"errors"
	"sync"
	"testing"
)

import (
	gxset "github.com/dubbogo/gost/container/set"
	"github.com/dubbogo/gost/gof/observer"

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
	"dubbo.apache.org/dubbo-go/v3/metadata/report"
	"dubbo.apache.org/dubbo-go/v3/metrics"
	metricsMetadata "dubbo.apache.org/dubbo-go/v3/metrics/metadata"
)

func TestGetNameMappingInstance(t *testing.T) {
	ins := GetNameMappingInstance()
	assert.NotNil(t, ins)
}

func TestNoReportInstance(t *testing.T) {
	ins := GetNameMappingInstance()
	lis := &listener{}
	serviceUrl := common.NewURLWithOptions(
		common.WithInterface("org.apache.dubbo.samples.proto.GreetService"),
		common.WithParamsValue(constant.ApplicationKey, "dubbo"),
	)
	_, err := ins.Get(serviceUrl, lis)
	require.Error(t, err, "test Get no report instance")
	err = ins.Map(serviceUrl)
	require.Error(t, err, "test Map with no report instance")
	err = ins.Remove(serviceUrl)
	require.Error(t, err, "test Remove with no report instance")
}

func TestServiceNameMappingNoReportMetersPerBusinessOperation(t *testing.T) {
	metadata.ClearMetadataReportInstances()
	t.Cleanup(metadata.ClearMetadataReportInstances)
	serviceNameMappingOnce = sync.Once{}
	serviceNameMappingInstance = nil

	ch := make(chan metrics.MetricsEvent, 10)
	metrics.Subscribe(constant.MetricsMetadata, ch)
	defer metrics.Unsubscribe(constant.MetricsMetadata)

	ins := GetNameMappingInstance()
	serviceUrl := common.NewURLWithOptions(
		common.WithInterface("org.example.NoReportService"),
		common.WithParamsValue(constant.ApplicationKey, "no-report-app"),
	)

	err := ins.Map(serviceUrl)
	require.Error(t, err)
	wantAttachment := mappingAttachment("org.example.NoReportService")
	wantAttachment[constant.ApplicationKey] = "no-report-app"
	assertMappingMetricEvent(t, <-ch, metricsMetadata.MetadataMappingRegister, false, false, wantAttachment)
	assert.Empty(t, ch)

	_, err = ins.Get(serviceUrl, nil)
	require.Error(t, err)
	assertMappingMetricEvent(t, <-ch, metricsMetadata.MetadataMappingGet, false, false, mappingAttachment("org.example.NoReportService"))
	assert.Empty(t, ch)

	_, err = ins.Get(serviceUrl, &listener{})
	require.Error(t, err)
	assertMappingMetricEvent(t, <-ch, metricsMetadata.MetadataMappingListen, false, false, mappingAttachment("org.example.NoReportService"))
	assert.Empty(t, ch)

	err = ins.Remove(serviceUrl)
	require.Error(t, err)
	assertMappingMetricEvent(t, <-ch, metricsMetadata.MetadataMappingRemove, false, false, mappingAttachment("org.example.NoReportService"))
	assert.Empty(t, ch)
}

func TestServiceNameMappingGet(t *testing.T) {
	ins := GetNameMappingInstance()
	lis := &listener{}
	serviceUrl := common.NewURLWithOptions(
		common.WithInterface("org.apache.dubbo.samples.proto.GreetService"),
		common.WithParamsValue(constant.ApplicationKey, "dubbo"),
	)
	mockReport, err := initMock()
	require.NoError(t, err)
	t.Run("test normal", func(t *testing.T) {
		mockReport.On("GetServiceAppMapping").Return(gxset.NewSet("dubbo"), nil).Once()
		apps, er := ins.Get(serviceUrl, lis)
		require.NoError(t, er)
		assert.False(t, apps.Empty())
	})
	t.Run("test error", func(t *testing.T) {
		mockReport.On("GetServiceAppMapping").Return(gxset.NewSet(), errors.New("mock error")).Once()
		_, err = ins.Get(serviceUrl, lis)
		require.Error(t, err)
	})
	mockReport.AssertExpectations(t)
}

func TestServiceNameMappingMap(t *testing.T) {
	ins := GetNameMappingInstance()
	serviceUrl := common.NewURLWithOptions(
		common.WithInterface("org.apache.dubbo.samples.proto.GreetService"),
		common.WithParamsValue(constant.ApplicationKey, "dubbo"),
	)
	mockReport, err := initMock()
	require.NoError(t, err)
	t.Run("test normal", func(t *testing.T) {
		mockReport.On("RegisterServiceAppMapping").Return(nil).Once()
		err = ins.Map(serviceUrl)
		require.NoError(t, err)
	})
	t.Run("non-conflict error returns immediately", func(t *testing.T) {
		// a generic error is not retriable, so RegisterServiceAppMapping is called exactly once
		mockReport.On("RegisterServiceAppMapping").Return(errors.New("mock error")).Once()
		err = ins.Map(serviceUrl)
		require.Error(t, err, "test mapping error")
	})
	t.Run("CAS conflict retries up to retryTimes", func(t *testing.T) {
		const conflictRetries = 3
		defer fastRetry(conflictRetries)()
		mockReport.On("RegisterServiceAppMapping").Return(report.ErrMappingCASConflict).Times(conflictRetries)
		err = ins.Map(serviceUrl)
		require.Error(t, err, "conflict exhausts the retry budget")
	})
	mockReport.AssertExpectations(t)
}

func TestServiceNameMappingRemove(t *testing.T) {
	ins := GetNameMappingInstance()
	serviceUrl := common.NewURLWithOptions(
		common.WithInterface("org.apache.dubbo.samples.proto.GreetService"),
		common.WithParamsValue(constant.ApplicationKey, "dubbo"),
	)
	mockReport, err := initMock()
	require.NoError(t, err)
	t.Run("test normal", func(t *testing.T) {
		mockReport.On("RemoveServiceAppMappingListener").Return(nil).Once()
		err = ins.Remove(serviceUrl)
		require.NoError(t, err)
	})
	t.Run("test error", func(t *testing.T) {
		mockReport.On("RemoveServiceAppMappingListener").Return(errors.New("mock error")).Once()
		err = ins.Remove(serviceUrl)
		require.Error(t, err)
	})
	mockReport.AssertExpectations(t)
}

func initMock() (*mockMetadataReport, error) {
	metadataReport := new(mockMetadataReport)
	extension.SetMetadataReportFactory("mock", func() report.MetadataReportFactory {
		return metadataReport
	})
	opts := metadata.NewReportOptions(
		metadata.WithProtocol("mock"),
		metadata.WithAddress("127.0.0.1"),
	)
	err := opts.Init()
	return metadataReport, err
}

func initMockWithId(t *testing.T, registryId string) *mockMetadataReport {
	t.Helper()
	mockReport := new(mockMetadataReport)
	extension.SetMetadataReportFactory(registryId, func() report.MetadataReportFactory {
		return mockReport
	})
	opts := metadata.NewReportOptions(
		metadata.WithRegistryId(registryId),
		metadata.WithProtocol(registryId),
		metadata.WithAddress("127.0.0.1"),
	)
	require.NoError(t, opts.Init())
	return mockReport
}

func TestServiceNameMappingRemoveFansOutToAllReports(t *testing.T) {
	metadata.ClearMetadataReportInstances()
	t.Cleanup(metadata.ClearMetadataReportInstances)
	serviceNameMappingOnce = sync.Once{}
	serviceNameMappingInstance = nil

	r1 := initMockWithId(t, "reg-a")
	r2 := initMockWithId(t, "reg-b")

	ins := GetNameMappingInstance()
	serviceUrl := common.NewURLWithOptions(
		common.WithInterface("org.example.FooService"),
		common.WithParamsValue(constant.ApplicationKey, "foo-app"),
	)

	r1.On("RemoveServiceAppMappingListener").Return(nil).Once()
	r2.On("RemoveServiceAppMappingListener").Return(nil).Once()

	err := ins.Remove(serviceUrl)
	require.NoError(t, err)
	r1.AssertExpectations(t)
	r2.AssertExpectations(t)
}

func TestServiceNameMappingRemoveCollectsAllErrors(t *testing.T) {
	metadata.ClearMetadataReportInstances()
	t.Cleanup(metadata.ClearMetadataReportInstances)
	serviceNameMappingOnce = sync.Once{}
	serviceNameMappingInstance = nil

	r1 := initMockWithId(t, "reg-c")
	r2 := initMockWithId(t, "reg-d")

	ins := GetNameMappingInstance()
	serviceUrl := common.NewURLWithOptions(
		common.WithInterface("org.example.BarService"),
		common.WithParamsValue(constant.ApplicationKey, "bar-app"),
	)

	err1 := errors.New("r1 failure")
	err2 := errors.New("r2 failure")

	// both reports fail
	r1.On("RemoveServiceAppMappingListener").Return(err1).Once()
	r2.On("RemoveServiceAppMappingListener").Return(err2).Once()

	err := ins.Remove(serviceUrl)
	require.Error(t, err)
	// both individual errors must be present in the returned error
	require.ErrorIs(t, err, err1)
	require.ErrorIs(t, err, err2)
	r1.AssertExpectations(t)
	r2.AssertExpectations(t)
}

func TestServiceNameMappingRemoveContinuesAfterPartialFailure(t *testing.T) {
	metadata.ClearMetadataReportInstances()
	t.Cleanup(metadata.ClearMetadataReportInstances)
	serviceNameMappingOnce = sync.Once{}
	serviceNameMappingInstance = nil

	r1 := initMockWithId(t, "reg-e")
	r2 := initMockWithId(t, "reg-f")

	ins := GetNameMappingInstance()
	serviceUrl := common.NewURLWithOptions(
		common.WithInterface("org.example.BazService"),
		common.WithParamsValue(constant.ApplicationKey, "baz-app"),
	)

	removeErr := errors.New("r1 partial failure")

	// r1 fails, r2 succeeds — the loop must not short-circuit
	r1.On("RemoveServiceAppMappingListener").Return(removeErr).Once()
	r2.On("RemoveServiceAppMappingListener").Return(nil).Once()

	err := ins.Remove(serviceUrl)
	// the joined error only contains r1's error; the call should error
	require.ErrorIs(t, err, removeErr)
	// both reports must have been called despite r1's failure
	r1.AssertExpectations(t)
	r2.AssertExpectations(t)
}

func TestServiceNameMappingGetMetersPerBusinessOperation(t *testing.T) {
	metadata.ClearMetadataReportInstances()
	t.Cleanup(metadata.ClearMetadataReportInstances)
	serviceNameMappingOnce = sync.Once{}
	serviceNameMappingInstance = nil

	r1 := initMockWithId(t, "reg-get-a")
	r2 := initMockWithId(t, "reg-get-b")

	ch := make(chan metrics.MetricsEvent, 10)
	metrics.Subscribe(constant.MetricsMetadata, ch)
	defer metrics.Unsubscribe(constant.MetricsMetadata)

	ins := GetNameMappingInstance()
	serviceUrl := common.NewURLWithOptions(
		common.WithInterface("org.example.MeteredService"),
	)

	// a listen fans out to every report but must count as a single listen event
	r1.On("GetServiceAppMapping").Return(gxset.NewSet("app-a"), nil).Once()
	r2.On("GetServiceAppMapping").Return(gxset.NewSet("app-b"), nil).Once()
	apps, err := ins.Get(serviceUrl, &listener{})
	require.NoError(t, err)
	assert.Equal(t, 2, apps.Size())
	assertMappingMetricEvent(t, <-ch, metricsMetadata.MetadataMappingListen, true, false, mappingAttachment("org.example.MeteredService"))
	assert.Empty(t, ch)

	// a plain get counts as a single get event
	r1.On("GetServiceAppMapping").Return(gxset.NewSet("app-a"), nil).Once()
	r2.On("GetServiceAppMapping").Return(gxset.NewSet("app-b"), nil).Once()
	apps, err = ins.Get(serviceUrl, nil)
	require.NoError(t, err)
	assert.Equal(t, 2, apps.Size())
	assertMappingMetricEvent(t, <-ch, metricsMetadata.MetadataMappingGet, true, false, mappingAttachment("org.example.MeteredService"))
	assert.Empty(t, ch)

	// partial failure is still a successful business operation (err == nil),
	// but the event must carry the partial flag
	getErr := errors.New("r1 failure")
	r1.On("GetServiceAppMapping").Return(gxset.NewSet(), getErr).Once()
	r2.On("GetServiceAppMapping").Return(gxset.NewSet("app-b"), nil).Once()
	apps, err = ins.Get(serviceUrl, &listener{})
	require.NoError(t, err)
	assert.False(t, apps.Empty())
	assertMappingMetricEvent(t, <-ch, metricsMetadata.MetadataMappingListen, true, true, mappingAttachment("org.example.MeteredService"))
	assert.Empty(t, ch)

	// all reports failing produces a single failed event
	r1.On("GetServiceAppMapping").Return(gxset.NewSet(), getErr).Once()
	r2.On("GetServiceAppMapping").Return(gxset.NewSet(), errors.New("r2 failure")).Once()
	_, err = ins.Get(serviceUrl, &listener{})
	require.Error(t, err)
	assertMappingMetricEvent(t, <-ch, metricsMetadata.MetadataMappingListen, false, false, mappingAttachment("org.example.MeteredService"))
	assert.Empty(t, ch)

	r1.AssertExpectations(t)
	r2.AssertExpectations(t)
}

func TestServiceNameMappingMapMetersPerBusinessOperation(t *testing.T) {
	metadata.ClearMetadataReportInstances()
	t.Cleanup(metadata.ClearMetadataReportInstances)
	serviceNameMappingOnce = sync.Once{}
	serviceNameMappingInstance = nil

	r1 := initMockWithId(t, "reg-map-a")
	r2 := initMockWithId(t, "reg-map-b")

	ch := make(chan metrics.MetricsEvent, 10)
	metrics.Subscribe(constant.MetricsMetadata, ch)
	defer metrics.Unsubscribe(constant.MetricsMetadata)

	ins := GetNameMappingInstance()
	serviceUrl := common.NewURLWithOptions(
		common.WithInterface("org.example.RegisterService"),
		common.WithParamsValue(constant.ApplicationKey, "reg-app"),
	)
	wantAttachment := mappingAttachment("org.example.RegisterService")
	wantAttachment[constant.ApplicationKey] = "reg-app"

	// all reports succeed: one register event
	r1.On("RegisterServiceAppMapping").Return(nil).Once()
	r2.On("RegisterServiceAppMapping").Return(nil).Once()
	err := ins.Map(serviceUrl)
	require.NoError(t, err)
	assertMappingMetricEvent(t, <-ch, metricsMetadata.MetadataMappingRegister, true, false, wantAttachment)
	assert.Empty(t, ch)

	// first report fails: Map stops immediately, one failed register event
	regErr := errors.New("r1 failure")
	// report iteration order is non-deterministic, so both may be the first to
	// fail; the call-count assertion below pins down that only one is invoked.
	r1.On("RegisterServiceAppMapping").Return(regErr).Maybe()
	r2.On("RegisterServiceAppMapping").Return(regErr).Maybe()
	callsBefore := len(r1.Calls) + len(r2.Calls)
	err = ins.Map(serviceUrl)
	require.Error(t, err)
	require.Equal(t, callsBefore+1, len(r1.Calls)+len(r2.Calls), "Map must stop at the first failing report")
	assertMappingMetricEvent(t, <-ch, metricsMetadata.MetadataMappingRegister, false, false, wantAttachment)
	assert.Empty(t, ch)

	r1.AssertExpectations(t)
	r2.AssertExpectations(t)
}

func TestServiceNameMappingRemoveMetersPerBusinessOperation(t *testing.T) {
	metadata.ClearMetadataReportInstances()
	t.Cleanup(metadata.ClearMetadataReportInstances)
	serviceNameMappingOnce = sync.Once{}
	serviceNameMappingInstance = nil

	r1 := initMockWithId(t, "reg-remove-a")
	r2 := initMockWithId(t, "reg-remove-b")

	ch := make(chan metrics.MetricsEvent, 10)
	metrics.Subscribe(constant.MetricsMetadata, ch)
	defer metrics.Unsubscribe(constant.MetricsMetadata)

	ins := GetNameMappingInstance()
	serviceUrl := common.NewURLWithOptions(
		common.WithInterface("org.example.RemoveService"),
	)
	wantAttachment := mappingAttachment("org.example.RemoveService")

	// all reports succeed: one remove event
	r1.On("RemoveServiceAppMappingListener").Return(nil).Once()
	r2.On("RemoveServiceAppMappingListener").Return(nil).Once()
	err := ins.Remove(serviceUrl)
	require.NoError(t, err)
	assertMappingMetricEvent(t, <-ch, metricsMetadata.MetadataMappingRemove, true, false, wantAttachment)
	assert.Empty(t, ch)

	// partial failure: best-effort still calls every report, one failed remove event
	removeErr := errors.New("r1 failure")
	r1.On("RemoveServiceAppMappingListener").Return(removeErr).Once()
	r2.On("RemoveServiceAppMappingListener").Return(nil).Once()
	err = ins.Remove(serviceUrl)
	require.ErrorIs(t, err, removeErr)
	assertMappingMetricEvent(t, <-ch, metricsMetadata.MetadataMappingRemove, false, false, wantAttachment)
	assert.Empty(t, ch)

	r1.AssertExpectations(t)
	r2.AssertExpectations(t)
}

func mappingAttachment(serviceInterface string) map[string]string {
	return map[string]string{
		constant.InterfaceKey: serviceInterface,
		constant.GroupKey:     DefaultGroup,
	}
}

func assertMappingMetricEvent(t *testing.T, metricEvent metrics.MetricsEvent, name metricsMetadata.MetricName, succ bool, partial bool, wantAttachment map[string]string) {
	t.Helper()
	assert.Equal(t, constant.MetricsMetadata, metricEvent.Type())
	event, ok := metricEvent.(*metricsMetadata.MetadataMetricEvent)
	assert.True(t, ok)
	assert.Equal(t, name, event.Name)
	assert.Equal(t, succ, event.Succ)
	assert.Equal(t, partial, event.Partial)
	assert.Equal(t, wantAttachment, event.Attachment)
	assert.NotNil(t, event.Start)
	assert.NotNil(t, event.End)
}

type listener struct {
}

func (l listener) OnEvent(e observer.Event) error {
	return nil
}

func (l listener) Stop() {
}

type mockMetadataReport struct {
	mock.Mock
}

func (m *mockMetadataReport) CreateMetadataReport(*common.URL) report.MetadataReport {
	return m
}

func (m *mockMetadataReport) GetAppMetadata(string, string) (*info.MetadataInfo, error) {
	args := m.Called()
	return args.Get(0).(*info.MetadataInfo), args.Error(1)
}

func (m *mockMetadataReport) PublishAppMetadata(string, string, *info.MetadataInfo) error {
	args := m.Called()
	return args.Error(0)
}

func (m *mockMetadataReport) RegisterServiceAppMapping(string, string, string) error {
	args := m.Called()
	return args.Error(0)
}

func (m *mockMetadataReport) GetServiceAppMapping(string, string, mapping.MappingListener) (*gxset.HashSet, error) {
	args := m.Called()
	return args.Get(0).(*gxset.HashSet), args.Error(1)
}

func (m *mockMetadataReport) RemoveServiceAppMappingListener(string, string) error {
	args := m.Called()
	return args.Error(0)
}

func (m *mockMetadataReport) UnPublishAppMetadata(string, string) error {
	args := m.Called()
	return args.Error(0)
}

func (m *mockMetadataReport) ListAppRevisions(string) ([]report.AppRevision, error) {
	args := m.Called()
	return args.Get(0).([]report.AppRevision), args.Error(1)
}

func (m *mockMetadataReport) URL() *common.URL {
	u, _ := common.NewURL("mock://127.0.0.1:8848")
	return u
}
