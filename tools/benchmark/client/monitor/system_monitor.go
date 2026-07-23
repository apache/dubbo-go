/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
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
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

type SystemMetrics struct {
	CPUUsage    float64
	MemoryUsage uint64
	Timestamp   time.Time
}

type SystemMonitor struct {
	pid         int
	interval    time.Duration
	metrics     []SystemMetrics
	mu          sync.Mutex
	stopChan    chan struct{}
	wg          sync.WaitGroup
}

func NewSystemMonitor(pid int, interval time.Duration) *SystemMonitor {
	return &SystemMonitor{
		pid:      pid,
		interval: interval,
		metrics:  make([]SystemMetrics, 0),
		stopChan: make(chan struct{}),
	}
}

func (sm *SystemMonitor) Start() {
	sm.wg.Add(1)
	go sm.monitor()
}

func (sm *SystemMonitor) Stop() {
	close(sm.stopChan)
	sm.wg.Wait()
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
	cmd := exec.Command("ps", "-p", strconv.Itoa(sm.pid), "-o", "%cpu=")
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	cpuStr := strings.TrimSpace(string(output))
	cpu, err := strconv.ParseFloat(cpuStr, 64)
	if err != nil {
		return 0, err
	}

	return cpu, nil
}

func (sm *SystemMonitor) getMemoryUsage() (uint64, error) {
	cmd := exec.Command("ps", "-p", strconv.Itoa(sm.pid), "-o", "rss=")
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	memStr := strings.TrimSpace(string(output))
	mem, err := strconv.ParseUint(memStr, 10, 64)
	if err != nil {
		return 0, err
	}

	return mem * 1024, nil
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
系统资源使用统计:
  平均CPU使用率:  %.2f%%
  内存峰值:      %.2f MB
`, avgCPU, float64(maxMemory)/1024/1024)
}
