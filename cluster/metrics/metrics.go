package metrics

import (
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"github.com/pkg/errors"
)

var (
	ErrMetricsNotFound = errors.New("metrics not found")
)

type Metric interface {
	// GetMethodMetrics returns method-level metrics, the format of key is "{ivk name}.{method name}.{key}"
	// ivk is the invoker of the method.
	// methodName is the method name.
	// key is the key of the metrics.
	GetMethodMetrics(ivk *protocol.Invoker, methodName, key string) (interface{}, error)

	// GetInvokerMetrics returns invoker-level metrics, the format of key is "{ivk name}.{key}"
	// DO NOT IMPLEMENT FOR EARLIER VERSION
	GetInvokerMetrics(ivk *protocol.Invoker, key string) (interface{}, error)

	// GetInstanceMetrics returns instance-level metrics, the format of key is "{key}"
	// DO NOT IMPLEMENT FOR EARLIER VERSION
	GetInstanceMetrics(key string) (interface{}, error)
}
