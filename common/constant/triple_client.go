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

package constant

import (
	"time"
)

// triple client
const (
	// TriClientPoolMaxSize is the maximum number of tri.Client instances per method in the pool.
	TriClientPoolMaxSize = 32
	// DefaultClientGetTimeout is the timeout for getting a tri.Client from the TriClientPool.
	DefaultClientGetTimeout = 50 * time.Millisecond
	// DefaultClientWarmingUp is the default warming up client count of TriClientPool.
	DefaultClientWarmingUp = 16
)

// triple client pool
const (
	AutoScalerPeriod    = 700 * time.Millisecond // minimum period that autoScaler expand or shrink
	MaxExpandPerCycle   = 16                     // maximum number of clients to expand per cycle
	HighIdleStreakLimit = 10                     // consecutive high idle count to trigger shrinking
	LowIdleThreshold    = 0.2                    // treat pool as busy if idle clients < 20% of total
	IdleShrinkThreshold = 0.6                    // shrink counter increases when idle ratio > 60%
	ShrinkRatio         = 0.125                  // shrink by 12.5% of current size when threshold is reached
	MaxExpandShift      = 30                     // maximum left-shift count to prevent integer overflow for expand
)

// protocol/triple/client.go
const (
	TriCliPoolPerClientTrans = "tri.per_client_transport"
)
