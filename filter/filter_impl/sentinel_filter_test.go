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

package filter_impl

import (
	"context"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/protocol/invocation"
)

func TestConsumerFilter_Invoke(t *testing.T) {
	f := GetSentinelConsumerFilter()
	url, err := common.NewURL("dubbo://127.0.0.1:20000/UserProvider?anyhost=true&" +
		"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&" +
		"environment=dev&interface=com.ikurento.user.UserProvider&ip=192.168.56.1&methods=GetUser%2C&" +
		"module=dubbogo+user-info+server&org=ikurento.com&owner=ZX&pid=1447&revision=0.0.1&" +
		"side=provider&timeout=3000&timestamp=1556509797245&bean.name=UserProvider")
	assert.NoError(t, err)
	mockInvoker := protocol.NewBaseInvoker(url)
	mockInvocation := invocation.NewRPCInvocation("hello", []interface{}{"OK"}, make(map[string]interface{}))
	result := f.Invoke(context.TODO(), mockInvoker, mockInvocation)
	assert.NoError(t, result.Error())
}

func TestProviderFilter_Invoke(t *testing.T) {
	f := GetSentinelProviderFilter()
	url, err := common.NewURL("dubbo://127.0.0.1:20000/UserProvider?anyhost=true&" +
		"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&" +
		"environment=dev&interface=com.ikurento.user.UserProvider&ip=192.168.56.1&methods=GetUser%2C&" +
		"module=dubbogo+user-info+server&org=ikurento.com&owner=ZX&pid=1447&revision=0.0.1&" +
		"side=provider&timeout=3000&timestamp=1556509797245&bean.name=UserProvider")
	assert.NoError(t, err)
	mockInvoker := protocol.NewBaseInvoker(url)
	mockInvocation := invocation.NewRPCInvocation("hello", []interface{}{"OK"}, make(map[string]interface{}))
	result := f.Invoke(context.TODO(), mockInvoker, mockInvocation)
	assert.NoError(t, result.Error())
}

func TestGetResourceName(t *testing.T) {
	url, err := common.NewURL("dubbo://127.0.0.1:20000/UserProvider?anyhost=true&" +
		"version=1.0.0&group=myGroup&" +
		"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&" +
		"environment=dev&interface=com.ikurento.user.UserProvider&ip=192.168.56.1&methods=GetUser%2C&" +
		"module=dubbogo+user-info+server&org=ikurento.com&owner=ZX&pid=1447&revision=0.0.1&" +
		"side=provider&timeout=3000&timestamp=1556509797245&bean.name=UserProvider")
	assert.NoError(t, err)
	mockInvoker := protocol.NewBaseInvoker(url)
	methodResourceName := getResourceName(mockInvoker,
		invocation.NewRPCInvocation("hello", []interface{}{"OK"}, make(map[string]interface{})), "prefix_")
	assert.Equal(t, "prefix_com.ikurento.user.UserProvider:myGroup:1.0.0:hello()", methodResourceName)
}
