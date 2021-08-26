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
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config"
)

func TestNewNacosClient(t *testing.T) {
	rc := &config.RemoteConfig{}
	rc.Protocol = "nacos"
	rc.Username = "nacos"
	client, err := NewNacosClient(rc)

	// address is nil
	assert.Nil(t, client)
	assert.NotNil(t, err)

	rc.Address = "console.nacos.io:80:123"
	client, err = NewNacosClient(rc)
	// invalid address
	assert.Nil(t, client)
	assert.NotNil(t, err)

	rc.Address = "console.nacos.io:80"
	rc.TimeoutStr = "10s"
	client, err = NewNacosClient(rc)
	assert.NotNil(t, client)
	assert.Nil(t, err)
}

func TestGetNacosConfig(t *testing.T) {
	regurl := getRegUrl()
	sc, cc, err := GetNacosConfig(regurl)

	assert.Nil(t, err)
	assert.NotNil(t, sc)
	assert.NotNil(t, cc)
	assert.Equal(t, cc.TimeoutMs, uint64(5000))
}

func TestNewNacosConfigClient(t *testing.T) {

	regurl := getRegUrl()
	client, err := NewNacosConfigClientByUrl(regurl)

	assert.Nil(t, err)
	assert.NotNil(t, client)
}

func TestNewNacosClientByUrl(t *testing.T) {
	regurl := getRegUrl()
	client, err := NewNacosClientByUrl(regurl)

	assert.Nil(t, err)
	assert.NotNil(t, client)
}

func TestTimeoutConfig(t *testing.T) {
	regurlMap := url.Values{}
	regurlMap.Set(constant.NACOS_NOT_LOAD_LOCAL_CACHE, "true")
	// regurlMap.Set(constant.NACOS_USERNAME, "nacos")
	// regurlMap.Set(constant.NACOS_PASSWORD, "nacos")
	regurlMap.Set(constant.NACOS_NAMESPACE_ID, "nacos")

	t.Run("default timeout", func(t *testing.T) {
		newURL, _ := common.NewURL("registry://console.nacos.io:80", common.WithParams(regurlMap))

		_, cc, err := GetNacosConfig(newURL)
		assert.Nil(t, err)

		assert.Equal(t, cc.TimeoutMs, uint64(int32(10*time.Second/time.Millisecond)))
	})

	t.Run("right timeout", func(t *testing.T) {

		regurlMap.Set(constant.CONFIG_TIMEOUT_KEY, "5s")

		newURL, _ := common.NewURL("registry://console.nacos.io:80", common.WithParams(regurlMap))

		_, cc, err := GetNacosConfig(newURL)
		assert.Nil(t, err)

		assert.Equal(t, cc.TimeoutMs, uint64(int32(5*time.Second/time.Millisecond)))
	})

	t.Run("invalid timeout", func(t *testing.T) {
		regurlMap.Set(constant.CONFIG_TIMEOUT_KEY, "5ab")

		newURL, _ := common.NewURL("registry://console.nacos.io:80", common.WithParams(regurlMap))
		_, cc, err := GetNacosConfig(newURL)
		assert.Nil(t, err)

		assert.Equal(t, cc.TimeoutMs, uint64(int32(3*time.Second/time.Millisecond)))
	})

}

func getRegUrl() *common.URL {

	regurlMap := url.Values{}
	regurlMap.Set(constant.NACOS_NOT_LOAD_LOCAL_CACHE, "true")
	// regurlMap.Set(constant.NACOS_USERNAME, "nacos")
	// regurlMap.Set(constant.NACOS_PASSWORD, "nacos")
	regurlMap.Set(constant.NACOS_NAMESPACE_ID, "nacos")
	regurlMap.Set(constant.CONFIG_TIMEOUT_KEY, "5s")

	regurl, _ := common.NewURL("registry://console.nacos.io:80", common.WithParams(regurlMap))

	return regurl
}
