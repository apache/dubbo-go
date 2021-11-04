package capeva

import "go.uber.org/atomic"

func after(crt, next *atomic.Uint64) bool {
	return crt.Load() > next.Load()
}

// setValueIfLess sets newValue to v if newValue is less than v
func setValueIfLess(v *atomic.Uint64, newValue uint64) {
	vuint64 := v.Load()
	for vuint64 > newValue {
		if !v.CAS(vuint64, newValue) {
			vuint64 = v.Load()
		}
	}
}

func slowStart(est, thresh *atomic.Uint64) {
	var estValue, threshValue, newEst uint64

	for {
		estValue = est.Load()
		threshValue = thresh.Load()

		newEst = estValue * 2
		if threshValue < newEst {
			newEst = threshValue
		}

		if est.CAS(estValue, newEst) {
			break
		}
	}
}
