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

package forking

import (
	"context"
	"dubbo.apache.org/dubbo-go/v3/cluster/cluster/base"
	"dubbo.apache.org/dubbo-go/v3/cluster/directory"
	"fmt"
	"time"
)

import (
	"github.com/Workiva/go-datastructures/queue"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

type clusterInvoker struct {
	base.ClusterInvoker
}

func newClusterInvoker(directory directory.Directory) protocol.Invoker {
	return &clusterInvoker{
		ClusterInvoker: base.NewClusterInvoker(directory),
	}
}

func (invoker *clusterInvoker) Invoke(ctx context.Context, invocation protocol.Invocation) protocol.Result {
	if err := invoker.CheckWhetherDestroyed(); err != nil {
		return &protocol.RPCResult{Err: err}
	}

	invokers := invoker.Directory.List(invocation)
	if err := invoker.CheckInvokers(invokers, invocation); err != nil {
		return &protocol.RPCResult{Err: err}
	}

	var selected []protocol.Invoker
	forks := invoker.GetURL().GetParamByIntValue(constant.FORKS_KEY, constant.DEFAULT_FORKS)
	timeouts := invoker.GetURL().GetParamInt(constant.TIMEOUT_KEY, constant.DEFAULT_TIMEOUT)
	if forks < 0 || forks > len(invokers) {
		selected = invokers
	} else {
		loadBalance := base.GetLoadBalance(invokers[0], invocation)
		for i := 0; i < forks; i++ {
			if ivk := invoker.DoSelect(loadBalance, invocation, invokers, selected); ivk != nil {
				selected = append(selected, ivk)
			}
		}
	}

	resultQ := queue.New(1)
	for _, ivk := range selected {
		go func(k protocol.Invoker) {
			result := k.Invoke(ctx, invocation)
			if err := resultQ.Put(result); err != nil {
				logger.Errorf("resultQ put failed with exception: %v.\n", err)
			}
		}(ivk)
	}

	rsps, err := resultQ.Poll(1, time.Millisecond*time.Duration(timeouts))
	if err != nil {
		return &protocol.RPCResult{
			Err: fmt.Errorf("failed to forking invoke provider %v, "+
				"but no luck to perform the invocation. Last error is: %v", selected, err),
		}
	}
	if len(rsps) == 0 {
		return &protocol.RPCResult{Err: fmt.Errorf("failed to forking invoke provider %v, but no resp", selected)}
	}

	result, ok := rsps[0].(protocol.Result)
	if !ok {
		return &protocol.RPCResult{Err: fmt.Errorf("failed to forking invoke provider %v, but not legal resp", selected)}
	}
	return result
}
