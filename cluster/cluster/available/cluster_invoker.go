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

package available

import (
	"context"
	"fmt"
)

import (
	"github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/cluster/base"
	"dubbo.apache.org/dubbo-go/v3/cluster/directory"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

type availableClusterInvoker struct {
	base.BaseClusterInvoker
}

// NewClusterInvoker returns a availableCluster invoker instance
func NewClusterInvoker(directory directory.Directory) protocol.Invoker {
	return &availableClusterInvoker{
		BaseClusterInvoker: base.NewBaseClusterInvoker(directory),
	}
}

func (invoker *availableClusterInvoker) Invoke(ctx context.Context, invocation protocol.Invocation) protocol.Result {
	invokers := invoker.Directory.List(invocation)
	err := invoker.CheckInvokers(invokers, invocation)
	if err != nil {
		return &protocol.RPCResult{Err: err}
	}

	err = invoker.CheckWhetherDestroyed()
	if err != nil {
		return &protocol.RPCResult{Err: err}
	}

	for _, ivk := range invokers {
		if ivk.IsAvailable() {
			return ivk.Invoke(ctx, invocation)
		}
	}
	return &protocol.RPCResult{Err: errors.New(fmt.Sprintf("no provider available in %v", invokers))}
}
