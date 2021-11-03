package capeva

import (
	"go.uber.org/atomic"
)

type baseCapacityEvaluator struct {
	estimated, actual *atomic.Int64
}

func newBaseCapacityEvaluator() *baseCapacityEvaluator {
	return &baseCapacityEvaluator{
		estimated: &atomic.Int64{},
		actual: &atomic.Int64{},
	}
}

func (ce *baseCapacityEvaluator) Estimated() int64 {
	return ce.estimated.Load()
}

func (ce *baseCapacityEvaluator) Actual() int64 {
	return ce.actual.Load()
}

func (ce *baseCapacityEvaluator) UpdateEstimated(value int64) {
	ce.actual.Store(value)
}

func (ce *baseCapacityEvaluator) UpdateActual(delta int64) {
	ce.actual.Add(delta)
}

func (ce *baseCapacityEvaluator) NewCapacityUpdater() CapacityUpdater {
	panic("implement me!")
}

type baseCapacityUpdater struct {
	capacityEvaluator CapacityEvaluator
}

func newBaseCapacityUpdater(eva CapacityEvaluator) *baseCapacityUpdater {
	cu := &baseCapacityUpdater{
		capacityEvaluator: eva,
	}
	cu.capacityEvaluator.UpdateActual(1)
	return cu
}

func (b *baseCapacityUpdater) Succeed() {
	panic("implement me")
}

func (b *baseCapacityUpdater) Failed() {
	panic("implement me")
}
