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

package healthcheck

import (
	"fmt"
	"math"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/cluster/router"
	"github.com/apache/dubbo-go/cluster/router/chain"
	"github.com/apache/dubbo-go/cluster/router/utils"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
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
	notify := make(chan struct{})
	go func() {
		for range notify {
		}
	}()
	consumerURL, _ := common.NewURL(healthCheck1001URL)
	consumerURL.SetParam(constant.HEALTH_ROUTE_ENABLED_KEY, "true")
	url1, _ := common.NewURL(fmt.Sprintf(healthCheckRouteUrlFormat, healthCheckRoute1010IP))
	url2, _ := common.NewURL(fmt.Sprintf(healthCheckRouteUrlFormat, healthCheckRoute1011IP))
	url3, _ := common.NewURL(fmt.Sprintf(healthCheckRouteUrlFormat, healthCheckRoute1012IP))
	hcr, _ := NewHealthCheckRouter(consumerURL, notify)

	var invokers []protocol.Invoker
	invoker1 := NewMockInvoker(url1)
	invoker2 := NewMockInvoker(url2)
	invoker3 := NewMockInvoker(url3)
	invokers = append(invokers, invoker1, invoker2, invoker3)
	inv := invocation.NewRPCInvocation(healthCheckRouteMethodNameTest, nil, nil)
	res := hcr.Route(utils.ToBitmap(invokers), setUpAddrCache(hcr.(*HealthCheckRouter), invokers), consumerURL, inv)
	// now all invokers are healthy
	assert.True(t, len(res.ToArray()) == len(invokers))

	for i := 0; i < 10; i++ {
		request(url1, healthCheckRouteMethodNameTest, 0, false, false)
	}
	res = hcr.Route(utils.ToBitmap(invokers), setUpAddrCache(hcr.(*HealthCheckRouter), invokers), consumerURL, inv)
	// invokers1  is unhealthy now
	assert.True(t, len(res.ToArray()) == 2 && !res.Contains(0))

	for i := 0; i < 10; i++ {
		request(url1, healthCheckRouteMethodNameTest, 0, false, false)
		request(url2, healthCheckRouteMethodNameTest, 0, false, false)
	}

	res = hcr.Route(utils.ToBitmap(invokers), setUpAddrCache(hcr.(*HealthCheckRouter), invokers), consumerURL, inv)
	// only invokers3  is healthy now
	assert.True(t, len(res.ToArray()) == 1 && !res.Contains(0) && !res.Contains(1))

	for i := 0; i < 10; i++ {
		request(url1, healthCheckRouteMethodNameTest, 0, false, false)
		request(url2, healthCheckRouteMethodNameTest, 0, false, false)
		request(url3, healthCheckRouteMethodNameTest, 0, false, false)
	}

	res = hcr.Route(utils.ToBitmap(invokers), setUpAddrCache(hcr.(*HealthCheckRouter), invokers), consumerURL, inv)
	// now all invokers are unhealthy, so downgraded to all
	assert.True(t, len(res.ToArray()) == 3)

	// reset the invoker1 successive failed count, so invoker1 go to healthy
	request(url1, healthCheckRouteMethodNameTest, 0, false, true)
	res = hcr.Route(utils.ToBitmap(invokers), setUpAddrCache(hcr.(*HealthCheckRouter), invokers), consumerURL, inv)
	assert.True(t, res.Contains(0))

	for i := 0; i < 6; i++ {
		request(url1, healthCheckRouteMethodNameTest, 0, false, false)
	}
	// now all invokers are unhealthy, so downgraded to all again
	res = hcr.Route(utils.ToBitmap(invokers), setUpAddrCache(hcr.(*HealthCheckRouter), invokers), consumerURL, inv)
	assert.True(t, len(res.ToArray()) == 3)
	time.Sleep(time.Second * 2)
	// invoker1 go to healthy again after 2s
	res = hcr.Route(utils.ToBitmap(invokers), setUpAddrCache(hcr.(*HealthCheckRouter), invokers), consumerURL, inv)
	assert.True(t, res.Contains(0))

}

func TestNewHealthCheckRouter(t *testing.T) {
	defer protocol.CleanAllStatus()
	notify := make(chan struct{})
	go func() {
		for range notify {
		}
	}()
	url, _ := common.NewURL(fmt.Sprintf(healthCheckDubboUrlFormat, healthCheckDubbo1010IP))
	hcr, _ := NewHealthCheckRouter(url, notify)
	h := hcr.(*HealthCheckRouter)
	assert.Nil(t, h.checker)

	url.SetParam(constant.HEALTH_ROUTE_ENABLED_KEY, "true")
	hcr, _ = NewHealthCheckRouter(url, notify)
	h = hcr.(*HealthCheckRouter)
	assert.NotNil(t, h.checker)

	dhc := h.checker.(*DefaultHealthChecker)
	assert.Equal(t, dhc.outStandingRequestConutLimit, int32(math.MaxInt32))
	assert.Equal(t, dhc.requestSuccessiveFailureThreshold, int32(constant.DEFAULT_SUCCESSIVE_FAILED_THRESHOLD))
	assert.Equal(t, dhc.circuitTrippedTimeoutFactor, int32(constant.DEFAULT_CIRCUIT_TRIPPED_TIMEOUT_FACTOR))

	url.SetParam(constant.CIRCUIT_TRIPPED_TIMEOUT_FACTOR_KEY, "500")
	url.SetParam(constant.SUCCESSIVE_FAILED_REQUEST_THRESHOLD_KEY, "10")
	url.SetParam(constant.OUTSTANDING_REQUEST_COUNT_LIMIT_KEY, "1000")
	hcr, _ = NewHealthCheckRouter(url, notify)
	h = hcr.(*HealthCheckRouter)
	dhc = h.checker.(*DefaultHealthChecker)
	assert.Equal(t, dhc.outStandingRequestConutLimit, int32(1000))
	assert.Equal(t, dhc.requestSuccessiveFailureThreshold, int32(10))
	assert.Equal(t, dhc.circuitTrippedTimeoutFactor, int32(500))
}

func setUpAddrCache(r router.Poolable, addrs []protocol.Invoker) router.Cache {
	pool, info := r.Pool(addrs)
	cache := chain.BuildCache(addrs)
	cache.SetAddrMeta(r.Name(), info)
	cache.SetAddrPool(r.Name(), pool)
	return cache
}
