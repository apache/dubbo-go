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

package local

import (
	"fmt"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/cluster/router"
	"github.com/apache/dubbo-go/cluster/router/chain"
	"github.com/apache/dubbo-go/cluster/router/utils"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/protocol/invocation"
)

const (
	localDiscRoute1010IP         = "192.168.10.10"
	localDiscRoute1011IP         = "192.168.10.11"
	localDiscRoute1012IP         = "192.168.10.12"
	localDiscRouteMethodNameTest = "test"
	localDiscRouteUrlFormat      = "dubbo://%s:20000/com.ikurento.user.UserProvider"
)

func TestLocalDiscRouterRoute(t *testing.T) {
	defer protocol.CleanAllStatus()
	consumerURL, _ := common.NewURL(fmt.Sprintf(localDiscRouteUrlFormat, localDiscRoute1010IP))
	url1, _ := common.NewURL(fmt.Sprintf(localDiscRouteUrlFormat, localDiscRoute1010IP))
	url2, _ := common.NewURL(fmt.Sprintf(localDiscRouteUrlFormat, localDiscRoute1011IP))
	url3, _ := common.NewURL(fmt.Sprintf(localDiscRouteUrlFormat, localDiscRoute1012IP))
	hcr, _ := NewLocalPriorityRouter(consumerURL)

	var invokers []protocol.Invoker
	invoker1 := NewMockInvoker(url1)
	invoker2 := NewMockInvoker(url2)
	invoker3 := NewMockInvoker(url3)
	invokers = append(invokers, invoker1, invoker2, invoker3)
	inv := invocation.NewRPCInvocation(localDiscRouteMethodNameTest, nil, nil)
	res := hcr.Route(utils.ToBitmap(invokers), setUpAddrCache(hcr.(*LocalPriorityRouter), invokers), consumerURL, inv)
	// now only same ip invoker is selected
	assert.True(t, len(res.ToArray()) == 1)

	// now all invoker with ip that not match client are selected
	invokers = invokers[1:]
	res = hcr.Route(utils.ToBitmap(invokers), setUpAddrCache(hcr.(*LocalPriorityRouter), invokers), consumerURL, inv)
	assert.True(t, len(res.ToArray()) == 2)
}

func setUpAddrCache(r router.Poolable, addrs []protocol.Invoker) router.Cache {
	pool, info := r.Pool(addrs)
	cache := chain.BuildCache(addrs)
	cache.SetAddrMeta(r.Name(), info)
	cache.SetAddrPool(r.Name(), pool)
	return cache
}
