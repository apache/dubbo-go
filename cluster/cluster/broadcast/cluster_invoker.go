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

package broadcast

import (
	"context"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/cluster/base"
	"dubbo.apache.org/dubbo-go/v3/cluster/directory"
	protocolbase "dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

type broadcastClusterInvoker struct {
	base.BaseClusterInvoker
}

func newBroadcastClusterInvoker(directory directory.Directory) protocolbase.Invoker {
	return &broadcastClusterInvoker{
		BaseClusterInvoker: base.NewBaseClusterInvoker(directory),
	}
}

// nolint
func (invoker *broadcastClusterInvoker) Invoke(ctx context.Context, invocation protocolbase.Invocation) result.Result {
	invokers := invoker.Directory.List(invocation)
	err := invoker.CheckInvokers(invokers, invocation)
	if err != nil {
		return &result.RPCResult{Err: err}
	}
	err = invoker.CheckWhetherDestroyed()
	if err != nil {
		return &result.RPCResult{Err: err}
	}

	var res result.Result
	for _, ivk := range invokers {
		res = ivk.Invoke(ctx, invocation)
		if res.Error() != nil {
			logger.Warnf("broadcast invoker invoke err: %v when use invoker: %v\n", res.Error(), ivk)
			err = res.Error()
		}
	}
	if err != nil {
		return &result.RPCResult{Err: err}
	}
	return res
}
