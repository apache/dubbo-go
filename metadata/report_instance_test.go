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
	"testing"
)

import (
	gxset "github.com/dubbogo/gost/container/set"
	"github.com/dubbogo/gost/gof/observer"

	"github.com/pkg/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
	"dubbo.apache.org/dubbo-go/v3/metadata/mapping"
	"dubbo.apache.org/dubbo-go/v3/metadata/report"
	"dubbo.apache.org/dubbo-go/v3/metrics"
	metricsMetadata "dubbo.apache.org/dubbo-go/v3/metrics/metadata"
)

func TestDelegateMetadataReportGetAppMetadata(t *testing.T) {
	mockReport := new(mockMetadataReport)
	defer mockReport.AssertExpectations(t)
	delegate := &DelegateMetadataReport{instance: mockReport}
	metadataInfo := info.NewAppMetadataInfo("dubbo")
	var ch = make(chan metrics.MetricsEvent, 10)
	metrics.Subscribe(constant.MetricsMetadata, ch)
	defer close(ch)
	t.Run("normal", func(t *testing.T) {
		mockReport.On("GetAppMetadata").Return(metadataInfo, nil).Once()
		got, err := delegate.GetAppMetadata("dubbo", "1")
		require.NoError(t, err)
		assert.Equal(t, metadataInfo, got)
		assert.Len(t, ch, 1)
		metricEvent := <-ch
		assert.Equal(t, constant.MetricsMetadata, metricEvent.Type())
		event, ok := metricEvent.(*metricsMetadata.MetadataMetricEvent)
		assert.True(t, ok)
		assert.Equal(t, metricsMetadata.MetadataSub, event.Name)
		assert.NotNil(t, event.Start)
		assert.NotNil(t, event.End)
		assert.True(t, event.Succ)
	})
	t.Run("error", func(t *testing.T) {
		mockReport.On("GetAppMetadata").Return(info.NewAppMetadataInfo("dubbo"), errors.New("mock error")).Once()
		_, err := delegate.GetAppMetadata("dubbo", "1111")
		require.Error(t, err)
		assert.Len(t, ch, 1)
		metricEvent := <-ch
		assert.Equal(t, constant.MetricsMetadata, metricEvent.Type())
		event, ok := metricEvent.(*metricsMetadata.MetadataMetricEvent)
		assert.True(t, ok)
		assert.Equal(t, metricsMetadata.MetadataSub, event.Name)
		assert.NotNil(t, event.Start)
		assert.NotNil(t, event.End)
		assert.False(t, event.Succ)
	})
}

func TestDelegateMetadataReportPublishAppMetadata(t *testing.T) {
	mockReport := new(mockMetadataReport)
	defer mockReport.AssertExpectations(t)
	delegate := &DelegateMetadataReport{instance: mockReport}
	metadataInfo := info.NewAppMetadataInfo("dubbo")
	var ch = make(chan metrics.MetricsEvent, 10)
	metrics.Subscribe(constant.MetricsMetadata, ch)
	defer close(ch)
	t.Run("normal", func(t *testing.T) {
		mockReport.On("PublishAppMetadata").Return(nil).Once()
		err := delegate.PublishAppMetadata("application", "revision", metadataInfo)
		require.NoError(t, err)
		assert.Len(t, ch, 1)
		metricEvent := <-ch
		assert.Equal(t, constant.MetricsMetadata, metricEvent.Type())
		event, ok := metricEvent.(*metricsMetadata.MetadataMetricEvent)
		assert.True(t, ok)
		assert.Equal(t, metricsMetadata.MetadataPush, event.Name)
		assert.NotNil(t, event.Start)
		assert.NotNil(t, event.End)
		assert.True(t, event.Succ)
	})
	t.Run("error", func(t *testing.T) {
		mockReport.On("PublishAppMetadata").Return(errors.New("mock error")).Once()
		err := delegate.PublishAppMetadata("application", "revision", metadataInfo)
		require.Error(t, err)
		assert.Len(t, ch, 1)
		metricEvent := <-ch
		assert.Equal(t, constant.MetricsMetadata, metricEvent.Type())
		event, ok := metricEvent.(*metricsMetadata.MetadataMetricEvent)
		assert.True(t, ok)
		assert.Equal(t, metricsMetadata.MetadataPush, event.Name)
		assert.NotNil(t, event.Start)
		assert.NotNil(t, event.End)
		assert.False(t, event.Succ)
	})
}

func TestDelegateMetadataReportGetServiceAppMapping(t *testing.T) {
	mockReport := new(mockMetadataReport)
	defer mockReport.AssertExpectations(t)
	delegate := &DelegateMetadataReport{instance: mockReport}
	t.Run("normal", func(t *testing.T) {
		mockReport.On("GetServiceAppMapping").Return(gxset.NewSet(), nil).Once()
		got, err := delegate.GetServiceAppMapping("dubbo", "dev", &listener{})
		require.NoError(t, err)
		assert.True(t, got.Empty())
	})
	t.Run("error", func(t *testing.T) {
		mockReport.On("GetServiceAppMapping").Return(gxset.NewSet(), errors.New("mock error")).Once()
		_, err := delegate.GetServiceAppMapping("dubbo", "dev", &listener{})
		require.Error(t, err)
	})
}

func TestDelegateMetadataReportRegisterServiceAppMapping(t *testing.T) {
	mockReport := new(mockMetadataReport)
	defer mockReport.AssertExpectations(t)
	delegate := &DelegateMetadataReport{instance: mockReport}
	t.Run("normal", func(t *testing.T) {
		mockReport.On("RegisterServiceAppMapping").Return(nil).Once()
		err := delegate.RegisterServiceAppMapping("interfaceName", "group", "application")
		require.NoError(t, err)
	})
	t.Run("error", func(t *testing.T) {
		mockReport.On("RegisterServiceAppMapping").Return(errors.New("mock error")).Once()
		err := delegate.RegisterServiceAppMapping("interfaceName", "group", "application")
		require.Error(t, err)
	})
}

func TestDelegateMetadataReportRemoveServiceAppMappingListener(t *testing.T) {
	mockReport := new(mockMetadataReport)
	defer mockReport.AssertExpectations(t)
	delegate := &DelegateMetadataReport{instance: mockReport}
	t.Run("normal", func(t *testing.T) {
		mockReport.On("RemoveServiceAppMappingListener").Return(nil).Once()
		err := delegate.RemoveServiceAppMappingListener("interfaceName", "group")
		require.NoError(t, err)
	})
	t.Run("error", func(t *testing.T) {
		mockReport.On("RemoveServiceAppMappingListener").Return(errors.New("mock error")).Once()
		err := delegate.RemoveServiceAppMappingListener("interfaceName", "group")
		require.Error(t, err)
	})
}

func TestGetMetadataReport(t *testing.T) {
	instances = make(map[string]report.MetadataReport)
	assert.Nil(t, GetMetadataReport())
	instances["default"] = new(mockMetadataReport)
	assert.NotNil(t, GetMetadataReport())
}

func TestGetMetadataReportByRegistry(t *testing.T) {
	instances = make(map[string]report.MetadataReport)
	assert.Nil(t, GetMetadataReportByRegistry("reg"))
	instances["default"] = new(mockMetadataReport)
	assert.NotNil(t, GetMetadataReportByRegistry("default"))
	assert.NotNil(t, GetMetadataReportByRegistry("reg"))
	assert.NotNil(t, GetMetadataReportByRegistry(""))
}

func TestGetMetadataReports(t *testing.T) {
	instances = make(map[string]report.MetadataReport)
	assert.Empty(t, GetMetadataReports())
	instances["default"] = new(mockMetadataReport)
	assert.Len(t, GetMetadataReports(), 1)
}

func TestGetMetadataType(t *testing.T) {
	assert.Equal(t, constant.DefaultMetadataStorageType, GetMetadataType())
	metadataOptions = &Options{}
	assert.Equal(t, constant.DefaultMetadataStorageType, GetMetadataType())
	metadataOptions = &Options{
		metadataType: constant.RemoteMetadataStorageType,
	}
	assert.Equal(t, constant.RemoteMetadataStorageType, GetMetadataType())
}

func TestAddMetadataReport(t *testing.T) {
	url := common.NewURLWithOptions(
		common.WithProtocol("registryId"),
	)
	err := addMetadataReport("registryId", url)
	require.NoError(t, err)
	assert.Nil(t, instances["registryId"])
	mockReport := new(mockMetadataReport)
	extension.SetMetadataReportFactory("registryId", func() report.MetadataReportFactory {
		return mockReport
	})
	err = addMetadataReport("registryId", url)
	require.NoError(t, err)
	assert.NotNil(t, instances["registryId"])
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

type listener struct {
}

func (l *listener) OnEvent(e observer.Event) error {
	return nil
}

func (l *listener) Stop() {
}
