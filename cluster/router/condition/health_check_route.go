package condition

import (
	"github.com/apache/dubbo-go/cluster/router"
	"github.com/apache/dubbo-go/common"
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
func NewHealthCheckRouter(url *common.URL) (router.Router, error) {
	r := &HealthCheckRouter{}
	r.url = url
	r.enabled = url.GetParamBool(HEALTH_ROUTE_ENABLED_KEY, false)
	if r.enabled {
		checkerName := url.GetParam(HEALTH_CHECKER, DEFAULT_HEALTH_CHECKER)
		r.checker = extension.GetHealthChecker(checkerName, url)
	}
	return r, nil
}

// Route gets a list of healthy invoker
func (r *HealthCheckRouter) Route(invokers []protocol.Invoker, url *common.URL, invocation protocol.Invocation) []protocol.Invoker {
	if !r.enabled {
		return invokers
	}
	var healthyInvokers []protocol.Invoker
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
	} else {
		return healthyInvokers
	}
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
