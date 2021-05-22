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
	"dubbo.apache.org/dubbo-go/v3/cluster"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

type forkingCluster struct{}

const forking = "forking"

func init() {
	extension.SetCluster(forking, NewForkingCluster)
}

// NewForkingCluster returns a forking cluster instance.
//
// Multiple servers are invoked in parallel, returning as soon as one succeeds.
// Usually it is used for real-time demanding read operations while wasting more service resources.
func NewForkingCluster() cluster.Cluster {
	return &forkingCluster{}
}

// Join returns a baseClusterInvoker instance
func (cluster *forkingCluster) Join(directory cluster.Directory) protocol.Invoker {
	return buildInterceptorChain(newForkingClusterInvoker(directory))
}
