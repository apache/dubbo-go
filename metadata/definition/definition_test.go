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

package definition

import (
	"fmt"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
)

func TestBuildServiceDefinition(t *testing.T) {
	serviceName := "com.ikurento.user.UserProvider"
	group := "group1"
	version := "0.0.1"
	protocol := "dubbo"
	beanName := "UserProvider"
	url, err := common.NewURL(fmt.Sprintf(
		"%v://127.0.0.1:20000/com.ikurento.user.UserProvider1?anyhost=true&"+
			"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&"+
			"environment=dev&interface=%v&ip=192.168.56.1&methods=GetUser&module=dubbogo+user-info+server&org=ikurento.com&"+
			"owner=ZX&pid=1447&revision=0.0.1&side=provider&timeout=3000&timestamp=1556509797245&group=%v&version=%v&bean.name=%v",
		protocol, serviceName, group, version, beanName))
	assert.NoError(t, err)
	_, err = common.ServiceMap.Register(serviceName, protocol, group, version, &UserProvider{})
	assert.NoError(t, err)
	service := common.ServiceMap.GetServiceByServiceKey(url.Protocol, url.ServiceKey())
	sd := BuildServiceDefinition(*service, url)
	assert.Equal(t, "{canonicalName:com.ikurento.user.UserProvider, codeSource:, methods:[{name:GetUser,parameterTypes:[{type:slice}],returnType:ptr,params:[] }], types:[]}", sd.String())
}
