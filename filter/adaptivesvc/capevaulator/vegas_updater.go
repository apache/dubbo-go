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

package capevaulator

import (
	"math"
	"time"
)

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

func (u *vegasUpdater) Succeed() {
	u.updateAfterReturn()
}

func (u *vegasUpdater) Failed() {
	u.updateAfterReturn()
}

func (u *vegasUpdater) updateAfterReturn() {
	u.eva.Actual.Add(-1)
	u.updateRTTs()

	u.reevaEstCap()
}

func (u *vegasUpdater) updateRTTs() {
	// update BaseRTT
	curRTT := uint64(time.Now().Sub(u.startedTime))
	setValueIfLess(u.eva.BaseRTT, curRTT)
	// update MinRTT
	setValueIfLess(u.eva.MinRTT, curRTT)
	// update CntRTT
	u.eva.CntRTT.Add(1)
}

// reevaEstCap reevaluates estimated capacity if the round
func (u *vegasUpdater) reevaEstCap() {
	var (
		nextRoundLeftBound uint64
		rtt                uint64
		baseRTT            uint64

		est       uint64
		newEst    uint64
		thresh    uint64
		newThresh uint64

		target uint64
		diff   uint64
	)
Loop:
	for after(u.eva.Seq, u.eva.NextRoundLeftBound) {
		nextRoundLeftBound = u.eva.NextRoundLeftBound.Load()
		rtt = u.eva.MinRTT.Load()
		baseRTT = u.eva.BaseRTT.Load()

		thresh = u.eva.Threshold.Load()
		est = u.eva.EstimatedCapacity()

		target = est * baseRTT / rtt
		diff = est * (rtt - baseRTT) / baseRTT

		if diff > u.eva.Gamma && est <= thresh {
			// going too fast, slow down.
			newEst = minUint64(est, target+1)
			newThresh = minUint64(thresh, newEst-1)
			if u.eva.Estimated.CAS(est, newEst) &&
				u.eva.Threshold.CAS(thresh, newThresh) {
				u.afterReevaEstCap(nextRoundLeftBound)
				break Loop
			}
		} else if est <= thresh {
			// slow start
			if slowStart(u.eva.Estimated, est, thresh) {
				u.afterReevaEstCap(nextRoundLeftBound)
				break Loop
			}
		} else {
			// congestion avoidance
			if diff > u.eva.Beta {
				// too fast, slow down
				newEst = est - 1
				newThresh = minUint64(thresh, newEst-1)
				if u.eva.Estimated.CAS(est, newEst) &&
					u.eva.Threshold.CAS(thresh, newThresh) {
					u.afterReevaEstCap(nextRoundLeftBound)
					break Loop
				}
			} else if diff < u.eva.Alpha {
				// too slow, speed up
				newEst = est + 1
				if u.eva.Estimated.CAS(est, newEst) {
					u.afterReevaEstCap(nextRoundLeftBound)
					break Loop
				}
			} else {
				// as fast as we should be
			}
		}
	}
}

func (u *vegasUpdater) afterReevaEstCap(nextRoundLeftBound uint64) {
	// update next round
	if !u.eva.NextRoundLeftBound.CAS(nextRoundLeftBound, nextRoundLeftBound+u.eva.RoundSize) {
		// if the round is updated, do nothing
		return
	}
	// reset MinRTT & CntRTT
	u.eva.MinRTT.Store(math.MaxUint64)
	u.eva.CntRTT.Store(0)
}
