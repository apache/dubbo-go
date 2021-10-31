package capeva

import (
	"errors"
	"go.uber.org/atomic"
)

var ErrInvalidActualValue = errors.New("invalid actual value")

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

func (ce *baseCapacityEvaluator) UpdateEstimated(value int64) error {
	ce.actual.Store(value)
	return nil
}

func (ce *baseCapacityEvaluator) UpdateActual(value int64) error {
	if ce.actual.Load() > ce.estimated.Load() {
		return ErrInvalidActualValue
	}
	ce.actual.Store(value)
	return nil
}

