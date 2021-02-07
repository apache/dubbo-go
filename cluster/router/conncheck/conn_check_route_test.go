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

package conncheck

import (
	"fmt"
	"testing"
)

import (
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/cluster/router"
	"github.com/apache/dubbo-go/cluster/router/chain"
	"github.com/apache/dubbo-go/cluster/router/utils"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/protocol/invocation"
	"github.com/apache/dubbo-go/protocol/mock"
)

const (
	connCheckRoute1010IP         = "192.168.10.10"
	connCheckRoute1011IP         = "192.168.10.11"
	connCheckRoute1012IP         = "192.168.10.12"
	connCheckRouteMethodNameTest = "test"
	connCheck1001URL             = "dubbo://192.168.10.1/com.ikurento.user.UserProvider"
	connCheckRouteUrlFormat      = "dubbo://%s:20000/com.ikurento.user.UserProvider"
)

func TestConnCheckRouterRoute(t *testing.T) {
	defer protocol.CleanAllStatus()
	notify := make(chan struct{})
	go func() {
		for range notify {
		}
	}()
	consumerURL, _ := common.NewURL(connCheck1001URL)
	url1, _ := common.NewURL(fmt.Sprintf(connCheckRouteUrlFormat, connCheckRoute1010IP))
	url2, _ := common.NewURL(fmt.Sprintf(connCheckRouteUrlFormat, connCheckRoute1011IP))
	url3, _ := common.NewURL(fmt.Sprintf(connCheckRouteUrlFormat, connCheckRoute1012IP))
	hcr, _ := NewConnCheckRouter(consumerURL, notify)

	var invokers []protocol.Invoker
	invoker1 := NewMockInvoker(url1)
	invoker2 := NewMockInvoker(url2)
	invoker3 := NewMockInvoker(url3)
	protocol.SetInvokerUnhealthyStatus(invoker1)
	protocol.SetInvokerUnhealthyStatus(invoker2)

	invokers = append(invokers, invoker1, invoker2, invoker3)
	inv := invocation.NewRPCInvocation(connCheckRouteMethodNameTest, nil, nil)
	res := hcr.Route(utils.ToBitmap(invokers), setUpAddrCache(hcr.(*ConnCheckRouter), invokers), consumerURL, inv)

	// now  invoker3 is healthy
	assert.True(t, len(res.ToArray()) == 1)

	// check blacklist remove
	protocol.RemoveInvokerUnhealthyStatus(invoker1)
	res = hcr.Route(utils.ToBitmap(invokers), setUpAddrCache(hcr.(*ConnCheckRouter), invokers), consumerURL, inv)
	// now  invoker3 invoker1 is healthy
	assert.True(t, len(res.ToArray()) == 2)

}

func TestRecovery(t *testing.T) {
	// check recover

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invoker1 := mock.NewMockInvoker(ctrl)
	invoker2 := mock.NewMockInvoker(ctrl)

	invoker1.EXPECT().GetUrl().Return(&common.URL{Path: "path1"}).AnyTimes()
	invoker2.EXPECT().GetUrl().Return(&common.URL{Path: "path2"}).AnyTimes()
	invoker1.EXPECT().IsAvailable().Return(true).AnyTimes()
	invoker2.EXPECT().IsAvailable().Return(true).AnyTimes()

	protocol.SetInvokerUnhealthyStatus(invoker1)
	protocol.SetInvokerUnhealthyStatus(invoker2)
	assert.Equal(t, len(protocol.GetBlackListInvokers(16)), 2)
	protocol.TryRefreshBlackList()
	assert.Equal(t, len(protocol.GetBlackListInvokers(16)), 0)
}

func setUpAddrCache(r router.Poolable, addrs []protocol.Invoker) router.Cache {
	pool, info := r.Pool(addrs)
	cache := chain.BuildCache(addrs)
	cache.SetAddrMeta(r.Name(), info)
	cache.SetAddrPool(r.Name(), pool)
	return cache
}
