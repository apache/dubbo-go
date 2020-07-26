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

package event

import (
	"testing"
)

import (
	gxset "github.com/dubbogo/gost/container/set"
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/metadata/service"
	"github.com/apache/dubbo-go/registry"
)

func prepareMetadataServiceForTest() {
	config.GetApplicationConfig().MetadataType = "mock"
	extension.SetMetadataService("mock", func() (service.MetadataService, error) {
		return &mockMetadataService{
			urls: []interface{}{"mock://localhost:8080?a=b"},
		}, nil
	})
}

func TestMetadataServiceURLParamsMetadataCustomizer(t *testing.T) {

	prepareMetadataServiceForTest()

	msup := &metadataServiceURLParamsMetadataCustomizer{exceptKeys: gxset.NewSet()}
	assert.Equal(t, 0, msup.GetPriority())

	msup.Customize(createInstance())
}

func createInstance() registry.ServiceInstance {
	ins := &registry.DefaultServiceInstance{}
	return ins
}

type mockMetadataService struct {
	urls []interface{}
}

func (m *mockMetadataService) Reference() string {
	panic("implement me")
}

func (m *mockMetadataService) ServiceName() (string, error) {
	panic("implement me")
}

func (m *mockMetadataService) ExportURL(url common.URL) (bool, error) {
	panic("implement me")
}

func (m *mockMetadataService) UnexportURL(url common.URL) error {
	panic("implement me")
}

func (m *mockMetadataService) SubscribeURL(url common.URL) (bool, error) {
	panic("implement me")
}

func (m *mockMetadataService) UnsubscribeURL(url common.URL) error {
	panic("implement me")
}

func (m *mockMetadataService) PublishServiceDefinition(url common.URL) error {
	panic("implement me")
}

func (m *mockMetadataService) GetExportedURLs(serviceInterface string, group string, version string, protocol string) ([]interface{}, error) {
	return m.urls, nil
}

func (m *mockMetadataService) MethodMapper() map[string]string {
	panic("implement me")
}

func (m *mockMetadataService) GetSubscribedURLs() ([]common.URL, error) {
	res := make([]common.URL, 0, len(m.urls))
	for _, ui := range m.urls {
		u, _ := common.NewURL(ui.(string))
		res = append(res, u)
	}
	return res, nil
}

func (m *mockMetadataService) GetServiceDefinition(interfaceName string, group string, version string) (string, error) {
	panic("implement me")
}

func (m *mockMetadataService) GetServiceDefinitionByServiceKey(serviceKey string) (string, error) {
	panic("implement me")
}

func (m *mockMetadataService) RefreshMetadata(exportedRevision string, subscribedRevision string) (bool, error) {
	panic("implement me")
}

func (m *mockMetadataService) Version() (string, error) {
	return "1.0.0", nil
}
