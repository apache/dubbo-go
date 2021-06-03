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
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

type zoneAwareCluster struct{}

func init() {
	extension.SetCluster(constant.ZONEAWARE_CLUSTER_NAME, NewZoneAwareCluster)
}

// NewZoneAwareCluster returns a zoneaware cluster instance.
//
// More than one registry for subscription.
// Usually it is used for choose between registries.
func NewZoneAwareCluster() cluster.Cluster {
	return &zoneAwareCluster{}
}

// Join returns a zoneAwareClusterInvoker instance
func (cluster *zoneAwareCluster) Join(directory cluster.Directory) protocol.Invoker {
	return buildInterceptorChain(newZoneAwareClusterInvoker(directory), getZoneAwareInterceptor())
}
