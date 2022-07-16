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

package meshrouter

import (
	"math/rand"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/router"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/remoting/xds"
)

const (
	name = "mesh-router"
)

// MeshRouter have
type MeshRouter struct {
	client *xds.WrappedClientImpl
}

// NewMeshRouter construct an NewConnCheckRouter via url
func NewMeshRouter() (router.PriorityRouter, error) {
	xdsWrappedClient := xds.GetXDSWrappedClient()
	if xdsWrappedClient == nil {
		logger.Debugf("[Mesh Router] xds wrapped client is not created.")
	}
	return &MeshRouter{
		client: xdsWrappedClient,
	}, nil
}

// Route gets a list of routed invoker
func (r *MeshRouter) Route(invokers []protocol.Invoker, url *common.URL, invocation protocol.Invocation) []protocol.Invoker {
	if r.client == nil {
		return invokers
	}
	hostAddr, err := r.client.GetHostAddrByServiceUniqueKey(common.GetSubscribeName(url))
	if err != nil {
		// todo deal with error
		return nil
	}
	rconf := r.client.GetRouterConfig(hostAddr)

	clusterInvokerMap := make(map[string][]protocol.Invoker)
	for _, v := range invokers {
		meshClusterID := v.GetURL().GetParam(constant.MeshClusterIDKey, "")
		if _, ok := clusterInvokerMap[meshClusterID]; !ok {
			clusterInvokerMap[meshClusterID] = make([]protocol.Invoker, 0)
		}
		clusterInvokerMap[meshClusterID] = append(clusterInvokerMap[meshClusterID], v)
	}
	route, err := r.client.MatchRoute(rconf, invocation)
	if err != nil {
		logger.Errorf("[Mesh Router] not found route,method=%s", invocation.MethodName())
		return nil
	}

	// Loop through routes in order and select first match.
	if route == nil || route.WeightedClusters == nil {
		logger.Errorf("[Mesh Router] route's WeightedClusters is empty, route: %+v", r)
		return invokers
	}
	invokersWeightPairs := make(invokerWeightPairs, 0)

	for clusterID, weight := range route.WeightedClusters {
		// cluster -> invokers
		targetInvokers := clusterInvokerMap[clusterID]
		invokersWeightPairs = append(invokersWeightPairs, invokerWeightPair{
			invokers: targetInvokers,
			weight:   weight.Weight,
		})
	}
	return invokersWeightPairs.GetInvokers()
}

// Process there is no process needs for uniform Router, as it upper struct RouterChain has done it
func (r *MeshRouter) Process(event *config_center.ConfigChangeEvent) {
}

// Name get name of ConnCheckerRouter
func (r *MeshRouter) Name() string {
	return name
}

// Priority get Router priority level
func (r *MeshRouter) Priority() int64 {
	return 0
}

// URL Return URL in router
func (r *MeshRouter) URL() *common.URL {
	return nil
}

// Notify the router the invoker list
func (r *MeshRouter) Notify(invokers []protocol.Invoker) {
}

type invokerWeightPair struct {
	invokers []protocol.Invoker
	weight   uint32
}

type invokerWeightPairs []invokerWeightPair

func (i *invokerWeightPairs) GetInvokers() []protocol.Invoker {
	if len(*i) == 0 {
		return nil
	}
	totalWeight := uint32(0)
	tempWeight := uint32(0)
	for _, v := range *i {
		totalWeight += v.weight
	}
	randFloat := rand.Float64()
	for _, v := range *i {
		tempWeight += v.weight
		tempPercent := float64(tempWeight) / float64(totalWeight)
		if tempPercent >= randFloat {
			return v.invokers
		}
	}
	return (*i)[0].invokers
}
