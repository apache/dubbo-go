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
	"strconv"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	_ "dubbo.apache.org/dubbo-go/v3/proxy/proxy_factory"
)

var testURL, _ = common.NewURL("dubbo://127.0.0.1:20000?application=foo&interface=com.foo.Bar&methods=GetPetByID&version=1.0.0")

func newTestMetadataMap() map[string]*info.MetadataInfo {
	metadataInfo := info.NewAppMetadataInfo("dubbo-app")
	metadataInfo.Revision = "1"
	metadataInfo.AddService(testURL)
	metadataInfo.AddSubscribeURL(testURL)
	registryMetadataInfo["default"] = metadataInfo
	return map[string]*info.MetadataInfo{"default": metadataInfo}
}

func TestDefaultMetadataServiceGetExportedServiceURLs(t *testing.T) {
	mts := &DefaultMetadataService{metadataMap: newTestMetadataMap()}
	got, err := mts.GetExportedServiceURLs()
	assert.Nil(t, err)
	assert.Equal(t, 1, len(got))
	assert.Equal(t, testURL, got[0])
}

func TestDefaultMetadataServiceGetExportedURLs(t *testing.T) {
	mts := &DefaultMetadataService{metadataMap: newTestMetadataMap()}

	tests := []struct {
		iface, group, version, protocol string
	}{
		{testURL.Interface(), testURL.Group(), testURL.Version(), testURL.Protocol},
		{"*", "*", "*", "*"},
	}

	for _, tt := range tests {
		got, err := mts.GetExportedURLs(tt.iface, tt.group, tt.version, tt.protocol)
		assert.Nil(t, err)
		assert.Equal(t, []*common.URL{testURL}, got)
	}
}

func TestDefaultMetadataServiceGetMetadataInfo(t *testing.T) {
	mts := &DefaultMetadataService{metadataMap: newTestMetadataMap()}

	got, err := mts.GetMetadataInfo("1")
	assert.Nil(t, err)
	assert.NotNil(t, got)

	got, err = mts.GetMetadataInfo("")
	assert.Nil(t, err)
	assert.Nil(t, got)

	got, err = mts.GetMetadataInfo("wrong-revision")
	assert.Nil(t, err)
	assert.Nil(t, got)
}

func TestDefaultMetadataServiceGetMetadataServiceURL(t *testing.T) {
	mts := &DefaultMetadataService{metadataUrl: testURL}
	got, err := mts.GetMetadataServiceURL()
	assert.Nil(t, err)
	assert.Equal(t, testURL, got)
}

func TestDefaultMetadataServiceGetSubscribedURLs(t *testing.T) {
	mts := &DefaultMetadataService{metadataMap: newTestMetadataMap()}
	got, err := mts.GetSubscribedURLs()
	assert.Nil(t, err)
	assert.Equal(t, []*common.URL{testURL}, got)
}

func TestDefaultMetadataServiceMethodMapper(t *testing.T) {
	mts := &DefaultMetadataService{}
	mapper := mts.MethodMapper()
	assert.Equal(t, "getExportedURLs", mapper["GetExportedURLs"])
	assert.Equal(t, "getMetadataInfo", mapper["GetMetadataInfo"])
}

func TestDefaultMetadataServiceVersion(t *testing.T) {
	mts := &DefaultMetadataService{}
	got, err := mts.Version()
	assert.Nil(t, err)
	assert.Equal(t, version, got)
}

func TestServiceExporterExport(t *testing.T) {
	mockExporter := new(mockExporter)
	dubboProtocol := new(mockProtocol)
	extension.SetProtocol("dubbo", func() base.Protocol { return dubboProtocol })

	port, _ := strconv.Atoi(common.GetRandomPort(""))
	opts := &Options{
		appName:      "dubbo-app",
		metadataType: constant.RemoteMetadataStorageType,
		protocol:     constant.DubboProtocol,
		port:         port,
	}

	dubboProtocol.On("Export").Return(mockExporter).Once()
	e := &serviceExporter{opts: opts, service: &DefaultMetadataService{}}
	err := e.Export()
	assert.Nil(t, err)
}
