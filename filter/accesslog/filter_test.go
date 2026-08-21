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

package accesslog

import (
	"context"
	"testing"
)

import (
	"github.com/golang/mock/gomock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

func TestFilter_Invoke_Not_Config(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	url, _ := common.NewURL(
		"dubbo://:20000/UserProvider?app.version=0.0.1&application=BDTService&bean.name=UserProvider" +
			"&cluster=failover&environment=dev&group=&interface=com.ikurento.user.UserProvider&loadbalance=random&methods.GetUser." +
			"loadbalance=random&methods.GetUser.retries=1&methods.GetUser.weight=0&module=dubbogo+user-info+server&name=" +
			"BDTService&organization=ikurento.com&owner=ZX&registry.role=3&retries=&" +
			"service.filter=echo%2Ctoken%2Caccesslog&timestamp=1569153406&token=934804bf-b007-4174-94eb-96e3e1d60cc7&version=&warmup=100")
	invoker := base.NewBaseInvoker(url)

	attach := make(map[string]any, 10)
	inv := invocation.NewRPCInvocation("MethodName", []any{"OK", "Hello"}, attach)

	filter := &Filter{}
	invokeResult := filter.Invoke(context.Background(), invoker, inv)
	assert.NoError(t, invokeResult.Error())
}

func TestFilterInvokeDefaultConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	url, _ := common.NewURL(
		"dubbo://:20000/UserProvider?app.version=0.0.1&application=BDTService&bean.name=UserProvider" +
			"&cluster=failover&accesslog=true&environment=dev&group=&interface=com.ikurento.user.UserProvider&loadbalance=random&methods.GetUser." +
			"loadbalance=random&methods.GetUser.retries=1&methods.GetUser.weight=0&module=dubbogo+user-info+server&name=" +
			"BDTService&organization=ikurento.com&owner=ZX&registry.role=3&retries=&" +
			"service.filter=echo%2Ctoken%2Caccesslog&timestamp=1569153406&token=934804bf-b007-4174-94eb-96e3e1d60cc7&version=&warmup=100")
	invoker := base.NewBaseInvoker(url)

	attach := make(map[string]any, 10)
	attach[constant.VersionKey] = "1.0"
	attach[constant.GroupKey] = "MyGroup"
	inv := invocation.NewRPCInvocation("MethodName", []any{"OK", "Hello"}, attach)

	filter := &Filter{}
	invokeResult := filter.Invoke(context.Background(), invoker, inv)
	assert.NoError(t, invokeResult.Error())
}

func TestFilterOnResponse(t *testing.T) {
	rpcResult := &result.RPCResult{}
	filter := &Filter{}
	response := filter.OnResponse(context.TODO(), rpcResult, nil, nil)
	assert.Equal(t, rpcResult, response)
}

func TestBuildAccessLogDataSkipsNonStringAttachments(t *testing.T) {
	attachments := map[string]any{
		constant.InterfaceKey: 42,
		constant.PathKey:      "fallback.Service",
		constant.MethodKey:    true,
		constant.VersionKey:   1,
		constant.GroupKey:     []string{"group"},
		constant.TimestampKey: 1234567890,
		constant.LocalAddr:    struct{}{},
		constant.RemoteAddr:   nil,
	}
	inv := invocation.NewRPCInvocation("MethodName", nil, attachments)

	var data map[string]string
	require.NotPanics(t, func() {
		data = (&Filter{}).buildAccessLogData(nil, inv)
	})
	assert.Equal(t, "fallback.Service", data[constant.InterfaceKey])
	assert.NotContains(t, data, constant.MethodKey)
	assert.NotContains(t, data, constant.VersionKey)
	assert.NotContains(t, data, constant.GroupKey)
	assert.NotContains(t, data, constant.TimestampKey)
	assert.NotContains(t, data, constant.LocalAddr)
	assert.NotContains(t, data, constant.RemoteAddr)
}
