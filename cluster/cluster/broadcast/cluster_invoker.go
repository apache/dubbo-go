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
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

type broadcastClusterInvoker struct {
	base.BaseClusterInvoker
}

func newBroadcastClusterInvoker(directory directory.Directory) protocol.Invoker {
	return &broadcastClusterInvoker{
		BaseClusterInvoker: base.NewBaseClusterInvoker(directory),
	}
}

// nolint
func (invoker *broadcastClusterInvoker) Invoke(ctx context.Context, invocation protocol.Invocation) protocol.Result {
	invokers := invoker.Directory.List(invocation)
	err := invoker.CheckInvokers(invokers, invocation)
	if err != nil {
		return &protocol.RPCResult{Err: err}
	}
	err = invoker.CheckWhetherDestroyed()
	if err != nil {
		return &protocol.RPCResult{Err: err}
	}

	var result protocol.Result
	for _, ivk := range invokers {
		result = ivk.Invoke(ctx, invocation)
		if result.Error() != nil {
			logger.Warnf("broadcast invoker invoke err: %v when use invoker: %v\n", result.Error(), ivk)
			err = result.Error()
		}
	}
	if err != nil {
		return &protocol.RPCResult{Err: err}
	}
	return result
}
