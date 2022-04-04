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
	"bytes"
	"math/rand"
	"strings"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/router"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/remoting/xds"
	"dubbo.apache.org/dubbo-go/v3/xds/client/resource"
	"dubbo.apache.org/dubbo-go/v3/xds/utils/resolver"
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
	hostAddr, err := r.client.GetHostAddrByServiceUniqueKey(getSubscribeName(url))
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

	if len(rconf.VirtualHosts) != 0 {
		// try to route to sub virtual host
		for _, vh := range rconf.VirtualHosts {
			// 1. match domain
			//vh.Domains == ["*"]

			// 2. match http route
			for _, r := range vh.Routes {
				//route.
				ctx := invocation.GetAttachmentAsContext()
				matcher, err := resource.RouteToMatcher(r)
				if err != nil {
					logger.Errorf("[Mesh Router] router to matcher failed with error %s", err)
					return invokers
				}
				if matcher.Match(resolver.RPCInfo{
					Context: ctx,
					Method:  "/" + invocation.MethodName(),
				}) {
					// Loop through routes in order and select first match.
					if r == nil || r.WeightedClusters == nil {
						logger.Errorf("[Mesh Router] route's WeightedClusters is empty, route: %+v", r)
						return invokers
					}
					invokersWeightPairs := make(invokerWeightPairs, 0)

					for clusterID, weight := range r.WeightedClusters {
						// cluster -> invokers
						targetInvokers := clusterInvokerMap[clusterID]
						invokersWeightPairs = append(invokersWeightPairs, invokerWeightPair{
							invokers: targetInvokers,
							weight:   weight.Weight,
						})
					}
					return invokersWeightPairs.GetInvokers()
				}
			}
		}
	}
	return invokers
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

func getSubscribeName(url *common.URL) string {
	var buffer bytes.Buffer

	buffer.Write([]byte(common.DubboNodes[common.PROVIDER]))
	appendParam(&buffer, url, constant.InterfaceKey)
	appendParam(&buffer, url, constant.VersionKey)
	appendParam(&buffer, url, constant.GroupKey)
	return buffer.String()
}

func appendParam(target *bytes.Buffer, url *common.URL, key string) {
	value := url.GetParam(key, "")
	target.Write([]byte(constant.NacosServiceNameSeparator))
	if strings.TrimSpace(value) != "" {
		target.Write([]byte(value))
	}
}
