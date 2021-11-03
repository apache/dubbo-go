package capeva

import (
	"dubbo.apache.org/dubbo-go/v3/filter/adaptivesvc/capupd"
	"go.uber.org/atomic"
)

type Vegas struct {
	*baseCapacityEvaluator

	cntRTT *atomic.Uint32
}

func NewVegas() *Vegas {
	return &Vegas{
		baseCapacityEvaluator: newBaseCapacityEvaluator(),
	}
}

func (v *Vegas) NewCapacityUpdater() capupd.CapacityUpdater {
	return capupd.NewVegas(v)
}