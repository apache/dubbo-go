package condition

import (
	"math"
)

import (
	"github.com/apache/dubbo-go/cluster/router"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/protocol"
)

const (
	HEALTH_CHECKER                             = "health.checker"
	DEFAULT_HEALTH_CHECKER                     = "default"
	OUTSTANDING_REQUEST_COUNT_LIMIT_KEY        = "outstanding.request.limit"
	SUCCESSIVE_FAILED_REQUEST_THRESHOLD_KEY    = "successive.failed.threshold"
	DEFAULT_SUCCESSIVE_FAILED_THRESHOLD        = 5
	CIRCUIT_TRIPPED_TIMEOUT_FACTOR_KEY         = "circuit.tripped.timeout.factor"
	DEFAULT_SUCCESSIVE_FAILED_REQUEST_MAX_DIFF = 5
	DEFAULT_CIRCUIT_TRIPPED_TIMEOUT_FACTOR     = 1000
	MAX_CIRCUIT_TRIPPED_TIMEOUT                = 30000
)

func init() {
	extension.SethealthChecker(DEFAULT_HEALTH_CHECKER, NewDefaultHealthChecker)
}

// DefaultHealthChecker is the default implementation of HealthChecker, which determines the health status of
// the invoker based on the number of successive bad request and the current active request.
type DefaultHealthChecker struct {
	// OutStandingRequestConutLimit
	OutStandingRequestConutLimit int32
	// RequestSuccessiveFailureThreshold
	RequestSuccessiveFailureThreshold int32
	// RequestSuccessiveFailureThreshold
	CircuitTrippedTimeoutFactor int32
}

// IsHealthy evaluates the healthy state on the given Invoker based on the number of successive bad request
// and the current active request
func (c *DefaultHealthChecker) IsHealthy(invoker protocol.Invoker) bool {
	urlStatus := protocol.GetURLStatus(invoker.GetUrl())
	if c.isCircuitBreakerTripped(urlStatus) || urlStatus.GetActive() > c.OutStandingRequestConutLimit {
		logger.Debugf("Invoker [%s] is currently in circuitbreaker tripped state", invoker.GetUrl().Key())
		return false
	}
	return true
}

// isCircuitBreakerTripped determine whether the invoker is in the tripped state by the number of successive bad request
func (c *DefaultHealthChecker) isCircuitBreakerTripped(status *protocol.RPCStatus) bool {
	circuitBreakerTimeout := c.getCircuitBreakerTimeout(status)
	currentTime := protocol.CurrentTimeMillis()
	if circuitBreakerTimeout <= 0 {
		return false
	}
	return circuitBreakerTimeout > currentTime
}

// getCircuitBreakerTimeout get the timestamp recovered from tripped state
func (c *DefaultHealthChecker) getCircuitBreakerTimeout(status *protocol.RPCStatus) int64 {
	sleepWindow := c.getCircuitBreakerSleepWindowTime(status)
	if sleepWindow <= 0 {
		return 0
	}
	return status.GetLastRequestFailedTimestamp() + sleepWindow
}

// getCircuitBreakerSleepWindowTime get the sleep window time of invoker
func (c *DefaultHealthChecker) getCircuitBreakerSleepWindowTime(status *protocol.RPCStatus) int64 {

	successiveFailureCount := status.GetSuccessiveRequestFailureCount()
	diff := successiveFailureCount - c.RequestSuccessiveFailureThreshold
	if diff < 0 {
		return 0
	} else if diff > DEFAULT_SUCCESSIVE_FAILED_REQUEST_MAX_DIFF {
		diff = DEFAULT_SUCCESSIVE_FAILED_REQUEST_MAX_DIFF
	}
	sleepWindow := (1 << diff) * DEFAULT_CIRCUIT_TRIPPED_TIMEOUT_FACTOR
	if sleepWindow > MAX_CIRCUIT_TRIPPED_TIMEOUT {
		sleepWindow = MAX_CIRCUIT_TRIPPED_TIMEOUT
	}
	return int64(sleepWindow)
}

// NewDefaultHealthChecker constructs a new DefaultHealthChecker based on the url
func NewDefaultHealthChecker(url *common.URL) router.HealthChecker {
	return &DefaultHealthChecker{
		OutStandingRequestConutLimit:      int32(url.GetParamInt(OUTSTANDING_REQUEST_COUNT_LIMIT_KEY, math.MaxInt32)),
		RequestSuccessiveFailureThreshold: int32(url.GetParamInt(SUCCESSIVE_FAILED_REQUEST_THRESHOLD_KEY, DEFAULT_SUCCESSIVE_FAILED_REQUEST_MAX_DIFF)),
		CircuitTrippedTimeoutFactor:       int32(url.GetParamInt(CIRCUIT_TRIPPED_TIMEOUT_FACTOR_KEY, DEFAULT_CIRCUIT_TRIPPED_TIMEOUT_FACTOR)),
	}
}
