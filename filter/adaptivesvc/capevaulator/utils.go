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

import "go.uber.org/atomic"

func after(crt, next *atomic.Uint64) bool {
	return crt.Load() > next.Load()
}

// setValueIfLess sets newValue to v if newValue is less than v
func setValueIfLess(v *atomic.Uint64, newValue uint64) {
	vuint64 := v.Load()
	for vuint64 > newValue {
		if !v.CAS(vuint64, newValue) {
			vuint64 = v.Load()
		}
	}
}

func slowStart(est *atomic.Uint64, estValue, threshValue uint64) bool {
	newEst := minUint64(estValue*2, threshValue)
	return est.CAS(estValue, newEst)
}

func minUint64(lhs, rhs uint64) uint64 {
	if lhs < rhs {
		return lhs
	}
	return rhs
}
