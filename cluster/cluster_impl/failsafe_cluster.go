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

type failsafeCluster struct{}

const failsafe = "failsafe"

func init() {
	extension.SetCluster(failsafe, NewFailsafeCluster)
}

// NewFailsafeCluster returns a failsafe cluster instance.
//
// Failure of security, anomalies, directly ignored. Usually it is
// used to write audit logs and other operations.
func NewFailsafeCluster() cluster.Cluster {
	return &failsafeCluster{}
}

// Join returns a baseClusterInvoker instance
func (cluster *failsafeCluster) Join(directory cluster.Directory) protocol.Invoker {
	return buildInterceptorChain(newFailsafeClusterInvoker(directory))
}
