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

package loadbalance

import (
	"net/url"
	"strconv"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

func TestGetWeightDefault(t *testing.T) {
	u, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService")
	require.NoError(t, err)
	invoker := base.NewBaseInvoker(u)
	inv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("GetUser"))

	weight := GetWeight(invoker, inv)
	assert.Equal(t, int64(constant.DefaultWeight), weight)
}

func TestGetWeightWithMethodLevel(t *testing.T) {
	urlParams := url.Values{}
	urlParams.Set("methods.GetUser."+constant.WeightKey, "200")
	u, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService", common.WithParams(urlParams))
	require.NoError(t, err)
	invoker := base.NewBaseInvoker(u)
	inv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("GetUser"))

	weight := GetWeight(invoker, inv)
	assert.Equal(t, int64(200), weight)
}

func TestGetWeightWithRegistryLevel(t *testing.T) {
	urlParams := url.Values{}
	urlParams.Set(constant.RegistryKey+"."+constant.RegistryLabelKey, "true")
	urlParams.Set(constant.RegistryKey+"."+constant.WeightKey, "150")
	u, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService", common.WithParams(urlParams))
	require.NoError(t, err)
	invoker := base.NewBaseInvoker(u)
	inv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("GetUser"))

	weight := GetWeight(invoker, inv)
	assert.Equal(t, int64(150), weight)
}

func TestGetWeightMethodLevelPriority(t *testing.T) {
	urlParams := url.Values{}
	urlParams.Set("methods.GetUser."+constant.WeightKey, "200")
	urlParams.Set(constant.RegistryKey+"."+constant.RegistryLabelKey, "true")
	urlParams.Set(constant.RegistryKey+"."+constant.WeightKey, "150")
	u, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService", common.WithParams(urlParams))
	require.NoError(t, err)
	invoker := base.NewBaseInvoker(u)
	inv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("GetUser"))

	weight := GetWeight(invoker, inv)
	// Method level should have higher priority
	assert.Equal(t, int64(200), weight)
}

func TestGetWeightWithWarmup(t *testing.T) {
	now := time.Now().Unix()
	// Set timestamp to 5 minutes ago (300 seconds)
	uptime := int64(300)
	timestamp := now - uptime

	urlParams := url.Values{}
	urlParams.Set(constant.RemoteTimestampKey, strconv.FormatInt(timestamp, 10))
	urlParams.Set(constant.WarmupKey, strconv.Itoa(constant.DefaultWarmup)) // 10 minutes = 600 seconds
	urlParams.Set("methods.GetUser."+constant.WeightKey, "100")
	u, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService", common.WithParams(urlParams))
	require.NoError(t, err)
	invoker := base.NewBaseInvoker(u)
	inv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("GetUser"))

	weight := GetWeight(invoker, inv)
	// Weight should be reduced due to warmup
	// Expected: (300 * 100) / 600 = 50
	assert.Equal(t, int64(50), weight)
}

func TestGetWeightWarmupComplete(t *testing.T) {
	now := time.Now().Unix()
	// Set timestamp to 11 minutes ago (660 seconds), beyond warmup period
	uptime := int64(660)
	timestamp := now - uptime

	urlParams := url.Values{}
	urlParams.Set(constant.RemoteTimestampKey, strconv.FormatInt(timestamp, 10))
	urlParams.Set(constant.WarmupKey, strconv.Itoa(constant.DefaultWarmup)) // 10 minutes = 600 seconds
	urlParams.Set("methods.GetUser."+constant.WeightKey, "100")
	u, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService", common.WithParams(urlParams))
	require.NoError(t, err)
	invoker := base.NewBaseInvoker(u)
	inv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("GetUser"))

	weight := GetWeight(invoker, inv)
	// Warmup complete, should return full weight
	assert.Equal(t, int64(100), weight)
}

func TestGetWeightWarmupLessThanOne(t *testing.T) {
	now := time.Now().Unix()
	// Set timestamp to 1 second ago
	uptime := int64(1)
	timestamp := now - uptime

	urlParams := url.Values{}
	urlParams.Set(constant.RemoteTimestampKey, strconv.FormatInt(timestamp, 10))
	urlParams.Set(constant.WarmupKey, strconv.Itoa(constant.DefaultWarmup))
	urlParams.Set("methods.GetUser."+constant.WeightKey, "100")
	u, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService", common.WithParams(urlParams))
	require.NoError(t, err)
	invoker := base.NewBaseInvoker(u)
	inv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("GetUser"))

	weight := GetWeight(invoker, inv)
	// Weight should be at least 1
	assert.Equal(t, int64(1), weight)
}

func TestGetWeightZero(t *testing.T) {
	urlParams := url.Values{}
	urlParams.Set("methods.GetUser."+constant.WeightKey, "0")
	u, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService", common.WithParams(urlParams))
	require.NoError(t, err)
	invoker := base.NewBaseInvoker(u)
	inv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("GetUser"))

	weight := GetWeight(invoker, inv)
	assert.Equal(t, int64(0), weight)
}

func TestGetWeightNegative(t *testing.T) {
	urlParams := url.Values{}
	urlParams.Set("methods.GetUser."+constant.WeightKey, "-10")
	u, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService", common.WithParams(urlParams))
	require.NoError(t, err)
	invoker := base.NewBaseInvoker(u)
	inv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("GetUser"))

	weight := GetWeight(invoker, inv)
	// Negative weight is treated as unspecified, so falls back to default weight
	assert.Equal(t, int64(constant.DefaultWeight), weight)
}

func TestGetWeightCustomWarmup(t *testing.T) {
	now := time.Now().Unix()
	// Set timestamp to 2 minutes ago (120 seconds)
	uptime := int64(120)
	timestamp := now - uptime
	customWarmup := 300 // 5 minutes

	urlParams := url.Values{}
	urlParams.Set(constant.RemoteTimestampKey, strconv.FormatInt(timestamp, 10))
	urlParams.Set(constant.WarmupKey, strconv.Itoa(customWarmup))
	urlParams.Set("methods.GetUser."+constant.WeightKey, "150")
	u, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService", common.WithParams(urlParams))
	require.NoError(t, err)
	invoker := base.NewBaseInvoker(u)
	inv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("GetUser"))

	weight := GetWeight(invoker, inv)
	// Expected: (120 * 150) / 300 = 60
	assert.Equal(t, int64(60), weight)
}

func TestGetWeightNoTimestamp(t *testing.T) {
	urlParams := url.Values{}
	urlParams.Set("methods.GetUser."+constant.WeightKey, "100")
	u, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService", common.WithParams(urlParams))
	require.NoError(t, err)
	invoker := base.NewBaseInvoker(u)
	inv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("GetUser"))

	weight := GetWeight(invoker, inv)
	// Without timestamp, uptime is 0, warmup doesn't apply
	assert.Equal(t, int64(100), weight)
}

func TestGetWeightDifferentMethods(t *testing.T) {
	urlParams := url.Values{}
	urlParams.Set("methods.GetUser."+constant.WeightKey, "100")
	urlParams.Set("methods.UpdateUser."+constant.WeightKey, "200")
	u, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService", common.WithParams(urlParams))
	require.NoError(t, err)
	invoker := base.NewBaseInvoker(u)

	inv1 := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("GetUser"))
	weight1 := GetWeight(invoker, inv1)
	assert.Equal(t, int64(100), weight1)

	inv2 := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("UpdateUser"))
	weight2 := GetWeight(invoker, inv2)
	assert.Equal(t, int64(200), weight2)
}

func TestGetWeightRegistryLabelFalse(t *testing.T) {
	urlParams := url.Values{}
	urlParams.Set(constant.RegistryKey+"."+constant.RegistryLabelKey, "false")
	urlParams.Set(constant.RegistryKey+"."+constant.WeightKey, "150")
	u, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService", common.WithParams(urlParams))
	require.NoError(t, err)
	invoker := base.NewBaseInvoker(u)
	inv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("GetUser"))

	weight := GetWeight(invoker, inv)
	// Registry label is false, should use default weight
	assert.Equal(t, int64(constant.DefaultWeight), weight)
}
