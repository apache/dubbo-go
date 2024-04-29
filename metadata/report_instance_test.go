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
	"reflect"
	"testing"
)

import (
	gxset "github.com/dubbogo/gost/container/set"
	"github.com/dubbogo/gost/gof/observer"

	"github.com/pkg/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
		assert.Nil(t, err)
		if !reflect.DeepEqual(got, metadataInfo) {
			t.Errorf("GetAppMetadata() got = %v, want %v", got, metadataInfo)
		}
		assert.True(t, len(ch) == 1)
		metricEvent := <-ch
		assert.Equal(t, metricEvent.Type(), constant.MetricsMetadata)
		event, ok := metricEvent.(*metricsMetadata.MetadataMetricEvent)
		assert.True(t, ok)
		assert.NotNil(t, event.Name, metricsMetadata.MetadataSub)
		assert.NotNil(t, event.Start)
		assert.NotNil(t, event.End)
		assert.True(t, event.Succ)
	})
	t.Run("error", func(t *testing.T) {
		mockReport.On("GetAppMetadata").Return(info.NewAppMetadataInfo("dubbo"), errors.New("mock error")).Once()
		_, err := delegate.GetAppMetadata("dubbo", "1111")
		assert.NotNil(t, err)
		assert.True(t, len(ch) == 1)
		metricEvent := <-ch
		assert.Equal(t, metricEvent.Type(), constant.MetricsMetadata)
		event, ok := metricEvent.(*metricsMetadata.MetadataMetricEvent)
		assert.True(t, ok)
		assert.NotNil(t, event.Name, metricsMetadata.MetadataSub)
		assert.NotNil(t, event.Start)
		assert.NotNil(t, event.End)
		assert.True(t, !event.Succ)
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
		assert.Nil(t, err)
		assert.True(t, len(ch) == 1)
		metricEvent := <-ch
		assert.Equal(t, metricEvent.Type(), constant.MetricsMetadata)
		event, ok := metricEvent.(*metricsMetadata.MetadataMetricEvent)
		assert.True(t, ok)
		assert.NotNil(t, event.Name, metricsMetadata.MetadataPush)
		assert.NotNil(t, event.Start)
		assert.NotNil(t, event.End)
		assert.True(t, event.Succ)
	})
	t.Run("error", func(t *testing.T) {
		mockReport.On("PublishAppMetadata").Return(errors.New("mock error")).Once()
		err := delegate.PublishAppMetadata("application", "revision", metadataInfo)
		assert.NotNil(t, err)
		assert.True(t, len(ch) == 1)
		metricEvent := <-ch
		assert.Equal(t, metricEvent.Type(), constant.MetricsMetadata)
		event, ok := metricEvent.(*metricsMetadata.MetadataMetricEvent)
		assert.True(t, ok)
		assert.NotNil(t, event.Name, metricsMetadata.MetadataPush)
		assert.NotNil(t, event.Start)
		assert.NotNil(t, event.End)
		assert.True(t, !event.Succ)
	})
}

func TestDelegateMetadataReportGetServiceAppMapping(t *testing.T) {
	mockReport := new(mockMetadataReport)
	defer mockReport.AssertExpectations(t)
	delegate := &DelegateMetadataReport{instance: mockReport}
	t.Run("normal", func(t *testing.T) {
		mockReport.On("GetServiceAppMapping").Return(gxset.NewSet(), nil).Once()
		got, err := delegate.GetServiceAppMapping("dubbo", "dev", &listener{})
		assert.Nil(t, err)
		assert.True(t, got.Empty())
	})
	t.Run("error", func(t *testing.T) {
		mockReport.On("GetServiceAppMapping").Return(gxset.NewSet(), errors.New("mock error")).Once()
		_, err := delegate.GetServiceAppMapping("dubbo", "dev", &listener{})
		assert.NotNil(t, err)
	})
}

func TestDelegateMetadataReportRegisterServiceAppMapping(t *testing.T) {
	mockReport := new(mockMetadataReport)
	defer mockReport.AssertExpectations(t)
	delegate := &DelegateMetadataReport{instance: mockReport}
	t.Run("normal", func(t *testing.T) {
		mockReport.On("RegisterServiceAppMapping").Return(nil).Once()
		err := delegate.RegisterServiceAppMapping("interfaceName", "group", "application")
		assert.Nil(t, err)
	})
	t.Run("error", func(t *testing.T) {
		mockReport.On("RegisterServiceAppMapping").Return(errors.New("mock error")).Once()
		err := delegate.RegisterServiceAppMapping("interfaceName", "group", "application")
		assert.NotNil(t, err)
	})
}

func TestDelegateMetadataReportRemoveServiceAppMappingListener(t *testing.T) {
	mockReport := new(mockMetadataReport)
	defer mockReport.AssertExpectations(t)
	delegate := &DelegateMetadataReport{instance: mockReport}
	t.Run("normal", func(t *testing.T) {
		mockReport.On("RemoveServiceAppMappingListener").Return(nil).Once()
		err := delegate.RemoveServiceAppMappingListener("interfaceName", "group")
		assert.Nil(t, err)
	})
	t.Run("error", func(t *testing.T) {
		mockReport.On("RemoveServiceAppMappingListener").Return(errors.New("mock error")).Once()
		err := delegate.RemoveServiceAppMappingListener("interfaceName", "group")
		assert.NotNil(t, err)
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
	assert.True(t, len(GetMetadataReports()) == 0)
	instances["default"] = new(mockMetadataReport)
	assert.True(t, len(GetMetadataReports()) == 1)
}

func TestGetMetadataType(t *testing.T) {
	assert.Equal(t, GetMetadataType(), constant.DefaultMetadataStorageType)
	metadataOptions = &Options{}
	assert.Equal(t, GetMetadataType(), constant.DefaultMetadataStorageType)
	metadataOptions = &Options{
		metadataType: constant.RemoteMetadataStorageType,
	}
	assert.Equal(t, GetMetadataType(), constant.RemoteMetadataStorageType)
}

func TestAddMetadataReport(t *testing.T) {
	url := common.NewURLWithOptions(
		common.WithProtocol("registryId"),
	)
	err := addMetadataReport("registryId", url)
	assert.Nil(t, err)
	assert.True(t, instances["registryId"] == nil)
	mockReport := new(mockMetadataReport)
	extension.SetMetadataReportFactory("registryId", func() report.MetadataReportFactory {
		return mockReport
	})
	err = addMetadataReport("registryId", url)
	assert.Nil(t, err)
	assert.True(t, instances["registryId"] != nil)
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
