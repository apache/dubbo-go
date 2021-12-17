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

package base

import (
	"encoding/base64"
	"fmt"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/router/chain"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

var (
	url, _ = common.NewURL(
		fmt.Sprintf("dubbo://%s:%d/com.ikurento.user.UserProvider", constant.LocalHostValue, constant.DefaultPort))
	anyURL, _ = common.NewURL(fmt.Sprintf("condition://%s/com.foo.BarService", constant.AnyHostValue))
)

func TestNewBaseDirectory(t *testing.T) {
	dir := NewDirectory(url)
	assert.Equal(t, url, dir.GetURL())
	assert.Equal(t, url, dir.GetDirectoryUrl())
}

func TestBuildRouterChain(t *testing.T) {
	regURL := url
	regURL.AddParam(constant.InterfaceKey, "mock-app")
	directory := NewDirectory(regURL)
	var err error
	directory.routerChain, err = chain.NewRouterChain()
	assert.Error(t, err)
}

func getRouteURL(rule string, u *common.URL) *common.URL {
	ru := u
	ru.AddParam("rule", rule)
	ru.AddParam("force", "true")
	ru.AddParam(constant.RouterKey, "router")
	return ru
}

func TestIsProperRouter(t *testing.T) {
	regURL := url
	regURL.AddParam(constant.ApplicationKey, "mock-app")
	d := NewDirectory(regURL)
	localIP := common.GetLocalIp()
	rule := base64.URLEncoding.EncodeToString([]byte("true => " + " host = " + localIP))
	routeURL := getRouteURL(rule, anyURL)
	routeURL.AddParam(constant.ApplicationKey, "mock-app")
	rst := d.isProperRouter(routeURL)
	assert.True(t, rst)

	regURL.AddParam(constant.ApplicationKey, "")
	regURL.AddParam(constant.InterfaceKey, "com.foo.BarService")
	d = NewDirectory(regURL)
	routeURL = getRouteURL(rule, anyURL)
	routeURL.AddParam(constant.InterfaceKey, "com.foo.BarService")
	rst = d.isProperRouter(routeURL)
	assert.True(t, rst)

	regURL.AddParam(constant.ApplicationKey, "")
	regURL.AddParam(constant.InterfaceKey, "")
	d = NewDirectory(regURL)
	routeURL = getRouteURL(rule, anyURL)
	rst = d.isProperRouter(routeURL)
	assert.True(t, rst)

	regURL.SetParam(constant.ApplicationKey, "")
	regURL.SetParam(constant.InterfaceKey, "")
	d = NewDirectory(regURL)
	routeURL = getRouteURL(rule, anyURL)
	routeURL.AddParam(constant.ApplicationKey, "mock-service")
	rst = d.isProperRouter(routeURL)
	assert.False(t, rst)

	regURL.SetParam(constant.ApplicationKey, "")
	regURL.SetParam(constant.InterfaceKey, "")
	d = NewDirectory(regURL)
	routeURL = getRouteURL(rule, anyURL)
	routeURL.AddParam(constant.InterfaceKey, "mock-service")
	rst = d.isProperRouter(routeURL)
	assert.False(t, rst)
}
