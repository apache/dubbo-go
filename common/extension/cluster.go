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

package extension

import (
	"dubbo.apache.org/dubbo-go/v3/cluster"
)

var clusters = make(map[string]func() cluster.Cluster)

// SetCluster sets the cluster fault-tolerant mode with @name
// For example: available/failfast/broadcast/failfast/failsafe/...
func SetCluster(name string, fcn func() cluster.Cluster) {
	clusters[name] = fcn
}

// GetCluster finds the cluster fault-tolerant mode with @name
func GetCluster(name string) cluster.Cluster {
	if clusters[name] == nil {
		panic("cluster for " + name + " is not existing, make sure you have import the package.")
	}
	return clusters[name]()
}
