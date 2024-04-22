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
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/metadata"
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
	"dubbo.apache.org/dubbo-go/v3/metadata/mapping"
	"dubbo.apache.org/dubbo-go/v3/metadata/report"
	"errors"
	gxset "github.com/dubbogo/gost/container/set"
	"github.com/dubbogo/gost/gof/observer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
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
	assert.NotNil(t, err, "test Get no report instance")
	err = ins.Map(serviceUrl)
	assert.NotNil(t, err, "test Map with no report instance")
	err = ins.Remove(serviceUrl)
	assert.NotNil(t, err, "test Remove with no report instance")
}

func TestServiceNameMappingGet(t *testing.T) {
	ins := GetNameMappingInstance()
	lis := &listener{}
	serviceUrl := common.NewURLWithOptions(
		common.WithInterface("org.apache.dubbo.samples.proto.GreetService"),
		common.WithParamsValue(constant.ApplicationKey, "dubbo"),
	)
	mockReport, err := initMock()
	assert.Nil(t, err)
	t.Run("test normal", func(t *testing.T) {
		mockReport.On("GetServiceAppMapping").Return(gxset.NewSet("dubbo"), nil).Once()
		apps, er := ins.Get(serviceUrl, lis)
		assert.Nil(t, er)
		assert.True(t, !apps.Empty())
	})
	t.Run("test error", func(t *testing.T) {
		mockReport.On("GetServiceAppMapping").Return(gxset.NewSet(), errors.New("mock error")).Once()
		_, err = ins.Get(serviceUrl, lis)
		assert.NotNil(t, err)
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
	assert.Nil(t, err)
	t.Run("test normal", func(t *testing.T) {
		mockReport.On("RegisterServiceAppMapping").Return(nil).Once()
		err = ins.Map(serviceUrl)
		assert.Nil(t, err)
	})
	t.Run("test error", func(t *testing.T) {
		mockReport.On("RegisterServiceAppMapping").Return(errors.New("mock error")).Times(retryTimes)
		err = ins.Map(serviceUrl)
		assert.NotNil(t, err, "test mapping error")
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
	assert.Nil(t, err)
	t.Run("test normal", func(t *testing.T) {
		mockReport.On("RemoveServiceAppMappingListener").Return(nil).Once()
		err = ins.Remove(serviceUrl)
		assert.Nil(t, err)
	})
	t.Run("test error", func(t *testing.T) {
		mockReport.On("RemoveServiceAppMappingListener").Return(errors.New("mock error")).Once()
		err = ins.Remove(serviceUrl)
		assert.NotNil(t, err)
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
