package capevaulator

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

func slowStart(est *atomic.Uint64, estValue, threshValue uint64) bool {
	newEst := minUint64(estValue*2, threshValue)
	return est.CAS(estValue, newEst)
}

func minUint64(lhs, rhs uint64) uint64 {
	if lhs < rhs {
		return lhs
	}
	return rhs
}
