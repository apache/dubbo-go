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
	"github.com/RoaringBitmap/roaring"
	"strconv"
	"strings"
)

import (
	"github.com/apache/dubbo-go/cluster/router"
	"github.com/apache/dubbo-go/cluster/router/utils"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/protocol"
)

const (
	connUnHealthy = "conn-unhealthy"
	connHealthy   = "conn-healthy"
	name          = "conn-check-router"
)

// ConnCheckRouter provides a health-first routing mechanism through ConnChecker
type ConnCheckRouter struct {
	url     *common.URL
	checker router.ConnChecker
	notify  chan struct{}
}

// NewConnCheckRouter construct an NewConnCheckRouter via url
func NewConnCheckRouter(url *common.URL, notify chan struct{}) (router.PriorityRouter, error) {
	r := &ConnCheckRouter{
		url:    url,
		notify: notify,
	}
	checkerName := url.GetParam(constant.HEALTH_CHECKER, constant.DEFAULT_HEALTH_CHECKER)
	r.checker = extension.GetConnChecker(checkerName, url)
	return r, nil
}

// Route gets a list of healthy invoker
func (r *ConnCheckRouter) Route(invokers *roaring.Bitmap, cache router.Cache, url *common.URL, invocation protocol.Invocation) *roaring.Bitmap {
	addrPool := cache.FindAddrPool(r)
	// Add healthy invoker to the list
	unhealthyBits := addrPool[connUnHealthy]
	healthyBits := addrPool[connHealthy]
	// check if empty, return
	if unhealthyBits == nil || unhealthyBits.IsEmpty() || healthyBits == nil || healthyBits.IsEmpty() {
		return invokers
	}
	healthyInvokers := utils.JoinIfNotEqual(healthyBits, invokers)
	// If all invokers are considered unhealthy, downgrade to all invoker
	if healthyInvokers.IsEmpty() {
		logger.Warnf(" Now all invokers are unhealthy, so downgraded to all! Service: [%s]", url.ServiceKey())
		return invokers
	}
	return healthyInvokers
}

func (r *ConnCheckRouter) RouteSnapshot(cache router.Cache) string {
	addrPool := cache.FindAddrPool(r)
	// Add healthy invoker to the list
	healthBit := addrPool[connHealthy]
	sb := strings.Builder{}
	sb.WriteString(r.Name())
	sb.WriteString(" -> ")
	if healthBit == nil {
		sb.WriteString(" No Unhealthy Invoker")
		return sb.String()
	}
	sb.WriteString("Count:")
	sb.WriteString(strconv.FormatUint(healthBit.GetCardinality(), 10))
	sb.WriteString(" ")
	sb.WriteString(healthBit.String())
	return sb.String()
}

// Pool separates healthy invokers from others.
func (r *ConnCheckRouter) Pool(invokers []protocol.Invoker) (router.AddrPool, router.AddrMetadata) {
	rb := make(router.AddrPool, 8)
	healthyInvokers := utils.ToBitmap(invokers)
	unhealthyInvokers := roaring.NewBitmap()
	for i, invoker := range invokers {
		if !r.checker.IsConnHealthy(invoker) {
			unhealthyInvokers.Add(uint32(i))
			healthyInvokers.Remove(uint32(i))
		}
	}
	if !unhealthyInvokers.IsEmpty() {
		rb[connHealthy] = healthyInvokers
		rb[connUnHealthy] = unhealthyInvokers
	}
	return rb, nil
}

// ShouldPool will always return true to make sure healthy check constantly.
func (r *ConnCheckRouter) ShouldPool() bool {
	return true
}

// Name get name of ConnCheckerRouter
func (r *ConnCheckRouter) Name() string {
	return name
}

// Priority get Router priority level
func (r *ConnCheckRouter) Priority() int64 {
	return 0
}

// URL Return URL in router
func (r *ConnCheckRouter) URL() *common.URL {
	return r.url
}

// ConnChecker returns the HealthChecker bound to this HealthCheckRouter
func (r *ConnCheckRouter) ConnChecker() router.ConnChecker {
	return r.checker
}
