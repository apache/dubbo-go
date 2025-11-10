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

package p2c

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"time"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/loadbalance"
	"dubbo.apache.org/dubbo-go/v3/cluster/metrics"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
)

func init() {
	extension.SetLoadbalance(constant.LoadBalanceKeyP2C, func() loadbalance.LoadBalance {
		return newP2CLoadBalance(nil)
	})
}

var (
	once     sync.Once
	instance loadbalance.LoadBalance
)

// p2cLoadBalance is a load balancer implementation based on the Power of Two Choices algorithm.
type p2cLoadBalance struct {
	// randomPicker is injectable for testing; allows deterministic random selection.
	randomPicker randomPicker
}

// randomPicker is a function type that randomly selects two distinct indices from a range [0, n).
// This function type is designed ONLY FOR TEST purposes to inject predictable values.
type randomPicker func(n int) (i, j int)

// secureRandomInt returns a secure random integer in [0, max)
func secureRandomInt(max int) int {
	if max <= 0 {
		return 0
	}

	// Generate a random uint32 using crypto/rand
	var b [4]byte
	_, err := rand.Read(b[:])
	if err != nil {
		// Fallback to time-based seed if crypto/rand fails
		return int(time.Now().UnixNano() % int64(max))
	}

	// Convert to int and scale to range
	randomUint := binary.LittleEndian.Uint32(b[:])
	return int(randomUint % uint32(max))
}

// defaultRnd is the default implementation of randomPicker.
// It handles edge cases for n <= 2 and ensures two distinct random indices for n > 2.
func defaultRnd(n int) (i, j int) {
	if n <= 1 {
		return 0, 0
	}
	if n == 2 {
		return 0, 1
	}

	i = secureRandomInt(n)
	j = secureRandomInt(n)
	for i == j {
		j = secureRandomInt(n)
	}
	return i, j
}

// newP2CLoadBalance creates or returns the singleton P2C load balancer.
// Uses the provided randomPicker if non-nil; otherwise defaults to defaultRnd.
// randomPicker parameter is designed ONLY FOR TEST purposes.
// Thread-safe via sync.Once.
func newP2CLoadBalance(r randomPicker) loadbalance.LoadBalance {
	if r == nil {
		r = defaultRnd
	}
	if instance == nil {
		once.Do(func() {
			instance = &p2cLoadBalance{randomPicker: r}
		})
	}
	return instance
}

func (l *p2cLoadBalance) Select(invokers []base.Invoker, invocation base.Invocation) base.Invoker {
	if len(invokers) == 0 {
		return nil
	}
	if len(invokers) == 1 {
		return invokers[0]
	}
	// m is the Metrics, which saves the metrics of instance, invokers and methods
	// The local metrics is available only for the earlier version.
	m := metrics.LocalMetrics
	// picks two nodes randomly
	i, j := l.randomPicker(len(invokers))
	logger.Debugf("[P2C select] Two invokers were selected, invoker[%d]: %s, invoker[%d]: %s.",
		i, invokers[i], j, invokers[j])

	methodName := invocation.ActualMethodName()
	// remainingIIface, remainingJIface means remaining capacity of node i and node j.
	// If one of the metrics is empty, invoke the invocation to that node directly.
	remainingIIface, err := m.GetMethodMetrics(invokers[i].GetURL(), methodName, metrics.HillClimbing)
	if err != nil {
		if errors.Is(err, metrics.ErrMetricsNotFound) {
			logger.Debugf("[P2C select] The invoker[%d] was selected, because it hasn't been selected before.", i)
			return invokers[i]
		}
		logger.Warnf("get method metrics err: %v", err)
		return nil
	}

	// TODO(justxuewei): It should have a strategy to drop some metrics after a period of time.
	remainingJIface, err := m.GetMethodMetrics(invokers[j].GetURL(), methodName, metrics.HillClimbing)
	if err != nil {
		if errors.Is(err, metrics.ErrMetricsNotFound) {
			logger.Debugf("[P2C select] The invoker[%d] was selected, because it hasn't been selected before.", j)
			return invokers[j]
		}
		logger.Warnf("get method metrics err: %v", err)
		return nil
	}

	// Convert interface to int, if the type is unexpected, panic immediately
	remainingI, ok := remainingIIface.(uint64)
	if !ok {
		panic(fmt.Sprintf("[P2C select] the type of %s expects to be uint64, but gets %T",
			metrics.HillClimbing, remainingIIface))
	}

	remainingJ, ok := remainingJIface.(uint64)
	if !ok {
		panic(fmt.Sprintf("the type of %s expects to be uint64, but gets %T", metrics.HillClimbing, remainingJIface))
	}

	logger.Debugf("[P2C select] The invoker[%d] remaining is %d, and the invoker[%d] is %d.", i, remainingI, j, remainingJ)

	// For the remaining capacity, the bigger, the better.
	if remainingI > remainingJ {
		logger.Debugf("[P2C select] The invoker[%d] was selected.", i)
		return invokers[i]
	}

	logger.Debugf("[P2C select] The invoker[%d] was selected.", j)
	return invokers[j]
}
