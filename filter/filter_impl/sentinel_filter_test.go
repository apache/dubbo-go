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

package filter_impl

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
)

import (
	"github.com/alibaba/sentinel-golang/core/flow"
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/protocol/invocation"
)

func TestSentinelFilter_QPS(t *testing.T) {
	url, err := common.NewURL("dubbo://127.0.0.1:20000/UserProvider?anyhost=true&" +
		"version=1.0.0&group=myGroup&" +
		"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&" +
		"environment=dev&interface=com.ikurento.user.UserProvider&ip=192.168.56.1&methods=GetUser%2C&" +
		"module=dubbogo+user-info+server&org=ikurento.com&owner=ZX&pid=1447&revision=0.0.1&" +
		"side=provider&timeout=3000&timestamp=1556509797245&bean.name=UserProvider")
	assert.NoError(t, err)
	mockInvoker := protocol.NewBaseInvoker(url)
	interfaceResourceName, _ := getResourceName(mockInvoker,
		invocation.NewRPCInvocation("hello", []interface{}{"OK"}, make(map[string]interface{})), "prefix_")
	mockInvocation := invocation.NewRPCInvocation("hello", []interface{}{"OK"}, make(map[string]interface{}))

	_, err = flow.LoadRules([]*flow.Rule{
		{
			Resource:               interfaceResourceName,
			MetricType:             flow.QPS,
			TokenCalculateStrategy: flow.Direct,
			ControlBehavior:        flow.Reject,
			Count:                  100,
			RelationStrategy:       flow.CurrentResource,
		},
	})
	assert.NoError(t, err)

	wg := sync.WaitGroup{}
	wg.Add(10)
	f := GetSentinelProviderFilter()
	pass := int64(0)
	block := int64(0)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 30; j++ {
				result := f.Invoke(context.TODO(), mockInvoker, mockInvocation)
				if result.Error() == nil {
					atomic.AddInt64(&pass, 1)
				} else {
					atomic.AddInt64(&block, 1)
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()
	assert.True(t, atomic.LoadInt64(&pass) == 100)
	assert.True(t, atomic.LoadInt64(&block) == 200)
}

func TestConsumerFilter_Invoke(t *testing.T) {
	f := GetSentinelConsumerFilter()
	url, err := common.NewURL("dubbo://127.0.0.1:20000/UserProvider?anyhost=true&" +
		"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&" +
		"environment=dev&interface=com.ikurento.user.UserProvider&ip=192.168.56.1&methods=GetUser%2C&" +
		"module=dubbogo+user-info+server&org=ikurento.com&owner=ZX&pid=1447&revision=0.0.1&" +
		"side=provider&timeout=3000&timestamp=1556509797245&bean.name=UserProvider")
	assert.NoError(t, err)
	mockInvoker := protocol.NewBaseInvoker(url)
	mockInvocation := invocation.NewRPCInvocation("hello", []interface{}{"OK"}, make(map[string]interface{}))
	result := f.Invoke(context.TODO(), mockInvoker, mockInvocation)
	assert.NoError(t, result.Error())
}

func TestProviderFilter_Invoke(t *testing.T) {
	f := GetSentinelProviderFilter()
	url, err := common.NewURL("dubbo://127.0.0.1:20000/UserProvider?anyhost=true&" +
		"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&" +
		"environment=dev&interface=com.ikurento.user.UserProvider&ip=192.168.56.1&methods=GetUser%2C&" +
		"module=dubbogo+user-info+server&org=ikurento.com&owner=ZX&pid=1447&revision=0.0.1&" +
		"side=provider&timeout=3000&timestamp=1556509797245&bean.name=UserProvider")
	assert.NoError(t, err)
	mockInvoker := protocol.NewBaseInvoker(url)
	mockInvocation := invocation.NewRPCInvocation("hello", []interface{}{"OK"}, make(map[string]interface{}))
	result := f.Invoke(context.TODO(), mockInvoker, mockInvocation)
	assert.NoError(t, result.Error())
}

func TestGetResourceName(t *testing.T) {
	url, err := common.NewURL("dubbo://127.0.0.1:20000/UserProvider?anyhost=true&" +
		"version=1.0.0&group=myGroup&" +
		"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&" +
		"environment=dev&interface=com.ikurento.user.UserProvider&ip=192.168.56.1&methods=GetUser%2C&" +
		"module=dubbogo+user-info+server&org=ikurento.com&owner=ZX&pid=1447&revision=0.0.1&" +
		"side=provider&timeout=3000&timestamp=1556509797245&bean.name=UserProvider")
	assert.NoError(t, err)
	mockInvoker := protocol.NewBaseInvoker(url)
	interfaceResourceName, methodResourceName := getResourceName(mockInvoker,
		invocation.NewRPCInvocation("hello", []interface{}{"OK"}, make(map[string]interface{})), "prefix_")
	assert.Equal(t, "com.ikurento.user.UserProvider:myGroup:1.0.0", interfaceResourceName)
	assert.Equal(t, "prefix_com.ikurento.user.UserProvider:myGroup:1.0.0:hello()", methodResourceName)
}
