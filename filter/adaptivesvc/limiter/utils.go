package limiter

import (
	"dubbo.apache.org/dubbo-go/v3/common/logger"
)

//var (
//	decay = 0.95
//	// cpu statistics interval (ms)
//	statInterval time.Duration = 500
//	cpu *atomic.Uint64
//)
//
//func init() {
//	cpu = new(atomic.Uint64)
//	// get cpu usage statistics regularly
//	go cpuStat()
//}
//
//func cpuStat() {
//	t := time.NewTicker(time.Microsecond * statInterval)
//
//	// prevent cpuStat method from crashing unexpectedly
//	defer func() {
//		t.Stop()
//		if err := recover(); err != nil {
//			logger.Warnf("[HillClimbing] cpuStat went down, err: %v, attempting to restart...", err)
//			go cpuStat()
//		}
//	}()
//
//	for range t.C {
//		stat :=
//	}
//}

func VerboseDebugf(msg string, args ...interface{})  {
	if !Verbose {
		return
	}
	logger.Debugf(msg, args...)
}

func minUint64(lhs, rhs uint64) uint64 {
	if lhs < rhs {
		return lhs
	}
	return rhs
}
