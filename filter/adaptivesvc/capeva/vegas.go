package capeva

import (
	"go.uber.org/atomic"
	"time"
)

const (
	defaultRoundSize uint64 = 100
)

// Vegas is a capacity evaluator using Vegas congestion avoid.
// RTT is not exactly same as TCP's,
// in this case, RTT means the time that the provider perform a method.
type Vegas struct {
	*baseCapacityEvaluator

	// RoundSize specifies the size of the round, which reflects on the speed of updating estimation.
	// The minRTT and cntRTT will be reset in the next round.
	RoundSize uint64

	seq          *atomic.Uint64
	nextRoundSeq *atomic.Uint64

	baseRTT *atomic.Uint64
	minRTT  *atomic.Uint64
	cntRTT  *atomic.Uint64
}

func NewVegas() *Vegas {
	return &Vegas{
		baseCapacityEvaluator: newBaseCapacityEvaluator(),
		RoundSize: defaultRoundSize,
		seq: &atomic.Uint64{},
		nextRoundSeq: &atomic.Uint64{},
		baseRTT: &atomic.Uint64{},
		minRTT: &atomic.Uint64{},
		cntRTT: &atomic.Uint64{},
	}
}

func (v *Vegas) NewCapacityUpdater() CapacityUpdater {
	return newVegasUpdater(v)
}

type VegasUpdater struct {
	*baseCapacityUpdater

	startedTime time.Time
}

func newVegasUpdater(eva CapacityEvaluator) CapacityUpdater {
	return &VegasUpdater{
		baseCapacityUpdater: newBaseCapacityUpdater(eva),
		startedTime:         time.Now(),
	}
}

func (v *VegasUpdater) Succeed() {
	v.update()
}

func (v *VegasUpdater) Failed() {
	v.update()
}

func (v *VegasUpdater) update() {
	v.capacityEvaluator.UpdateActual(-1)
}
