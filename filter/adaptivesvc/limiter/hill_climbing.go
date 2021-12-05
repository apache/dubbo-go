/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package limiter

import (
	"math"
	"sync"
	"time"
)

import (
	"go.uber.org/atomic"
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
	seq   *atomic.Uint64
	round *atomic.Uint64

	inflight   *atomic.Uint64
	limitation *atomic.Uint64

	mutex *sync.Mutex
	// nextUpdateTime = lastUpdatedTime + updateInterval
	updateInterval  *atomic.Uint64
	lastUpdatedTime *atomic.Time

	// indicators of the current round
	successCounter *atomic.Uint64
	rttAvg         *atomic.Float64

	// indicators of history
	bestConcurrency *atomic.Uint64
	bestRTTAvg      *atomic.Float64
	bestLimitation  *atomic.Uint64
	bestSuccessRate *atomic.Uint64
}

func NewHillClimbing() Limiter {
	l := &HillClimbing{
		seq:             new(atomic.Uint64),
		round:           new(atomic.Uint64),
		inflight:        new(atomic.Uint64),
		limitation:      atomic.NewUint64(initialLimitation),
		mutex:           new(sync.Mutex),
		updateInterval:  atomic.NewUint64(radicalPeriod),
		lastUpdatedTime: atomic.NewTime(time.Now()),
		successCounter:  new(atomic.Uint64),
		rttAvg:          new(atomic.Float64),
		bestConcurrency: new(atomic.Uint64),
		bestRTTAvg:      new(atomic.Float64),
		bestLimitation:  new(atomic.Uint64),
		bestSuccessRate: new(atomic.Uint64),
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

func (u *HillClimbingUpdater) DoUpdate() error {
	defer func() {
		u.limiter.inflight.Dec()
	}()
	VerboseDebugf("[HillClimbingUpdater] A request finished, the limiter will be updated, seq: %d.", u.seq)

	rtt := uint64(time.Now().Sub(u.startTime))
	inflight := u.limiter.Inflight()

	option, err := u.getOption(rtt, inflight)
	if err != nil {
		return err
	}
	if err = u.adjustLimitation(option); err != nil {
		return err
	}
	return nil
}

func (u *HillClimbingUpdater) getOption(rtt, _ uint64) (HillClimbingOption, error) {
	u.limiter.mutex.Lock()
	defer u.limiter.mutex.Unlock()

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

		if successRate > u.limiter.bestSuccessRate.Load() {
			// successRate is the best in the history, update
			// all best-indicators.
			u.limiter.bestSuccessRate.Store(successRate)
			u.limiter.bestRTTAvg.Store(rttAvg)
			u.limiter.bestConcurrency.Store(uint64(concurrency))
			u.limiter.bestLimitation.Store(u.limiter.limitation.Load())
			VerboseDebugf("[HillClimbingUpdater] Best-indicators are up-to-date, "+
				"seq: %d, bestSuccessRate: %d, bestRTTAvg: %.4f, bestConcurrency: %d,"+
				" bestLimitation: %d.", u.seq, u.limiter.bestSuccessRate.Load(),
				u.limiter.bestRTTAvg.Load(), u.limiter.bestConcurrency.Load(),
				u.limiter.bestLimitation.Load())
		} else {
			if u.shouldShrink(successCounter, uint64(concurrency), successRate, rttAvg) {
				if updateInterval == radicalPeriod {
					option = HillClimbingOptionShrinkPlus
				} else {
					option = HillClimbingOptionShrink
				}
				// shrinking limitation means the process of adjusting
				// limitation goes to stable, so extends the update
				// interval to avoid adjusting frequently.
				u.limiter.updateInterval.Store(minUint64(updateInterval*2, stablePeriod))
			}
		}

		// reset indicators for the new round
		u.limiter.successCounter.Store(0)
		u.limiter.rttAvg.Store(float64(rtt))
		u.limiter.lastUpdatedTime.Store(time.Now())
		VerboseDebugf("[HillClimbingUpdater] A new round is applied, all indicators are reset.")
	} else {
		// still in the current round

		u.limiter.successCounter.Add(1)
		// ra = (ra * c  + r) / (c + 1), where ra denotes rttAvg,
		// c denotes successCounter, r denotes rtt.
		u.limiter.rttAvg.Store((rttAvg*float64(successCounter) + float64(rtt)) / float64(successCounter+1))
		option = HillClimbingOptionDoNothing
	}

	return option, nil
}

func (u *HillClimbingUpdater) shouldShrink(counter, concurrency, successRate uint64, rttAvg float64) bool {
	bestSuccessRate := u.limiter.bestSuccessRate.Load()
	bestRTTAvg := u.limiter.bestRTTAvg.Load()

	diff := bestSuccessRate - successRate
	diffPct := uint64(100.0 * float64(successRate) / float64(bestSuccessRate))

	if diff <= 300 && diffPct <= 10 {
		// diff is acceptable, shouldn't shrink
		return false
	}

	if concurrency > bestSuccessRate || rttAvg > bestRTTAvg {
		// The unacceptable diff dues to too large
		// concurrency or rttAvg.
		concDiff := concurrency - bestSuccessRate
		concDiffPct := uint64(100.0 * float64(concurrency) / float64(bestSuccessRate))
		rttAvgDiff := rttAvg - bestRTTAvg
		rttAvgPctDiff := uint64(100.0 * rttAvg / bestRTTAvg)

		// TODO(justxuewei): Hard-coding here is not proper, but
		// 	it should refactor after testing.
		var (
			rttAvgDiffThreshold    uint64
			rttAvgPctDiffThreshold uint64
		)
		if bestRTTAvg < 5 {
			rttAvgDiffThreshold = 3
			rttAvgPctDiffThreshold = 80
		} else if bestRTTAvg < 10 {
			rttAvgDiffThreshold = 2
			rttAvgPctDiffThreshold = 30
		} else if bestRTTAvg < 50 {
			rttAvgDiffThreshold = 5
			rttAvgPctDiffThreshold = 20
		} else if bestRTTAvg < 100 {
			rttAvgDiffThreshold = 10
			rttAvgPctDiffThreshold = 10
		} else {
			rttAvgDiffThreshold = 20
			rttAvgPctDiffThreshold = 5
		}

		return (concDiffPct > 10 && concDiff > 5) && (uint64(rttAvgDiff) > rttAvgDiffThreshold || rttAvgPctDiff >= rttAvgPctDiffThreshold)
	}

	return false
}

func (u *HillClimbingUpdater) adjustLimitation(option HillClimbingOption) error {
	limitation := float64(u.limiter.limitation.Load())
	oldLimitation := limitation
	bestLimitation := float64(u.limiter.bestLimitation.Load())
	alpha := 1.5 * math.Log(limitation)
	beta := 0.8 * math.Log(limitation)
	logUpdateInterval := math.Log2(float64(u.limiter.updateInterval.Load()) / 1000.0)

	switch option {
	case HillClimbingOptionExtendPlus:
		limitation += alpha / logUpdateInterval
	case HillClimbingOptionExtend:
		limitation += beta / logUpdateInterval
	case HillClimbingOptionShrinkPlus:
		limitation = bestLimitation - alpha/logUpdateInterval
	case HillClimbingOptionShrink:
		limitation = bestLimitation - beta/logUpdateInterval
	}

	limitation = math.Max(1.0, math.Min(limitation, float64(maxLimitation)))
	u.limiter.limitation.Store(uint64(limitation))
	VerboseDebugf("[HillClimbingUpdater] The limitation is update from %d to %d.", uint64(oldLimitation), uint64(limitation))
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
