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
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

type mockCluster struct{}

// NewMockCluster returns a mock cluster instance.
//
// Mock cluster is usually used for service degradation, such as an authentication service.
// When the service provider is completely hung up, the client does not throw an exception,
// return an authorization failure through the Mock data instead.
func NewMockCluster() cluster.Cluster {
	return &mockCluster{}
}

// nolint
func (cluster *mockCluster) Join(directory cluster.Directory) protocol.Invoker {
	return buildInterceptorChain(protocol.NewBaseInvoker(directory.GetURL()))
}
