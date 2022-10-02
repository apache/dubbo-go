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

	// For the remaining capacity, the bigger, the better.
	if weightI > weightJ {
		logger.Debugf("[P2C select] The invoker[%d] was selected.", i)
		return invokers[i]
	}

	logger.Debugf("[P2C select] The invoker[%d] was selected.", j)
	return invokers[j]
}

//Weight w_i = s_i + ε*t_i
func Weight(url1, url2 *common.URL, methodName string) (weight1, weight2 float64) {

	s1 := successRateWeight(url1, methodName)
	s2 := successRateWeight(url2, methodName)

	rtt1Iface, _ := metrics.EMAMetrics.GetMethodMetrics(url1, methodName, metrics.RTT)
	rtt2Iface, _ := metrics.EMAMetrics.GetMethodMetrics(url2, methodName, metrics.RTT)
	rtt1 := metrics.ToFloat64(rtt1Iface)
	rtt2 := metrics.ToFloat64(rtt2Iface)
	logger.Debugf("[P2C Weight Metrics] [invoker1] %s's s score: %f, rtt: %f; [invoker2] %s's s score: %f, rtt: %f.",
		url1.Ip, s1, rtt1, url2.Ip, s2, rtt2)
	avgRtt := (rtt1 + rtt2) / 2
	t1 := normalize((1 + avgRtt) / (1 + rtt1))
	t2 := normalize((1 + avgRtt) / (1 + rtt2))

	avgS := (s1 + s2) / 2
	avgT := (t1 + t2) / 2
	e := avgS / avgT

	weight1 = s1 + e*t1
	weight2 = s2 + e*t2
	logger.Debugf("[P2C Weight] [invoker1] %s's s score: %f, t score: %f, weight: %f; [invoker2] %s's s score: %f, t score: %f, weight: %f.",
		url1.Ip, s1, t1, weight1, url2.Ip, s2, t2, weight2)
	return weight1, weight2
}

func normalize(x float64) float64 {
	return 1.5 * x / (x + 1)
}

func successRateWeight(url *common.URL, methodName string) float64 {
	requestsIface, _ := metrics.SlidingWindowCounterMetrics.GetMethodMetrics(url, methodName, metrics.Requests)
	acceptsIface, _ := metrics.SlidingWindowCounterMetrics.GetMethodMetrics(url, methodName, metrics.Accepts)

	requests := metrics.ToFloat64(requestsIface)
	accepts := metrics.ToFloat64(acceptsIface)
	r := (1 + accepts) / (1 + requests)

	//r will greater than 1 because SlidingWindowCounter collects the most recent data and there is a delay in receiving a response.
	if r > 1 {
		r = 1
	}
	logger.Debugf("[P2C Weight] [Success Rate] %s requests: %f, accepts: %f, success rate: %f.",
		url.Ip, requests, accepts, r)
	return r
}
