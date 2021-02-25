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
)

import (
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
	invoke := &zoneAwareClusterInvoker{
		baseClusterInvoker: newBaseClusterInvoker(directory),
	}
	// add local to interceptor
	invoke.interceptor = invoke
	return invoke
}

// nolint
func (invoker *zoneAwareClusterInvoker) DoInvoke(ctx context.Context, invocation protocol.Invocation) protocol.Result {
	invokers := invoker.directory.List(invocation)

	err := invoker.checkInvokers(invokers, invocation)
	if err != nil {
		return &protocol.RPCResult{Err: err}
	}

	// First, pick the invoker (XXXClusterInvoker) that comes from the local registry, distinguish by a 'preferred' key.
	for _, invoker := range invokers {
		key := constant.REGISTRY_KEY + "." + constant.PREFERRED_KEY
		if invoker.IsAvailable() && matchParam("true", key, "false", invoker) {
			return invoker.Invoke(ctx, invocation)
		}
	}

	// providers in the registry with the same zone
	key := constant.REGISTRY_KEY + "." + constant.ZONE_KEY
	zone := invocation.AttachmentsByKey(key, "")
	if "" != zone {
		for _, invoker := range invokers {
			if invoker.IsAvailable() && matchParam(zone, key, "", invoker) {
				return invoker.Invoke(ctx, invocation)
			}
		}

		force := invocation.AttachmentsByKey(constant.REGISTRY_KEY+"."+constant.ZONE_FORCE_KEY, "")
		if "true" == force {
			return &protocol.RPCResult{
				Err: fmt.Errorf("no registry instance in zone or "+
					"no available providers in the registry, zone: %v, "+
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

func (invoker *zoneAwareClusterInvoker) BeforeInvoker(ctx context.Context, invocation protocol.Invocation) {
	key := constant.REGISTRY_KEY + "." + constant.ZONE_FORCE_KEY
	force := ctx.Value(key)

	if force != nil {
		switch value := force.(type) {
		case bool:
			if value {
				invocation.SetAttachments(key, "true")
			}
		case string:
			if "true" == value {
				invocation.SetAttachments(key, "true")
			}
		default:
			// ignore
		}
	}
}

func (invoker *zoneAwareClusterInvoker) AfterInvoker(ctx context.Context, invocation protocol.Invocation) {

}

func matchParam(target, key, def string, invoker protocol.Invoker) bool {
	return target == invoker.GetUrl().GetParam(key, def)
}
