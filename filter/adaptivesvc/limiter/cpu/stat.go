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

package cpu

import (
	"fmt"
	"time"
)

import (
	"go.uber.org/atomic"
)

const (
	interval time.Duration = time.Millisecond * 500
)

var (
	stats CPU
	usage = atomic.NewUint64(0)
)

// CPU is cpu stat usage.
type CPU interface {
	Usage() (u uint64, e error)
	Info() Info
}

func init() {
	var (
		err error
	)
	stats, err = newCgroupCPU()
	if err != nil {
		panic(fmt.Sprintf("cgroup cpu init failed! err:=%v", err))
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			<-ticker.C
			u, err := stats.Usage()
			if err == nil && u != 0 {
				usage.Store(u)
			}
		}
	}()
}

// Info cpu info.
type Info struct {
	Frequency uint64
	Quota     float64
}

// CpuUsage read cpu stat.
func CpuUsage() uint64 {
	return usage.Load()
}

// GetInfo get cpu info.
func GetInfo() Info {
	return stats.Info()
}
