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

package polaris

import (
	"testing"
)

import (
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
)

const (
	baseServiceURL    = "dubbo://127.0.0.1:20000/com.xxx.Service"
	baseSubServiceURL = "dubbo://127.0.0.1:20001/com.xxx.Service"
	serviceURLWithApp = baseServiceURL + "?application=param-app"
	errMissingAppName = "polaris router must set application name"
	attrAppName       = "attr-app"
	subAttrAppName    = "sub-attr-app"
)

func mustNewURL(t *testing.T, rawURL string) *common.URL {
	t.Helper()
	url, err := common.NewURL(rawURL)
	require.NoError(t, err)
	return url
}

func TestNewPolarisRouterApplicationNameFromParam(t *testing.T) {
	url := mustNewURL(t, serviceURLWithApp)

	r, err := newPolarisRouter(url)
	require.NoError(t, err)
	require.Equal(t, "param-app", r.currentApplication)
}

func TestNewPolarisRouterApplicationNameFromAttribute(t *testing.T) {
	url := mustNewURL(t, baseServiceURL)
	url.SetAttribute(constant.ApplicationKey, &global.ApplicationConfig{Name: attrAppName})

	r, err := newPolarisRouter(url)
	require.NoError(t, err)
	require.Equal(t, attrAppName, r.currentApplication)
}

func TestNewPolarisRouterApplicationNameFromSubURLAttribute(t *testing.T) {
	url := mustNewURL(t, baseServiceURL)
	subURL := mustNewURL(t, baseSubServiceURL)
	subURL.SetAttribute(constant.ApplicationKey, &global.ApplicationConfig{Name: subAttrAppName})
	url.SubURL = subURL

	r, err := newPolarisRouter(url)
	require.NoError(t, err)
	require.Equal(t, subAttrAppName, r.currentApplication)
}

func TestNewPolarisRouterApplicationNameMissing(t *testing.T) {
	url := mustNewURL(t, baseServiceURL)

	_, err := newPolarisRouter(url)
	require.EqualError(t, err, errMissingAppName)
}

func TestNewPolarisRouterApplicationNameFromSubURLParam(t *testing.T) {
	url := mustNewURL(t, baseServiceURL)
	subURL := mustNewURL(t, serviceURLWithApp)
	url.SubURL = subURL

	r, err := newPolarisRouter(url)
	require.NoError(t, err)
	require.Equal(t, "param-app", r.currentApplication)
}
