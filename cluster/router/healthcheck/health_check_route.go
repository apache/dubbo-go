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
	"github.com/apache/dubbo-go/cluster/router"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/protocol"
)

const (
	HEALTH_ROUTE_ENABLED_KEY = "health.route.enabled"
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
func (r *HealthCheckRouter) Route(invokers []protocol.Invoker, url *common.URL, invocation protocol.Invocation) []protocol.Invoker {
	if !r.enabled {
		return invokers
	}
	healthyInvokers := make([]protocol.Invoker, 0, len(invokers))
	// Add healthy invoker to the list
	for _, invoker := range invokers {
		if r.checker.IsHealthy(invoker) {
			healthyInvokers = append(healthyInvokers, invoker)
		}
	}
	// If all Invoke are considered unhealthy, downgrade to all inovker
	if len(healthyInvokers) == 0 {
		logger.Warnf(" Now all invokers are unhealthy, so downgraded to all! Service: [%s]", url.ServiceKey())
		return invokers
	}
	return healthyInvokers
}

// Priority
func (r *HealthCheckRouter) Priority() int64 {
	return 0
}

// URL Return URL in router
func (r *HealthCheckRouter) URL() common.URL {
	return *r.url
}

// HealthyChecker returns the HealthChecker bound to this HealthCheckRouter
func (r *HealthCheckRouter) HealthyChecker() router.HealthChecker {
	return r.checker
}
