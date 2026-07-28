/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *       http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package monitor

import (
	"fmt"
	"sync"
	"time"
)

import (
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/process"
)

type SystemMetrics struct {
	CPUUsage    float64
	MemoryUsage uint64
	Timestamp   time.Time
}

type SystemMonitor struct {
	pid      int
	interval time.Duration
	metrics  []SystemMetrics
	mu       sync.Mutex
	stopChan chan struct{}
	wg       sync.WaitGroup
	stopOnce sync.Once
	proc     *process.Process
}

func NewSystemMonitor(pid int, interval time.Duration) *SystemMonitor {
	sm := &SystemMonitor{
		pid:      pid,
		interval: interval,
		metrics:  make([]SystemMetrics, 0),
		stopChan: make(chan struct{}),
	}

	proc, err := process.NewProcess(int32(pid))
	if err != nil {
		return sm
	}
	sm.proc = proc

	return sm
}

func (sm *SystemMonitor) Start() {
	if sm.proc == nil {
		return
	}
	sm.wg.Add(1)
	go sm.monitor()
}

func (sm *SystemMonitor) Stop() {
	sm.stopOnce.Do(func() {
		close(sm.stopChan)
		sm.wg.Wait()
	})
}

func (sm *SystemMonitor) monitor() {
	defer sm.wg.Done()

	ticker := time.NewTicker(sm.interval)
	defer ticker.Stop()

	for {
		select {
		case <-sm.stopChan:
			return
		case <-ticker.C:
			metrics, err := sm.collectMetrics()
			if err != nil {
				continue
			}
			sm.mu.Lock()
			sm.metrics = append(sm.metrics, metrics)
			sm.mu.Unlock()
		}
	}
}

func (sm *SystemMonitor) collectMetrics() (SystemMetrics, error) {
	metrics := SystemMetrics{
		Timestamp: time.Now(),
	}

	cpu, err := sm.getCPUUsage()
	if err != nil {
		return metrics, err
	}
	metrics.CPUUsage = cpu

	mem, err := sm.getMemoryUsage()
	if err != nil {
		return metrics, err
	}
	metrics.MemoryUsage = mem

	return metrics, nil
}

func (sm *SystemMonitor) getCPUUsage() (float64, error) {
	if sm.proc != nil {
		cpuPercent, err := sm.proc.Percent(0)
		if err == nil {
			return cpuPercent, nil
		}
	}

	cpuPercents, err := cpu.Percent(0, false)
	if err != nil {
		return 0, err
	}

	if len(cpuPercents) > 0 {
		return cpuPercents[0], nil
	}

	return 0, nil
}

func (sm *SystemMonitor) getMemoryUsage() (uint64, error) {
	if sm.proc != nil {
		memInfo, err := sm.proc.MemoryInfo()
		if err == nil {
			return memInfo.RSS, nil
		}
	}

	virtualMem, err := mem.VirtualMemory()
	if err != nil {
		return 0, err
	}

	return virtualMem.Used, nil
}

func (sm *SystemMonitor) GetMetrics() []SystemMetrics {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	result := make([]SystemMetrics, len(sm.metrics))
	copy(result, sm.metrics)
	return result
}

func (sm *SystemMonitor) GetSummary() (avgCPU float64, maxMemory uint64) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if len(sm.metrics) == 0 {
		return 0, 0
	}

	var totalCPU float64
	maxMemory = 0

	for _, m := range sm.metrics {
		totalCPU += m.CPUUsage
		if m.MemoryUsage > maxMemory {
			maxMemory = m.MemoryUsage
		}
	}

	return totalCPU / float64(len(sm.metrics)), maxMemory
}

func (sm *SystemMonitor) String() string {
	avgCPU, maxMemory := sm.GetSummary()
	return fmt.Sprintf(`
System Resource Usage:
  Avg CPU Usage:   %.2f%%
  Memory Peak:     %.2f MB
`, avgCPU, float64(maxMemory)/1024/1024)
}
