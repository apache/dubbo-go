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

package cluster_impl

import (
	"context"
	"fmt"
	"github.com/apache/dubbo-go/cluster"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/protocol"
)

// When there're more than one registry for subscription.
//
// This extension provides a strategy to decide how to distribute traffics among them:
// 1. registry marked as 'preferred=true' has the highest priority.
// 2. check the zone the current request belongs, pick the registry that has the same zone first.
// 3. Evenly balance traffic between all registries based on each registry's weight.
// 4. Pick anyone that's available.
type zoneAwareClusterInvoker struct {
	baseClusterInvoker
}

func newZoneAwareClusterInvoker(directory cluster.Directory) protocol.Invoker {
	return &zoneAwareClusterInvoker{
		baseClusterInvoker: newBaseClusterInvoker(directory),
	}
}

func (invoker *zoneAwareClusterInvoker) Invoke(ctx context.Context, invocation protocol.Invocation) protocol.Result {
	invokers := invoker.directory.List(invocation)

	err := invoker.checkInvokers(invokers, invocation)
	if err != nil {
		return &protocol.RPCResult{Err: err}
	}

	// First, pick the invoker (XXXClusterInvoker) that comes from the local registry, distinguish by a 'preferred' key.
	for _, invoker := range invokers {
		if invoker.IsAvailable() && "true" == invoker.GetUrl().GetParam(constant.REGISTRY_KEY+"."+constant.PREFERRED_KEY, "false") {
			return invoker.Invoke(ctx, invocation)
		}
	}

	// providers in the registry with the same zone
	zone := invocation.AttachmentsByKey(constant.REGISTRY_ZONE, "")
	if "" != zone {
		for _, invoker := range invokers {
			if invoker.IsAvailable() && zone == invoker.GetUrl().GetParam(constant.REGISTRY_KEY+"."+constant.ZONE_KEY, "") {
				return invoker.Invoke(ctx, invocation)
			}
		}

		force := invocation.AttachmentsByKey(constant.REGISTRY_ZONE_FORCE, "")
		if "true" == force {
			return &protocol.RPCResult{
				Err: fmt.Errorf("no registry instance in zone or no available providers in the registry, zone: %v, "+
					" registries: %v", zone, invoker.GetUrl()),
			}
		}
	}

	// load balance among all registries, with registry weight count in.
	loadBalance := getLoadBalance(invokers[0], invocation)
	ivk := invoker.doSelect(loadBalance, invocation, invokers, nil)
	if ivk != nil && ivk.IsAvailable() {
		return ivk.Invoke(ctx, invocation)
	}

	// If none of the invokers has a preferred signal or is picked by the loadBalancer, pick the first one available.
	for _, invoker := range invokers {
		if invoker.IsAvailable() {
			return invoker.Invoke(ctx, invocation)
		}
	}

	return &protocol.RPCResult{
		Err: fmt.Errorf("no provider available in %v", invokers),
	}
}
