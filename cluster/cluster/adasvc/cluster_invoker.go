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

package adasvc

import (
	"context"
	"dubbo.apache.org/dubbo-go/v3/cluster/cluster/base"
	"dubbo.apache.org/dubbo-go/v3/cluster/directory"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

type clusterInvoker struct {
	base.ClusterInvoker
}

func NewClusterInvoker(directory directory.Directory) protocol.Invoker {
	return &clusterInvoker{
		ClusterInvoker: base.NewClusterInvoker(directory),
	}
}

func (ivk *clusterInvoker) Invoke(_ context.Context, invocation protocol.Invocation) protocol.Result {
	// invokers for the invocation
	ivks := ivk.Directory.List(invocation)
}
