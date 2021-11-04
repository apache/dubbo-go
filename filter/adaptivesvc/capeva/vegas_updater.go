package capeva

import (
	"go.uber.org/atomic"
	"math"
	"time"
)

type vegasUpdater struct {
	eva *Vegas

	startedTime time.Time
}

func newVegasUpdater(eva *Vegas) CapacityUpdater {
	return &vegasUpdater{
		eva:         eva,
		startedTime: time.Now(),
	}
}

func (u *vegasUpdater) Succeed() {
	u.updateAfterReturn()
}

func (u *vegasUpdater) Failed() {
	u.updateAfterReturn()
}

func (u *vegasUpdater) updateAfterReturn() {
	u.eva.Actual.Add(-1)
	u.updateRTTs()

	u.reevaEstCap()
}

func (u *vegasUpdater) updateRTTs() {
	// update BaseRTT
	curRTT := uint64(time.Now().Sub(u.startedTime))
	setValueIfLess(u.eva.BaseRTT, curRTT)
	// update MinRTT
	setValueIfLess(u.eva.MinRTT, curRTT)
	// update CntRTT
	u.eva.CntRTT.Add(1)
}

// reevaEstCap reevaluates estimated capacity if the round
func (u *vegasUpdater) reevaEstCap() {
	if after(u.eva.Seq, u.eva.NextRoundLeftBound) {
		// update next round
		u.eva.NextRoundLeftBound.Add(u.eva.RoundSize)

		rtt := u.eva.MinRTT.Load()
		baseRTT := u.eva.BaseRTT.Load()

		thresh := u.eva.Threshold.Load()
		estCap := u.eva.EstimatedCapacity()

		targetEstCap := estCap * baseRTT / rtt
		diff := estCap * (rtt - baseRTT) / baseRTT

		if diff > u.eva.Gamma && estCap <= thresh {

		}

		// reset MinRTT & CntRTT
		u.eva.MinRTT.Store(math.MaxUint64)
		u.eva.CntRTT.Store(0)
	}
}
