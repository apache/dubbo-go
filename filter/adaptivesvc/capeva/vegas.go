package capeva

import (
	"go.uber.org/atomic"
	"math"
)

const (
	defaultThreshold uint64 = 100

	defaultRoundSize uint64 = 10

	defaultAlpha uint64 = 2
	defaultBeta  uint64 = 4
	defaultGamma uint64 = 1
)

// Vegas is a capacity evaluator using Vegas congestion avoid.
// RTT is not exactly same as TCP's,
// in this case, RTT means the time that the provider perform a method.
type Vegas struct {
	Estimated, Actual, Threshold *atomic.Uint64

	// TODO(justxuewei): load values from config
	Alpha, Beta, Gamma uint64

	// RoundSize specifies the size of the round, which reflects on the speed of updating estimation.
	// The MinRTT and CntRTT will be reset in the next round.
	// The smaller RoundSize is, the faster reevaluating estimated capacity is.
	RoundSize uint64

	Seq                *atomic.Uint64
	NextRoundLeftBound *atomic.Uint64

	BaseRTT *atomic.Uint64
	MinRTT  *atomic.Uint64
	CntRTT  *atomic.Uint64 // not used so far
}

func NewVegas() *Vegas {
	estimated := atomic.NewUint64(1)
	threshold := atomic.NewUint64(defaultThreshold)

	minRTT := atomic.NewUint64(math.MaxUint64)

	return &Vegas{
		Estimated: estimated,
		Actual:    &atomic.Uint64{},
		Threshold: threshold,

		Alpha:     defaultAlpha,
		Beta:      defaultBeta,
		Gamma:     defaultGamma,
		RoundSize: defaultRoundSize,

		Seq:                &atomic.Uint64{},
		NextRoundLeftBound: &atomic.Uint64{},

		BaseRTT: &atomic.Uint64{},
		MinRTT:  minRTT,
		CntRTT:  &atomic.Uint64{},
	}
}

func (v *Vegas) EstimatedCapacity() uint64 {
	return v.Estimated.Load()
}

func (v *Vegas) ActualCapacity() uint64 {
	return v.Actual.Load()
}

func (v *Vegas) NewCapacityUpdater() CapacityUpdater {
	return newVegasUpdater(v)
}
