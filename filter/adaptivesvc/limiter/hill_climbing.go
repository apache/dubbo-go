package limiter

import (
	"go.uber.org/atomic"
	"time"
)

var (
	_ Limiter = (*HillClimbing)(nil)
	_ Updater = (*HillClimbingUpdater)(nil)
)

type HillClimbingOption int64

const (
	HillClimbingOptionShrinkPlus HillClimbingOption = -2
	HillClimbingOptionShrink     HillClimbingOption = -1
	HillClimbingOptionDoNothing  HillClimbingOption = 0
	HillClimbingOptionExtend     HillClimbingOption = 1
	HillClimbingOptionExtendPlus HillClimbingOption = 2
)

// HillClimbing is a limiter using HillClimbing algorithm
type HillClimbing struct {
	seq *atomic.Uint64

	inflight   *atomic.Uint64
	limitation *atomic.Uint64
}

func NewHillClimbing() Limiter {
	l := &HillClimbing{
		seq:        new(atomic.Uint64),
		inflight:   new(atomic.Uint64),
		limitation: new(atomic.Uint64),
	}

	return l
}

func (l *HillClimbing) Inflight() uint64 {
	return l.inflight.Load()
}

func (l *HillClimbing) Remaining() uint64 {
	limitation := l.limitation.Load()
	inflight := l.Inflight()
	if limitation < inflight {
		return 0
	}
	return limitation - inflight
}

func (l *HillClimbing) Acquire() (Updater, error) {
	if l.Remaining() == 0 {
		return nil, ErrReachLimitation
	}
	return NewHillClimbingUpdater(l), nil
}

type HillClimbingUpdater struct {
	startTime time.Time
	seq       uint64
	limiter   *HillClimbing

	updateInterval  *atomic.Uint64
	lastUpdatedTime time.Time
	successCounter *atomic.Uint64
}

func NewHillClimbingUpdater(limiter *HillClimbing) *HillClimbingUpdater {
	inflight := limiter.inflight.Add(1)
	u := &HillClimbingUpdater{
		startTime: time.Now(),
		seq:       limiter.seq.Add(1) - 1,
		limiter:   limiter,
	}
	VerboseDebugf("[NewHillClimbingUpdater] A new request arrived, seq: %d, inflight: %d, time: %s.",
		u.seq, inflight, u.startTime.String())
	return u
}

func (u *HillClimbingUpdater) DoUpdate(rtt, inflight uint64) error {
	defer func() {
		u.limiter.inflight.Add(-1)
	}()
	VerboseDebugf("[HillClimbingUpdater.DoUpdate] A request finished, the limiter will be updated, seq: %d.", u.seq)
	option, err := u.getOption(rtt, inflight)
	if err != nil {
		return err
	}
	if err = u.adjustLimitation(option); err != nil {
		return err
	}
	return nil
}

func (u *HillClimbingUpdater) getOption(rtt, inflight uint64) (HillClimbingOption, error) {
	var option HillClimbingOption

	return option, nil
}

func (u *HillClimbingUpdater) shouldShrink(counter uint64, rttAvg float64) bool {
	return false
}

func (u *HillClimbingUpdater) adjustLimitation(option HillClimbingOption) error {
	return nil
}
