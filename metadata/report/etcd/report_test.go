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

package etcd

import (
	"encoding/json"
	"net/url"
	"strconv"
	"testing"
)

import (
	"github.com/coreos/etcd/embed"
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/metadata/identifier"
)

const defaultEtcdV3WorkDir = "/tmp/default-dubbo-go-registry.etcd"

func initEtcd(t *testing.T) *embed.Etcd {
	DefaultListenPeerURLs := "http://localhost:2380"
	DefaultListenClientURLs := "http://localhost:2379"
	lpurl, _ := url.Parse(DefaultListenPeerURLs)
	lcurl, _ := url.Parse(DefaultListenClientURLs)
	cfg := embed.NewConfig()
	cfg.LPUrls = []url.URL{*lpurl}
	cfg.LCUrls = []url.URL{*lcurl}
	cfg.Dir = defaultEtcdV3WorkDir
	e, err := embed.StartEtcd(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func TestEtcdMetadataReportFactory_CreateMetadataReport(t *testing.T) {
	e := initEtcd(t)
	url, err := common.NewURL("registry://127.0.0.1:2379", common.WithParamsValue(constant.ROLE_KEY, strconv.Itoa(common.PROVIDER)))
	if err != nil {
		t.Fatal(err)
	}
	metadataReportFactory := &etcdMetadataReportFactory{}
	metadataReport := metadataReportFactory.CreateMetadataReport(url)
	assert.NotNil(t, metadataReport)
	e.Close()
}

func TestEtcdMetadataReport_CRUD(t *testing.T) {
	e := initEtcd(t)
	url, err := common.NewURL("registry://127.0.0.1:2379", common.WithParamsValue(constant.ROLE_KEY, strconv.Itoa(common.PROVIDER)))
	if err != nil {
		t.Fatal(err)
	}
	metadataReportFactory := &etcdMetadataReportFactory{}
	metadataReport := metadataReportFactory.CreateMetadataReport(url)
	assert.NotNil(t, metadataReport)

	err = metadataReport.StoreConsumerMetadata(newMetadataIdentifier("consumer"), "consumer metadata")
	assert.Nil(t, err)

	err = metadataReport.StoreProviderMetadata(newMetadataIdentifier("provider"), "provider metadata")
	assert.Nil(t, err)

	serviceMi := newServiceMetadataIdentifier()
	serviceUrl, _ := common.NewURL("registry://localhost:8848", common.WithParamsValue(constant.ROLE_KEY, strconv.Itoa(common.PROVIDER)))
	metadataReport.SaveServiceMetadata(serviceMi, serviceUrl)
	assert.Nil(t, err)

	subMi := newSubscribeMetadataIdentifier()
	urlList := make([]string, 0, 1)
	urlList = append(urlList, serviceUrl.String())
	urls, _ := json.Marshal(urlList)
	err = metadataReport.SaveSubscribedData(subMi, string(urls))
	assert.Nil(t, err)

	err = metadataReport.RemoveServiceMetadata(serviceMi)
	assert.Nil(t, err)

	e.Close()
}

func newSubscribeMetadataIdentifier() *identifier.SubscriberMetadataIdentifier {
	return &identifier.SubscriberMetadataIdentifier{
		Revision:           "subscribe",
		MetadataIdentifier: *newMetadataIdentifier("provider"),
	}

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
