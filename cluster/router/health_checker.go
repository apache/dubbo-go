package router

import (
	"github.com/apache/dubbo-go/protocol"
)

type HealthChecker interface {
	IsHealthy(invoker protocol.Invoker) bool
}
