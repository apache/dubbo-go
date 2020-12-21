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

package zookeeper

import (
	"encoding/json"
	"net/url"
	"strconv"
	"testing"
)

import (
	"github.com/dubbogo/go-zookeeper/zk"
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/metadata/identifier"
	"github.com/apache/dubbo-go/metadata/report"
)

func newProviderRegistryUrl(host string, port int) *common.URL {
	return common.NewURLWithOptions(
		common.WithIp(host),
		common.WithPort(strconv.Itoa(port)),
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.ROLE_KEY, strconv.Itoa(common.PROVIDER)),
	)
}

func newBaseMetadataIdentifier(side string) *identifier.BaseMetadataIdentifier {
	return &identifier.BaseMetadataIdentifier{
		ServiceInterface: "org.apache.HelloWorld",
		Version:          "1.0.0",
		Group:            "group",
		Side:             side,
	}
}

func newMetadataIdentifier(side string) *identifier.MetadataIdentifier {
	return &identifier.MetadataIdentifier{
		Application:            "application",
		BaseMetadataIdentifier: *newBaseMetadataIdentifier(side),
	}
}

func newServiceMetadataIdentifier(side string) *identifier.ServiceMetadataIdentifier {
	return &identifier.ServiceMetadataIdentifier{
		Revision:               "1.0",
		Protocol:               "dubbo",
		BaseMetadataIdentifier: *newBaseMetadataIdentifier(side),
	}
}

func newSubscribeMetadataIdentifier(side string) *identifier.SubscriberMetadataIdentifier {
	return &identifier.SubscriberMetadataIdentifier{
		Revision:           "1.0",
		MetadataIdentifier: *newMetadataIdentifier(side),
	}
}

type zookeeperMetadataReportTestSuite struct {
	t *testing.T
	m report.MetadataReport
}

func newZookeeperMetadataReportTestSuite(t *testing.T, m report.MetadataReport) *zookeeperMetadataReportTestSuite {
	return &zookeeperMetadataReportTestSuite{t: t, m: m}
}

func (suite *zookeeperMetadataReportTestSuite) testStoreProviderMetadata() {
	providerMi := newMetadataIdentifier("provider")
	providerMeta := "provider"
	err := suite.m.StoreProviderMetadata(providerMi, providerMeta)
	assert.NoError(suite.t, err)
}

func (suite *zookeeperMetadataReportTestSuite) testStoreConsumerMetadata() {
	consumerMi := newMetadataIdentifier("consumer")
	consumerMeta := "consumer"
	err := suite.m.StoreProviderMetadata(consumerMi, consumerMeta)
	assert.NoError(suite.t, err)
}

func (suite *zookeeperMetadataReportTestSuite) testSaveServiceMetadata(url *common.URL) {
	serviceMi := newServiceMetadataIdentifier("provider")
	err := suite.m.SaveServiceMetadata(serviceMi, url)
	assert.NoError(suite.t, err)
}

func (suite *zookeeperMetadataReportTestSuite) testRemoveServiceMetadata() {
	serviceMi := newServiceMetadataIdentifier("provider")
	err := suite.m.RemoveServiceMetadata(serviceMi)
	assert.NoError(suite.t, err)
}

func (suite *zookeeperMetadataReportTestSuite) testGetExportedURLs() {
	serviceMi := newServiceMetadataIdentifier("provider")
	urls, err := suite.m.GetExportedURLs(serviceMi)
	assert.Equal(suite.t, 1, len(urls))
	assert.NoError(suite.t, err)
}

func (suite *zookeeperMetadataReportTestSuite) testSaveSubscribedData(url *common.URL) {
	subscribeMi := newSubscribeMetadataIdentifier("provider")
	urls := []string{url.String()}
	bytes, _ := json.Marshal(urls)
	err := suite.m.SaveSubscribedData(subscribeMi, string(bytes))
	assert.Nil(suite.t, err)
}

func (suite *zookeeperMetadataReportTestSuite) testGetSubscribedURLs() {
	subscribeMi := newSubscribeMetadataIdentifier("provider")
	urls, err := suite.m.GetSubscribedURLs(subscribeMi)
	assert.Equal(suite.t, 1, len(urls))
	assert.NoError(suite.t, err)
}

func (suite *zookeeperMetadataReportTestSuite) testGetServiceDefinition() {
	providerMi := newMetadataIdentifier("provider")
	providerMeta, err := suite.m.GetServiceDefinition(providerMi)
	assert.Equal(suite.t, "provider", providerMeta)
	assert.NoError(suite.t, err)
}

func test1(t *testing.T) {
	testCluster, err := zk.StartTestCluster(1, nil, nil)
	assert.NoError(t, err)
	defer testCluster.Stop()

	url := newProviderRegistryUrl("127.0.0.1", testCluster.Servers[0].Port)
	mf := extension.GetMetadataReportFactory("zookeeper")
	m := mf.CreateMetadataReport(url)

	suite := newZookeeperMetadataReportTestSuite(t, m)
	suite.testStoreProviderMetadata()
	suite.testStoreConsumerMetadata()
	suite.testSaveServiceMetadata(url)
	suite.testGetExportedURLs()
	suite.testRemoveServiceMetadata()
	suite.testSaveSubscribedData(url)
	suite.testGetSubscribedURLs()
	suite.testGetServiceDefinition()
}

func TestZookeeperMetadataReport(t *testing.T) {
	t.Run("test1", test1)
}
