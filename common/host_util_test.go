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

func TestGetLocalHostName(t *testing.T) {
	assert.NotNil(t, GetLocalHostName())
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

func TestGetRandomPort(t *testing.T) {
	port := GetRandomPort("")
	assert.True(t, port != "")
}

func TestIsMatchGlobPattern(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		value    string
		expected bool
	}{
		// * matches anything
		{"any pattern", "*", "anything", true},
		{"any pattern empty value", "*", "", true},

		// both empty
		{"both empty", "", "", true},

		// one empty
		{"pattern empty", "", "value", false},
		{"value empty", "pattern", "", false},

		// no wildcard - exact match
		{"exact match", "hello", "hello", true},
		{"exact not match", "hello", "world", false},

		// * at the end - prefix match
		{"prefix match", "hello*", "hello world", true},
		{"prefix exact", "hello*", "hello", true},
		{"prefix not match", "hello*", "world", false},

		// * at the beginning - suffix match
		{"suffix match", "*world", "hello world", true},
		{"suffix exact", "*world", "world", true},
		{"suffix not match", "*world", "hello", false},

		// * in the middle - prefix and suffix match
		{"middle match", "hello*world", "hello beautiful world", true},
		{"middle exact", "hello*world", "helloworld", true},
		{"middle prefix not match", "hello*world", "hi beautiful world", false},
		{"middle suffix not match", "hello*world", "hello beautiful earth", false},
		{"middle both not match", "hello*world", "hi earth", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsMatchGlobPattern(tt.pattern, tt.value))
		})
	}
}

func TestGetLocalIpCached(t *testing.T) {
	// First call
	ip1 := GetLocalIp()
	assert.NotEmpty(t, ip1)

	// Second call should return cached value
	ip2 := GetLocalIp()
	assert.Equal(t, ip1, ip2)
}

func TestGetLocalHostNameCached(t *testing.T) {
	// First call
	hostname1 := GetLocalHostName()
	assert.NotEmpty(t, hostname1)

	// Second call should return cached value
	hostname2 := GetLocalHostName()
	assert.Equal(t, hostname1, hostname2)
}
