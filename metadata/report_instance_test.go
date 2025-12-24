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
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
	"dubbo.apache.org/dubbo-go/v3/metadata/mapping"
	"dubbo.apache.org/dubbo-go/v3/metadata/report"

	gxset "github.com/dubbogo/gost/container/set"
	"github.com/dubbogo/gost/gof/observer"

	"github.com/pkg/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockMetadataReport struct {
	mock.Mock
}

func (m *mockMetadataReport) CreateMetadataReport(*common.URL) report.MetadataReport { return m }

func (m *mockMetadataReport) GetAppMetadata(string, string) (*info.MetadataInfo, error) {
	args := m.Called()
	return args.Get(0).(*info.MetadataInfo), args.Error(1)
}

func (m *mockMetadataReport) PublishAppMetadata(string, string, *info.MetadataInfo) error {
	return m.Called().Error(0)
}

func (m *mockMetadataReport) RegisterServiceAppMapping(string, string, string) error {
	return m.Called().Error(0)
}

func (m *mockMetadataReport) GetServiceAppMapping(string, string, mapping.MappingListener) (*gxset.HashSet, error) {
	args := m.Called()
	return args.Get(0).(*gxset.HashSet), args.Error(1)
}

func (m *mockMetadataReport) RemoveServiceAppMappingListener(string, string) error {
	return m.Called().Error(0)
}

type mockListener struct{}

func (l *mockListener) OnEvent(e observer.Event) error { return nil }
func (l *mockListener) Stop()                          {}

func TestDelegateMetadataReportGetAppMetadata(t *testing.T) {
	mockReport := new(mockMetadataReport)
	delegate := &DelegateMetadataReport{instance: mockReport}
	metadataInfo := info.NewAppMetadataInfo("dubbo")

	mockReport.On("GetAppMetadata").Return(metadataInfo, nil).Once()
	got, err := delegate.GetAppMetadata("dubbo", "1")
	assert.Nil(t, err)
	assert.Equal(t, metadataInfo, got)

	mockReport.On("GetAppMetadata").Return(info.NewAppMetadataInfo(""), errors.New("error")).Once()
	_, err = delegate.GetAppMetadata("dubbo", "2")
	assert.NotNil(t, err)
}

func TestDelegateMetadataReportPublishAppMetadata(t *testing.T) {
	mockReport := new(mockMetadataReport)
	delegate := &DelegateMetadataReport{instance: mockReport}

	mockReport.On("PublishAppMetadata").Return(nil).Once()
	assert.Nil(t, delegate.PublishAppMetadata("app", "rev", info.NewAppMetadataInfo("dubbo")))

	mockReport.On("PublishAppMetadata").Return(errors.New("error")).Once()
	assert.NotNil(t, delegate.PublishAppMetadata("app", "rev", info.NewAppMetadataInfo("dubbo")))
}

func TestDelegateMetadataReportServiceAppMapping(t *testing.T) {
	mockReport := new(mockMetadataReport)
	delegate := &DelegateMetadataReport{instance: mockReport}

	mockReport.On("GetServiceAppMapping").Return(gxset.NewSet(), nil).Once()
	got, err := delegate.GetServiceAppMapping("dubbo", "dev", &mockListener{})
	assert.Nil(t, err)
	assert.True(t, got.Empty())

	mockReport.On("RegisterServiceAppMapping").Return(nil).Once()
	assert.Nil(t, delegate.RegisterServiceAppMapping("interface", "group", "app"))

	mockReport.On("RemoveServiceAppMappingListener").Return(nil).Once()
	assert.Nil(t, delegate.RemoveServiceAppMappingListener("interface", "group"))
}

func TestGetMetadataReport(t *testing.T) {
	instances = make(map[string]report.MetadataReport)
	assert.Nil(t, GetMetadataReport())

	instances["default"] = new(mockMetadataReport)
	assert.NotNil(t, GetMetadataReport())
}

func TestGetMetadataReportByRegistry(t *testing.T) {
	instances = make(map[string]report.MetadataReport)
	instances["default"] = new(mockMetadataReport)

	assert.NotNil(t, GetMetadataReportByRegistry("default"))
	assert.NotNil(t, GetMetadataReportByRegistry("other"))
	assert.NotNil(t, GetMetadataReportByRegistry(""))
}

func TestGetMetadataReports(t *testing.T) {
	instances = make(map[string]report.MetadataReport)
	assert.Equal(t, 0, len(GetMetadataReports()))

	instances["default"] = new(mockMetadataReport)
	assert.Equal(t, 1, len(GetMetadataReports()))
}

func TestGetMetadataType(t *testing.T) {
	metadataOptions = nil
	assert.Equal(t, constant.DefaultMetadataStorageType, GetMetadataType())

	metadataOptions = &Options{metadataType: constant.RemoteMetadataStorageType}
	assert.Equal(t, constant.RemoteMetadataStorageType, GetMetadataType())
}

func TestAddMetadataReport(t *testing.T) {
	instances = make(map[string]report.MetadataReport)
	url := common.NewURLWithOptions(common.WithProtocol("test-protocol"))

	// No factory registered
	err := addMetadataReport("reg1", url)
	assert.Nil(t, err)
	assert.Nil(t, instances["reg1"])

	// With factory
	extension.SetMetadataReportFactory("test-protocol", func() report.MetadataReportFactory {
		return new(mockMetadataReport)
	})
	err = addMetadataReport("reg2", url)
	assert.Nil(t, err)
	assert.NotNil(t, instances["reg2"])
}
