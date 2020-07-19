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
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/protocol"
)

type broadcastCluster struct{}

const broadcast = "broadcast"

func init() {
	extension.SetCluster(broadcast, NewBroadcastCluster)
}

// NewBroadcastCluster returns a broadcast cluster instance.
//
// Calling all providers' broadcast one by one. All errors will be reported.
// It is usually used to notify all providers to update local resource information such as caches or logs.
func NewBroadcastCluster() cluster.Cluster {
	return &broadcastCluster{}
}

// Join returns a baseClusterInvoker instance
func (cluster *broadcastCluster) Join(directory cluster.Directory) protocol.Invoker {
	return newBroadcastClusterInvoker(directory)
}
