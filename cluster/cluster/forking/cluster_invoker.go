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
	"fmt"
	"time"
)

import (
	"github.com/Workiva/go-datastructures/queue"

	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/cluster/base"
	"dubbo.apache.org/dubbo-go/v3/cluster/directory"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	protocolbase "dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

type forkingClusterInvoker struct {
	base.BaseClusterInvoker
}

func newForkingClusterInvoker(directory directory.Directory) protocolbase.Invoker {
	return &forkingClusterInvoker{
		BaseClusterInvoker: base.NewBaseClusterInvoker(directory),
	}
}

func (invoker *forkingClusterInvoker) Invoke(ctx context.Context, invocation protocolbase.Invocation) result.Result {
	if err := invoker.CheckWhetherDestroyed(); err != nil {
		return &result.RPCResult{Err: err}
	}

	invokers := invoker.Directory.List(invocation)
	if err := invoker.CheckInvokers(invokers, invocation); err != nil {
		return &result.RPCResult{Err: err}
	}

	var selected []protocolbase.Invoker
	forks := invoker.GetURL().GetParamByIntValue(constant.ForksKey, constant.DefaultForks)
	timeouts := invoker.GetURL().GetParamInt(constant.TimeoutKey, constant.DefaultTimeout)
	if forks < 0 || forks > len(invokers) {
		selected = invokers
	} else {
		loadBalance := base.GetLoadBalance(invokers[0], invocation.ActualMethodName())
		for i := 0; i < forks; i++ {
			if ivk := invoker.DoSelect(loadBalance, invocation, invokers, selected); ivk != nil {
				selected = append(selected, ivk)
			}
		}
	}

	resultQ := queue.New(1)
	for _, ivk := range selected {
		go func(k protocolbase.Invoker) {
			result := k.Invoke(ctx, invocation)
			if err := resultQ.Put(result); err != nil {
				logger.Errorf("resultQ put failed with exception: %v.\n", err)
			}
		}(ivk)
	}

	rsps, err := resultQ.Poll(1, time.Millisecond*time.Duration(timeouts))
	if err != nil {
		return &result.RPCResult{
			Err: fmt.Errorf("failed to forking invoke provider %v, "+
				"but no luck to perform the invocation. Last error is: %v", selected, err),
		}
	}
	if len(rsps) == 0 {
		return &result.RPCResult{Err: fmt.Errorf("failed to forking invoke provider %v, but no resp", selected)}
	}

	res, ok := rsps[0].(result.Result)
	if !ok {
		return &result.RPCResult{Err: fmt.Errorf("failed to forking invoke provider %v, but not legal resp", selected)}
	}
	return res
}
