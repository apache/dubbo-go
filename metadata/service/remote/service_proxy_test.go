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

package remote

import (
	"testing"
)
import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/config/instance"
	"github.com/apache/dubbo-go/metadata/identifier"
	"github.com/apache/dubbo-go/metadata/report"
	"github.com/apache/dubbo-go/metadata/report/factory"
	"github.com/apache/dubbo-go/metadata/service"
	"github.com/apache/dubbo-go/registry"
)

func TestMetadataServiceProxy_GetExportedURLs(t *testing.T) {
	pxy := createProxy()
	res, err := pxy.GetExportedURLs(constant.ANY_VALUE, constant.ANY_VALUE, constant.ANY_VALUE, constant.ANY_VALUE)
	assert.Nil(t, err)
	assert.Len(t, res, 2)
}

func TestMetadataServiceProxy_GetServiceDefinition(t *testing.T) {
	pxy := createProxy()
	res, err := pxy.GetServiceDefinition(constant.ANY_VALUE, constant.ANY_VALUE, constant.ANY_VALUE)
	assert.Nil(t, err)
	assert.Equal(t, "definition", res)
}

// TestMetadataServiceProxy test those unimportant method
// in fact, we don't use them
func TestMetadataServiceProxy(t *testing.T) {
	pxy := createProxy()
	pxy.ServiceName()
	pxy.PublishServiceDefinition(common.URL{})
	pxy.Version()
	pxy.GetSubscribedURLs()
	pxy.UnsubscribeURL(common.URL{})
	pxy.GetServiceDefinitionByServiceKey("any")
	pxy.ExportURL(common.URL{})
	pxy.SubscribeURL(common.URL{})
	pxy.MethodMapper()
	pxy.UnexportURL(common.URL{})
	pxy.Reference()
	pxy.RefreshMetadata(constant.ANY_VALUE, constant.ANY_VALUE)
}

func createProxy() service.MetadataService {

	prepareTest()

	ins := &registry.DefaultServiceInstance{
		Id:          "test-id",
		ServiceName: "com.dubbo",
		Host:        "localhost",
		Port:        8080,
		Enable:      true,
		Healthy:     true,
		Metadata:    map[string]string{constant.METADATA_SERVICE_URL_PARAMS_PROPERTY_NAME: `{"mock":{"timeout":"10000","version":"1.0.0","dubbo":"2.0.2","release":"2.7.6","port":"20880"}}`},
	}
	return newMetadataServiceProxy(ins)
}

func prepareTest() {
	extension.SetMetadataReportFactory("mock", func() factory.MetadataReportFactory {
		return &mockMetadataReportFactory{}
	})
	u, _ := common.NewURL("mock://localhost")
	instance.GetMetadataReportInstance(&u)
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

func (m mockMetadataReport) SaveServiceMetadata(*identifier.ServiceMetadataIdentifier, common.URL) error {
	return nil
}

func (m mockMetadataReport) RemoveServiceMetadata(*identifier.ServiceMetadataIdentifier) error {
	panic("implement me")
}

func (m mockMetadataReport) GetExportedURLs(*identifier.ServiceMetadataIdentifier) ([]string, error) {
	return []string{"mock://localhost1", "mock://localhost2"}, nil
}

func (m mockMetadataReport) SaveSubscribedData(*identifier.SubscriberMetadataIdentifier, string) error {
	return nil
}

func (m mockMetadataReport) GetSubscribedURLs(*identifier.SubscriberMetadataIdentifier) ([]string, error) {
	panic("implement me")
}

func (m mockMetadataReport) GetServiceDefinition(*identifier.MetadataIdentifier) (string, error) {
	return "definition", nil
}
