package capeva

import "sync"

type Vegas struct {
	*baseCapacityEvaluator

	// mutex protects vegasDataMap
	mutex *sync.Mutex
	vegasDataMap map[string]*vegasData
}

func NewVegas() *Vegas {
	return &Vegas{
		baseCapacityEvaluator: newBaseCapacityEvaluator(),
		mutex: &sync.Mutex{},
		vegasDataMap: make(map[string]*vegasData),
	}
}

type vegasData struct {

}
