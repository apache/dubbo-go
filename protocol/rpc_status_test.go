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

package protocol

import (
	"strconv"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
)

const (
	mockCommonDubboUrl = "dubbo://192.168.10.10:20000/com.ikurento.user.UserProvider"
)

func TestBeginCount(t *testing.T) {
	defer CleanAllStatus()

	url, _ := common.NewURL(mockCommonDubboUrl)
	BeginCount(url, "test")
	urlStatus := GetURLStatus(url)
	methodStatus := GetMethodStatus(url, "test")
	methodStatus1 := GetMethodStatus(url, "test1")
	assert.Equal(t, int32(1), methodStatus.active)
	assert.Equal(t, int32(1), urlStatus.active)
	assert.Equal(t, int32(0), methodStatus1.active)

}

func TestEndCount(t *testing.T) {
	defer CleanAllStatus()

	url, _ := common.NewURL(mockCommonDubboUrl)
	EndCount(url, "test", 100, true)
	urlStatus := GetURLStatus(url)
	methodStatus := GetMethodStatus(url, "test")
	assert.Equal(t, int32(-1), methodStatus.active)
	assert.Equal(t, int32(-1), urlStatus.active)
	assert.Equal(t, int32(1), methodStatus.total)
	assert.Equal(t, int32(1), urlStatus.total)
}

func TestGetMethodStatus(t *testing.T) {
	defer CleanAllStatus()

	url, _ := common.NewURL(mockCommonDubboUrl)
	status := GetMethodStatus(url, "test")
	assert.NotNil(t, status)
	assert.Equal(t, int32(0), status.total)
}

func TestGetUrlStatus(t *testing.T) {
	defer CleanAllStatus()

	url, _ := common.NewURL(mockCommonDubboUrl)
	status := GetURLStatus(url)
	assert.NotNil(t, status)
	assert.Equal(t, int32(0), status.total)
}

func TestBeginCount0(t *testing.T) {
	defer CleanAllStatus()

	url, _ := common.NewURL(mockCommonDubboUrl)
	status := GetURLStatus(url)
	beginCount0(status)
	assert.Equal(t, int32(1), status.active)
}

func TestAll(t *testing.T) {
	defer CleanAllStatus()

	url, _ := common.NewURL(mockCommonDubboUrl)
	request(url, "test", 100, false, true)
	urlStatus := GetURLStatus(url)
	methodStatus := GetMethodStatus(url, "test")
	assert.Equal(t, int32(1), methodStatus.total)
	assert.Equal(t, int32(1), urlStatus.total)
	assert.Equal(t, int32(0), methodStatus.active)
	assert.Equal(t, int32(0), urlStatus.active)
	assert.Equal(t, int32(0), methodStatus.failed)
	assert.Equal(t, int32(0), urlStatus.failed)
	assert.Equal(t, int32(0), methodStatus.successiveRequestFailureCount)
	assert.Equal(t, int32(0), urlStatus.successiveRequestFailureCount)
	assert.Equal(t, int64(100), methodStatus.totalElapsed)
	assert.Equal(t, int64(100), urlStatus.totalElapsed)
	request(url, "test", 100, false, false)
	request(url, "test", 100, false, false)
	request(url, "test", 100, false, false)
	request(url, "test", 100, false, false)
	request(url, "test", 100, false, false)
	assert.Equal(t, int32(6), methodStatus.total)
	assert.Equal(t, int32(6), urlStatus.total)
	assert.Equal(t, int32(5), methodStatus.failed)
	assert.Equal(t, int32(5), urlStatus.failed)
	assert.Equal(t, int32(5), methodStatus.successiveRequestFailureCount)
	assert.Equal(t, int32(5), urlStatus.successiveRequestFailureCount)
	assert.Equal(t, int64(600), methodStatus.totalElapsed)
	assert.Equal(t, int64(600), urlStatus.totalElapsed)
	assert.Equal(t, int64(500), methodStatus.failedElapsed)
	assert.Equal(t, int64(500), urlStatus.failedElapsed)

	request(url, "test", 100, false, true)
	assert.Equal(t, int32(0), methodStatus.successiveRequestFailureCount)
	assert.Equal(t, int32(0), urlStatus.successiveRequestFailureCount)

	request(url, "test", 200, false, false)
	request(url, "test", 200, false, false)
	assert.Equal(t, int32(2), methodStatus.successiveRequestFailureCount)
	assert.Equal(t, int32(2), urlStatus.successiveRequestFailureCount)
	assert.Equal(t, int64(200), methodStatus.maxElapsed)
	assert.Equal(t, int64(200), urlStatus.maxElapsed)

	request(url, "test1", 200, false, false)
	request(url, "test1", 200, false, false)
	request(url, "test1", 200, false, false)
	assert.Equal(t, int32(5), urlStatus.successiveRequestFailureCount)
	methodStatus1 := GetMethodStatus(url, "test1")
	assert.Equal(t, int32(2), methodStatus.successiveRequestFailureCount)
	assert.Equal(t, int32(3), methodStatus1.successiveRequestFailureCount)

}

func request(url *common.URL, method string, elapsed int64, active, succeeded bool) {
	BeginCount(url, method)
	if !active {
		EndCount(url, method, elapsed, succeeded)
	}
}

func TestCurrentTimeMillis(t *testing.T) {
	defer CleanAllStatus()
	c := CurrentTimeMillis()
	assert.NotNil(t, c)
	str := strconv.FormatInt(c, 10)
	i, _ := strconv.ParseInt(str, 10, 64)
	assert.Equal(t, c, i)
}
