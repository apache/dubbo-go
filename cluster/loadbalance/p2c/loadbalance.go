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
	"errors"
	"fmt"
	"math/rand"
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
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

var (
	randSeed = func() int64 {
		return time.Now().Unix()
	}
)

func init() {
	extension.SetLoadbalance(constant.LoadBalanceKeyP2C, newP2CLoadBalance)
}

var (
	once     sync.Once
	instance loadbalance.LoadBalance
)

type p2cLoadBalance struct{}

func newP2CLoadBalance() loadbalance.LoadBalance {
	if instance == nil {
		once.Do(func() {
			instance = &p2cLoadBalance{}
		})
	}
	return instance
}

func (l *p2cLoadBalance) Select(invokers []protocol.Invoker, invocation protocol.Invocation) protocol.Invoker {
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
	var i, j int
	if len(invokers) == 2 {
		i, j = 0, 1
	} else {
		rand.Seed(randSeed())
		i = rand.Intn(len(invokers))
		j = i
		for i == j {
			j = rand.Intn(len(invokers))
		}
	}
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
