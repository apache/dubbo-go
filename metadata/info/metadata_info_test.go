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
	"sync"
	"testing"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
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

func TestMetadataInfoAddServiceBackfillsApplicationTag(t *testing.T) {
	metadataInfo := &MetadataInfo{
		Services:              make(map[string]*ServiceInfo),
		exportedServiceURLs:   make(map[string][]*common.URL),
		subscribedServiceURLs: make(map[string][]*common.URL),
	}
	url, err := common.NewURL("tri://127.0.0.1:20000?application=foo&application.tag=gray&interface=com.foo.Bar&methods=GetPetByID")
	require.NoError(t, err)

	metadataInfo.AddService(url)

	assert.Equal(t, "foo", metadataInfo.App)
	assert.Equal(t, "gray", metadataInfo.Tag)
}

func TestMetadataInfoCopiesExportedServiceURLs(t *testing.T) {
	metadataInfo := NewMetadataInfo("foo", "")
	url := serviceUrl.Clone()
	expectedPath := url.Path
	expectedMethods := append([]string(nil), url.Methods...)
	url.SubURL = common.NewURLWithOptions(common.WithPath("nested"))
	expectedSubURLPath := url.SubURL.Path
	sharedAttribute := &struct{ Value string }{Value: "shared"}
	url.SetParam("snapshot-param", "before")
	url.SetAttribute("snapshot-attribute", "before")
	url.SetAttribute("shared-attribute", sharedAttribute)

	metadataInfo.AddService(url)
	url.Path = "/caller-mutated"
	url.Methods[0] = "caller-mutated"
	url.SubURL.Path = "/caller-mutated"
	url.SetParam("snapshot-param", "caller-mutated")
	url.SetAttribute("snapshot-attribute", "caller-mutated")

	urls := metadataInfo.GetExportedServiceURLs()
	require.Len(t, urls, 1)
	assert.NotSame(t, url, urls[0])
	assert.Equal(t, expectedPath, urls[0].Path)
	assert.Equal(t, expectedMethods, urls[0].Methods)
	require.NotNil(t, urls[0].SubURL)
	assert.NotSame(t, url.SubURL, urls[0].SubURL)
	assert.Equal(t, expectedSubURLPath, urls[0].SubURL.Path)
	assert.Equal(t, "before", urls[0].GetParam("snapshot-param", ""))
	attribute, ok := urls[0].GetAttribute("snapshot-attribute")
	require.True(t, ok)
	assert.Equal(t, "before", attribute)
	attribute, ok = urls[0].GetAttribute("shared-attribute")
	require.True(t, ok)
	assert.Same(t, sharedAttribute, attribute)

	urls[0].Path = "/getter-mutated"
	urls[0].Methods[0] = "getter-mutated"
	urls[0].SubURL.Path = "/getter-mutated"
	urls[0].SetParam("snapshot-param", "getter-mutated")
	urls[0].SetAttribute("snapshot-attribute", "getter-mutated")

	freshURLs := metadataInfo.GetExportedServiceURLs()
	require.Len(t, freshURLs, 1)
	assert.NotSame(t, urls[0], freshURLs[0])
	assert.Equal(t, expectedPath, freshURLs[0].Path)
	assert.Equal(t, expectedMethods, freshURLs[0].Methods)
	require.NotNil(t, freshURLs[0].SubURL)
	assert.NotSame(t, urls[0].SubURL, freshURLs[0].SubURL)
	assert.Equal(t, expectedSubURLPath, freshURLs[0].SubURL.Path)
	assert.Equal(t, "before", freshURLs[0].GetParam("snapshot-param", ""))
	attribute, ok = freshURLs[0].GetAttribute("snapshot-attribute")
	require.True(t, ok)
	assert.Equal(t, "before", attribute)
}

func TestMetadataInfoReplaceExportedServicesCopiesURLs(t *testing.T) {
	metadataInfo := NewMetadataInfo("foo", "")
	url := serviceUrl.Clone()
	expectedPath := url.Path
	url.SetParam("snapshot-param", "before")

	metadataInfo.ReplaceExportedServices([]*common.URL{url})
	url.Path = "/caller-mutated"
	url.SetParam("snapshot-param", "caller-mutated")

	urls := metadataInfo.GetExportedServiceURLs()
	require.Len(t, urls, 1)
	assert.NotSame(t, url, urls[0])
	assert.Equal(t, expectedPath, urls[0].Path)
	assert.Equal(t, "before", urls[0].GetParam("snapshot-param", ""))

	service := requireOnlyService(t, metadataInfo.GetServices())
	assert.NotSame(t, url, service.URL)
	assert.Equal(t, expectedPath, service.URL.Path)
	assert.Equal(t, "before", service.URL.GetParam("snapshot-param", ""))
}

func TestMetadataInfoRemoveServiceWithClonedURL(t *testing.T) {
	metadataInfo := NewMetadataInfo("foo", "")
	url, err := common.NewURL("dubbo://127.0.0.1:20000?application=foo&interface=com.foo.Bar&methods=GetPetByID%2CGetPetTypes&side=provider&version=1.0.0")
	require.NoError(t, err)

	metadataInfo.AddService(url)
	metadataInfo.RemoveService(url.Clone())

	assert.Empty(t, metadataInfo.Services)
	assert.Empty(t, metadataInfo.GetExportedServiceURLs())
}

func TestMetadataInfoRemoveServiceKeepsRemainingMatchKeyService(t *testing.T) {
	metadataInfo := NewMetadataInfo("foo", "")
	url1, err := common.NewURL("dubbo://127.0.0.1:20000?application=foo&interface=com.foo.Bar&methods=GetPetByID&side=provider&version=1.0.0")
	require.NoError(t, err)
	url2, err := common.NewURL("dubbo://127.0.0.1:20001?application=foo&interface=com.foo.Bar&methods=GetPetByID&side=provider&version=1.0.0")
	require.NoError(t, err)

	metadataInfo.AddService(url1)
	metadataInfo.AddService(url2)
	metadataInfo.RemoveService(url1.Clone())

	require.Len(t, metadataInfo.Services, 1)
	remainingURLs := metadataInfo.GetExportedServiceURLs()
	require.Len(t, remainingURLs, 1)
	assert.True(t, url2.URLEqual(remainingURLs[0]))
	assert.NotSame(t, url2, remainingURLs[0])
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

func TestMetadataInfoCopiesSubscribedURLs(t *testing.T) {
	metadataInfo := NewMetadataInfo("foo", "")
	url := serviceUrl.Clone()
	expectedPath := url.Path
	url.SetParam("snapshot-param", "before")

	metadataInfo.AddSubscribeURL(url)
	url.Path = "/caller-mutated"
	url.SetParam("snapshot-param", "caller-mutated")

	urls := metadataInfo.GetSubscribedURLs()
	require.Len(t, urls, 1)
	assert.NotSame(t, url, urls[0])
	assert.Equal(t, expectedPath, urls[0].Path)
	assert.Equal(t, "before", urls[0].GetParam("snapshot-param", ""))

	urls[0].Path = "/getter-mutated"
	urls[0].SetParam("snapshot-param", "getter-mutated")

	freshURLs := metadataInfo.GetSubscribedURLs()
	require.Len(t, freshURLs, 1)
	assert.NotSame(t, urls[0], freshURLs[0])
	assert.Equal(t, expectedPath, freshURLs[0].Path)
	assert.Equal(t, "before", freshURLs[0].GetParam("snapshot-param", ""))
}

func TestMetadataInfoRemoveSubscribeURLWithClonedURL(t *testing.T) {
	info := NewMetadataInfo("dubbo", "tag")
	info.AddSubscribeURL(serviceUrl)
	info.RemoveSubscribeURL(serviceUrl.Clone())
	assert.Empty(t, info.GetSubscribedURLs())
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

func TestServiceInfoExcludesInstanceLevelParams(t *testing.T) {
	serviceURL, err := common.NewURL("tri://127.0.0.1:20000/org.apache.dubbo.samples.proto.GreetService",
		common.WithInterface("org.apache.dubbo.samples.proto.GreetService"),
		common.WithParamsValue(constant.EnvironmentKey, "pre"),
		common.WithMethods([]string{"Greet"}),
	)
	require.NoError(t, err)

	service := NewServiceInfoWithURL(serviceURL)

	// Environment is instance-level metadata, not service-level.
	// It should NOT appear in ServiceInfo.Params and thus not affect revision.
	assert.Empty(t, service.GetParams()[constant.EnvironmentKey])
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

func TestMetadataInfoGetServices(t *testing.T) {
	metadataInfo := &MetadataInfo{
		Services:              make(map[string]*ServiceInfo),
		exportedServiceURLs:   make(map[string][]*common.URL),
		subscribedServiceURLs: make(map[string][]*common.URL),
	}
	url, _ := common.NewURL("dubbo://127.0.0.1:20000?application=foo&category=providers&check=false&dubbo=dubbo-go+v1.5.0&interface=com.foo.Bar&methods=GetPetByID%2CGetPetTypes&organization=Apache&owner=foo&revision=1.0.0&side=provider&version=1.0.0")
	metadataInfo.AddService(url)

	services := metadataInfo.GetServices()
	require.Len(t, services, 1)
	assert.NotEmpty(t, services)

	// GetServices returns a copy: modifying the original does not affect the snapshot
	metadataInfo.RemoveService(url)
	assert.Len(t, services, 1)

	// A fresh call reflects the removal
	assert.Empty(t, metadataInfo.GetServices())
}

func TestMetadataInfoServiceSnapshotsCopyURL(t *testing.T) {
	metadataInfo := NewMetadataInfo("foo", "")
	url := serviceUrl.Clone()
	expectedPath := url.Path
	metadataInfo.AddService(url)
	url.Path = "/caller-mutated"

	service := requireOnlyService(t, metadataInfo.GetServices())
	assert.NotSame(t, url, service.URL)
	assert.Equal(t, expectedPath, service.URL.Path)
	service.URL.Path = "/getter-mutated"
	service.Params["caller-mutation"] = "getter"

	freshService := requireOnlyService(t, metadataInfo.GetServices())
	assert.Equal(t, expectedPath, freshService.URL.Path)
	assert.NotContains(t, freshService.Params, "caller-mutation")

	snapshotService := requireOnlyService(t, metadataInfo.Snapshot().Services)
	snapshotService.URL.Path = "/snapshot-mutated"
	snapshotService.Params["caller-mutation"] = "snapshot"

	freshService = requireOnlyService(t, metadataInfo.GetServices())
	assert.Equal(t, expectedPath, freshService.URL.Path)
	assert.NotContains(t, freshService.Params, "caller-mutation")
}

func TestMetadataInfoSnapshotDoesNotInitializeSourceLazyFields(t *testing.T) {
	metadataInfo := NewMetadataInfo("foo", "")
	service := &ServiceInfo{
		Name:     "com.foo.Bar",
		Group:    "test",
		Version:  "1.0.0",
		Protocol: "tri",
		Params:   map[string]string{"timeout": "1000"},
	}
	metadataInfo.Services["service"] = service

	snapshotService := metadataInfo.Snapshot().Services["service"]

	assert.Empty(t, service.ServiceKey)
	assert.Empty(t, service.MatchKey)
	assert.Equal(t, common.ServiceKey(service.Name, service.Group, service.Version), snapshotService.ServiceKey)
	assert.Equal(t, common.MatchKey(snapshotService.ServiceKey, service.Protocol), snapshotService.MatchKey)
	assert.Nil(t, snapshotService.URL)
	snapshotService.Params["timeout"] = "2000"
	assert.Equal(t, "1000", service.Params["timeout"])
}

func TestMetadataInfoConcurrentURLSnapshotMutation(t *testing.T) {
	metadataInfo := NewMetadataInfo("foo", "")
	metadataInfo.AddService(serviceUrl)
	metadataInfo.AddSubscribeURL(serviceUrl)

	const iterations = 100
	var waitGroup sync.WaitGroup
	waitGroup.Add(2)
	go func() {
		defer waitGroup.Done()
		for i := range iterations {
			replacement := testServiceURL("replacement." + strconv.Itoa(i))
			metadataInfo.ReplaceExportedServices([]*common.URL{replacement})

			transient := testServiceURL("transient." + strconv.Itoa(i))
			metadataInfo.AddService(transient)
			metadataInfo.RemoveService(transient.Clone())

			subscription := testServiceURL("subscription." + strconv.Itoa(i))
			metadataInfo.AddSubscribeURL(subscription)
			metadataInfo.RemoveSubscribeURL(subscription.Clone())
		}
	}()
	go func() {
		defer waitGroup.Done()
		for i := range iterations {
			mutateURLSnapshots(metadataInfo.GetExportedServiceURLs(), i)
			mutateURLSnapshots(metadataInfo.GetSubscribedURLs(), i)
			mutateServiceSnapshots(metadataInfo.GetServices(), i)
			mutateServiceSnapshots(metadataInfo.Snapshot().Services, i)
		}
	}()
	waitGroup.Wait()

	exportedURLs := metadataInfo.GetExportedServiceURLs()
	require.Len(t, exportedURLs, 1)
	assert.Equal(t, "replacement.99", exportedURLs[0].Interface())
	require.Len(t, metadataInfo.GetSubscribedURLs(), 1)
}

func testServiceURL(service string) *common.URL {
	url := serviceUrl.Clone()
	url.Path = "/" + service
	url.SetParam(constant.InterfaceKey, service)
	return url
}

func mutateURLSnapshots(urls []*common.URL, iteration int) {
	for _, url := range urls {
		url.Path = "/snapshot-" + strconv.Itoa(iteration)
		url.SetParam("snapshot-param", strconv.Itoa(iteration))
		url.SetAttribute("snapshot-attribute", iteration)
		if len(url.Methods) > 0 {
			url.Methods[0] = "snapshot-mutated"
		}
		if url.SubURL != nil {
			url.SubURL.Path = "/snapshot-mutated"
		}
	}
}

func mutateServiceSnapshots(services map[string]*ServiceInfo, iteration int) {
	for _, service := range services {
		service.Params["snapshot-param"] = strconv.Itoa(iteration)
		if service.URL != nil {
			mutateURLSnapshots([]*common.URL{service.URL}, iteration)
		}
	}
}

func requireOnlyService(t *testing.T, services map[string]*ServiceInfo) *ServiceInfo {
	t.Helper()
	require.Len(t, services, 1)
	var onlyService *ServiceInfo
	for _, service := range services {
		onlyService = service
	}
	return onlyService
}

func TestServiceInfoJavaClassName(t *testing.T) {
	assert.Equalf(t, "org.apache.dubbo.metadata.MetadataInfo", NewAppMetadataInfo("dubbo").JavaClassName(), "JavaClassName()")
}
