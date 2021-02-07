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
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/cluster/router/chain"
	_ "github.com/apache/dubbo-go/cluster/router/condition"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
)

var (
	url, _ = common.NewURL(
		fmt.Sprintf("dubbo://%s:%d/com.ikurento.user.UserProvider", constant.LOCAL_HOST_VALUE, constant.DEFAULT_PORT))
	anyURL, _ = common.NewURL(fmt.Sprintf("condition://%s/com.foo.BarService", constant.ANYHOST_VALUE))
)

func TestNewBaseDirectory(t *testing.T) {
	dir := NewBaseDirectory(url)
	assert.Equal(t, url, dir.GetUrl())
	assert.Equal(t, url, dir.GetDirectoryUrl())
}

func TestBuildRouterChain(t *testing.T) {

	regURL := url
	regURL.AddParam(constant.INTERFACE_KEY, "mock-app")
	directory := NewBaseDirectory(regURL)
	var err error
	directory.routerChain, err = chain.NewRouterChain(regURL)
	assert.Nil(t, err)
	localIP := common.GetLocalIp()
	rule := base64.URLEncoding.EncodeToString([]byte("true => " + " host = " + localIP))
	routeURL := getRouteURL(rule, anyURL)
	routeURL.AddParam(constant.INTERFACE_KEY, "mock-app")
	routerURLs := make([]*common.URL, 0)
	routerURLs = append(routerURLs, routeURL)
	directory.SetRouters(routerURLs)
	chain := directory.RouterChain()

	assert.NotNil(t, chain)
}

func getRouteURL(rule string, u *common.URL) *common.URL {
	ru := u
	ru.AddParam("rule", rule)
	ru.AddParam("force", "true")
	ru.AddParam(constant.ROUTER_KEY, "router")
	return ru
}

func TestIsProperRouter(t *testing.T) {
	regURL := url
	regURL.AddParam(constant.APPLICATION_KEY, "mock-app")
	d := NewBaseDirectory(regURL)
	localIP := common.GetLocalIp()
	rule := base64.URLEncoding.EncodeToString([]byte("true => " + " host = " + localIP))
	routeURL := getRouteURL(rule, anyURL)
	routeURL.AddParam(constant.APPLICATION_KEY, "mock-app")
	rst := d.isProperRouter(routeURL)
	assert.True(t, rst)

	regURL.AddParam(constant.APPLICATION_KEY, "")
	regURL.AddParam(constant.INTERFACE_KEY, "com.foo.BarService")
	d = NewBaseDirectory(regURL)
	routeURL = getRouteURL(rule, anyURL)
	routeURL.AddParam(constant.INTERFACE_KEY, "com.foo.BarService")
	rst = d.isProperRouter(routeURL)
	assert.True(t, rst)

	regURL.AddParam(constant.APPLICATION_KEY, "")
	regURL.AddParam(constant.INTERFACE_KEY, "")
	d = NewBaseDirectory(regURL)
	routeURL = getRouteURL(rule, anyURL)
	rst = d.isProperRouter(routeURL)
	assert.True(t, rst)

	regURL.SetParam(constant.APPLICATION_KEY, "")
	regURL.SetParam(constant.INTERFACE_KEY, "")
	d = NewBaseDirectory(regURL)
	routeURL = getRouteURL(rule, anyURL)
	routeURL.AddParam(constant.APPLICATION_KEY, "mock-service")
	rst = d.isProperRouter(routeURL)
	assert.False(t, rst)

	regURL.SetParam(constant.APPLICATION_KEY, "")
	regURL.SetParam(constant.INTERFACE_KEY, "")
	d = NewBaseDirectory(regURL)
	routeURL = getRouteURL(rule, anyURL)
	routeURL.AddParam(constant.INTERFACE_KEY, "mock-service")
	rst = d.isProperRouter(routeURL)
	assert.False(t, rst)
}
