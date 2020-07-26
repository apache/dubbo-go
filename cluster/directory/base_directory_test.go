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

package directory

import (
	"encoding/base64"
	"fmt"
	"testing"
)

import (
	gxnet "github.com/dubbogo/gost/net"
	"github.com/stretchr/testify/assert"
)

import (
	_ "github.com/apache/dubbo-go/cluster/router/condition"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
)

var (
	url, _ = common.NewURL(
		fmt.Sprintf("dubbo://%s:%d/com.ikurento.user.UserProvider", constant.LOCAL_HOST_VALUE, constant.DEFAULT_PORT))
	anyUrl, _ = common.NewURL(fmt.Sprintf("condition://%s/com.foo.BarService", constant.ANYHOST_VALUE))
)

func TestNewBaseDirectory(t *testing.T) {
	directory := NewBaseDirectory(&url)
	assert.NotNil(t, directory)
	assert.Equal(t, url, directory.GetUrl())
	assert.Equal(t, &url, directory.GetDirectoryUrl())
}

func TestBuildRouterChain(t *testing.T) {

	regURL := url
	regURL.AddParam(constant.INTERFACE_KEY, "mock-app")
	directory := NewBaseDirectory(&regURL)

	assert.NotNil(t, directory)

	localIP, _ := gxnet.GetLocalIP()
	rule := base64.URLEncoding.EncodeToString([]byte("true => " + " host = " + localIP))
	routeURL := getRouteURL(rule)
	routeURL.AddParam(constant.INTERFACE_KEY, "mock-app")
	routerURLs := make([]*common.URL, 0)
	routerURLs = append(routerURLs, routeURL)
	directory.SetRouters(routerURLs)
	chain := directory.RouterChain()

	assert.NotNil(t, chain)
}

func getRouteURL(rule string) *common.URL {
	anyUrl.AddParam("rule", rule)
	anyUrl.AddParam("force", "true")
	anyUrl.AddParam(constant.ROUTER_KEY, "router")
	return &anyUrl
}
