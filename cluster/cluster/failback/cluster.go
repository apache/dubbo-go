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

package failback

import (
	clusterpkg "dubbo.apache.org/dubbo-go/v3/cluster/cluster"
	"dubbo.apache.org/dubbo-go/v3/cluster/directory"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

func init() {
	extension.SetCluster(constant.ClusterKeyFailBack, NewCluster)
}

type cluster struct{}

// NewCluster returns a failback cluster instance
//
// Failure automatically restored, failed to record the background request,
// regular retransmission. Usually used for message notification operations.
func NewCluster() clusterpkg.Cluster {
	return &cluster{}
}

// Join returns a baseClusterInvoker instance
func (cluster *cluster) Join(directory directory.Directory) protocol.Invoker {
	return clusterpkg.BuildInterceptorChain(newClusterInvoker(directory))
}
