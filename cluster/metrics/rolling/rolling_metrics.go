package rolling

import (
	"dubbo.apache.org/dubbo-go/v3/common"
)

type Metrics interface {
	// GetMethodMetrics returns method-level metrics, the format of key is "{instance key}.{invoker key}.{method key}.{key}"
	// url is invoker's url, which contains information about instance and invoker.
	// methodName is the method name.
	// key is the key of the metrics.
	GetMethodMetrics(url *common.URL, methodName, key string) (float64, error)
	AppendMethodMetrics(url *common.URL, methodName, key string, value float64) error
}
