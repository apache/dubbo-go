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
	"testing"
)

import (
	gxset "github.com/dubbogo/gost/container/set"
	"github.com/dubbogo/gost/gof/observer"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/metadata"
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
	"dubbo.apache.org/dubbo-go/v3/metadata/mapping"
	"dubbo.apache.org/dubbo-go/v3/metadata/report"
)

type mockMetadataReport struct{ mock.Mock }

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

func (l mockListener) OnEvent(e observer.Event) error { return nil }
func (l mockListener) Stop()                          {}

func initMockReport() (*mockMetadataReport, error) {
	mockReport := new(mockMetadataReport)
	extension.SetMetadataReportFactory("mock", func() report.MetadataReportFactory { return mockReport })
	opts := metadata.NewReportOptions(metadata.WithProtocol("mock"), metadata.WithAddress("127.0.0.1"))
	return mockReport, opts.Init()
}

func createServiceURL() *common.URL {
	return common.NewURLWithOptions(
		common.WithInterface("org.apache.dubbo.samples.proto.GreetService"),
		common.WithParamsValue(constant.ApplicationKey, "dubbo"),
	)
}

func TestGetNameMappingInstance(t *testing.T) {
	assert.NotNil(t, GetNameMappingInstance())
}

func TestNoReportInstance(t *testing.T) {
	ins := GetNameMappingInstance()
	url := createServiceURL()

	_, err := ins.Get(url, &mockListener{})
	assert.NotNil(t, err)

	assert.NotNil(t, ins.Map(url))
	assert.NotNil(t, ins.Remove(url))
}

func TestServiceNameMappingGet(t *testing.T) {
	mockReport, err := initMockReport()
	assert.Nil(t, err)

	ins := GetNameMappingInstance()
	url := createServiceURL()

	mockReport.On("GetServiceAppMapping").Return(gxset.NewSet("dubbo"), nil).Once()
	apps, err := ins.Get(url, &mockListener{})
	assert.Nil(t, err)
	assert.False(t, apps.Empty())

	mockReport.On("GetServiceAppMapping").Return(gxset.NewSet(), errors.New("error")).Once()
	_, err = ins.Get(url, &mockListener{})
	assert.NotNil(t, err)
}

func TestServiceNameMappingMap(t *testing.T) {
	mockReport, err := initMockReport()
	assert.Nil(t, err)

	ins := GetNameMappingInstance()
	url := createServiceURL()

	mockReport.On("RegisterServiceAppMapping").Return(nil).Once()
	assert.Nil(t, ins.Map(url))

	mockReport.On("RegisterServiceAppMapping").Return(errors.New("error")).Times(retryTimes)
	assert.NotNil(t, ins.Map(url))
}

func TestServiceNameMappingRemove(t *testing.T) {
	mockReport, err := initMockReport()
	assert.Nil(t, err)

	ins := GetNameMappingInstance()
	url := createServiceURL()

	mockReport.On("RemoveServiceAppMappingListener").Return(nil).Once()
	assert.Nil(t, ins.Remove(url))

	mockReport.On("RemoveServiceAppMappingListener").Return(errors.New("error")).Once()
	assert.NotNil(t, ins.Remove(url))
}
