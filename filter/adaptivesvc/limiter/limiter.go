package limiter

import (
	"fmt"
)

var ErrReachLimitation = fmt.Errorf("reach limitation")

var (
	Verbose = false
)

type Limiter interface {
	Inflight() uint64
	Remaining() uint64
	Acquire() (Updater, error)
}

type Updater interface {
	DoUpdate(rtt, inflight uint64) error
}
