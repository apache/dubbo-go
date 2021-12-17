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
	"dubbo.apache.org/dubbo-go/v3/common/logger"
)

//var (
//	decay = 0.95
//	// cpu statistics interval (ms)
//	statInterval time.Duration = 500
//	cpu *atomic.Uint64
//)
//
//func init() {
//	cpu = new(atomic.Uint64)
//	// get cpu usage statistics regularly
//	go cpuStat()
//}
//
//func cpuStat() {
//	t := time.NewTicker(time.Microsecond * statInterval)
//
//	// prevent cpuStat method from crashing unexpectedly
//	defer func() {
//		t.Stop()
//		if err := recover(); err != nil {
//			logger.Warnf("[HillClimbing] cpuStat went down, err: %v, attempting to restart...", err)
//			go cpuStat()
//		}
//	}()
//
//	for range t.C {
//		stat :=
//	}
//}
func VerboseDebugf(msg string, args ...interface{}) {
	if !Verbose {
		return
	}
	logger.Debugf(msg, args...)
}

func minUint64(lhs, rhs uint64) uint64 {
	if lhs < rhs {
		return lhs
	}
	return rhs
}
