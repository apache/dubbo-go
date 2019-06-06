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
	"github.com/apache/dubbo-go/cluster"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/protocol"
)

type registryAwareClusterInvoker struct {
	baseClusterInvoker
}

func newRegistryAwareClusterInvoker(directory cluster.Directory) protocol.Invoker {
	return &registryAwareClusterInvoker{
		baseClusterInvoker: newBaseClusterInvoker(directory),
	}
}

func (invoker *registryAwareClusterInvoker) Invoke(invocation protocol.Invocation) protocol.Result {
	invokers := invoker.directory.List(invocation)
	//First, pick the invoker (XXXClusterInvoker) that comes from the local registry, distinguish by a 'default' key.
	for _, invoker := range invokers {
		if invoker.IsAvailable() && invoker.GetUrl().GetParam(constant.REGISTRY_DEFAULT_KEY, "false") == "true" {
			return invoker.Invoke(invocation)
		}
	}

	//If none of the invokers has a local signal, pick the first one available.
	for _, invoker := range invokers {
		if invoker.IsAvailable() {
			return invoker.Invoke(invocation)
		}
	}
	return nil
}
