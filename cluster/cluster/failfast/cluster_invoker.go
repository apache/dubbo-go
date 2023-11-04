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

package failfast

import (
	"context"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/cluster/base"
	"dubbo.apache.org/dubbo-go/v3/cluster/directory"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

type failfastClusterInvoker struct {
	base.BaseClusterInvoker
}

func newFailfastClusterInvoker(directory directory.Directory) protocol.Invoker {
	return &failfastClusterInvoker{
		BaseClusterInvoker: base.NewBaseClusterInvoker(directory),
	}
}

// nolint
func (invoker *failfastClusterInvoker) Invoke(ctx context.Context, invocation protocol.Invocation) protocol.Result {
	invokers := invoker.Directory.List(invocation)
	err := invoker.CheckInvokers(invokers, invocation)
	if err != nil {
		return &protocol.RPCResult{Err: err}
	}

	loadbalance := base.GetLoadBalance(invokers[0], invocation.ActualMethodName())

	err = invoker.CheckWhetherDestroyed()
	if err != nil {
		return &protocol.RPCResult{Err: err}
	}

	ivk := invoker.DoSelect(loadbalance, invocation, invokers, nil)
	return ivk.Invoke(ctx, invocation)
}
