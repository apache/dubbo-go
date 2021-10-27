package metrics

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"github.com/pkg/errors"
)

var (
	ErrMetricsNotFound = errors.New("metrics not found")
)

type Metrics interface {
	// GetMethodMetrics returns method-level metrics, the format of key is "{instance key}.{invoker key}.{method key}.{key}"
	// url is invoker's url, which contains information about instance and invoker.
	// methodName is the method name.
	// key is the key of the metrics.
	GetMethodMetrics(url *common.URL, methodName, key string) (interface{}, error)

	// GetInvokerMetrics returns invoker-level metrics, the format of key is "{instance key}.{invoker key}.{key}"
	// DO NOT IMPLEMENT FOR EARLIER VERSION
	GetInvokerMetrics(url *common.URL, key string) (interface{}, error)

	// GetInstanceMetrics returns instance-level metrics, the format of key is "{instance key}.{key}"
	// DO NOT IMPLEMENT FOR EARLIER VERSION
	GetInstanceMetrics(url *common.URL, key string) (interface{}, error)
}
