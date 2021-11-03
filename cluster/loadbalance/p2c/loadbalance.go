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
	"dubbo.apache.org/dubbo-go/v3/cluster/loadbalance"
	"dubbo.apache.org/dubbo-go/v3/cluster/metrics"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"errors"
	"fmt"
	"math/rand"
	"time"
)

func init() {
	extension.SetLoadbalance(constant.LoadBalanceKeyP2C, newLoadBalance)
}

type loadBalance struct {
}

func newLoadBalance() loadbalance.LoadBalance {
	return &loadBalance{}
}

func (l *loadBalance) Select(invokers []protocol.Invoker, invocation protocol.Invocation) protocol.Invoker {
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
		rand.Seed(time.Now().Unix())
		i = rand.Intn(len(invokers))
		j = i
		for i == j {
			j = rand.Intn(len(invokers))
		}
	}
	// TODO(justxuewei): please consider how to get the real method name from $invoke,
	// 	see also [#1511](https://github.com/apache/dubbo-go/issues/1511)
	methodName := invocation.MethodName()
	// viInterface, vjInterface means vegas latency of node i and node j
	// If one of the metrics is empty, invoke the invocation to that node directly.
	viInterface, err := m.GetMethodMetrics(invokers[i].GetURL(), methodName, metrics.Vegas)
	if err != nil {
		if errors.Is(err, metrics.ErrMetricsNotFound) {
			return invokers[i]
		}
		logger.Warnf("get method metrics err: %v", err)
		return nil
	}

	vjInterface, err := m.GetMethodMetrics(invokers[j].GetURL(), methodName, metrics.Vegas)
	if err != nil {
		if errors.Is(err, metrics.ErrMetricsNotFound) {
			return invokers[j]
		}
		logger.Warnf("get method metrics err: %v", err)
		return nil
	}

	// Convert interface to int, if the type is unexpected, panic immediately
	vi, ok := viInterface.(int)
	if !ok {
		panic(fmt.Sprintf("the type of %s expects to be int, but gets %T", metrics.Vegas, viInterface))
	}

	vj, ok := vjInterface.(int)
	if !ok {
		panic(fmt.Sprintf("the type of %s expects to be int, but gets %T", metrics.Vegas, viInterface))
	}

	// For the latency time, the smaller, the better.
	if vi < vj {
		return invokers[i]
	}

	return invokers[j]
}
