package metrics

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"fmt"
	"sync"
)

var LocalMetrics Metrics

func init() {
	LocalMetrics = newLocalMetrics()
}

type localMetrics struct {
	// protect metrics
	lock    *sync.Mutex
	metrics map[string]interface{}
}

func newLocalMetrics() *localMetrics {
	return &localMetrics{
		lock:    &sync.Mutex{},
		metrics: make(map[string]interface{}),
	}
}

func (m *localMetrics) GetMethodMetrics(url *common.URL, methodName, key string) (interface{}, error) {
	metricsKey := fmt.Sprintf("%s.%s.%s.%s", getInstanceKey(url), getInvokerKey(url), methodName, key)
	if metrics, ok := m.metrics[metricsKey]; ok {
		return metrics, nil
	}
	return nil, ErrMetricsNotFound
}

func (m *localMetrics) SetMethodMetrics(url *common.URL, methodName, key string, value interface{}) error {
	metricsKey := fmt.Sprintf("%s.%s.%s.%s", getInstanceKey(url), getInvokerKey(url), methodName, key)
	m.metrics[metricsKey] = value
	return nil
}

func (m *localMetrics) GetInvokerMetrics(url *common.URL, key string) (interface{}, error) {
	panic("implement me")
}

func (m *localMetrics) SetInvokerMetrics(url *common.URL, key string, value interface{}) error {
	panic("implement me")
}

func (m *localMetrics) GetInstanceMetrics(url *common.URL, key string) (interface{}, error) {
	panic("implement me")
}

func (m *localMetrics) SetInstanceMetrics(url *common.URL, key string, value interface{}) error {
	panic("implement me")
}
