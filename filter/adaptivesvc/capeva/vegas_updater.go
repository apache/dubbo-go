package capeva

import "time"

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

func (v *vegasUpdater) Succeed() {
	v.updateAfterReturn()
}

func (v *vegasUpdater) Failed() {
	v.updateAfterReturn()
}

func (v *vegasUpdater) updateAfterReturn() {
	v.eva.Actual.Add(-1)
	v.updateRTTs()
}

func (v *vegasUpdater) updateRTTs() {
	// update BaseRTT
	curRTT := uint64(time.Now().Sub(v.startedTime))
	oldBaseRTT := v.eva.BaseRTT.Load()
	for oldBaseRTT > curRTT {
		if !v.eva.BaseRTT.CAS(oldBaseRTT, curRTT) {
			oldBaseRTT = v.eva.BaseRTT.Load()
		}
	}
	// update MinRTT
	// update CntRTT
}
