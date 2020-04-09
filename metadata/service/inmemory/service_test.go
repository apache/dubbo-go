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
	u, _ := common.NewURL(fmt.Sprintf("%v://127.0.0.1:20000/com.ikurento.user.UserProvider?anyhost=true&"+
		"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&"+
		"environment=dev&interface=%v&ip=192.168.56.1&methods=GetUser&"+
		"module=dubbogo+user-info+server&org=ikurento.com&owner=ZX&pid=1447&revision=0.0.1&"+
		"side=provider&timeout=3000&timestamp=1556509797245&group=%v&version=%v&bean.name=%v", protocol, serviceName, group, version, beanName))
	mts.ExportURL(u)
	sets := mts.GetExportedURLs(serviceName, group, version, protocol)
	assert.Equal(t, 1, sets.Size())
	mts.SubscribeURL(u)

	mts.SubscribeURL(u)
	sets2 := mts.GetSubscribedURLs()
	assert.Equal(t, 1, sets2.Size())

	mts.UnexportURL(u)
	sets11 := mts.GetExportedURLs(serviceName, group, version, protocol)
	assert.Equal(t, 0, sets11.Size())

	mts.UnsubscribeURL(u)
	sets22 := mts.GetSubscribedURLs()
	assert.Equal(t, 0, sets22.Size())

	userProvider := &UserProvider{}
	common.ServiceMap.Register(protocol, userProvider)
	mts.PublishServiceDefinition(u)
	expected := `{"CanonicalName":"com.ikurento.user.UserProvider","CodeSource":"","Methods":[{"Name":"GetUser","ParameterTypes":["slice"],"ReturnType":"ptr","Parameters":null}],"Types":null}`
	assert.Equal(t, mts.GetServiceDefinition(serviceName, group, version), expected)
	serviceKey := definition.ServiceDescriperBuild(serviceName, group, version)
	assert.Equal(t, mts.GetServiceDefinitionByServiceKey(serviceKey), expected)
}
