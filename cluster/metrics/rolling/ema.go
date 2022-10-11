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

package rolling

import (
	"sync"
)

// EMA is a struct implemented Exponential Moving Average.
// val = old * (1 - alpha) + new * alpha
type EMA struct {
	mu    sync.RWMutex
	alpha float64
	val   float64
}

type EMAOpts struct {
	Alpha float64
}

// NewEMA creates a new EMA based on the given EMAOpts.
func NewEMA(opts EMAOpts) *EMA {
	return &EMA{
		alpha: opts.Alpha,
		val:   0,
	}
}

func (e *EMA) Append(v float64) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.val = e.val*(1-e.alpha) + v*e.alpha
}

func (e *EMA) Value() float64 {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.val
}
