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
	"github.com/RoaringBitmap/roaring"
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
	HEALTH_ROUTE_ENABLED_KEY = "health.route.enabled"
	healthy                  = "healthy"
	name                     = "health-check-router"
)

// HealthCheckRouter provides a health-first routing mechanism through HealthChecker
type HealthCheckRouter struct {
	url     *common.URL
	enabled bool
	checker router.HealthChecker
}

// NewHealthCheckRouter construct an HealthCheckRouter via url
func NewHealthCheckRouter(url *common.URL) (router.PriorityRouter, error) {
	r := &HealthCheckRouter{
		url:     url,
		enabled: url.GetParamBool(HEALTH_ROUTE_ENABLED_KEY, false),
	}
	if r.enabled {
		checkerName := url.GetParam(constant.HEALTH_CHECKER, constant.DEFAULT_HEALTH_CHECKER)
		r.checker = extension.GetHealthChecker(checkerName, url)
	}
	return r, nil
}

// Route gets a list of healthy invoker
func (r *HealthCheckRouter) Route(invokers *roaring.Bitmap, cache router.Cache, url *common.URL, invocation protocol.Invocation) *roaring.Bitmap {
	if !r.enabled {
		return invokers
	}

	addrPool := cache.FindAddrPool(r)
	// Add healthy invoker to the list
	healthyInvokers := utils.JoinIfNotEqual(addrPool[healthy], invokers)
	// If all invokers are considered unhealthy, downgrade to all invoker
	if healthyInvokers.IsEmpty() {
		logger.Warnf(" Now all invokers are unhealthy, so downgraded to all! Service: [%s]", url.ServiceKey())
		return invokers
	}
	return healthyInvokers
}

// Pool separates healthy invokers from others.
func (r *HealthCheckRouter) Pool(invokers []protocol.Invoker) (router.AddrPool, router.AddrMetadata) {
	if !r.enabled {
		return nil, nil
	}

	rb := make(router.AddrPool, 8)
	rb[healthy] = roaring.NewBitmap()
	for i, invoker := range invokers {
		if r.checker.IsHealthy(invoker) {
			rb[healthy].Add(uint32(i))
		}
	}

	return rb, nil
}

// ShouldPool will always return true to make sure healthy check constantly.
func (r *HealthCheckRouter) ShouldPool() bool {
	return r.enabled
}

func (r *HealthCheckRouter) Name() string {
	return name
}

// Priority
func (r *HealthCheckRouter) Priority() int64 {
	return 0
}

// URL Return URL in router
func (r *HealthCheckRouter) URL() *common.URL {
	return r.url
}

// HealthyChecker returns the HealthChecker bound to this HealthCheckRouter
func (r *HealthCheckRouter) HealthyChecker() router.HealthChecker {
	return r.checker
}
