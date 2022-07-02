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

package ringhash

import (
	"strconv"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/loadbalance"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/remoting/xds"
)

func init() {
	extension.SetLoadbalance(constant.LoadXDSRingHash, newRingHashLoadBalance)
}

type invokerWrapper struct {
	invoker protocol.Invoker
	weight  int
}

type ringhashLoadBalance struct {
	client xds.XDSWrapperClient
}

// newRingHashLoadBalance xds ring hash
//
// The same parameters of the request is always sent to the same provider.
func newRingHashLoadBalance() loadbalance.LoadBalance {
	return &ringhashLoadBalance{client: xds.GetXDSWrappedClient()}
}

// Select gets invoker based on load balancing strategy
func (lb *ringhashLoadBalance) Select(invokers []protocol.Invoker, invocation protocol.Invocation) protocol.Invoker {
	url := invocation.Invoker().GetURL()
	serviceUniqueKey := common.GetSubscribeName(url)
	hostAddr, err := lb.client.GetHostAddrByServiceUniqueKey(serviceUniqueKey)
	if err != nil {
		logger.Errorf("[xds ringhash] GetHostAddrByServiceUniqueKey failed,error=%v", err)
		return nil
	}
	clusterUpdate := lb.client.GetClusterUpdateIgnoreVersion(hostAddr)
	policy := clusterUpdate.LBPolicy
	if policy.MaximumRingSize < policy.MinimumRingSize {
		logger.Errorf("[xds ringhash] ringsize parameter is invalid. MinimumRingSize=%d,MaximumRingSize=%d", policy.MaximumRingSize, policy.MaximumRingSize)
		return nil
	}
	invokerWrappers := make([]invokerWrapper, 0, len(invokers))
	for _, v := range invokers {
		weight, _ := strconv.Atoi(v.GetURL().GetParam(constant.EndPointWeight, "1"))
		invokerWrappers = append(invokerWrappers, invokerWrapper{invoker: v, weight: weight})
	}
	ring, err := lb.generateRing(invokerWrappers, policy.MinimumRingSize, policy.MaximumRingSize)
	if err != nil {
		logger.Errorf("[xds ringhash] ringsize parameter is invalid. MinimumRingSize=%d,MaximumRingSize=%d", policy.MaximumRingSize, policy.MaximumRingSize)
		return nil
	}

	routerConfig := lb.client.GetRouterConfig(hostAddr)
	router, err := lb.client.MatchRoute(routerConfig, invocation)
	if err != nil {
		logger.Errorf("[xds ringhash] not found route,method=%s", invocation.MethodName())
		return nil
	}
	return lb.pick(lb.generateHash(invocation, router.HashPolicies), ring).invoker
}
