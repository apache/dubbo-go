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

package adaptivesvc

import (
	"sync"
)

import (
	clusterpkg "dubbo.apache.org/dubbo-go/v3/cluster/cluster"
	"dubbo.apache.org/dubbo-go/v3/cluster/directory"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

var (
	once     sync.Once
	instance clusterpkg.Cluster
)

func init() {
	extension.SetCluster(constant.ClusterKeyAdaptiveService, newAdaptiveServiceCluster)
}

// adaptiveServiceCluster is a cluster for adaptive service.
type adaptiveServiceCluster struct{}

func newAdaptiveServiceCluster() clusterpkg.Cluster {
	if instance == nil {
		once.Do(func() {
			instance = &adaptiveServiceCluster{}
		})
	}
	return instance
}

func (c *adaptiveServiceCluster) Join(directory directory.Directory) protocol.Invoker {
	return clusterpkg.BuildInterceptorChain(newAdaptiveServiceClusterInvoker(directory))
}
