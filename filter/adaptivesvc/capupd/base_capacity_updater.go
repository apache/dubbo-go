package capupd

import "dubbo.apache.org/dubbo-go/v3/filter/adaptivesvc/capeva"

type baseCapacityUpdater struct {
	capacityEvaluator capeva.CapacityEvaluator
}

func newBaseCapacityUpdater(eva capeva.CapacityEvaluator) *baseCapacityUpdater {
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

