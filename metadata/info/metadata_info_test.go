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

package info

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
)

var (
	serviceUrl = common.NewURLWithOptions(
		common.WithProtocol("tri"),
		common.WithIp("127.0.0.1"),
		common.WithPort("20035"),
		common.WithPath("/org.apache.dubbo.samples.proto.GreetService"),
		common.WithInterface("org.apache.dubbo.samples.proto.GreetService"),
		common.WithMethods([]string{"Greet", "SayHello"}),
		common.WithParamsValue("loadbalance", "random"),
		common.WithParamsValue("methods.Greet.timeout", "1000"),
	)
)

func TestMetadataInfoAddService(t *testing.T) {
	metadataInfo := &MetadataInfo{
		Services:              make(map[string]*ServiceInfo),
		exportedServiceURLs:   make(map[string][]*common.URL),
		subscribedServiceURLs: make(map[string][]*common.URL),
	}

	url, _ := common.NewURL("dubbo://127.0.0.1:20000?application=foo&category=providers&check=false&dubbo=dubbo-go+v1.5.0&interface=com.foo.Bar&methods=GetPetByID%2CGetPetTypes&organization=Apache&owner=foo&revision=1.0.0&side=provider&version=1.0.0")
	metadataInfo.AddService(url)
	assert.NotEmpty(t, metadataInfo.Services)
	assert.NotEmpty(t, metadataInfo.GetExportedServiceURLs())

	metadataInfo.RemoveService(url)
	assert.Empty(t, metadataInfo.Services)
	assert.Empty(t, metadataInfo.GetExportedServiceURLs())
}

func TestHessian(t *testing.T) {
	metadataInfo := &MetadataInfo{
		App:                   "test",
		Revision:              "1",
		Services:              make(map[string]*ServiceInfo),
		exportedServiceURLs:   make(map[string][]*common.URL),
		subscribedServiceURLs: make(map[string][]*common.URL),
	}
	metadataInfo.Services["1"] = NewServiceInfo("dubbo.io", "default", "1.0.0", "dubbo", "", make(map[string]string))
	e := hessian.NewEncoder()
	err := e.Encode(metadataInfo)
	require.NoError(t, err)
	obj, err := hessian.NewDecoder(e.Buffer()).Decode()
	require.NoError(t, err)
	objJson, _ := json.Marshal(obj)
	metaJson, _ := json.Marshal(metadataInfo)
	assert.JSONEq(t, string(metaJson), string(objJson))
}

func TestMetadataInfoAddSubscribeURL(t *testing.T) {
	info := NewMetadataInfo("dubbo", "tag")
	info.AddSubscribeURL(serviceUrl)
	assert.NotEmpty(t, info.GetSubscribedURLs())
	info.RemoveSubscribeURL(serviceUrl)
	assert.Empty(t, info.GetSubscribedURLs())
}

func TestMetadataInfoCalAndGetRevision(t *testing.T) {
	metadata := NewAppMetadataInfo("dubbo")
	assert.Equalf(t, "0", metadata.CalAndGetRevision(), "CalAndGetRevision()")
	metadata.AddService(serviceUrl)
	assert.NotEqual(t, "0", metadata.CalAndGetRevision())

	v := metadata.Revision
	assert.Equal(t, v, metadata.CalAndGetRevision(), "CalAndGetRevision() test cache")

	metadata = NewAppMetadataInfo("dubbo")
	url1 := serviceUrl.Clone()
	url1.Methods = []string{}
	metadata.AddService(url1)
	assert.NotEqual(t, "0", metadata.CalAndGetRevision(), "CalAndGetRevision() test empty methods")
}

func TestNewMetadataInfo(t *testing.T) {
	info := NewMetadataInfo("dubbo", "tag")
	assert.Equal(t, "dubbo", info.App)
	assert.Equal(t, "tag", info.Tag)
}

func TestNewMetadataInfoWithParams(t *testing.T) {
	info := NewMetadataInfoWithParams("dubbo", "",
		map[string]*ServiceInfo{"org.apache.dubbo.samples.proto.GreetService": NewServiceInfoWithURL(serviceUrl)})
	assert.Equal(t, "dubbo", info.App)
	assert.Empty(t, info.Revision)
	assert.Equal(t, map[string]*ServiceInfo{"org.apache.dubbo.samples.proto.GreetService": NewServiceInfoWithURL(serviceUrl)}, info.Services)
}

func TestNewServiceInfoWithURL(t *testing.T) {
	info := NewServiceInfoWithURL(serviceUrl)
	assert.Same(t, serviceUrl, info.URL)
	assert.Equal(t, info.Protocol, serviceUrl.Protocol)
	assert.Equal(t, info.Name, serviceUrl.Interface())
	assert.Equal(t, info.Group, serviceUrl.Group())
	assert.Equal(t, info.Version, serviceUrl.Version())
	assert.Equal(t, strconv.Itoa(info.Port), serviceUrl.Port)
	assert.Equal(t, info.Path, strings.TrimPrefix(serviceUrl.Path, "/"))
	assert.Equal(t, "1000", info.Params["Greet.timeout"])
}

func TestServiceInfoGetMethods(t *testing.T) {
	service := NewServiceInfoWithURL(serviceUrl)
	assert.Equal(t, []string{"Greet", "SayHello"}, service.GetMethods())
}

func TestServiceInfoGetParams(t *testing.T) {
	service := NewServiceInfoWithURL(serviceUrl)
	assert.Equal(t, []string{"random"}, service.GetParams()["loadbalance"])
}

func TestServiceInfoGetMatchKey(t *testing.T) {
	si := NewServiceInfoWithURL(serviceUrl)
	matchKey := si.MatchKey
	assert.Equal(t, si.GetMatchKey(), matchKey)
	si.MatchKey = ""
	assert.NotEmpty(t, si.GetMatchKey())
	si.MatchKey = ""
	si.ServiceKey = ""
	assert.NotEmpty(t, si.GetMatchKey())
}

func TestServiceInfoJavaClassName(t *testing.T) {
	assert.Equalf(t, "org.apache.dubbo.metadata.MetadataInfo", NewAppMetadataInfo("dubbo").JavaClassName(), "JavaClassName()")
}
