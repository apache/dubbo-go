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
package invocation

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
)

func TestRPCInvocation_ServiceKey(t *testing.T) {
	providerURL := "dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?anyhost=true&" +
		"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&" +
		"environment=dev&interface=com.ikurento.user.UserProvider&ip=192.168.56.1&methods=GetUser%2C&" +
		"module=dubbogo+user-info+server&org=ikurento.com&owner=ZX&pid=1447&revision=0.0.1&" +
		"side=provider&timeout=3000&timestamp=1556509797245"

	sameInfPathConsumerURL := "dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?anyhost=true&" +
		"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&" +
		"environment=dev&interface=com.ikurento.user.UserProvider&ip=192.168.56.1&methods=GetUser%2C&" +
		"module=dubbogo+user-info+server&org=ikurento.com&owner=ZX&pid=1447&revision=0.0.1&" +
		"side=provider&timeout=3000&timestamp=1556509797245"
	diffInfPathConsumerURL := "dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?anyhost=true&" +
		"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&" +
		"environment=dev&interface=com.ikurento.user.UserProviderFoo&ip=192.168.56.1&methods=GetUser%2C&" +
		"module=dubbogo+user-info+server&org=ikurento.com&owner=ZX&pid=1447&revision=0.0.1&" +
		"side=provider&timeout=3000&timestamp=1556509797245"

	providerUrl, err := common.NewURL(providerURL)
	assert.NoError(t, err)

	// invocation with same interface and path value
	sameInfPathConsumerUrl, err := common.NewURL(sameInfPathConsumerURL)
	assert.NoError(t, err)
	invocation := NewRPCInvocationWithOptions(WithAttachments(map[string]interface{}{
		constant.INTERFACE_KEY: sameInfPathConsumerUrl.GetParam(constant.INTERFACE_KEY, ""),
		constant.PATH_KEY:      sameInfPathConsumerUrl.Path,
		constant.GROUP_KEY:     sameInfPathConsumerUrl.GetParam(constant.GROUP_KEY, ""),
		constant.VERSION_KEY:   sameInfPathConsumerUrl.GetParam(constant.VERSION_KEY, ""),
	}))
	assert.Equal(t, providerUrl.ServiceKey(), invocation.ServiceKey())

	// invocation with different interface and path value
	diffInfPathConsumerUrl, err := common.NewURL(diffInfPathConsumerURL)
	assert.NoError(t, err)
	invocation = NewRPCInvocationWithOptions(WithAttachments(map[string]interface{}{
		constant.INTERFACE_KEY: diffInfPathConsumerUrl.GetParam(constant.INTERFACE_KEY, ""),
		constant.PATH_KEY:      diffInfPathConsumerUrl.Path,
		constant.GROUP_KEY:     diffInfPathConsumerUrl.GetParam(constant.GROUP_KEY, ""),
		constant.VERSION_KEY:   diffInfPathConsumerUrl.GetParam(constant.VERSION_KEY, ""),
	}))
	assert.Equal(t, providerUrl.ServiceKey(), invocation.ServiceKey())
}
