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
	"os"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

func TestGetLocalIp(t *testing.T) {
	assert.NotNil(t, GetLocalIp())
}

func TestHandleRegisterIPAndPort(t *testing.T) {
	url := NewURLWithOptions(WithIp("1.2.3.4"), WithPort("20000"))
	HandleRegisterIPAndPort(url)
	assert.Equal(t, "1.2.3.4", url.Ip)
	assert.Equal(t, "20000", url.Port)
}

func TestHandleRegisterIPAndPortBlank(t *testing.T) {
	url, _ := NewURL("")
	HandleRegisterIPAndPort(url)
	assert.Equal(t, GetLocalIp(), url.Ip)
	assert.Equal(t, constant.DubboDefaultPortToRegistry, url.Port)
}

func TestHandleRegisterIPAndPortWithEnv(t *testing.T) {
	url, _ := NewURL("")
	_ = os.Setenv(constant.DubboIpToRegistryKey, "1.2.3.4")
	_ = os.Setenv(constant.DubboPortToRegistryKey, "20000")
	HandleRegisterIPAndPort(url)
	assert.Equal(t, "1.2.3.4", url.Ip)
	assert.Equal(t, "20000", url.Port)
}

func TestHandleRegisterIPAndPortWithEnvInvalidPort(t *testing.T) {
	url, _ := NewURL("")
	_ = os.Setenv(constant.DubboIpToRegistryKey, "1.2.3.4")
	_ = os.Setenv(constant.DubboPortToRegistryKey, "0")
	HandleRegisterIPAndPort(url)
	assert.Equal(t, "1.2.3.4", url.Ip)
	assert.Equal(t, constant.DubboDefaultPortToRegistry, url.Port)
}

func TestIsValidPort(t *testing.T) {
	assert.Equal(t, false, isValidPort(""))
	assert.Equal(t, false, isValidPort("abc"))
	assert.Equal(t, false, isValidPort("0"))
	assert.Equal(t, false, isValidPort("65536"))
	assert.Equal(t, true, isValidPort("20000"))
}
