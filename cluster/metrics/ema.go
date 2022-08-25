package metrics

import (
	"fmt"
	"sync"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
)

// EMA is a struct implemented Exponential Moving Average.
// val = old * (1 - alpha) + new * alpha
type EMA struct {
	mu    sync.Mutex
	alpha float64
	val   float64
}

type EMAOpts struct {
	Alpha float64
}

// NewEMA creates a new EMA based on the given EMAOpts.
func NewEMA(opts EMAOpts) *EMA {
	return &EMA{
		alpha: opts.Alpha,
		val:   0,
	}
}

func (e *EMA) Add(v float64) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.val = e.val*(1-e.alpha) + v*e.alpha
}

func (e *EMA) Value() float64 {
	return e.val
}

var EMAMetrics Metrics

func init() {
	EMAMetrics = newEMAMetrics()
}

var _ Metrics = (*emaMetrics)(nil)

type emaMetrics struct {
	opts    EMAOpts
	metrics sync.Map
}

func newEMAMetrics() *emaMetrics {
	return &emaMetrics{
		opts: EMAOpts{
			Alpha: 0.7,
		},
	}
}

func (m *emaMetrics) GetMethodMetrics(url *common.URL, methodName, key string) (interface{}, error) {
	metricsKey := fmt.Sprintf("%s.%s.%s.%s", getInstanceKey(url), getInvokerKey(url), methodName, key)
	if metrics, ok := m.metrics.Load(metricsKey); ok {
		return metrics.(*EMA).Value(), nil
	}
	return float64(0), ErrMetricsNotFound
}

func (m *emaMetrics) SetMethodMetrics(url *common.URL, methodName, key string, val interface{}) error {
	v := ToFloat64(val)
	metricsKey := fmt.Sprintf("%s.%s.%s.%s", getInstanceKey(url), getInvokerKey(url), methodName, key)
	if metrics, ok := m.metrics.Load(metricsKey); ok {
		metrics.(*EMA).Add(v)
	} else {
		metrics, _ = m.metrics.LoadOrStore(metricsKey, NewEMA(m.opts))
		metrics.(*EMA).Add(v)
	}
	return nil
}

func (m *emaMetrics) GetInvokerMetrics(url *common.URL, key string) (interface{}, error) {
	panic("implement me")
}

func (m *emaMetrics) SetInvokerMetrics(url *common.URL, key string, value interface{}) error {
	panic("implement me")
}

func (m *emaMetrics) GetInstanceMetrics(url *common.URL, key string) (interface{}, error) {
	panic("implement me")
}

func (m *emaMetrics) SetInstanceMetrics(url *common.URL, key string, value interface{}) error {
	panic("implement me")
}
