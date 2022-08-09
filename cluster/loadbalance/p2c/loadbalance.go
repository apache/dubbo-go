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
	"dubbo.apache.org/dubbo-go/v3/common"
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
	rand.Seed(randSeed())
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
	// picks two nodes randomly
	var i, j int
	if len(invokers) == 2 {
		i, j = 0, 1
	} else {
		i = rand.Intn(len(invokers))
		j = i
		for i == j {
			j = rand.Intn(len(invokers))
		}
	}
	logger.Debugf("[P2C select] Two invokers were selected, invoker[%d]: %s, invoker[%d]: %s.",
		i, invokers[i], j, invokers[j])

	methodName := invocation.ActualMethodName()

	weightI, weightJ := Weight(invokers[i].GetURL(), invokers[j].GetURL(), methodName)

	logger.Debugf("[P2C I] %s: %f", invokers[i].GetURL().Ip, weightI)
	logger.Debugf("[P2C J] %s: %f", invokers[j].GetURL().Ip, weightJ)

	// For the remaining capacity, the bigger, the better.
	if weightI > weightJ {
		logger.Debugf("[P2C select] The invoker[%d] was selected.", i)
		return invokers[i]
	}

	logger.Debugf("[P2C select] The invoker[%d] was selected.", j)
	return invokers[j]
}

//Weight w_i = s_i + ε*t_i
func Weight(urlI, urlJ *common.URL, methodName string) (weightI, weightJ float64) {

	sI := successRateWeight(urlI, methodName)
	sJ := successRateWeight(urlJ, methodName)

	rttIFace, _ := metrics.EMAMetrics.GetMethodMetrics(urlI, methodName, metrics.RTT)
	rttJFace, _ := metrics.EMAMetrics.GetMethodMetrics(urlJ, methodName, metrics.RTT)
	rttI := metrics.ToFloat64(rttIFace)
	rttJ := metrics.ToFloat64(rttJFace)

	avgRtt := (rttI + rttJ) / 2
	tI := norm((1 + avgRtt) / (1 + rttI))
	tJ := norm((1 + avgRtt) / (1 + rttJ))

	avgS := (sI + sJ) / 2
	avgL := (tI + tJ) / 2
	e := avgS / avgL

	weightI = sI + e*tI
	weightJ = sJ + e*tJ

	//weightI = sI + norm((1+avgRtt)/(1+rttI))
	//weightJ = sJ + norm((1+avgRtt)/(1+rttJ))

	return weightI, weightJ
}

func norm(x float64) float64 {
	return x / (x + 1)
}

func successRateWeight(url *common.URL, methodName string) float64 {
	requestsIFace, _ := metrics.SlidingWindowCounterMetrics.GetMethodMetrics(url, methodName, metrics.Requests)
	acceptsIFace, _ := metrics.SlidingWindowCounterMetrics.GetMethodMetrics(url, methodName, metrics.Accepts)

	requests := metrics.ToFloat64(requestsIFace)
	accepts := metrics.ToFloat64(acceptsIFace)
	r := (1 + accepts) / (1 + requests)

	if r > 1 {
		r = 1
	}
	return r
}
