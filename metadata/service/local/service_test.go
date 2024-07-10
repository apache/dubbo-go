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

package local

import (
	"fmt"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/metadata/definition"
)

func TestMetadataService(t *testing.T) {
	mts, _ := GetLocalMetadataService()
	serviceName := "com.ikurento.user.UserProvider"
	group := "group1"
	version := "0.0.1"
	protocol := "dubbo"
	beanName := "UserProvider"

	u2, err := common.NewURL(fmt.Sprintf(
		"%v://127.0.0.1:20000/com.ikurento.user.UserProvider2?anyhost=true&"+
			"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&"+
			"environment=dev&interface=%v&ip=192.168.56.1&methods=GetUser&module=dubbogo+user-info+server&org=ikurento.com&"+
			"owner=ZX&pid=1447&revision=0.0.1&side=provider&timeout=3000&timestamp=1556509797245&group=%v&version=%v&bean.name=%v",
		protocol, serviceName, group, version, beanName))
	assert.NoError(t, err)
	ok, err := mts.ExportURL(u2)
	assert.True(t, ok)
	assert.NoError(t, err)

	u3, err := common.NewURL(fmt.Sprintf(
		"%v://127.0.0.1:20000/com.ikurento.user.UserProvider3?anyhost=true&"+
			"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&"+
			"environment=dev&interface=%v&ip=192.168.56.1&methods=GetUser&module=dubbogo+user-info+server&org=ikurento.com&"+
			"owner=ZX&pid=1447&revision=0.0.1&side=provider&timeout=3000&timestamp=1556509797245&group=%v&version=%v&bean.name=%v",
		protocol, serviceName, group, version, beanName))
	assert.NoError(t, err)
	ok, err = mts.ExportURL(u3)
	assert.True(t, ok)
	assert.NoError(t, err)

	u, err := common.NewURL(fmt.Sprintf(
		"%v://127.0.0.1:20000/com.ikurento.user.UserProvider1?anyhost=true&"+
			"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&"+
			"environment=dev&interface=%v&ip=192.168.56.1&methods=GetUser&module=dubbogo+user-info+server&org=ikurento.com&"+
			"owner=ZX&pid=1447&revision=0.0.1&side=provider&timeout=3000&timestamp=1556509797245&group=%v&version=%v&bean.name=%v",
		protocol, serviceName, group, version, beanName))
	assert.NoError(t, err)
	ok, err = mts.ExportURL(u)
	assert.True(t, ok)
	assert.NoError(t, err)
	list, _ := mts.GetExportedURLs(serviceName, group, version, protocol)
	assert.Equal(t, 3, len(list))
	ok, err = mts.SubscribeURL(u)
	assert.True(t, ok)
	assert.NoError(t, err)

	ok, err = mts.SubscribeURL(u)
	assert.False(t, ok)
	assert.NoError(t, err)
	list2, err := mts.GetSubscribedURLs()
	assert.Equal(t, 1, len(list2))
	assert.NoError(t, err)

	err = mts.UnexportURL(u)
	assert.NoError(t, err)
	list3, _ := mts.GetExportedURLs(serviceName, group, version, protocol)
	assert.Equal(t, 2, len(list3))

	err = mts.UnsubscribeURL(u)
	assert.NoError(t, err)
	list4, _ := mts.GetSubscribedURLs()
	assert.Equal(t, 0, len(list4))

	userProvider := &definition.UserProvider{}
	_, err = common.ServiceMap.Register(serviceName, protocol, group, version, userProvider)
	assert.NoError(t, err)
	err = mts.PublishServiceDefinition(u)
	assert.NoError(t, err)
	expected := "{\"parameters\":{\"anyhost\":\"true\",\"application\":\"BDTService\"," +
		"\"bean.name\":\"UserProvider\",\"category\":\"providers\",\"default.timeout\":\"10000\"," +
		"\"dubbo\":\"dubbo-provider-golang-1.0.0\",\"environment\":\"dev\",\"group\":\"group1\"," +
		"\"interface\":\"com.ikurento.user.UserProvider\",\"ip\":\"192.168.56.1\"," +
		"\"methods\":\"GetUser\",\"module\":\"dubbogo user-info server\",\"org\":\"ikurento.com\"," +
		"\"owner\":\"ZX\",\"pid\":\"1447\",\"revision\":\"0.0.1\",\"side\":\"provider\"," +
		"\"timeout\":\"3000\",\"timestamp\":\"1556509797245\",\"version\":\"0.0.1\"}," +
		"\"canonicalName\":\"com.ikurento.user.UserProvider\",\"codeSource\":\"\"," +
		"\"methods\":[{\"name\":\"GetUser\",\"parameterTypes\":[\"slice\"],\"returnType\":\"ptr\"," +
		"\"parameters\":null},{\"name\":\"getUser\",\"parameterTypes\":[\"slice\"],\"returnType\":\"ptr\"" +
		",\"parameters\":null}],\"types\":null}"
	expected2 := "{\"parameters\":{\"anyhost\":\"true\",\"application\":\"BDTService\"," +
		"\"bean.name\":\"UserProvider\",\"category\":\"providers\",\"default.timeout\":\"10000\"," +
		"\"dubbo\":\"dubbo-provider-golang-1.0.0\",\"environment\":\"dev\",\"group\":\"group1\"," +
		"\"interface\":\"com.ikurento.user.UserProvider\",\"ip\":\"192.168.56.1\"," +
		"\"methods\":\"GetUser\",\"module\":\"dubbogo user-info server\",\"org\":\"ikurento.com\"," +
		"\"owner\":\"ZX\",\"pid\":\"1447\",\"revision\":\"0.0.1\",\"side\":\"provider\"," +
		"\"timeout\":\"3000\",\"timestamp\":\"1556509797245\",\"version\":\"0.0.1\"}," +
		"\"canonicalName\":\"com.ikurento.user.UserProvider\",\"codeSource\":\"\"," +
		"\"methods\":[{\"name\":\"getUser\",\"parameterTypes\":[\"slice\"],\"returnType\":\"ptr\"," +
		"\"parameters\":null},{\"name\":\"GetUser\",\"parameterTypes\":[\"slice\"],\"returnType\":\"ptr\"" +
		",\"parameters\":null}],\"types\":null}"
	def1, err := mts.GetServiceDefinition(serviceName, group, version)
	assert.Equal(t, true, (def1 == expected || def1 == expected2))
	assert.NoError(t, err)
	serviceKey := definition.ServiceDescriperBuild(serviceName, group, version)
	def2, err := mts.GetServiceDefinitionByServiceKey(serviceKey)
	assert.Equal(t, true, (def2 == expected || def2 == expected2))
	assert.NoError(t, err)
}
