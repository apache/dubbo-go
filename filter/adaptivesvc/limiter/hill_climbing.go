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

	radicalPeriod = 1000 * time.Millisecond
	stablePeriod  = 32000 * time.Millisecond
)

// HillClimbing is a limiter using HillClimbing algorithm
type HillClimbing struct {
	seq   *atomic.Uint64
	round *atomic.Uint64

	inflight   *atomic.Uint64
	limitation *atomic.Uint64

	mutex *sync.Mutex
	// nextUpdateTime = lastUpdatedTime + updateInterval
	updateInterval  *atomic.Duration
	lastUpdatedTime *atomic.Time

	// metrics of the current round
	transactionNum *atomic.Uint64
	rttAvg         *atomic.Float64

	// best metrics in the history
	bestMaxCapacity *atomic.Float64
	bestRTTAvg      *atomic.Float64
	bestLimitation  *atomic.Uint64
	bestTPS         *atomic.Uint64
}

func NewHillClimbing() Limiter {
	l := &HillClimbing{
		seq:             new(atomic.Uint64),
		round:           new(atomic.Uint64),
		inflight:        new(atomic.Uint64),
		limitation:      atomic.NewUint64(initialLimitation),
		mutex:           new(sync.Mutex),
		updateInterval:  atomic.NewDuration(radicalPeriod),
		lastUpdatedTime: atomic.NewTime(time.Now()),
		transactionNum:  new(atomic.Uint64),
		rttAvg:          new(atomic.Float64),
		bestMaxCapacity: new(atomic.Float64),
		bestRTTAvg:      atomic.NewFloat64(math.MaxFloat64),
		bestLimitation:  new(atomic.Uint64),
		bestTPS:         new(atomic.Uint64),
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
		u.seq, inflight, u.startTime)
	return u
}

func (u *HillClimbingUpdater) DoUpdate() error {
	defer func() {
		u.limiter.inflight.Dec()
	}()
	VerboseDebugf("[HillClimbingUpdater] A request finished, the limiter will be updated, seq: %d.", u.seq)

	rtt := uint64(time.Now().Sub(u.startTime).Milliseconds())
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
	transactionNum := u.limiter.transactionNum.Load()
	limitation := u.limiter.limitation.Load()

	// the current option is expired
	if now.Before(lastUpdatedTime) {
		return option, nil
	}

	if now.Sub(lastUpdatedTime) > updateInterval || rttAvg == 0 {
		// the current req is on the next round or no rttAvg.

		// FIXME(justxuewei): If all requests in one round not receive responses, rttAvg will be 0, and maxCapacity will
		//  be 0 as well, the actual maxCapacity, however, is not 0.
		maxCapacity := float64(transactionNum) * float64(updateInterval.Milliseconds()) / rttAvg
		VerboseDebugf("[HillClimbingUpdater] maxCapacity: %f, transactionNum: %d, rttAvg: %f, bestRTTAvg: %f, "+
			"updateInterval: %d",
			maxCapacity, transactionNum, rttAvg, u.limiter.bestRTTAvg.Load(), updateInterval.Milliseconds())

		// Consider extending limitation if concurrent is about to reach the limitation.
		if u.limiter.bestRTTAvg.Load() == math.MaxFloat64 || uint64(maxCapacity*1.5) > limitation {
			if updateInterval == radicalPeriod {
				option = HillClimbingOptionExtendPlus
			} else {
				option = HillClimbingOptionExtend
			}
		}

		tps := uint64(1000.0 * float64(transactionNum) / float64(updateInterval.Milliseconds()))
		VerboseDebugf("[HillClimbingUpdater] The TPS is %d, transactionNum: %d, updateInterval: %d.",
			tps, transactionNum, updateInterval)

		if tps > u.limiter.bestTPS.Load() {
			VerboseDebugf("[HillClimbingUpdater] The best TPS is updated from %d to %d.",
				u.limiter.bestTPS.Load(), tps)
			// tps is the best in the history, update
			// all best metrics.
			u.limiter.bestTPS.Store(tps)
			u.limiter.bestRTTAvg.Store(rttAvg)
			u.limiter.bestMaxCapacity.Store(maxCapacity)
			u.limiter.bestLimitation.Store(u.limiter.limitation.Load())
			VerboseDebugf("[HillClimbingUpdater] Best-metrics are up-to-date, "+
				"seq: %d, bestTPS: %d, bestRTTAvg: %.4f, bestMaxCapacity: %d,"+
				" bestLimitation: %d.", u.seq, u.limiter.bestTPS.Load(),
				u.limiter.bestRTTAvg.Load(), u.limiter.bestMaxCapacity.Load(),
				u.limiter.bestLimitation.Load())
		} else {
			VerboseDebugf("[HillClimbingUpdater] The best TPS is not updated, best TPS is %d, "+
				"the current TPS is %d",
				u.limiter.bestTPS.Load(), tps)
			if u.shouldShrink(transactionNum, maxCapacity, tps, rttAvg) {
				if updateInterval == radicalPeriod {
					option = HillClimbingOptionShrinkPlus
				} else {
					option = HillClimbingOptionShrink
				}
				// shrinking limitation means the process of adjusting
				// limitation goes to stable, so extends the update
				// interval to avoid adjusting frequently.
				u.limiter.updateInterval.Store(minDuration(updateInterval*2, stablePeriod))
			}
		}

		// reset metrics for the new round
		u.limiter.transactionNum.Store(0)
		u.limiter.rttAvg.Store(float64(rtt))
		u.limiter.lastUpdatedTime.Store(time.Now())
		VerboseDebugf("[HillClimbingUpdater] A new round is applied, all metrics are reset.")
	} else {
		// still on the current round

		u.limiter.transactionNum.Add(1)
		// ra = (ra * c  + r) / (c + 1), where ra denotes rttAvg,
		// c denotes transactionNum, r denotes rtt.
		u.limiter.rttAvg.Store((rttAvg*float64(transactionNum) + float64(rtt)) / float64(transactionNum+1))
		option = HillClimbingOptionDoNothing
	}

	return option, nil
}

func (u *HillClimbingUpdater) shouldShrink(transactionNum uint64, maxCapacity float64, tps uint64, rttAvg float64) bool {
	//bestTPS := u.limiter.bestTPS.Load()
	bestMaxCapacity := u.limiter.bestMaxCapacity.Load()
	bestRTTAvg := u.limiter.bestRTTAvg.Load()

	diff := bestMaxCapacity - maxCapacity
	diffPct := uint64(100.0 * diff / bestMaxCapacity)

	VerboseDebugf("[HillClimbingUpdater] shouldShrink maxCapacity diff: %f, diffPct: %d.", diff, diffPct)

	if diff <= 300 && diffPct <= 10 {
		// diff is acceptable, shouldn't shrink
		return false
	}

	if diff > 0 || rttAvg > bestRTTAvg {
		// The unacceptable diff dues to too large maxCapacity or rttAvg.
		rttAvgDiff := uint64(rttAvg - bestRTTAvg)
		rttAvgDiffPct := uint64(100.0 * rttAvg / bestRTTAvg)

		// TODO(justxuewei): Hard-coding here is not proper, but it should refactor after testing.
		var (
			rttAvgDiffThreshold    uint64
			rttAvgDiffPctThreshold uint64
		)
		if bestRTTAvg < 5 {
			rttAvgDiffThreshold = 3
			rttAvgDiffPctThreshold = 80
		} else if bestRTTAvg < 10 {
			rttAvgDiffThreshold = 2
			rttAvgDiffPctThreshold = 30
		} else if bestRTTAvg < 50 {
			rttAvgDiffThreshold = 5
			rttAvgDiffPctThreshold = 20
		} else if bestRTTAvg < 100 {
			rttAvgDiffThreshold = 10
			rttAvgDiffPctThreshold = 10
		} else {
			rttAvgDiffThreshold = 20
			rttAvgDiffPctThreshold = 5
		}

		VerboseDebugf("[HillClimbingUpdater] shouldShrink bestRTTAvg: %d, rttAvgDiff: %d, rttAvgDiffPct: %d, "+
			"rttAvgDiffThreshold: %d, rttAvgDiffPctThreshold: %d.", bestRTTAvg, rttAvgDiff, rttAvgDiffPct,
			rttAvgDiffPctThreshold, rttAvgDiffPctThreshold)

		return (diffPct > 10 && diff > 5) &&
			(rttAvgDiff > rttAvgDiffThreshold || rttAvgDiffPct >= rttAvgDiffPctThreshold)
	}

	return false
}

func (u *HillClimbingUpdater) adjustLimitation(option HillClimbingOption) error {
	if option == HillClimbingOptionDoNothing {
		VerboseDebugf("[HillClimbingUpdater] The option is do nothing, the limitation will not be updated.")
		return nil
	}

	limitation := float64(u.limiter.limitation.Load())
	oldLimitation := limitation
	bestLimitation := float64(u.limiter.bestLimitation.Load())
	updateInterval := u.limiter.updateInterval.Load()
	alpha := 1.5 * math.Log(limitation)
	beta := 0.8 * math.Log(limitation)
	logUpdateInterval := math.Max(1.0, math.Log2(float64(updateInterval.Milliseconds())/1000.0))

	VerboseDebugf("[HillClimbingUpdater] Before calculating new limitation, option: %d, limitation: %f, "+
		"bestLimitation: %f, alpha: %f, beta: %f, logUpdateInterval: %f, updateInterval: %d", option, limitation,
		bestLimitation, alpha, beta, logUpdateInterval, updateInterval.Milliseconds())

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
