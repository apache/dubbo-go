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

package nacos

import (
	"net/url"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
)

func TestNacosServiceDiscovery_Destroy(t *testing.T) {
	serviceDiscovry, err := extension.GetServiceDiscovery(constant.NACOS_KEY, mockUrl())
	assert.Nil(t, err)
	assert.NotNil(t, serviceDiscovry)
}

func mockUrl() *common.URL {
	urlMap := url.Values{}
	urlMap.Set(constant.GROUP_KEY, "guangzhou-idc")
	urlMap.Set(constant.ROLE_KEY, strconv.Itoa(common.PROVIDER))
	urlMap.Set(constant.INTERFACE_KEY, "com.ikurento.user.UserProvider")
	urlMap.Set(constant.VERSION_KEY, "1.0.0")
	urlMap.Set(constant.CLUSTER_KEY, "mock")
	url, _ := common.NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider", common.WithParams(urlMap), common.WithMethods([]string{"GetUser", "AddUser"}))
	return &url
}
