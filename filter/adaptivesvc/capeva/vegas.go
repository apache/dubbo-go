package capeva

import "sync"

type Vegas struct {
	*baseCapacityEvaluator

	// mutex protects invocationMetricsMap
	mutex *sync.Mutex
	invocationMetricsMap map[string]*vegasInvocationMetrics
}

func NewVegas() *Vegas {
	return &Vegas{
		baseCapacityEvaluator: newBaseCapacityEvaluator(),
		mutex: &sync.Mutex{},
		invocationMetricsMap: make(map[string]*vegasInvocationMetrics),
	}
}

type vegasInvocationMetrics struct {

}
