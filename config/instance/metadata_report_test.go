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

package instance

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/metadata/identifier"
	"github.com/apache/dubbo-go/metadata/report"
	"github.com/apache/dubbo-go/metadata/report/factory"
)

func TestGetMetadataReportInstance(t *testing.T) {
	extension.SetMetadataReportFactory("mock", func() factory.MetadataReportFactory {
		return &mockMetadataReportFactory{}
	})
	u, _ := common.NewURL("mock://127.0.0.1")
	rpt := GetMetadataReportInstance(u)
	assert.NotNil(t, rpt)
}

type mockMetadataReportFactory struct {
}

func (m *mockMetadataReportFactory) CreateMetadataReport(*common.URL) report.MetadataReport {
	return &mockMetadataReport{}
}

type mockMetadataReport struct {
}

func (m mockMetadataReport) StoreProviderMetadata(*identifier.MetadataIdentifier, string) error {
	panic("implement me")
}

func (m mockMetadataReport) StoreConsumerMetadata(*identifier.MetadataIdentifier, string) error {
	panic("implement me")
}

func (m mockMetadataReport) SaveServiceMetadata(*identifier.ServiceMetadataIdentifier, *common.URL) error {
	panic("implement me")
}

func (m mockMetadataReport) RemoveServiceMetadata(*identifier.ServiceMetadataIdentifier) error {
	panic("implement me")
}

func (m mockMetadataReport) GetExportedURLs(*identifier.ServiceMetadataIdentifier) ([]string, error) {
	panic("implement me")
}

func (m mockMetadataReport) SaveSubscribedData(*identifier.SubscriberMetadataIdentifier, string) error {
	panic("implement me")
}

func (m mockMetadataReport) GetSubscribedURLs(*identifier.SubscriberMetadataIdentifier) ([]string, error) {
	panic("implement me")
}

func (m mockMetadataReport) GetServiceDefinition(*identifier.MetadataIdentifier) (string, error) {
	panic("implement me")
}
