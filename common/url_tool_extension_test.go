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
	"github.com/apache/dubbo-go/common/constant"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDefaultURLTool(t *testing.T) {
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
	assert.False(t, GetURLTool().CompareURLEqual(&url1, &url2))
	assert.True(t, GetURLTool().CompareURLEqual(&url1, &url2, constant.TIMESTAMP_KEY, constant.REMOTE_TIMESTAMP_KEY))
}

func TestNewCustomURLTool(t *testing.T) {
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
	assert.True(t, GetURLTool().CompareURLEqual(&url1, &url2, constant.TIMESTAMP_KEY, constant.REMOTE_TIMESTAMP_KEY))
	SetURLTool(customURLTool{})
	assert.False(t, GetURLTool().CompareURLEqual(&url1, &url2))
	assert.False(t, GetURLTool().CompareURLEqual(&url1, &url2, constant.TIMESTAMP_KEY, constant.REMOTE_TIMESTAMP_KEY))

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
	assert.True(t, GetURLTool().CompareURLEqual(&url1, &url2))
	assert.True(t, GetURLTool().CompareURLEqual(&url1, &url2, constant.TIMESTAMP_KEY, constant.REMOTE_TIMESTAMP_KEY))
	SetURLTool(customURLTool{})
	assert.True(t, GetURLTool().CompareURLEqual(&url1, &url2))
	assert.True(t, GetURLTool().CompareURLEqual(&url1, &url2, constant.TIMESTAMP_KEY, constant.REMOTE_TIMESTAMP_KEY))
}

// just for no timestamp, it depend on write data.
type customURLTool struct {
}

func (customURLTool) Priority() uint8 {
	//default is 16.
	return 32
}
func (customURLTool) CompareURLEqual(l *URL, r *URL, execludeParam ...string) bool {
	return l.PrimitiveURL == r.PrimitiveURL
}
