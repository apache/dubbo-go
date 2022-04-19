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

package ringhash

import (
	"errors"
	"fmt"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/remoting/xds/mocks"
	"dubbo.apache.org/dubbo-go/v3/xds/client/resource"
)

func TestSelect(t *testing.T) {
	var invokers []protocol.Invoker
	url, _ := common.NewURL(fmt.Sprintf("dubbo://%s:%d/org.apache.demo.HelloService",
		"192.168.1.1", constant.DefaultPort))
	url.SetParam(constant.InterfaceKey, "org.apache.demo.HelloService")
	url.SetParam(constant.EndPointWeight, "3")
	invokers = append(invokers, protocol.NewBaseInvoker(url))
	url1, _ := common.NewURL(fmt.Sprintf("dubbo://%s:%d/org.apache.demo.HelloService",
		"192.168.1.2", constant.DefaultPort))
	url1.SetParam(constant.InterfaceKey, "org.apache.demo.HelloService")
	url1.SetParam(constant.EndPointWeight, "3")
	invokers = append(invokers, protocol.NewBaseInvoker(url1))
	url2, _ := common.NewURL(fmt.Sprintf("dubbo://%s:%d/org.apache.demo.HelloService",
		"192.168.1.3", constant.DefaultPort))
	url2.SetParam(constant.InterfaceKey, "org.apache.demo.HelloService")
	url2.SetParam(constant.EndPointWeight, "4")
	invokers = append(invokers, protocol.NewBaseInvoker(url2))
	t.Run("normal", func(t *testing.T) {
		mockXDSClient := &mocks.WrappedClientMock{}
		mockXDSClient.On("GetRouterConfig", mock.Anything).Return(resource.RouteConfigUpdate{})
		mockXDSClient.On("GetClusterUpdateIgnoreVersion", mock.Anything).
			Return(resource.ClusterUpdate{LBPolicy: &resource.ClusterLBPolicyRingHash{MinimumRingSize: 9, MaximumRingSize: 10}})
		mockXDSClient.On("GetHostAddrByServiceUniqueKey", mock.Anything).Return("HelloService", nil)
		mockXDSClient.On("MatchRoute", mock.Anything, mock.Anything).
			Return(&resource.Route{HashPolicies: []*resource.HashPolicy{{HashPolicyType: resource.HashPolicyTypeHeader,
				HeaderName: "key"}}}, nil)
		lb := &ringhashLoadBalance{client: mockXDSClient}
		inv := &invocation.RPCInvocation{}
		inv.SetInvoker(protocol.NewBaseInvoker(url))
		inv.SetAttachment("key", "1")
		assert.NotNil(t, lb.Select(invokers, inv))
	})

	t.Run("normal_hash_channelID", func(t *testing.T) {
		mockXDSClient := &mocks.WrappedClientMock{}
		mockXDSClient.On("GetRouterConfig", mock.Anything).Return(resource.RouteConfigUpdate{})
		mockXDSClient.On("GetClusterUpdateIgnoreVersion", mock.Anything).
			Return(resource.ClusterUpdate{LBPolicy: &resource.ClusterLBPolicyRingHash{MinimumRingSize: 9, MaximumRingSize: 10}})
		mockXDSClient.On("GetHostAddrByServiceUniqueKey", mock.Anything).Return("HelloService", nil)
		mockXDSClient.On("MatchRoute", mock.Anything, mock.Anything).
			Return(&resource.Route{HashPolicies: []*resource.HashPolicy{
				{HashPolicyType: resource.HashPolicyTypeChannelID}}}, nil)
		lb := &ringhashLoadBalance{client: mockXDSClient}
		inv := &invocation.RPCInvocation{}
		inv.SetInvoker(protocol.NewBaseInvoker(url))
		assert.NotNil(t, lb.Select(invokers, inv))
	})

	t.Run("GetHostAddrByServiceUniqueKey_faild", func(t *testing.T) {
		mockXDSClient := &mocks.WrappedClientMock{}
		mockXDSClient.On("GetHostAddrByServiceUniqueKey", mock.Anything).
			Return("HelloService", errors.New("error"))
		lb := &ringhashLoadBalance{client: mockXDSClient}
		inv := &invocation.RPCInvocation{}
		inv.SetInvoker(protocol.NewBaseInvoker(url))
		inv.SetAttachment("key", "1")
		assert.Nil(t, lb.Select(invokers, inv))
	})
}
