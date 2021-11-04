package capeva

import (
	"go.uber.org/atomic"
)

const (
	defaultRoundSize uint64 = 100
)

// Vegas is a capacity evaluator using Vegas congestion avoid.
// RTT is not exactly same as TCP's,
// in this case, RTT means the time that the provider perform a method.
type Vegas struct {
	Estimated, Actual *atomic.Uint64

	// RoundSize specifies the size of the round, which reflects on the speed of updating estimation.
	// The MinRTT and CntRTT will be reset in the next round.
	RoundSize uint64

	Seq          *atomic.Uint64
	NextRoundSeq *atomic.Uint64

	BaseRTT *atomic.Uint64
	MinRTT  *atomic.Uint64
	CntRTT  *atomic.Uint64
}

func NewVegas() *Vegas {
	return &Vegas{
		Estimated:    &atomic.Uint64{},
		Actual:       &atomic.Uint64{},
		RoundSize:    defaultRoundSize,
		Seq:          &atomic.Uint64{},
		NextRoundSeq: &atomic.Uint64{},
		BaseRTT:      &atomic.Uint64{},
		MinRTT:       &atomic.Uint64{},
		CntRTT:       &atomic.Uint64{},
	}
}

func (v *Vegas) EstimatedCapacity() int64 {
	panic("implement me")
}

func (v *Vegas) ActualCapacity() int64 {
	panic("implement me")
}

func (v *Vegas) NewCapacityUpdater() CapacityUpdater {
	return newVegasUpdater(v)
}
