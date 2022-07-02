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
	"fmt"
	"testing"
)

import (
	gxset "github.com/dubbogo/gost/container/set"
	"github.com/dubbogo/gost/log/logger"

	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config/instance"
	"dubbo.apache.org/dubbo-go/v3/metadata/definition"
	"dubbo.apache.org/dubbo-go/v3/metadata/identifier"
	"dubbo.apache.org/dubbo-go/v3/metadata/report"
	"dubbo.apache.org/dubbo-go/v3/metadata/report/factory"
	"dubbo.apache.org/dubbo-go/v3/metadata/service/local"
)

var (
	serviceMetadata    = make(map[*identifier.ServiceMetadataIdentifier]*common.URL, 4)
	subscribedMetadata = make(map[*identifier.SubscriberMetadataIdentifier]string, 4)
)

func getMetadataReportFactory() factory.MetadataReportFactory {
	return &metadataReportFactory{}
}

type metadataReportFactory struct{}

func (mrf *metadataReportFactory) CreateMetadataReport(*common.URL) report.MetadataReport {
	return &metadataReport{}
}

type metadataReport struct{}

func (mr metadataReport) RegisterServiceAppMapping(string, string, string) error {
	panic("implement me")
}

func (mr metadataReport) GetServiceAppMapping(string, string) (*gxset.HashSet, error) {
	panic("implement me")
}

func (mr metadataReport) GetAppMetadata(*identifier.SubscriberMetadataIdentifier) (*common.MetadataInfo, error) {
	panic("implement me")
}

func (mr metadataReport) PublishAppMetadata(*identifier.SubscriberMetadataIdentifier, *common.MetadataInfo) error {
	panic("implement me")
}

func (metadataReport) StoreProviderMetadata(*identifier.MetadataIdentifier, string) error {
	return nil
}

func (metadataReport) StoreConsumerMetadata(*identifier.MetadataIdentifier, string) error {
	return nil
}

func (mr *metadataReport) SaveServiceMetadata(id *identifier.ServiceMetadataIdentifier, url *common.URL) error {
	logger.Infof("SaveServiceMetadata , url is %v", url)
	serviceMetadata[id] = url
	return nil
}

func (metadataReport) RemoveServiceMetadata(*identifier.ServiceMetadataIdentifier) error {
	return nil
}

func (metadataReport) GetExportedURLs(*identifier.ServiceMetadataIdentifier) ([]string, error) {
	return nil, nil
}

func (mr *metadataReport) SaveSubscribedData(id *identifier.SubscriberMetadataIdentifier, urls string) error {
	logger.Infof("SaveSubscribedData, , url is %v", urls)
	subscribedMetadata[id] = urls
	return nil
}

func (metadataReport) GetSubscribedURLs(*identifier.SubscriberMetadataIdentifier) ([]string, error) {
	return nil, nil
}

func (metadataReport) GetServiceDefinition(*identifier.MetadataIdentifier) (string, error) {
	return "", nil
}

func TestMetadataService(t *testing.T) {
	extension.SetMetadataReportFactory("mock", getMetadataReportFactory)
	u, err := common.NewURL("mock://127.0.0.1:20000/?sync.report=true")
	assert.NoError(t, err)
	instance.GetMetadataReportInstance(u)
	mts, err := GetRemoteMetadataService()
	assert.NoError(t, err)
	assert.NotNil(t, mts)
}

func TestMockInmemoryProc(t *testing.T) {
	mts, _ := local.GetLocalMetadataService()
	serviceName := "com.ikurento.user.UserProvider"
	group := "group1"
	version := "0.0.1"
	protocol := "dubbo"
	beanName := "UserProvider"
	userProvider := &definition.UserProvider{}

	u, err := common.NewURL(fmt.Sprintf(
		"%v://127.0.0.1:20000/com.ikurento.user.UserProvider1?anyhost=true&"+
			"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&"+
			"environment=dev&interface=%v&ip=192.168.56.1&methods=GetUser&module=dubbogo+user-info+server&org=ikurento.com&"+
			"owner=ZX&pid=1447&revision=0.0.1&side=provider&timeout=3000&timestamp=1556509797245&group=%v&version=%v&bean.name=%v",
		protocol, serviceName, group, version, beanName))
	assert.NoError(t, err)

	_, err = mts.ExportURL(u)
	assert.NoError(t, err)
	_, err = mts.SubscribeURL(u)
	assert.NoError(t, err)

	_, err = common.ServiceMap.Register(serviceName, protocol, group, version, userProvider)
	assert.NoError(t, err)
	err = mts.PublishServiceDefinition(u)
	assert.NoError(t, err)

	expected := "{\"CanonicalName\":\"com.ikurento.user.UserProvider\",\"CodeSource\":\"\"," +
		"\"Methods\":[{\"Name\":\"GetUser\",\"ParameterTypes\":[\"slice\"],\"ReturnType\":\"ptr\"," +
		"\"Parameters\":null}],\"Types\":null}"
	def1, _ := mts.GetServiceDefinition(serviceName, group, version)
	assert.Equal(t, expected, def1)
	serviceKey := definition.ServiceDescriperBuild(serviceName, group, version)
	def2, _ := mts.GetServiceDefinitionByServiceKey(serviceKey)
	assert.Equal(t, expected, def2)
}
