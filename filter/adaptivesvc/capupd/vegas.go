package capupd

import "dubbo.apache.org/dubbo-go/v3/filter/adaptivesvc/capeva"

type Vegas struct {
	*baseCapacityUpdater
}

func NewVegas(eva capeva.CapacityEvaluator) *Vegas {
	return &Vegas{
		baseCapacityUpdater: newBaseCapacityUpdater(eva),
	}
}

func (v *Vegas) Succeed() {
	v.capacityEvaluator.UpdateActual(-1)
}

func (v *Vegas) Failed() {
	v.capacityEvaluator.UpdateActual(-1)
}
