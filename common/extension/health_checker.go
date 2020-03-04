package extension

import (
	"github.com/apache/dubbo-go/cluster/router"
	"github.com/apache/dubbo-go/common"
)

var (
	healthCheckers = make(map[string]func(url *common.URL) router.HealthChecker)
)

// SethealthChecker set the HealthChecker with name
func SethealthChecker(name string, fcn func(url *common.URL) router.HealthChecker) {
	healthCheckers[name] = fcn
}

// GetHealthChecker get the HealthChecker with name
func GetHealthChecker(name string, url *common.URL) router.HealthChecker {
	if healthCheckers[name] == nil {
		panic("healthCheckers for " + name + " is not existing, make sure you have import the package.")
	}
	return healthCheckers[name](url)
}
