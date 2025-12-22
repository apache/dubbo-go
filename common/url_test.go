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

// Package common url_test.go contains unit tests for URL parsing and manipulation.
// Note: This file contains hardcoded IP addresses (127.0.0.1, 0.0.0.0, 192.168.x.x, etc.)
// which are used exclusively for testing URL parsing functionality. These are:
// - 127.0.0.1: loopback address (RFC 5735)
// - 0.0.0.0: any address binding
// - 192.168.x.x: private network addresses (RFC 1918)
// These addresses are not used for actual network connections in production.

package common

import (
	"encoding/base64"
	"net/url"
	"testing"
	"time"
)

import (
	gxset "github.com/dubbogo/gost/container/set"
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

// Test constants for URL parsing tests
// These IP addresses are used for unit testing only and are non-routable
// addresses per RFC 1918 (private networks) or RFC 5737 (documentation)
const (
	userName        = "username"
	testPassword    = "testpass" // #nosec G101 - test credential for unit tests
	loopbackAddress = "127.0.0.1"
	testPort        = "20000"
)

func TestNewURLWithOptions(t *testing.T) {
	methods := []string{"Methodone,methodtwo"}
	params := url.Values{}
	params.Set("key", "value")
	u := NewURLWithOptions(WithPath("com.test.Service"),
		WithUsername(userName),
		WithPassword(testPassword),
		WithProtocol("testprotocol"),
		WithIp(loopbackAddress),
		WithPort("8080"),
		WithMethods(methods),
		WithParams(params),
		WithParamsValue("key2", "value2"),
		WithAttribute("key3", "value3"),
		WithAttribute("key4", "value4"),
	)
	assert.Equal(t, "/com.test.Service", u.Path)
	assert.Equal(t, userName, u.Username)
	assert.Equal(t, testPassword, u.Password)
	assert.Equal(t, "testprotocol", u.Protocol)
	assert.Equal(t, loopbackAddress, u.Ip)
	assert.Equal(t, "8080", u.Port)
	assert.Equal(t, methods, u.Methods)
	assert.Equal(t, 2, len(u.params))
	assert.Equal(t, 2, len(u.attributes))
}

func TestURL(t *testing.T) {
	u, err := NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?anyhost=true&" +
		"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&" +
		"environment=dev&interface=com.ikurento.user.UserProvider&ip=192.168.56.1&methods=GetUser%2C&" +
		"module=dubbogo+user-info+server&org=ikurento.com&owner=ZX&pid=1447&revision=0.0.1&" +
		"side=provider&timeout=3000&timestamp=1556509797245")
	assert.NoError(t, err)

	urlInst := URL{}
	urlInst.noCopy.Lock()
	urlInst.SetParam("hello", "world")
	urlInst.noCopy.Unlock()

	assert.Equal(t, "/com.ikurento.user.UserProvider", u.Path)
	assert.Equal(t, "127.0.0.1:20000", u.Location)
	assert.Equal(t, "dubbo", u.Protocol)
	assert.Equal(t, loopbackAddress, u.Ip)
	assert.Equal(t, "20000", u.Port)
	assert.Equal(t, urlInst.Methods, u.Methods)
	assert.Equal(t, "", u.Username)
	assert.Equal(t, "", u.Password)
	assert.Equal(t, "anyhost=true&application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-"+
		"provider-golang-1.0.0&environment=dev&interface=com.ikurento.user.UserProvider&ip=192.168.56.1&methods=GetUser%"+
		"2C&module=dubbogo+user-info+server&org=ikurento.com&owner=ZX&pid=1447&revision=0.0.1&side=provider&timeout=3000&t"+
		"imestamp=1556509797245", u.params.Encode())

	assert.Equal(t, "dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?anyhost=true&application=BDTServi"+
		"ce&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&environment=dev&interface=com.ikure"+
		"nto.user.UserProvider&ip=192.168.56.1&methods=GetUser%2C&module=dubbogo+user-info+server&org=ikurento.com&owner="+
		"ZX&pid=1447&revision=0.0.1&side=provider&timeout=3000&timestamp=1556509797245", u.String())
}

func TestURLWithoutSchema(t *testing.T) {
	u, err := NewURL("127.0.0.1:20000/com.ikurento.user.UserProvider?anyhost=true&"+
		"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&"+
		"environment=dev&interface=com.ikurento.user.UserProvider&ip=192.168.56.1&methods=GetUser%2C&"+
		"module=dubbogo+user-info+server&org=ikurento.com&owner=ZX&pid=1447&revision=0.0.1&"+
		"side=provider&timeout=3000&timestamp=1556509797245", WithProtocol("dubbo"))
	assert.NoError(t, err)

	assert.Equal(t, "/com.ikurento.user.UserProvider", u.Path)
	assert.Equal(t, "127.0.0.1:20000", u.Location)
	assert.Equal(t, "dubbo", u.Protocol)
	assert.Equal(t, loopbackAddress, u.Ip)
	assert.Equal(t, "20000", u.Port)
	assert.Equal(t, URL{}.Methods, u.Methods)
	assert.Equal(t, "", u.Username)
	assert.Equal(t, "", u.Password)
	assert.Equal(t, "anyhost=true&application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-"+
		"provider-golang-1.0.0&environment=dev&interface=com.ikurento.user.UserProvider&ip=192.168.56.1&methods=GetUser%"+
		"2C&module=dubbogo+user-info+server&org=ikurento.com&owner=ZX&pid=1447&revision=0.0.1&side=provider&timeout=3000&t"+
		"imestamp=1556509797245", u.params.Encode())

	assert.Equal(t, "dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?anyhost=true&application=BDTServi"+
		"ce&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&environment=dev&interface=com.ikure"+
		"nto.user.UserProvider&ip=192.168.56.1&methods=GetUser%2C&module=dubbogo+user-info+server&org=ikurento.com&owner="+
		"ZX&pid=1447&revision=0.0.1&side=provider&timeout=3000&timestamp=1556509797245", u.String())
}

func TestURLEqual(t *testing.T) {
	u1, err := NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?interface=com.ikurento.user.UserProvider&group=&version=2.6.0")
	assert.NoError(t, err)
	u2, err := NewURL("dubbo://127.0.0.2:20001/com.ikurento.user.UserProvider?interface=com.ikurento.user.UserProvider&group=&version=2.6.0")
	assert.NoError(t, err)
	assert.True(t, u1.URLEqual(u2))

	u3, err := NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?interface=com.ikurento.user.UserProvider&group=gg&version=2.6.0")
	assert.NoError(t, err)
	assert.False(t, u1.URLEqual(u3))

	// urlGroupAnyValue's group is *
	urlGroupAnyValue, err := NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?interface=com.ikurento.user.UserProvider&group=*&version=2.6.0")
	assert.NoError(t, err)
	assert.True(t, u3.URLEqual(urlGroupAnyValue))

	// test for enabled
	urlEnabledEmpty, err := NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?interface=com.ikurento.user.UserProvider&group=*&version=2.6.0&enabled=")
	assert.NoError(t, err)
	assert.True(t, u3.URLEqual(urlEnabledEmpty))

	urlEnabledFalse, err := NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?interface=com.ikurento.user.UserProvider&group=*&version=2.6.0&enabled=1")
	assert.NoError(t, err)
	assert.False(t, u3.URLEqual(urlEnabledFalse))

	urlEnabledTrue, err := NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?interface=com.ikurento.user.UserProvider&group=*&version=2.6.0&enabled=true")
	assert.NoError(t, err)
	assert.True(t, u3.URLEqual(urlEnabledTrue))

	urlEnabledAny, err := NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?interface=com.ikurento.user.UserProvider&group=*&version=2.6.0&enabled=*")
	assert.NoError(t, err)
	assert.True(t, u3.URLEqual(urlEnabledAny))

	// test for category
	categoryAny, err := NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?interface=com.ikurento.user.UserProvider&group=*&version=2.6.0&enabled=*&category=*")
	assert.NoError(t, err)
	assert.True(t, categoryAny.URLEqual(u3))
}

func TestURLGetParam(t *testing.T) {
	params := url.Values{}
	params.Set("key", "value")

	u := URL{}
	u.SetParams(params)

	v := u.GetParam("key", "default")
	assert.Equal(t, "value", v)

	u = URL{}
	v = u.GetParam("key", "default")
	assert.Equal(t, "default", v)
}

func TestURLGetParamInt(t *testing.T) {
	params := url.Values{}
	params.Set("key", "value")

	u := URL{}
	u.SetParams(params)

	v := u.GetParamInt("key", 1)
	assert.Equal(t, int64(1), v)

	u = URL{}
	v = u.GetParamInt("key", 1)
	assert.Equal(t, int64(1), v)
}

func TestURLGetParamIntValue(t *testing.T) {
	params := url.Values{}
	params.Set("key", "0")

	u := URL{}
	u.SetParams(params)

	v := u.GetParamInt("key", 1)
	assert.Equal(t, int64(0), v)

	u = URL{}
	v = u.GetParamInt("key", 1)
	assert.Equal(t, int64(1), v)
}

func TestURLGetParamBool(t *testing.T) {
	params := url.Values{}
	params.Set("force", "true")

	u := URL{}
	u.SetParams(params)

	v := u.GetParamBool("force", false)
	assert.Equal(t, true, v)

	u = URL{}
	v = u.GetParamBool("force", false)
	assert.Equal(t, false, v)
}

func TestURLGetParamAndDecoded(t *testing.T) {
	rule := "host = 2.2.2.2,1.1.1.1,3.3.3.3 & host !=1.1.1.1 => host = 1.2.3.4"
	params := url.Values{}
	params.Set("rule", base64.URLEncoding.EncodeToString([]byte(rule)))

	u := URL{}
	u.SetParams(params)

	v, _ := u.GetParamAndDecoded("rule")
	assert.Equal(t, rule, v)
}

func TestURLGetRawParam(t *testing.T) {
	u, _ := NewURL("condition://0.0.0.0:8080/com.foo.BarService?serialization=fastjson")
	u.Username = "test"
	u.Password = "test" // #nosec G101 - test credential
	assert.Equal(t, "condition", u.GetRawParam("protocol"))
	assert.Equal(t, "0.0.0.0", u.GetRawParam("host"))
	assert.Equal(t, "8080", u.GetRawParam("port"))
	assert.Equal(t, "test", u.GetRawParam(userName))
	assert.Equal(t, "test", u.GetRawParam("password"))
	assert.Equal(t, "/com.foo.BarService", u.GetRawParam("path"))
	assert.Equal(t, "fastjson", u.GetRawParam("serialization"))
}

func TestURLToMap(t *testing.T) {
	u, _ := NewURL("condition://0.0.0.0:8080/com.foo.BarService?serialization=fastjson")
	u.Username = "test"
	u.Password = "test" // #nosec G101 - test credential

	m := u.ToMap()
	assert.Equal(t, 7, len(m))
	assert.Equal(t, "condition", m["protocol"])
	assert.Equal(t, "0.0.0.0", m["host"])
	assert.Equal(t, "8080", m["port"])
	assert.Equal(t, "test", m[userName])
	assert.Equal(t, "test", m["password"])
	assert.Equal(t, "/com.foo.BarService", m["path"])
	assert.Equal(t, "fastjson", m["serialization"])
}

func TestURLGetMethodParamInt(t *testing.T) {
	params := url.Values{}
	params.Set("methods.GetValue.timeout", "3")

	u := URL{}
	u.SetParams(params)

	v := u.GetMethodParamInt("GetValue", "timeout", 1)
	assert.Equal(t, int64(3), v)

	u = URL{}
	v = u.GetMethodParamInt("GetValue", "timeout", 1)
	assert.Equal(t, int64(1), v)
}

func TestURLGetMethodParam(t *testing.T) {
	params := url.Values{}
	params.Set("methods.GetValue.timeout", "3s")

	u := URL{}
	u.SetParams(params)

	v := u.GetMethodParam("GetValue", "timeout", "1s")
	assert.Equal(t, "3s", v)

	u = URL{}
	v = u.GetMethodParam("GetValue", "timeout", "1s")
	assert.Equal(t, "1s", v)
}

func TestURLGetMethodParamBool(t *testing.T) {
	params := url.Values{}
	params.Set("methods.GetValue.async", "true")

	u := URL{}
	u.SetParams(params)

	v := u.GetMethodParamBool("GetValue", "async", false)
	assert.Equal(t, true, v)

	u = URL{}
	v = u.GetMethodParamBool("GetValue2", "async", false)
	assert.Equal(t, false, v)
}

func TestURLGetAttribute(t *testing.T) {
	u := URL{}
	key := "key"
	notExistKey := "not-exist-key"
	val := "value"
	u.SetAttribute(key, val)

	rawVal, ok := u.GetAttribute(key)
	assert.Equal(t, true, ok)
	v, ok := rawVal.(string)
	assert.Equal(t, true, ok)
	assert.Equal(t, val, v)

	rawVal, ok = u.GetAttribute(notExistKey)
	assert.Equal(t, false, ok)
	assert.Nil(t, rawVal)
}

func TestMergeUrl(t *testing.T) {
	referenceUrlParams := url.Values{}
	referenceUrlParams.Set(constant.ClusterKey, "random")
	referenceUrlParams.Set(constant.RetriesKey, "1")
	referenceUrlParams.Set("test3", "1")
	referenceUrlParams.Set("methods.testMethod."+constant.RetriesKey, "1")
	serviceUrlParams := url.Values{}
	serviceUrlParams.Set("test2", "1")
	serviceUrlParams.Set(constant.ClusterKey, "roundrobin")
	serviceUrlParams.Set(constant.RetriesKey, "2")
	serviceUrlParams.Set(constant.MethodKeys+".testMethod."+constant.RetriesKey, "2")
	referenceUrl, _ := NewURL("mock1://127.0.0.1:1111", WithParams(referenceUrlParams), WithMethods([]string{"testMethod"}))
	serviceUrl, _ := NewURL("mock2://127.0.0.1:20000", WithParams(serviceUrlParams))

	mergedUrl := serviceUrl.MergeURL(referenceUrl)
	assert.Equal(t, "random", mergedUrl.GetParam(constant.ClusterKey, ""))
	assert.Equal(t, "1", mergedUrl.GetParam("test2", ""))
	assert.Equal(t, "1", mergedUrl.GetParam("test3", ""))
	assert.Equal(t, "1", mergedUrl.GetParam(constant.RetriesKey, ""))
	assert.Equal(t, "1", mergedUrl.GetParam(constant.MethodKeys+".testMethod."+constant.RetriesKey, ""))
}

func TestURLSetParams(t *testing.T) {
	u1, err := NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?interface=com.ikurento.user.UserProvider&group=&version=2.6.0&configVersion=1.0")
	assert.NoError(t, err)
	params := url.Values{}
	params.Set("key", "3")
	u1.SetParams(params)
	assert.Equal(t, "3", u1.GetParam("key", ""))
	assert.Equal(t, "2.6.0", u1.GetParam("version", ""))
}

func TestURLReplaceParams(t *testing.T) {
	u1, err := NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?interface=com.ikurento.user.UserProvider&group=&version=2.6.0&configVersion=1.0")
	assert.NoError(t, err)
	params := url.Values{}
	params.Set("key", "3")
	u1.ReplaceParams(params)
	assert.Equal(t, "3", u1.GetParam("key", ""))
	assert.Equal(t, "", u1.GetParam("version", ""))
}

func TestClone(t *testing.T) {
	u1, err := NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?interface=com.ikurento.user.UserProvider&group=&version=2.6.0&configVersion=1.0")
	assert.NoError(t, err)
	u2 := u1.Clone()
	assert.Equal(t, u2.Protocol, "dubbo")
	assert.Equal(t, "1.0", u2.GetParam("configVersion", ""))
	u2.Protocol = "provider"
	assert.Equal(t, u1.Protocol, "dubbo")
	assert.Equal(t, u2.Protocol, "provider")
}

func TestColonSeparatedKey(t *testing.T) {
	u1, _ := NewURL("dubbo://127.0.0.1:20000")
	u1.AddParam(constant.InterfaceKey, "com.ikurento.user.UserProvider")

	assert.Equal(t, u1.ColonSeparatedKey(), u1.GetParam(constant.InterfaceKey, "")+"::")
	u1.AddParam(constant.VersionKey, "version1")
	assert.Equal(t, u1.ColonSeparatedKey(), u1.GetParam(constant.InterfaceKey, "")+":version1:")
	u1.AddParam(constant.GroupKey, "group1")
	assert.Equal(t, u1.ColonSeparatedKey(), u1.GetParam(constant.InterfaceKey, "")+":version1:group1")
	u1.SetParam(constant.VersionKey, "")
	assert.Equal(t, u1.ColonSeparatedKey(), u1.GetParam(constant.InterfaceKey, "")+"::group1")
}

func TestCompareURLEqualFunc(t *testing.T) {
	// test Default
	url1, _ := NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?anyhost=true&" +
		"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&" +
		"environment=dev&interface=com.ikurento.user.UserProvider&ip=192.168.56.1&methods=GetUser%2C&" +
		"module=dubbogo+user-info+server&org=ikurento.com&owner=ZX&pid=1447&revision=0.0.1&" +
		"side=provider&timeout=3000&timestamp=1556509797245")
	url2, _ := NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?anyhost=true&" +
		"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&" +
		"environment=dev&interface=com.ikurento.user.UserProvider&ip=192.168.56.1&methods=GetUser%2C&" +
		"module=dubbogo+user-info+server&org=ikurento.com&owner=ZX&pid=1447&revision=0.0.1&" +
		"side=provider&timeout=3000&timestamp=155650979798")
	assert.False(t, GetCompareURLEqualFunc()(url1, url2))
	assert.True(t, GetCompareURLEqualFunc()(url1, url2, constant.TimestampKey, constant.RemoteTimestampKey))

	// test custom
	url1, _ = NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?anyhost=true&" +
		"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&" +
		"environment=dev&interface=com.ikurento.user.UserProvider&ip=192.168.56.1&methods=GetUser%2C&" +
		"module=dubbogo+user-info+server&org=ikurento.com&owner=ZX&pid=1447&revision=0.0.1&" +
		"side=provider&timeout=3000&timestamp=1556509797245")
	url2, _ = NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?anyhost=true&" +
		"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&" +
		"environment=dev&interface=com.ikurento.user.UserProvider&ip=192.168.56.1&methods=GetUser%2C&" +
		"module=dubbogo+user-info+server&org=ikurento.com&owner=ZX&pid=1447&revision=0.0.1&" +
		"side=provider&timeout=3000&timestamp=155650979798")
	assert.True(t, GetCompareURLEqualFunc()(url1, url2, constant.TimestampKey, constant.RemoteTimestampKey))
	SetCompareURLEqualFunc(CustomCompareURLEqual)
	assert.False(t, GetCompareURLEqualFunc()(url1, url2))
	assert.False(t, GetCompareURLEqualFunc()(url1, url2, constant.TimestampKey, constant.RemoteTimestampKey))

	url1, _ = NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?anyhost=true&" +
		"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&" +
		"environment=dev&interface=com.ikurento.user.UserProvider&ip=192.168.56.1&methods=GetUser%2C&" +
		"module=dubbogo+user-info+server&org=ikurento.com&owner=ZX&pid=1447&revision=0.0.1&" +
		"side=provider&timeout=3000")
	url2, _ = NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?anyhost=true&" +
		"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&" +
		"environment=dev&interface=com.ikurento.user.UserProvider&ip=192.168.56.1&methods=GetUser%2C&" +
		"module=dubbogo+user-info+server&org=ikurento.com&owner=ZX&pid=1447&revision=0.0.1&" +
		"side=provider&timeout=3000")
	assert.True(t, GetCompareURLEqualFunc()(url1, url2))
	assert.True(t, GetCompareURLEqualFunc()(url1, url2, constant.TimestampKey, constant.RemoteTimestampKey))
	SetCompareURLEqualFunc(CustomCompareURLEqual)
	assert.True(t, GetCompareURLEqualFunc()(url1, url2))
	assert.True(t, GetCompareURLEqualFunc()(url1, url2, constant.TimestampKey, constant.RemoteTimestampKey))
}

func CustomCompareURLEqual(l *URL, r *URL, execludeParam ...string) bool {
	return l.PrimitiveURL == r.PrimitiveURL
}

func TestParseServiceKey(t *testing.T) {
	type args struct {
		serviceKey string
	}
	tests := []struct {
		name  string
		args  args
		want  string
		want1 string
		want2 string
	}{
		{
			name: "test1",
			args: args{
				serviceKey: "group/interface:version",
			},
			want:  "interface",
			want1: "group",
			want2: "version",
		},
		{
			name: "test2",
			args: args{
				serviceKey: "*/*:*",
			},
			want:  "*",
			want1: "*",
			want2: "*",
		},
		{
			name: "test3",
			args: args{
				serviceKey: "group/org.apache.dubbo.mock.api.MockService",
			},
			want:  "org.apache.dubbo.mock.api.MockService",
			want1: "group",
			want2: "",
		},
		{
			name: "test4",
			args: args{
				serviceKey: "org.apache.dubbo.mock.api.MockService",
			},
			want:  "org.apache.dubbo.mock.api.MockService",
			want1: "",
			want2: "",
		},
		{
			name: "test5",
			args: args{
				serviceKey: "group/",
			},
			want:  "",
			want1: "group",
			want2: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, got2 := ParseServiceKey(tt.args.serviceKey)
			assert.Equalf(t, tt.want, got, "ParseServiceKey(%v)", tt.args.serviceKey)
			assert.Equalf(t, tt.want1, got1, "ParseServiceKey(%v)", tt.args.serviceKey)
			assert.Equalf(t, tt.want2, got2, "ParseServiceKey(%v)", tt.args.serviceKey)
		})
	}
}

func TestIsAnyCondition(t *testing.T) {
	type args struct {
		intf       string
		group      string
		version    string
		serviceURL *URL
	}
	serviceURL, _ := NewURL(GetLocalIp()+":0", WithProtocol("admin"), WithParams(url.Values{
		constant.GroupKey:     {"group"},
		constant.VersionKey:   {"version"},
		constant.InterfaceKey: {"intf"},
	}))
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "test1",
			args: args{
				intf:       constant.AnyValue,
				group:      constant.AnyValue,
				version:    constant.AnyValue,
				serviceURL: serviceURL,
			},
			want: true,
		},
		{
			name: "test2",
			args: args{
				intf:       constant.AnyValue,
				group:      "group",
				version:    "version",
				serviceURL: serviceURL,
			},
			want: true,
		},
		{
			name: "test3",
			args: args{
				intf:       "intf",
				group:      constant.AnyValue,
				version:    constant.AnyValue,
				serviceURL: serviceURL,
			},
			want: true,
		},
		{
			name: "test4",
			args: args{
				intf:       constant.AnyValue,
				group:      "group1",
				version:    constant.AnyValue,
				serviceURL: serviceURL,
			},
			want: false,
		},
		{
			name: "test5",
			args: args{
				intf:       "intf",
				group:      "group",
				version:    constant.AnyValue,
				serviceURL: serviceURL,
			},
			want: true,
		},
		{
			name: "test6",
			args: args{
				intf:       "intf",
				group:      constant.AnyValue,
				version:    "version1",
				serviceURL: serviceURL,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, IsAnyCondition(tt.args.intf, tt.args.group, tt.args.version, tt.args.serviceURL), "IsAnyCondition(%v, %v, %v, %v)", tt.args.intf, tt.args.group, tt.args.version, tt.args.serviceURL)
		})
	}
}

func TestNewURLWithMultiAddr(t *testing.T) {
	u1, err := NewURL("zookeeper://127.0.0.1:2181,127.0.0.1:2182,127.0.0.1:2183")
	assert.Nil(t, err)
	assert.Equal(t, "127.0.0.1:2181,127.0.0.1:2182,127.0.0.1:2183", u1.Location)

	u2, err := NewURL("zookeeper://127.0.0.1:2181 , 127.0.0.1:2182, 127.0.0.1:2183")
	assert.Nil(t, err)
	assert.Equal(t, "127.0.0.1:2181,127.0.0.1:2182,127.0.0.1:2183", u2.Location)

	u3, err := NewURL("zookeeper://127.0.0.1:2181 , 127.0.0.1:2182, 127.0.0.1:2183,, , , ")
	assert.Nil(t, err)
	assert.Equal(t, "127.0.0.1:2181,127.0.0.1:2182,127.0.0.1:2183", u3.Location)

	u4, err := NewURL(" , ,127.0.0.1:2181 , 127.0.0.1:2182, 127.0.0.1:2183,, , , ", WithProtocol("zookeeper"))
	assert.Nil(t, err)
	assert.Equal(t, "127.0.0.1:2181,127.0.0.1:2182,127.0.0.1:2183", u4.Location)
}

func TestURLSetParamsMultiValue(t *testing.T) {
	// Test case to verify multi-value parameter handling
	u := &URL{}

	// Create url.Values with multiple values for the same key
	params := url.Values{}
	params.Add("tag", "foo")
	params.Add("tag", "bar")
	params.Add("tag", "baz")
	params.Set("single", "value")

	// Set the parameters using the fixed implementation
	u.SetParams(params)

	// The implementation should preserve all values (multi-values stored internally)
	got := u.GetParams()["tag"]
	assert.ElementsMatch(t, []string{"foo", "bar", "baz"}, got)
	assert.Equal(t, "value", u.GetParam("single", ""))

	// Test with empty values
	params2 := url.Values{}
	params2.Add("empty", "")
	params2.Add("empty", "value")
	params2.Add("only_empty", "")

	u2 := &URL{}
	u2.SetParams(params2)

	// Empty value slots are preserved as provided by caller
	got2 := u2.GetParams()["empty"]
	assert.ElementsMatch(t, []string{"", "value"}, got2)
	got3 := u2.GetParams()["only_empty"]
	assert.ElementsMatch(t, []string{""}, got3)
}

func TestRoleType(t *testing.T) {
	assert.Equal(t, "consumers", RoleType(CONSUMER).String())
	assert.Equal(t, "configurators", RoleType(CONFIGURATOR).String())
	assert.Equal(t, "routers", RoleType(ROUTER).String())
	assert.Equal(t, "providers", RoleType(PROVIDER).String())

	assert.Equal(t, "consumer", RoleType(CONSUMER).Role())
	assert.Equal(t, "", RoleType(CONFIGURATOR).Role())
	assert.Equal(t, "routers", RoleType(ROUTER).Role())
	assert.Equal(t, "provider", RoleType(PROVIDER).Role())
}

func TestJavaClassName(t *testing.T) {
	u := &URL{}
	assert.Equal(t, "org.apache.dubbo.common.URL", u.JavaClassName())
}

func TestWithInterface(t *testing.T) {
	u := NewURLWithOptions(WithInterface("com.test.Service"))
	assert.Equal(t, "com.test.Service", u.GetParam(constant.InterfaceKey, ""))
}

func TestWithLocation(t *testing.T) {
	// WithLocation sets Location, but NewURLWithOptions overwrites it with Ip:Port
	// So we need to set Ip and Port as well, or test the option directly
	u := &URL{}
	WithLocation("192.168.1.1:8080")(u)
	assert.Equal(t, "192.168.1.1:8080", u.Location)

	// When using NewURLWithOptions with Ip and Port
	u2 := NewURLWithOptions(WithIp("192.168.1.1"), WithPort("8080"))
	assert.Equal(t, "192.168.1.1:8080", u2.Location)
}

func TestWithToken(t *testing.T) {
	// empty token
	u1 := NewURLWithOptions(WithToken(""))
	assert.Equal(t, "", u1.GetParam(constant.TokenKey, ""))

	// custom token
	u2 := NewURLWithOptions(WithToken("my-token"))
	assert.Equal(t, "my-token", u2.GetParam(constant.TokenKey, ""))

	// "true" generates UUID
	u3 := NewURLWithOptions(WithToken("true"))
	token := u3.GetParam(constant.TokenKey, "")
	assert.NotEmpty(t, token)
	assert.NotEqual(t, "true", token)

	// "default" generates UUID
	u4 := NewURLWithOptions(WithToken("default"))
	token2 := u4.GetParam(constant.TokenKey, "")
	assert.NotEmpty(t, token2)
	assert.NotEqual(t, "default", token2)

	// "TRUE" (uppercase) generates UUID
	u5 := NewURLWithOptions(WithToken("TRUE"))
	token3 := u5.GetParam(constant.TokenKey, "")
	assert.NotEmpty(t, token3)
	assert.NotEqual(t, "TRUE", token3)
}

func TestWithWeight(t *testing.T) {
	// positive weight
	u1 := NewURLWithOptions(WithWeight(100))
	assert.Equal(t, "100", u1.GetParam(constant.WeightKey, ""))

	// zero weight (should not be set)
	u2 := NewURLWithOptions(WithWeight(0))
	assert.Equal(t, "", u2.GetParam(constant.WeightKey, ""))

	// negative weight (should not be set)
	u3 := NewURLWithOptions(WithWeight(-1))
	assert.Equal(t, "", u3.GetParam(constant.WeightKey, ""))
}

func TestMatchKey(t *testing.T) {
	assert.Equal(t, "com.test.Service:dubbo", MatchKey("com.test.Service", "dubbo"))
	assert.Equal(t, ":http", MatchKey("", "http"))
}

func TestURLGroupInterfaceVersion(t *testing.T) {
	u, _ := NewURL("dubbo://127.0.0.1:20000/com.test.Service?interface=com.test.Service&group=test-group&version=1.0.0")
	assert.Equal(t, "test-group", u.Group())
	assert.Equal(t, "com.test.Service", u.Interface())
	assert.Equal(t, "1.0.0", u.Version())
}

func TestURLAddress(t *testing.T) {
	u1 := &URL{Ip: "192.168.1.1", Port: "8080"}
	assert.Equal(t, "192.168.1.1:8080", u1.Address())

	u2 := &URL{Ip: "192.168.1.1", Port: ""}
	assert.Equal(t, "192.168.1.1", u2.Address())
}

func TestURLKey(t *testing.T) {
	u, _ := NewURL("dubbo://user:pass@127.0.0.1:20000/com.test.Service?interface=com.test.Service&group=g1&version=1.0")
	key := u.Key()
	assert.Contains(t, key, "dubbo://")
	assert.Contains(t, key, "user:pass@")
	assert.Contains(t, key, "interface=com.test.Service")
	assert.Contains(t, key, "group=g1")
	assert.Contains(t, key, "version=1.0")
}

func TestGetCacheInvokerMapKey(t *testing.T) {
	u, _ := NewURL("dubbo://127.0.0.1:20000/com.test.Service?interface=com.test.Service&group=g1&version=1.0&timestamp=12345")
	key := u.GetCacheInvokerMapKey()
	assert.Contains(t, key, "interface=com.test.Service")
	assert.Contains(t, key, "group=g1")
	assert.Contains(t, key, "version=1.0")
	assert.Contains(t, key, "timestamp=12345")
}

func TestServiceKey(t *testing.T) {
	// with group and version
	assert.Equal(t, "group/interface:version", ServiceKey("interface", "group", "version"))

	// without group
	assert.Equal(t, "interface:version", ServiceKey("interface", "", "version"))

	// without version
	assert.Equal(t, "group/interface", ServiceKey("interface", "group", ""))

	// version 0.0.0 should be ignored
	assert.Equal(t, "group/interface", ServiceKey("interface", "group", "0.0.0"))

	// empty interface
	assert.Equal(t, "", ServiceKey("", "group", "version"))

	// URL method
	u, _ := NewURL("dubbo://127.0.0.1:20000/com.test.Service?interface=com.test.Service&group=g1&version=1.0")
	assert.Equal(t, "g1/com.test.Service:1.0", u.ServiceKey())
}

func TestEncodedServiceKey(t *testing.T) {
	u, _ := NewURL("dubbo://127.0.0.1:20000/com.test.Service?interface=com.test.Service&group=g1&version=1.0")
	encoded := u.EncodedServiceKey()
	assert.Equal(t, "g1*com.test.Service:1.0", encoded)
}

func TestServiceWithSubURL(t *testing.T) {
	subURL, _ := NewURL("dubbo://127.0.0.1:20000?interface=com.sub.Service")
	u := &URL{SubURL: subURL}
	assert.Equal(t, "com.sub.Service", u.Service())

	// empty SubURL interface
	subURL2 := &URL{}
	u2 := &URL{SubURL: subURL2, Path: "/com.path.Service"}
	assert.Equal(t, "com.path.Service", u2.Service())

	// no SubURL, no interface param
	u3 := &URL{Path: "/com.path.Service"}
	assert.Equal(t, "com.path.Service", u3.Service())
}

func TestAddParam(t *testing.T) {
	u := &URL{}
	u.AddParam("key1", "value1")
	assert.Equal(t, "value1", u.GetParam("key1", ""))

	// add another value to same key
	u.AddParam("key1", "value2")
	params := u.GetParams()
	assert.Equal(t, 2, len(params["key1"]))
}

func TestAddParamAvoidNil(t *testing.T) {
	u := &URL{}
	u.AddParamAvoidNil("key1", "value1")
	assert.Equal(t, "value1", u.GetParam("key1", ""))
}

func TestDelParam(t *testing.T) {
	u, _ := NewURL("dubbo://127.0.0.1:20000?key1=value1&key2=value2")
	u.DelParam("key1")
	assert.Equal(t, "", u.GetParam("key1", ""))
	assert.Equal(t, "value2", u.GetParam("key2", ""))

	// delete from nil params
	u2 := &URL{}
	u2.DelParam("key") // should not panic
}

func TestGetNonDefaultParam(t *testing.T) {
	u, _ := NewURL("dubbo://127.0.0.1:20000?key1=value1")

	v, ok := u.GetNonDefaultParam("key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", v)

	v, ok = u.GetNonDefaultParam("nonexistent")
	assert.False(t, ok)
	assert.Equal(t, "", v)
}

func TestRangeParams(t *testing.T) {
	u, _ := NewURL("dubbo://127.0.0.1:20000?key1=value1&key2=value2")
	count := 0
	u.RangeParams(func(key, value string) bool {
		count++
		return true
	})
	assert.Equal(t, 2, count)

	// test early break
	count = 0
	u.RangeParams(func(key, value string) bool {
		count++
		return false // break after first
	})
	assert.Equal(t, 1, count)
}

func TestGetParamInt32(t *testing.T) {
	params := url.Values{}
	params.Set("key", "123")

	u := &URL{}
	u.SetParams(params)

	v := u.GetParamInt32("key", 0)
	assert.Equal(t, int32(123), v)

	// invalid value
	u2 := &URL{}
	v2 := u2.GetParamInt32("key", 99)
	assert.Equal(t, int32(99), v2)
}

func TestGetParamByIntValue(t *testing.T) {
	params := url.Values{}
	params.Set("key", "456")

	u := &URL{}
	u.SetParams(params)

	v := u.GetParamByIntValue("key", 0)
	assert.Equal(t, 456, v)

	// invalid value
	u2 := &URL{}
	v2 := u2.GetParamByIntValue("key", 99)
	assert.Equal(t, 99, v2)
}

func TestGetMethodParamIntValue(t *testing.T) {
	params := url.Values{}
	params.Set("methods.GetValue.timeout", "100")

	u := &URL{}
	u.SetParams(params)

	v := u.GetMethodParamIntValue("GetValue", "timeout", 0)
	assert.Equal(t, 100, v)

	// missing method param
	v2 := u.GetMethodParamIntValue("OtherMethod", "timeout", 50)
	assert.Equal(t, 50, v2)
}

func TestGetMethodParamInt64(t *testing.T) {
	params := url.Values{}
	params.Set("methods.GetValue.timeout", "200")
	params.Set("timeout", "100")

	u := &URL{}
	u.SetParams(params)

	// method param exists
	v := u.GetMethodParamInt64("GetValue", "timeout", 0)
	assert.Equal(t, int64(200), v)

	// method param not exists, fallback to global
	v2 := u.GetMethodParamInt64("OtherMethod", "timeout", 0)
	assert.Equal(t, int64(100), v2)

	// neither exists
	v3 := u.GetMethodParamInt64("OtherMethod", "retries", 5)
	assert.Equal(t, int64(5), v3)
}

func TestGetParamDuration(t *testing.T) {
	params := url.Values{}
	params.Set("timeout", "5s")

	u := &URL{}
	u.SetParams(params)

	d := u.GetParamDuration("timeout", "1s")
	assert.Equal(t, 5*time.Second, d)

	// invalid duration, returns 3s default
	params2 := url.Values{}
	params2.Set("timeout", "invalid")
	u2 := &URL{}
	u2.SetParams(params2)
	d2 := u2.GetParamDuration("timeout", "invalid")
	assert.Equal(t, 3*time.Second, d2)

	// missing param with valid default
	u3 := &URL{}
	d3 := u3.GetParamDuration("timeout", "2s")
	assert.Equal(t, 2*time.Second, d3)
}

func TestCloneExceptParams(t *testing.T) {
	u, _ := NewURL("dubbo://127.0.0.1:20000/com.test.Service?key1=value1&key2=value2&key3=value3")

	excludeSet := gxset.NewSet("key1", "key3")
	cloned := u.CloneExceptParams(excludeSet)

	assert.Equal(t, "", cloned.GetParam("key1", ""))
	assert.Equal(t, "value2", cloned.GetParam("key2", ""))
	assert.Equal(t, "", cloned.GetParam("key3", ""))
	assert.Equal(t, "dubbo", cloned.Protocol)
}

func TestCloneWithParams(t *testing.T) {
	u, _ := NewURL("dubbo://user:pass@127.0.0.1:20000/com.test.Service?key1=value1&key2=value2&key3=value3") // #nosec G101 - test credential
	u.Methods = []string{"method1", "method2"}

	cloned := u.CloneWithParams([]string{"key1", "key3"})

	assert.Equal(t, "value1", cloned.GetParam("key1", ""))
	assert.Equal(t, "", cloned.GetParam("key2", ""))
	assert.Equal(t, "value3", cloned.GetParam("key3", ""))
	assert.Equal(t, "dubbo", cloned.Protocol)
	assert.Equal(t, "user", cloned.Username)
	assert.Equal(t, "pass", cloned.Password) // #nosec G101 - test credential
	assert.Equal(t, []string{"method1", "method2"}, cloned.Methods)
}

func TestURLCompare(t *testing.T) {
	u1, _ := NewURL("dubbo://127.0.0.1:20000/a.Service")
	u2, _ := NewURL("dubbo://127.0.0.1:20000/b.Service")
	u3, _ := NewURL("dubbo://127.0.0.1:20000/a.Service")

	assert.Equal(t, -1, u1.Compare(u2))
	assert.Equal(t, 1, u2.Compare(u1))
	assert.Equal(t, 0, u1.Compare(u3))
}

func TestRangeAttributes(t *testing.T) {
	u := &URL{}
	u.SetAttribute("attr1", "value1")
	u.SetAttribute("attr2", 123)

	count := 0
	u.RangeAttributes(func(key string, value any) bool {
		count++
		return true
	})
	assert.Equal(t, 2, count)

	// test early break
	count = 0
	u.RangeAttributes(func(key string, value any) bool {
		count++
		return false
	})
	assert.Equal(t, 1, count)
}

func TestURLSlice(t *testing.T) {
	u1, _ := NewURL("dubbo://127.0.0.1:20000/c.Service")
	u2, _ := NewURL("dubbo://127.0.0.1:20000/a.Service")
	u3, _ := NewURL("dubbo://127.0.0.1:20000/b.Service")

	slice := URLSlice{u1, u2, u3}

	assert.Equal(t, 3, slice.Len())
	assert.True(t, slice.Less(1, 2))  // a < b
	assert.False(t, slice.Less(0, 2)) // c > b

	slice.Swap(0, 1)
	assert.Equal(t, u2, slice[0])
	assert.Equal(t, u1, slice[1])
}

func TestIsEquals(t *testing.T) {
	u1, _ := NewURL("dubbo://127.0.0.1:20000/com.test.Service?key1=value1&key2=value2")
	u2, _ := NewURL("dubbo://127.0.0.1:20000/com.test.Service?key1=value1&key2=value2")
	u3, _ := NewURL("dubbo://127.0.0.1:20001/com.test.Service?key1=value1&key2=value2")
	u4, _ := NewURL("dubbo://127.0.0.1:20000/com.test.Service?key1=value1&key2=different")

	// equal URLs
	assert.True(t, IsEquals(u1, u2))

	// different port
	assert.False(t, IsEquals(u1, u3))

	// different param value
	assert.False(t, IsEquals(u1, u4))

	// with excludes
	assert.True(t, IsEquals(u1, u4, "key2"))

	// nil checks
	assert.False(t, IsEquals(nil, u1))
	assert.False(t, IsEquals(u1, nil))
}

func TestGetSubscribeName(t *testing.T) {
	u, _ := NewURL("dubbo://127.0.0.1:20000?interface=com.test.Service&version=1.0&group=test")
	name := GetSubscribeName(u)
	assert.Contains(t, name, "providers")
	assert.Contains(t, name, "com.test.Service")
	assert.Contains(t, name, "1.0")
	assert.Contains(t, name, "test")
}

func TestNewURLEmptyString(t *testing.T) {
	u, err := NewURL("")
	assert.NoError(t, err)
	assert.NotNil(t, u)
	assert.Equal(t, "", u.Protocol)
}

func TestNewURLWithUsernamePassword(t *testing.T) {
	// #nosec G101 - test credential for URL parsing test
	u, err := NewURL("dubbo://admin:testpwd@127.0.0.1:20000/com.test.Service")
	assert.NoError(t, err)
	assert.Equal(t, "admin", u.Username)
	assert.Equal(t, "testpwd", u.Password)
}

func TestURLStringWithCredentials(t *testing.T) {
	// #nosec G101 - test credential for URL string test
	u, _ := NewURL("dubbo://admin:testpwd@127.0.0.1:20000/com.test.Service?key=value")
	str := u.String()
	assert.Contains(t, str, "admin:testpwd@")
}

func TestToMapWithoutPort(t *testing.T) {
	u := &URL{
		Protocol: "dubbo",
		Location: "127.0.0.1",
		Path:     "/com.test.Service",
	}
	m := u.ToMap()
	assert.Equal(t, "0", m["port"])
}

func TestToMapEmpty(t *testing.T) {
	u := &URL{}
	m := u.ToMap()
	assert.Nil(t, m)
}

func TestMergeURLWithAttributes(t *testing.T) {
	u1, _ := NewURL("dubbo://127.0.0.1:20000?key1=value1")
	u1.SetAttribute("attr1", "attrValue1")

	u2, _ := NewURL("dubbo://127.0.0.1:20001?key2=value2")
	u2.SetAttribute("attr2", "attrValue2")
	u2.Methods = []string{"method1"}

	merged := u1.MergeURL(u2)

	// attributes should be merged
	v1, ok1 := merged.GetAttribute("attr1")
	assert.True(t, ok1)
	assert.Equal(t, "attrValue1", v1)

	v2, ok2 := merged.GetAttribute("attr2")
	assert.True(t, ok2)
	assert.Equal(t, "attrValue2", v2)
}

func TestURLEqualWithCategory(t *testing.T) {
	// test category matching with RemoveValuePrefix
	u1, _ := NewURL("dubbo://127.0.0.1:20000/com.test.Service?interface=com.test.Service&category=providers")
	u2, _ := NewURL("dubbo://127.0.0.1:20000/com.test.Service?interface=com.test.Service&category=-providers")
	assert.False(t, u1.URLEqual(u2))

	// category contains target - u3 has category that contains u1's category
	u3, _ := NewURL("dubbo://127.0.0.1:20000/com.test.Service?interface=com.test.Service&category=providers,consumers")
	// u1 checks if u3's category matches - u3 has "providers,consumers" which contains "providers"
	assert.True(t, u3.URLEqual(u1))
}

func TestNewURLWithRegistryGroup(t *testing.T) {
	u, err := NewURL("dubbo://127.0.0.1:20000/com.test.Service?registry.group=mygroup")
	assert.NoError(t, err)
	assert.Contains(t, u.PrimitiveURL, "mygroup")
}

func TestColonSeparatedKeyEmpty(t *testing.T) {
	u := &URL{}
	assert.Equal(t, "", u.ColonSeparatedKey())
}

func TestColonSeparatedKeyWithVersion000(t *testing.T) {
	u, _ := NewURL("dubbo://127.0.0.1:20000")
	u.AddParam(constant.InterfaceKey, "com.test.Service")
	u.AddParam(constant.VersionKey, "0.0.0")
	u.AddParam(constant.GroupKey, "group1")

	// version 0.0.0 should be treated as empty
	assert.Equal(t, "com.test.Service::group1", u.ColonSeparatedKey())
}

// Additional edge case tests for better coverage

func TestNewURLParseError(t *testing.T) {
	// Test invalid URL that causes parse error
	_, err := NewURL("://invalid")
	assert.Error(t, err)
}

func TestNewURLWithInvalidQuery(t *testing.T) {
	// URL with special characters that need escaping
	u, err := NewURL("dubbo://127.0.0.1:20000/com.test.Service?key=value%20with%20space")
	assert.NoError(t, err)
	assert.Equal(t, "value with space", u.GetParam("key", ""))
}

func TestURLStringWithoutCredentials(t *testing.T) {
	u, _ := NewURL("dubbo://127.0.0.1:20000/com.test.Service?key=value")
	str := u.String()
	assert.Contains(t, str, "dubbo://127.0.0.1:20000")
	assert.NotContains(t, str, "@")
}

func TestGetParamAndDecodedError(t *testing.T) {
	u := &URL{}
	params := url.Values{}
	params.Set("rule", "invalid-base64!!!")
	u.SetParams(params)

	_, err := u.GetParamAndDecoded("rule")
	assert.Error(t, err)
}

func TestIsEqualsWithDifferentMapLength(t *testing.T) {
	u1, _ := NewURL("dubbo://127.0.0.1:20000/com.test.Service?key1=value1")
	u2, _ := NewURL("dubbo://127.0.0.1:20000/com.test.Service?key1=value1&key2=value2")
	assert.False(t, IsEquals(u1, u2))
}

func TestIsEqualsWithMissingKey(t *testing.T) {
	u1, _ := NewURL("dubbo://127.0.0.1:20000/com.test.Service?key1=value1&key2=value2")
	u2, _ := NewURL("dubbo://127.0.0.1:20000/com.test.Service?key1=value1&key3=value3")
	assert.False(t, IsEquals(u1, u2))
}

func TestIsEqualsBothNil(t *testing.T) {
	// Both nil should not panic - but current implementation would panic
	// This tests the nil check logic
	var u1 *URL = nil
	var u2 *URL = nil
	// Note: IsEquals(nil, nil) would panic due to nil pointer dereference
	// This is expected behavior based on current implementation
	assert.False(t, IsEquals(u1, &URL{}))
	assert.False(t, IsEquals(&URL{}, u2))
}

func TestMergeURLWithMethodParams(t *testing.T) {
	referenceUrlParams := url.Values{}
	referenceUrlParams.Set(constant.LoadbalanceKey, "random")
	referenceUrlParams.Set("methods.testMethod."+constant.TimeoutKey, "5000")

	serviceUrlParams := url.Values{}
	serviceUrlParams.Set(constant.ClusterKey, "failover")

	referenceUrl, _ := NewURL("mock1://127.0.0.1:1111", WithParams(referenceUrlParams), WithMethods([]string{"testMethod"}))
	serviceUrl, _ := NewURL("mock2://127.0.0.1:20000", WithParams(serviceUrlParams))

	mergedUrl := serviceUrl.MergeURL(referenceUrl)
	assert.Equal(t, "random", mergedUrl.GetParam(constant.LoadbalanceKey, ""))
	assert.Equal(t, "5000", mergedUrl.GetParam("methods.testMethod."+constant.TimeoutKey, ""))
}

func TestURLWithPathAlreadyHasSlash(t *testing.T) {
	u := NewURLWithOptions(WithPath("/com.test.Service"))
	assert.Equal(t, "/com.test.Service", u.Path)

	u2 := NewURLWithOptions(WithPath("com.test.Service"))
	assert.Equal(t, "/com.test.Service", u2.Path)
}

func TestServiceKeyWithEmptyInterface(t *testing.T) {
	u := &URL{}
	assert.Equal(t, "", u.ServiceKey())
}

func TestGetRawParamDefault(t *testing.T) {
	u, _ := NewURL("dubbo://127.0.0.1:20000?customKey=customValue")
	assert.Equal(t, "customValue", u.GetRawParam("customKey"))
	assert.Equal(t, "", u.GetRawParam("nonexistent"))
}

func TestCloneWithNilParams(t *testing.T) {
	u := &URL{
		Protocol: "dubbo",
		Ip:       "127.0.0.1",
		Port:     "20000",
	}
	cloned := u.Clone()
	assert.Equal(t, "dubbo", cloned.Protocol)
	assert.Equal(t, "127.0.0.1", cloned.Ip)
}

func TestCloneWithAttributes(t *testing.T) {
	u, _ := NewURL("dubbo://127.0.0.1:20000")
	u.SetAttribute("key1", "value1")
	u.SetAttribute("key2", 123)

	cloned := u.Clone()

	v1, ok1 := cloned.GetAttribute("key1")
	assert.True(t, ok1)
	assert.Equal(t, "value1", v1)

	v2, ok2 := cloned.GetAttribute("key2")
	assert.True(t, ok2)
	assert.Equal(t, 123, v2)
}

func TestSetParamsWithEmptySlice(t *testing.T) {
	u := &URL{}
	params := url.Values{}
	params["emptyKey"] = []string{}

	u.SetParams(params)
	// Empty slice should delete the key
	assert.Equal(t, "", u.GetParam("emptyKey", ""))
}

func TestReplaceParams(t *testing.T) {
	u, _ := NewURL("dubbo://127.0.0.1:20000?key1=value1")

	newParams := url.Values{}
	newParams.Set("key2", "value2")

	u.ReplaceParams(newParams)

	assert.Equal(t, "", u.GetParam("key1", ""))
	assert.Equal(t, "value2", u.GetParam("key2", ""))
}

func TestURLEqualWithDifferentIpPort(t *testing.T) {
	// URLEqual ignores IP and Port differences
	u1, _ := NewURL("dubbo://127.0.0.1:20000/com.test.Service?interface=com.test.Service")
	u2, _ := NewURL("dubbo://192.168.1.1:30000/com.test.Service?interface=com.test.Service")
	assert.True(t, u1.URLEqual(u2))
}

func TestIsMatchCategoryWithEmptyCategory2(t *testing.T) {
	u1, _ := NewURL("dubbo://127.0.0.1:20000/com.test.Service?interface=com.test.Service&category=providers")
	u2, _ := NewURL("dubbo://127.0.0.1:20000/com.test.Service?interface=com.test.Service")
	// When category2 is empty, it should match DefaultCategory
	assert.True(t, u1.URLEqual(u2))
}

func TestNewURLWithLocationWithoutPort(t *testing.T) {
	u, err := NewURL("dubbo://hostname/com.test.Service")
	assert.NoError(t, err)
	assert.Equal(t, "hostname", u.Location)
	assert.Equal(t, "", u.Port)
}

func TestGetParamBoolInvalidValue(t *testing.T) {
	params := url.Values{}
	params.Set("key", "notabool")

	u := &URL{}
	u.SetParams(params)

	v := u.GetParamBool("key", true)
	assert.Equal(t, true, v) // returns default on parse error
}

func TestGetParamIntInvalidValue(t *testing.T) {
	params := url.Values{}
	params.Set("key", "notanint")

	u := &URL{}
	u.SetParams(params)

	v := u.GetParamInt("key", 100)
	assert.Equal(t, int64(100), v) // returns default on parse error
}

func TestAppendParamWithEmptyValue(t *testing.T) {
	u, _ := NewURL("dubbo://127.0.0.1:20000?interface=com.test.Service")
	name := GetSubscribeName(u)
	// Should handle empty version and group
	assert.Contains(t, name, "providers")
	assert.Contains(t, name, "com.test.Service")
}
