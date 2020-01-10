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

package impl

import (
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Workiva/go-datastructures/slice/skip"

	"github.com/apache/dubbo-go/metrics"
)

const (
	// edr = ExponentiallyDecayingReservoir
	EdrDefaultSize  = 1028
	EdrDefaultAlpha = 0.015
)

var (
	edrRescaleThreshold = time.Hour.Nanoseconds()
)

/**
 * An exponentially-decaying random reservoir of longs. Uses Cormode et al's
 * forward-decaying priority reservoir sampling method to produce a statistically representative
 * sampling reservoir, exponentially biased towards newer entries.
 *
 * see http://dimacs.rutgers.edu/~graham/pubs/papers/fwddecay.pdf
 * Cormode et al. Forward Decay: A Practical Time Decay Model for Streaming Systems. ICDE '09:
 *      Proceedings of the 2009 IEEE International Conference on Data Engineering (2009)
 *
 * In a word, this struct is the priority, forward exponentially-decaying implementation
 */
type ExponentiallyDecayingReservoir struct {
	values        *skip.SkipList
	rwMutex       sync.RWMutex
	alpha         float64
	size          int32
	count         int64
	startTime     int64
	nextScaleTime int64
	clock         metrics.Clock
}

func (rsv *ExponentiallyDecayingReservoir) Size() int {
	if int64(rsv.size) > rsv.count {
		return int(rsv.count)
	}
	return int(rsv.size)
}

func (rsv *ExponentiallyDecayingReservoir) UpdateN(value int64) {
	timestamp := currentTimeInSecond(rsv.clock.GetTime())
	
}

func (rsv *ExponentiallyDecayingReservoir) rescaleIfNeeded() {
	now := rsv.clock.GetTick()
	if now >= rsv.nextScaleTime {
		rsv.rescale(now, rsv.nextScaleTime)
	}
}

/* "A common feature of the above techniques—indeed, the key technique that
 * allows us to track the decayed weights efficiently—is that they maintain
 * counts and other quantities based on g(ti − L), and only scale by g(t − L)
 * at query time. But while g(ti −L)/g(t−L) is guaranteed to lie between zero
 * and one, the intermediate values of g(ti − L) could become very large. For
 * polynomial functions, these values should not grow too large, and should be
 * effectively represented in practice by floating point values without loss of
 * precision. For exponential functions, these values could grow quite large as
 * new values of (ti − L) become large, and potentially exceed the capacity of
 * common floating point types. However, since the values stored by the
 * algorithms are linear combinations of g values (scaled sums), they can be
 * rescaled relative to a new landmark. That is, by the analysis of exponential
 * decay in Section III-A, the choice of L does not affect the final result. We
 * can therefore multiply each value based on L by a factor of exp(−α(L′ − L)),
 * and obtain the correct value as if we had instead computed relative to a new
 * landmark L′ (and then use this new L′ at query time). This can be done with
 * a linear pass over whatever data structure is being used."
 */
func (rsv *ExponentiallyDecayingReservoir) rescale(now int64, next int64)  {

	if atomic.CompareAndSwapInt64(&rsv.nextScaleTime, next, now + edrRescaleThreshold) {
		// win the race condition, so we will lock and then rescale
		rsv.rwMutex.Lock()
		defer rsv.rwMutex.Unlock()
		oldStartTime := rsv.startTime
		rsv.startTime = currentTimeInSecond(rsv.clock.GetTime())

		scalingFactor := math.Exp(-rsv.alpha * float64(rsv.startTime - oldStartTime))

		if scalingFactor == 0 {
			rsv.values = skip.New(int32(0))
			rsv.count = 0
			return
		}

		// rsv.values.Iter(NewWeightSample(math.Min))
	}
}

func (rsv *ExponentiallyDecayingReservoir) GetSnapshot() metrics.Snapshot {
	panic("implement me")
}

func currentTimeInSecond(timeInMs int64) int64 {
	return timeInMs / secondToMsRate
}

func NewExponentiallyDecayingReservoir(size int32, alpha float64, clock metrics.Clock) metrics.Reservoir {
	return &ExponentiallyDecayingReservoir{
		clock:         clock,
		values:        skip.New(int32(1)),
		startTime:     currentTimeInSecond(clock.GetTime()),
		nextScaleTime: clock.GetTick() + edrRescaleThreshold,
	}
}