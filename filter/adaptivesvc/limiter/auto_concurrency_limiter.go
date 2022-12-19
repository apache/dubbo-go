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
	"math/rand"
	"sync"
	"time"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"go.uber.org/atomic"
)

import (
	"dubbo.apache.org/dubbo-go/v3/filter/adaptivesvc/limiter/cpu"
)

var (
	_       Limiter        = (*AutoConcurrency)(nil)
	_       Updater        = (*AutoConcurrencyUpdater)(nil)
	cpuLoad *atomic.Uint64 = atomic.NewUint64(0) // range from 0 to 1000
)

// These parameters may need to be different between services
const (
	MaxExploreRatio    = 0.3
	MinExploreRatio    = 0.06
	SampleWindowSizeMs = 1000
	MinSampleCount     = 40
	MaxSampleCount     = 500
	CPUDecay           = 0.95
)

type AutoConcurrency struct {
	sync.RWMutex

	exploreRatio         float64
	emaFactor            float64
	noLoadLatency        float64 // duration
	maxQPS               float64
	halfSampleIntervalMS int64
	maxConcurrency       uint64

	// metrics of the current round
	startSampleTimeUs  int64
	lastSamplingTimeUs *atomic.Int64
	resetLatencyUs     int64 // time to reset noLoadLatency
	remeasureStartUs   int64 // time to reset req data (sampleCount, totalSampleUs, totalReqCount)
	sampleCount        int64
	totalSampleUs      int64
	totalReqCount      *atomic.Int64

	inflight *atomic.Uint64
}

func init() {
	go cpuproc()
}

// cpu = cpuᵗ⁻¹ * decay + cpuᵗ * (1 - decay)
func cpuproc() {
	ticker := time.NewTicker(time.Millisecond * 500) // same to cpu sample rate
	defer func() {
		ticker.Stop()
		if err := recover(); err != nil {
			logger.Warnf("cpu usage collector panic: %v", err)
			go cpuproc()
		}
	}()

	for range ticker.C {
		usage := cpu.CpuUsage()
		prevCPU := cpuLoad.Load()
		curCPU := uint64(float64(prevCPU)*CPUDecay + float64(usage)*(1.0-CPUDecay))
		logger.Debugf("current cpu usage: %d", curCPU)
		cpuLoad.Store(curCPU)
	}
}

func CPUUsage() uint64 {
	return cpuLoad.Load()
}

func NewAutoConcurrencyLimiter() *AutoConcurrency {
	l := &AutoConcurrency{
		exploreRatio:         MaxExploreRatio,
		emaFactor:            0.1,
		noLoadLatency:        -1,
		maxQPS:               -1,
		maxConcurrency:       40,
		halfSampleIntervalMS: 25000,
		resetLatencyUs:       0,
		inflight:             atomic.NewUint64(0),
		lastSamplingTimeUs:   atomic.NewInt64(0),
		totalReqCount:        atomic.NewInt64(0),
	}
	l.remeasureStartUs = l.NextResetTime(time.Now().UnixNano() / 1e3)
	return l
}

func (l *AutoConcurrency) updateNoLoadLatency(latency float64) {
	emaFactor := l.emaFactor
	if l.noLoadLatency <= 0 {
		l.noLoadLatency = latency
	} else if latency < l.noLoadLatency {
		l.noLoadLatency = latency*emaFactor + l.noLoadLatency*(1-emaFactor)
	}
}

func (l *AutoConcurrency) updateQPS(qps float64) {
	emaFactor := l.emaFactor / 10
	if l.maxQPS <= qps {
		l.maxQPS = qps
	} else {
		l.maxQPS = qps*emaFactor + l.maxQPS*(1-emaFactor)
	}
}

func (l *AutoConcurrency) updateMaxConcurrency(v uint64) {
	if l.maxConcurrency <= v {
		l.maxConcurrency = v
	} else {
		l.maxConcurrency = uint64(float64(v)*l.emaFactor + float64(l.maxConcurrency)*(1-l.emaFactor))
	}
}

func (l *AutoConcurrency) Inflight() uint64 {
	return l.inflight.Load()
}

func (l *AutoConcurrency) Remaining() uint64 {
	return l.maxConcurrency - l.inflight.Load()
}

func (l *AutoConcurrency) Acquire() (Updater, error) {
	now := time.Now()
	if l.inflight.Inc() > l.maxConcurrency && CPUUsage() >= 500 { // only when cpu load is above 50%
		l.inflight.Dec()
		return nil, ErrReachLimitation
	}
	u := &AutoConcurrencyUpdater{
		startTime: now,
		limiter:   l,
	}
	return u, nil
}

func (l *AutoConcurrency) Reset(startTimeUs int64) {
	l.startSampleTimeUs = startTimeUs
	l.sampleCount = 0
	l.totalSampleUs = 0
	l.totalReqCount.Store(0)
}

func (l *AutoConcurrency) NextResetTime(samplingTimeUs int64) int64 {
	return samplingTimeUs + (l.halfSampleIntervalMS+rand.Int63n(l.halfSampleIntervalMS))*1000
}

func (l *AutoConcurrency) Update(latency int64, samplingTimeUs int64) {
	l.Lock()
	defer l.Unlock()
	if l.resetLatencyUs != 0 { // wait to reset noLoadLatency and other data
		if l.resetLatencyUs > samplingTimeUs {
			return
		}
		l.noLoadLatency = -1
		l.resetLatencyUs = 0
		l.remeasureStartUs = l.NextResetTime(samplingTimeUs)
		l.Reset(samplingTimeUs)
	}

	if l.startSampleTimeUs == 0 {
		l.startSampleTimeUs = samplingTimeUs
	}

	l.sampleCount++
	l.totalSampleUs += latency

	logger.Debugf("[Auto Concurrency Limiter Test] samplingTimeUs: %v, startSampleTimeUs: %v", samplingTimeUs, l.startSampleTimeUs)

	if l.sampleCount < MinSampleCount {
		if samplingTimeUs-l.startSampleTimeUs >= SampleWindowSizeMs*1000 { // QPS is too small
			l.Reset(samplingTimeUs)
		}
		return
	}

	logger.Debugf("[Auto Concurrency Limiter Test] samplingTimeUs: %v, startSampleTimeUs: %v", samplingTimeUs, l.startSampleTimeUs)

	// sampling time is too short. If sample count is bigger than MaxSampleCount, just update.
	if samplingTimeUs-l.startSampleTimeUs < SampleWindowSizeMs*1000 && l.sampleCount < MaxSampleCount {
		return
	}

	qps := float64(l.totalReqCount.Load()) * 1000000.0 / float64(samplingTimeUs-l.startSampleTimeUs)
	l.updateQPS(qps)

	avgLatency := l.totalSampleUs / l.sampleCount
	l.updateNoLoadLatency(float64(avgLatency))

	nextMaxConcurrency := uint64(0)
	if l.remeasureStartUs <= samplingTimeUs { // should reset
		l.Reset(samplingTimeUs)
		l.resetLatencyUs = samplingTimeUs + avgLatency*2
		nextMaxConcurrency = uint64(math.Ceil(l.maxQPS * l.noLoadLatency * 0.9 / 1000000))
	} else {
		// use explore ratio to adjust MaxConcurrency
		if float64(avgLatency) <= l.noLoadLatency*(1.0+MinExploreRatio) ||
			qps >= l.maxQPS*(1.0+MinExploreRatio) {
			l.exploreRatio = math.Min(MaxExploreRatio, l.exploreRatio+0.02)
		} else {
			l.exploreRatio = math.Max(MinExploreRatio, l.exploreRatio-0.02)
		}
		nextMaxConcurrency = uint64(math.Ceil(l.noLoadLatency * l.maxQPS * (1 + l.exploreRatio) / 1000000))
	}
	l.maxConcurrency = nextMaxConcurrency

	// maxConcurrency should be no less than 1
	if l.maxConcurrency <= 0 {
		l.maxConcurrency = 1
	}

	logger.Debugf("[Auto Concurrency Limiter] Qps: %v, NoLoadLatency: %f, MaxConcurrency: %d, limiter: %+v",
		l.maxQPS, l.noLoadLatency, l.maxConcurrency, l)

	// Update completed, resample
	l.Reset(samplingTimeUs)

}

type AutoConcurrencyUpdater struct {
	startTime time.Time
	limiter   *AutoConcurrency
}

func (u *AutoConcurrencyUpdater) DoUpdate() error {
	defer func() {
		u.limiter.inflight.Dec()
	}()
	u.limiter.totalReqCount.Add(1)
	now := time.Now().UnixNano() / 1e3
	lastSamplingTimeUs := u.limiter.lastSamplingTimeUs.Load()
	if lastSamplingTimeUs == 0 || now-lastSamplingTimeUs >= 100 {
		sample := u.limiter.lastSamplingTimeUs.CAS(lastSamplingTimeUs, now)
		if sample {
			logger.Debugf("[Auto Concurrency Updater] sample, %v, %v", u.limiter.resetLatencyUs, u.limiter.remeasureStartUs)
			latency := now - u.startTime.UnixNano()/1e3
			u.limiter.Update(latency, now)
		}
	}

	return nil
}
