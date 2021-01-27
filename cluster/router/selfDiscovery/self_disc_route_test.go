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

package self_disc

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
	healthCheckRoute1010IP         = "192.168.10.10"
	healthCheckRoute1011IP         = "192.168.10.11"
	healthCheckRoute1012IP         = "192.168.10.12"
	healthCheckRouteMethodNameTest = "test"
	healthCheck1001URL             = "dubbo://192.168.10.1/com.ikurento.user.UserProvider"
	healthCheckRouteUrlFormat      = "dubbo://%s:20000/com.ikurento.user.UserProvider"
)

func TestHealthCheckRouterRoute(t *testing.T) {
	defer protocol.CleanAllStatus()
	consumerURL, _ := common.NewURL(fmt.Sprintf(healthCheckRouteUrlFormat, healthCheckRoute1010IP))
	url1, _ := common.NewURL(fmt.Sprintf(healthCheckRouteUrlFormat, healthCheckRoute1010IP))
	url2, _ := common.NewURL(fmt.Sprintf(healthCheckRouteUrlFormat, healthCheckRoute1011IP))
	url3, _ := common.NewURL(fmt.Sprintf(healthCheckRouteUrlFormat, healthCheckRoute1012IP))
	hcr, _ := NewSelfDiscRouter(consumerURL)

	var invokers []protocol.Invoker
	invoker1 := NewMockInvoker(url1)
	invoker2 := NewMockInvoker(url2)
	invoker3 := NewMockInvoker(url3)
	invokers = append(invokers, invoker1, invoker2, invoker3)
	inv := invocation.NewRPCInvocation(healthCheckRouteMethodNameTest, nil, nil)
	res := hcr.Route(utils.ToBitmap(invokers), setUpAddrCache(hcr.(*SelfDiscRouter), invokers), consumerURL, inv)
	// now all invokers are healthy
	assert.True(t, len(res.ToArray()) == 1)

	invokers = invokers[1:]
	res = hcr.Route(utils.ToBitmap(invokers), setUpAddrCache(hcr.(*SelfDiscRouter), invokers), consumerURL, inv)
	assert.True(t, len(res.ToArray()) == 2)
}

func setUpAddrCache(r router.Poolable, addrs []protocol.Invoker) router.Cache {
	pool, info := r.Pool(addrs)
	cache := chain.BuildCache(addrs)
	cache.SetAddrMeta(r.Name(), info)
	cache.SetAddrPool(r.Name(), pool)
	return cache
}
