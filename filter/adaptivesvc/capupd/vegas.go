package capupd

import (
	"dubbo.apache.org/dubbo-go/v3/filter/adaptivesvc/capeva"
	"time"
)

type Vegas struct {
	*baseCapacityUpdater

	startedTime time.Time
}

func NewVegas(eva capeva.CapacityEvaluator) *Vegas {
	return &Vegas{
		baseCapacityUpdater: newBaseCapacityUpdater(eva),
		startedTime:         time.Now(),
	}
}

func (v *Vegas) Succeed() {
	v.capacityEvaluator.UpdateActual(-1)
}

func (v *Vegas) Failed() {
	v.capacityEvaluator.UpdateActual(-1)
}
