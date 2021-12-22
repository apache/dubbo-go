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
	"net/url"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config"
)

func TestGetPolarisConfig(t *testing.T) {

	rc := &config.RemoteConfig{}
	rc.Params = make(map[string]string)

	rc.Protocol = "polaris"
	rc.Address = "127.0.0.1:8091"

	rc.Params[constant.PolarisNamespace] = "default"

	url, err := rc.ToURL()
	if err != nil {
		t.Fatal(err)
	}

	sdkCtx, namespace, err := GetPolarisConfig(url)

	if err != nil {
		t.Fatal(err)
	}
	assert.NotNil(t, sdkCtx, "SDKContext")

	assert.Equal(t, "default", namespace, "namespace")
	assert.ElementsMatch(t, []string{"127.0.0.1:8091"}, sdkCtx.GetConfig().GetGlobal().GetServerConnector().GetAddresses(), "server address")
}

func TestGetPolarisConfigWithExternalFile(t *testing.T) {

	rc := &config.RemoteConfig{}
	rc.Params = make(map[string]string)

	rc.Protocol = "polaris"
	rc.Address = "127.0.0.1:8091"

	rc.Params[constant.PolarisNamespace] = "default"
	rc.Params[constant.PolarisConfigFilePath] = "./polaris.yaml"

	url, err := rc.ToURL()
	if err != nil {
		t.Fatal(err)
	}

	sdkCtx, namespace, err := GetPolarisConfig(url)

	if err != nil {
		t.Fatal(err)
	}
	assert.NotNil(t, sdkCtx, "SDKContext")

	assert.Equal(t, "default", namespace, "namespace")
	assert.ElementsMatch(t, []string{"127.0.0.1:8091", "127.0.0.2:8091"}, sdkCtx.GetConfig().GetGlobal().GetServerConnector().GetAddresses(), "server address")
}

func TestGetPolarisConfigByUrl(t *testing.T) {
	regurl := getRegUrl()
	sdkCtx, namespace, err := GetPolarisConfig(regurl)

	assert.Nil(t, err)
	assert.Equal(t, "default", namespace, "namespace")
	assert.ElementsMatch(t, []string{"127.0.0.1:8091", "127.0.0.2:8091"}, sdkCtx.GetConfig().GetGlobal().GetServerConnector().GetAddresses(), "server address")
}

func getRegUrl() *common.URL {

	regurlMap := url.Values{}
	regurlMap.Set(constant.PolarisNamespace, "default")
	regurlMap.Set(constant.PolarisConfigFilePath, "./polaris.yaml")

	regurl, _ := common.NewURL("registry://127.0.0.1:8091", common.WithParams(regurlMap))

	return regurl
}
