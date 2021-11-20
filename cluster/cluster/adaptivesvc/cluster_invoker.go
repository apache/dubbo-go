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

package adaptivesvc

import (
	"context"
	"dubbo.apache.org/dubbo-go/v3/cluster/cluster/base"
	"dubbo.apache.org/dubbo-go/v3/cluster/directory"
	"dubbo.apache.org/dubbo-go/v3/cluster/metrics"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	perrors "github.com/pkg/errors"
)

type adaptiveServiceClusterInvoker struct {
	base.ClusterInvoker
}

func newAdaptiveServiceClusterInvoker(directory directory.Directory) protocol.Invoker {
	return &adaptiveServiceClusterInvoker{
		ClusterInvoker: base.NewClusterInvoker(directory),
	}
}

func (ivk *adaptiveServiceClusterInvoker) Invoke(ctx context.Context, invocation protocol.Invocation) protocol.Result {
	invokers := ivk.Directory.List(invocation)
	if err := ivk.CheckInvokers(invokers, invocation); err != nil {
		return &protocol.RPCResult{Err: err}
	}

	// get loadBalance
	lbKey := invokers[0].GetURL().GetParam(constant.LoadbalanceKey, constant.LoadBalanceKeyP2C)
	if lbKey != constant.LoadBalanceKeyP2C {
		return &protocol.RPCResult{
			Err: perrors.Errorf("adaptive service not supports %s load balance", lbKey),
		}
	}
	lb := extension.GetLoadbalance(lbKey)

	// select a node by the loadBalance
	invoker := lb.Select(invokers, invocation)

	// invoke
	result := invoker.Invoke(ctx, invocation)

	// update metrics
	remaining := invocation.Attachments()[constant.AdaptiveServiceRemainingKey]
	err := metrics.LocalMetrics.SetMethodMetrics(invoker.GetURL(),
		invocation.MethodName(), metrics.HillClimbing, remaining)
	if err != nil {
		return &protocol.RPCResult{Err: err}
	}

	return result
}
