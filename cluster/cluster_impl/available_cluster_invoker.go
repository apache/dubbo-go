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
	"fmt"
)

import (
	"github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/cluster"
	"github.com/apache/dubbo-go/protocol"
)

type availableClusterInvoker struct {
	baseClusterInvoker
}

func NewAvailableClusterInvoker(directory cluster.Directory) protocol.Invoker {
	return &availableClusterInvoker{
		baseClusterInvoker: newBaseClusterInvoker(directory),
	}
}

func (invoker *availableClusterInvoker) Invoke(invocation protocol.Invocation) protocol.Result {
	invokers := invoker.directory.List(invocation)
	err := invoker.checkInvokers(invokers, invocation)
	if err != nil {
		return &protocol.RPCResult{Err: err}
	}

	err = invoker.checkWhetherDestroyed()
	if err != nil {
		return &protocol.RPCResult{Err: err}
	}

	for _, ivk := range invokers {
		if ivk.IsAvailable() {
			return ivk.Invoke(invocation)
		}
	}
	return &protocol.RPCResult{Err: errors.New(fmt.Sprintf("no provider available in %v", invokers))}
}
