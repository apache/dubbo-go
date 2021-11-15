package limiter

import "dubbo.apache.org/dubbo-go/v3/common/logger"

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
