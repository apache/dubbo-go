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

package inmemory

import (
	"context"
	"fmt"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/metadata/definition"
)

type User struct {
	Id   string
	Name string
	Age  int32
	Time time.Time
}

type UserProvider struct {
}

func (u *UserProvider) GetUser(ctx context.Context, req []interface{}) (*User, error) {
	rsp := User{"A001", "Alex Stocks", 18, time.Now()}
	return &rsp, nil
}

func (u *UserProvider) Reference() string {
	return "UserProvider"
}

func (u User) JavaClassName() string {
	return "com.ikurento.user.User"
}

func TestMetadataService(t *testing.T) {
	mts := NewMetadataService()
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
	mts.ExportURL(u2)

	u3, err := common.NewURL(fmt.Sprintf(
		"%v://127.0.0.1:20000/com.ikurento.user.UserProvider3?anyhost=true&"+
			"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&"+
			"environment=dev&interface=%v&ip=192.168.56.1&methods=GetUser&module=dubbogo+user-info+server&org=ikurento.com&"+
			"owner=ZX&pid=1447&revision=0.0.1&side=provider&timeout=3000&timestamp=1556509797245&group=%v&version=%v&bean.name=%v",
		protocol, serviceName, group, version, beanName))
	assert.NoError(t, err)
	mts.ExportURL(u3)

	u, err := common.NewURL(fmt.Sprintf(
		"%v://127.0.0.1:20000/com.ikurento.user.UserProvider1?anyhost=true&"+
			"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&"+
			"environment=dev&interface=%v&ip=192.168.56.1&methods=GetUser&module=dubbogo+user-info+server&org=ikurento.com&"+
			"owner=ZX&pid=1447&revision=0.0.1&side=provider&timeout=3000&timestamp=1556509797245&group=%v&version=%v&bean.name=%v",
		protocol, serviceName, group, version, beanName))
	assert.NoError(t, err)
	mts.ExportURL(u)
	list, _ := mts.GetExportedURLs(serviceName, group, version, protocol)
	assert.Equal(t, uint64(3), list.Len())
	iter := list.IterAtPosition(0)
	for iter.Next() {
		comparator := iter.Value()
		fmt.Println(comparator)
	}
	mts.SubscribeURL(u)

	mts.SubscribeURL(u)
	list2, _ := mts.GetSubscribedURLs()
	assert.Equal(t, uint64(1), list2.Len())

	mts.UnexportURL(u)
	list3, _ := mts.GetExportedURLs(serviceName, group, version, protocol)
	assert.Equal(t, uint64(2), list3.Len())

	mts.UnsubscribeURL(u)
	list4, _ := mts.GetSubscribedURLs()
	assert.Equal(t, uint64(0), list4.Len())

	userProvider := &UserProvider{}
	common.ServiceMap.Register(serviceName, protocol, userProvider)
	mts.PublishServiceDefinition(u)
	expected := "{\"CanonicalName\":\"com.ikurento.user.UserProvider\",\"CodeSource\":\"\"," +
		"\"Methods\":[{\"Name\":\"GetUser\",\"ParameterTypes\":[\"slice\"],\"ReturnType\":\"ptr\"," +
		"\"Parameters\":null}],\"Types\":null}"
	def1, _ := mts.GetServiceDefinition(serviceName, group, version)
	assert.Equal(t, expected, def1)
	serviceKey := definition.ServiceDescriperBuild(serviceName, group, version)
	def2, _ := mts.GetServiceDefinitionByServiceKey(serviceKey)
	assert.Equal(t, expected, def2)
}
