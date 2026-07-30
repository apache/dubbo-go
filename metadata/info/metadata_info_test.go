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
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"sort"
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
	assert.Len(t, metadataInfo.GetExportedServiceURLs(), 1)
	assert.Equal(t, url2, metadataInfo.GetExportedServiceURLs()[0])
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

func TestServiceInfoJavaClassName(t *testing.T) {
	assert.Equalf(t, "org.apache.dubbo.metadata.MetadataInfo", NewAppMetadataInfo("dubbo").JavaClassName(), "JavaClassName()")
}

// referenceCalRevision is an INDEPENDENT, minimal re-implementation of the Java
// dubbo MetadataInfo.calRevision() algorithm. It is used purely as a cross-check
// for the production info.CalRevision and MUST NOT call it.
//
// Java algorithm (per dubbo MetadataInfo.calRevision / ServiceInfo.toDescString):
//
//	calRevision():
//	    sb.append(app)
//	    for each service (TreeMap sorted by key): sb.append(toDescString())
//	    return md5(sb.toString())   // 32-char lowercase hex
//
//	toDescString():
//	    getMatchKey() + port + path + new TreeMap<>(getParams()).toString()
//	    where TreeMap.toString() == "{k1=v1, k2=v2}" (braces, "k=v", ", " separator,
//	    keys in natural ascending order); empty params render as "{}".
//	    Methods are intentionally NOT part of the serialization.
func referenceCalRevision(app string, services map[string]*ServiceInfo) string {
	if len(services) == 0 {
		return "0"
	}

	matchKeys := make([]string, 0, len(services))
	for mk := range services {
		matchKeys = append(matchKeys, mk)
	}
	sort.Strings(matchKeys)

	var b strings.Builder
	b.WriteString(app)
	for _, mk := range matchKeys {
		si := services[mk]
		b.WriteString(si.GetMatchKey())
		b.WriteString(strconv.Itoa(si.Port))
		b.WriteString(si.Path)

		keys := make([]string, 0, len(si.Params))
		for k := range si.Params {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		b.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(k)
			b.WriteByte('=')
			b.WriteString(si.Params[k])
		}
		b.WriteByte('}')
	}

	sum := md5.Sum([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}

// goldenRevision is the 32-char lowercase hex MD5 produced by the Java
// MetadataInfo.calRevision() algorithm for the deterministic MetadataInfo built in
// TestCalRevisionJavaAlignmentGolden. It is hard-coded (independently derived from
// the Java spec, not from the Go implementation) so that any future drift in
// info.CalRevision breaks this regression guard.
//
// Input:
//
//	app = "dubbo-go-app"
//	service #1: name=com.foo.Bar group="" version=1.0.0 protocol=dubbo port=20880
//	           path=com.foo.Bar params={timeout=3000, weight=100}
//	service #2: name=com.foo.Baz group=g1 version=2.0.0 protocol=tri port=50051
//	           path=com.foo.Baz params={cluster=failfast}
//
// Concatenated (app + sorted-by-matchKey toDescString):
//
//	dubbo-go-app
//	+ com.foo.Bar:1.0.0:dubbo20880com.foo.Bar{timeout=3000, weight=100}
//	+ g1/com.foo.Baz:2.0.0:tri50051com.foo.Baz{cluster=failfast}
const goldenRevision = "8ab6351c86cccf642be0a2cda8be847a"

// TestCalRevisionJavaAlignmentGolden verifies that info.CalRevision produces exactly
// the same revision as the Java dubbo implementation for identical service definitions.
func TestCalRevisionJavaAlignmentGolden(t *testing.T) {
	app := "dubbo-go-app"

	s1 := NewServiceInfo("com.foo.Bar", "", "1.0.0", "dubbo", "com.foo.Bar", map[string]string{
		"timeout": "3000",
		"weight":  "100",
	})
	s1.Port = 20880

	s2 := NewServiceInfo("com.foo.Baz", "g1", "2.0.0", "tri", "com.foo.Baz", map[string]string{
		"cluster": "failfast",
	})
	s2.Port = 50051

	services := map[string]*ServiceInfo{
		s1.GetMatchKey(): s1,
		s2.GetMatchKey(): s2,
	}

	// 1. Production CalRevision must equal the independent Java-spec reference.
	got := CalRevision(app, services)
	expected := referenceCalRevision(app, services)
	assert.Equal(t, expected, got, "CalRevision must match the independent Java-spec reference implementation")

	// 2. Output is a 32-char lowercase hex MD5 digest.
	assert.Regexp(t, `^[0-9a-f]{32}$`, got, "revision must be a 32-char lowercase hex MD5")

	// 3. Output matches the hard-coded golden vector (regression guard).
	assert.Equal(t, goldenRevision, got, "revision must equal the Java-derived golden vector")

	// 4. Same input produces a stable, deterministic revision.
	assert.Equal(t, got, CalRevision(app, services), "CalRevision must be deterministic for identical input")
}

// TestCalRevisionEmptyServicesReturnsZero guards the Java EMPTY_REVISION contract:
// an empty service set yields "0".
func TestCalRevisionEmptyServicesReturnsZero(t *testing.T) {
	assert.Equal(t, "0", CalRevision("app", nil))
	assert.Equal(t, "0", CalRevision("app", map[string]*ServiceInfo{}))
}

// TestServiceInfoToDescStringFormat verifies the exact Java toDescString() format
// (matchKey + port + path + TreeMap(params).toString()), independent of methods.
func TestServiceInfoToDescStringFormat(t *testing.T) {
	si := NewServiceInfo("com.foo.Bar", "", "1.0.0", "dubbo", "com.foo.Bar", map[string]string{
		"timeout": "3000",
		"weight":  "100",
	})
	si.Port = 20880
	si.Methods = []string{"sayHello", "sayGoodbye"}

	// methods are NOT part of toDescString; params are rendered as a Java TreeMap.
	assert.Equal(t, "com.foo.Bar:1.0.0:dubbo20880com.foo.Bar{timeout=3000, weight=100}", si.toDescString())
}

// TestServiceInfoToDescStringEmptyParams verifies an empty params map renders as "{}".
func TestServiceInfoToDescStringEmptyParams(t *testing.T) {
	si := NewServiceInfo("com.foo.Bar", "", "1.0.0", "dubbo", "com.foo.Bar", map[string]string{})
	si.Port = 20880
	assert.Equal(t, "com.foo.Bar:1.0.0:dubbo20880com.foo.Bar{}", si.toDescString())
}
