/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cluster_impl

import (
	"context"
)
import (
	"github.com/apache/dubbo-go/cluster"
	"github.com/apache/dubbo-go/protocol"
)

type failfastClusterInvoker struct {
	baseClusterInvoker
}

func newFailFastClusterInvoker(directory cluster.Directory) protocol.Invoker {
	return &failfastClusterInvoker{
		baseClusterInvoker: newBaseClusterInvoker(directory),
	}
}

func (invoker *failfastClusterInvoker) Invoke(ctx context.Context, invocation protocol.Invocation) protocol.Result {
	invokers := invoker.directory.List(invocation)
	err := invoker.checkInvokers(invokers, invocation)
	if err != nil {
		return &protocol.RPCResult{Err: err}
	}

	loadbalance := getLoadBalance(invokers[0], invocation)

	err = invoker.checkWhetherDestroyed()
	if err != nil {
		return &protocol.RPCResult{Err: err}
	}

	ivk := invoker.doSelect(loadbalance, invocation, invokers, nil)
	return ivk.Invoke(ctx, invocation)
}
