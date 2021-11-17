package limiter

import (
	"go.uber.org/atomic"
	"sync"
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

var (
	initialLimitation uint64 = 50
	maxLimitation     uint64 = 500
	radicalPeriod     uint64 = 1000
	stablePeriod      uint64 = 32000
)

// HillClimbing is a limiter using HillClimbing algorithm
type HillClimbing struct {
	seq *atomic.Uint64

	inflight   *atomic.Uint64
	limitation *atomic.Uint64

	lastUpdatedTimeMutex *sync.Mutex
	// nextUpdateTime = lastUpdatedTime + updateInterval
	updateInterval  *atomic.Uint64
	lastUpdatedTime *atomic.Time

	// indicators of the current round
	successCounter *atomic.Uint64
	rttAvg         *atomic.Float64

	bestMutex *sync.Mutex
	// indicators of history
	bestConcurrency *atomic.Uint64
	bestRTTAvg      *atomic.Float64
	bestLimitation  *atomic.Uint64
	bestSuccessRate *atomic.Uint64
}

func NewHillClimbing() Limiter {
	l := &HillClimbing{
		seq:                  new(atomic.Uint64),
		inflight:             new(atomic.Uint64),
		limitation:           new(atomic.Uint64),
		lastUpdatedTimeMutex: new(sync.Mutex),
		updateInterval:       atomic.NewUint64(radicalPeriod),
		lastUpdatedTime:      atomic.NewTime(time.Now()),
		successCounter:       new(atomic.Uint64),
		rttAvg:               new(atomic.Float64),
		bestMutex:            new(sync.Mutex),
		bestConcurrency:      new(atomic.Uint64),
		bestRTTAvg:           new(atomic.Float64),
		bestLimitation:       new(atomic.Uint64),
		bestSuccessRate:      new(atomic.Uint64),
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
	limiter   *HillClimbing

	// for debug purposes
	seq uint64
}

func NewHillClimbingUpdater(limiter *HillClimbing) *HillClimbingUpdater {
	inflight := limiter.inflight.Add(1)
	u := &HillClimbingUpdater{
		startTime: time.Now(),
		limiter:   limiter,
		seq:       limiter.seq.Add(1) - 1,
	}
	VerboseDebugf("[NewHillClimbingUpdater] A new request arrived, seq: %d, inflight: %d, time: %s.",
		u.seq, inflight, u.startTime.String())
	return u
}

func (u *HillClimbingUpdater) DoUpdate(rtt, inflight uint64) error {
	defer func() {
		u.limiter.inflight.Add(-1)
	}()
	VerboseDebugf("[HillClimbingUpdater] A request finished, the limiter will be updated, seq: %d.", u.seq)

	u.limiter.lastUpdatedTimeMutex.Lock()
	// if lastUpdatedTime is updated, terminate DoUpdate immediately
	lastUpdatedTime := u.limiter.lastUpdatedTime.Load()
	u.limiter.lastUpdatedTimeMutex.Unlock()

	option, err := u.getOption(rtt, inflight)
	if err != nil {
		return err
	}
	if u.shouldDrop(lastUpdatedTime) {
		return nil
	}

	if err = u.adjustLimitation(option); err != nil {
		return err
	}
	return nil
}

func (u *HillClimbingUpdater) getOption(rtt, inflight uint64) (HillClimbingOption, error) {
	now := time.Now()
	option := HillClimbingOptionDoNothing

	lastUpdatedTime := u.limiter.lastUpdatedTime.Load()
	updateInterval := u.limiter.updateInterval.Load()
	rttAvg := u.limiter.rttAvg.Load()
	successCounter := u.limiter.successCounter.Load()
	limitation := u.limiter.limitation.Load()

	if now.Sub(lastUpdatedTime) > time.Duration(updateInterval) ||
		rttAvg == 0 {
		// Current req is at the next round or no rttAvg.

		// FIXME(justxuewei): If all requests in one round
		// 	not receive responses, rttAvg will be 0, and
		// 	concurrency will be 0 as well, the actual
		// 	concurrency, however, is not 0.
		concurrency := float64(successCounter) * rttAvg / float64(updateInterval)

		// Consider extending limitation if concurrent is
		// about to reach the limitation.
		if uint64(concurrency*1.5) > limitation {
			if updateInterval == radicalPeriod {
				option = HillClimbingOptionExtendPlus
			} else {
				option = HillClimbingOptionExtend
			}
		}

		successRate := uint64(1000.0 * float64(successCounter) / float64(updateInterval))

		// Wrap the code into an anonymous function due to
		// use defer to ensure the bestMutex is unlocked
		// once the best-indicators is updated.
		isUpdated := func() bool {
			u.limiter.bestMutex.Lock()
			defer u.limiter.bestMutex.Unlock()
			if successRate > u.limiter.bestSuccessRate.Load() {
				// successRate is the best in the history, update
				// all best-indicators.
				u.limiter.bestSuccessRate.Store(successRate)
				u.limiter.bestRTTAvg.Store(rttAvg)
				u.limiter.bestConcurrency.Store(uint64(concurrency))
				u.limiter.bestLimitation.Store(u.limiter.limitation.Load())
				VerboseDebugf("[HillClimbingUpdater] Best-indicators are up-to-date, " +
					"seq: %d, bestSuccessRate: %d, bestRTTAvg: %.4f, bestConcurrency: %d," +
					" bestLimitation: %d.", u.seq, u.limiter.bestSuccessRate.Load(),
					u.limiter.bestRTTAvg.Load(), u.limiter.bestConcurrency.Load(),
					u.limiter.bestLimitation.Load())
				return true
			}
			return false
		}()

		if !isUpdated && u.shouldShrink(successCounter, rttAvg) {

		}

		// reset data for the last round
		u.limiter.successCounter.Store(0)
		u.limiter.rttAvg.Store(float64(rtt))
	} else {
		// still in the current round
		// TODO(justxuewei): [TBD] if needs to protect here using mutex??
		u.limiter.successCounter.Add(1)
	}

	return option, nil
}

func (u *HillClimbingUpdater) shouldShrink(counter uint64, rttAvg float64) bool {


	return false
}

// TODO(justxuewei): update lastUpdatedTime
func (u *HillClimbingUpdater) adjustLimitation(option HillClimbingOption) error {
	u.limiter.lastUpdatedTimeMutex.Lock()
	defer u.limiter.lastUpdatedTimeMutex.Unlock()

	return nil
}

func (u *HillClimbingUpdater) shouldDrop(lastUpdatedTime time.Time) (isDropped bool) {
	if !u.limiter.lastUpdatedTime.Load().Equal(lastUpdatedTime) {
		VerboseDebugf("[HillClimbingUpdater] The limitation is updated by others, drop this update, seq: %d.", u.seq)
		isDropped = true
		return
	}
	return
}
