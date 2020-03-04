package router

import (
	"github.com/apache/dubbo-go/protocol"
)

// HealthChecker is used to determine whether the invoker is healthy or not
type HealthChecker interface {
	// IsHealthy evaluates the healthy state on the given Invoker
	IsHealthy(invoker protocol.Invoker) bool
}
