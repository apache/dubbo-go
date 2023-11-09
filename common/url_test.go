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

package common

import (
	"encoding/base64"
	"net/url"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

const (
	userName        = "username"
	password        = "password"
	loopbackAddress = "127.0.0.1"
)

func TestNewURLWithOptions(t *testing.T) {
	methods := []string{"Methodone,methodtwo"}
	params := url.Values{}
	params.Set("key", "value")
	u := NewURLWithOptions(WithPath("com.test.Service"),
		WithUsername(userName),
		WithPassword(password),
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
	assert.Equal(t, password, u.Password)
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
	u.Password = "test"
	assert.Equal(t, "condition", u.GetRawParam("protocol"))
	assert.Equal(t, "0.0.0.0", u.GetRawParam("host"))
	assert.Equal(t, "8080", u.GetRawParam("port"))
	assert.Equal(t, "test", u.GetRawParam(userName))
	assert.Equal(t, "test", u.GetRawParam(password))
	assert.Equal(t, "/com.foo.BarService", u.GetRawParam("path"))
	assert.Equal(t, "fastjson", u.GetRawParam("serialization"))
}

func TestURLToMap(t *testing.T) {
	u, _ := NewURL("condition://0.0.0.0:8080/com.foo.BarService?serialization=fastjson")
	u.Username = "test"
	u.Password = "test"

	m := u.ToMap()
	assert.Equal(t, 7, len(m))
	assert.Equal(t, "condition", m["protocol"])
	assert.Equal(t, "0.0.0.0", m["host"])
	assert.Equal(t, "8080", m["port"])
	assert.Equal(t, "test", m[userName])
	assert.Equal(t, "test", m[password])
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

	mergedUrl := MergeURL(serviceUrl, referenceUrl)
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
		constant.GroupKey:   {"group"},
		constant.VersionKey: {"version"},
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
			want: false,
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, IsAnyCondition(tt.args.intf, tt.args.group, tt.args.version, tt.args.serviceURL), "IsAnyCondition(%v, %v, %v, %v)", tt.args.intf, tt.args.group, tt.args.version, tt.args.serviceURL)
		})
	}
}
