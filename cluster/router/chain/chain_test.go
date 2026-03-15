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

package chain

import (
	"testing"
)

import (
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

const (
	testServiceURL     = "dubbo://127.0.0.1:20000/com.xxx.Service?interface=com.xxx.Service"
	testServiceURLMain = testServiceURL + "&application=main-app"
	testSubURLSub      = "dubbo://127.0.0.1:20000/com.xxx.Service?application=sub-app"
)

func TestNewRouterChainApplicationKeyFromSubURL(t *testing.T) {
	mainURL, err := common.NewURL(testServiceURL)
	require.NoError(t, err)
	require.Empty(t, mainURL.GetParam(constant.ApplicationKey, ""))

	subURL, err := common.NewURL("dubbo://127.0.0.1:20000/com.xxx.Service?application=test-app")
	require.NoError(t, err)

	mainURL.SubURL = subURL

	_, err = NewRouterChain(mainURL)
	require.Error(t, err)
	require.Equal(t, "test-app", mainURL.GetParam(constant.ApplicationKey, ""))
}

func TestNewRouterChainApplicationKeyFromMainURL(t *testing.T) {
	mainURL, err := common.NewURL(testServiceURLMain)
	require.NoError(t, err)
	require.Equal(t, "main-app", mainURL.GetParam(constant.ApplicationKey, ""))

	subURL, err := common.NewURL(testSubURLSub)
	require.NoError(t, err)

	mainURL.SubURL = subURL

	_, err = NewRouterChain(mainURL)
	require.Error(t, err)
	require.Equal(t, "main-app", mainURL.GetParam(constant.ApplicationKey, ""))
}

func TestNewRouterChainNoApplicationKey(t *testing.T) {
	mainURL, err := common.NewURL(testServiceURL)
	require.NoError(t, err)
	require.Empty(t, mainURL.GetParam(constant.ApplicationKey, ""))

	subURL, err := common.NewURL(testServiceURL)
	require.NoError(t, err)

	mainURL.SubURL = subURL

	_, err = NewRouterChain(mainURL)
	require.Error(t, err)
}
