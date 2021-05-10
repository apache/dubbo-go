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

package nacos

import (
	"net/http"
	"strconv"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/metadata/identifier"
	"dubbo.apache.org/dubbo-go/v3/metadata/report"
)

func TestNacosMetadataReport_CRUD(t *testing.T) {
	if !checkNacosServerAlive() {
		return
	}
	rpt := newTestReport()
	assert.NotNil(t, rpt)

	providerMi := newMetadataIdentifier("server")
	providerMeta := "provider"
	err := rpt.StoreProviderMetadata(providerMi, providerMeta)
	assert.Nil(t, err)

	consumerMi := newMetadataIdentifier("client")
	consumerMeta := "consumer"
	err = rpt.StoreConsumerMetadata(consumerMi, consumerMeta)
	assert.Nil(t, err)

	serviceMi := newServiceMetadataIdentifier()
	serviceUrl, _ := common.NewURL("registry://console.nacos.io:80", common.WithParamsValue(constant.ROLE_KEY, strconv.Itoa(common.PROVIDER)))
	err = rpt.SaveServiceMetadata(serviceMi, serviceUrl)
	assert.Nil(t, err)

	exportedUrls, err := rpt.GetExportedURLs(serviceMi)
	assert.Equal(t, 1, len(exportedUrls))
	assert.Nil(t, err)

	err = rpt.RemoveServiceMetadata(serviceMi)
	assert.Nil(t, err)
}

func newServiceMetadataIdentifier() *identifier.ServiceMetadataIdentifier {
	return &identifier.ServiceMetadataIdentifier{
		Protocol: "nacos",
		Revision: "a",
		BaseMetadataIdentifier: identifier.BaseMetadataIdentifier{
			ServiceInterface: "com.test.MyTest",
			Version:          "1.0.0",
			Group:            "test_group",
			Side:             "service",
		},
	}
}

func newMetadataIdentifier(side string) *identifier.MetadataIdentifier {
	return &identifier.MetadataIdentifier{
		Application: "test",
		BaseMetadataIdentifier: identifier.BaseMetadataIdentifier{
			ServiceInterface: "com.test.MyTest",
			Version:          "1.0.0",
			Group:            "test_group",
			Side:             side,
		},
	}
}

func TestNacosMetadataReportFactory_CreateMetadataReport(t *testing.T) {
	res := newTestReport()
	assert.NotNil(t, res)
}

func newTestReport() report.MetadataReport {
	regurl, _ := common.NewURL("registry://console.nacos.io:80", common.WithParamsValue(constant.ROLE_KEY, strconv.Itoa(common.PROVIDER)))
	res := extension.GetMetadataReportFactory("nacos").CreateMetadataReport(regurl)
	return res
}

func checkNacosServerAlive() bool {
	c := http.Client{Timeout: time.Second}
	if _, err := c.Get("http://console.nacos.io/nacos/"); err != nil {
		return false
	}
	return true
}
