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
	"dubbo.apache.org/dubbo-go/v3/cluster/loadbalance"
)

var loadbalances = NewRegistry[func() loadbalance.LoadBalance]("loadbalance")

// SetLoadbalance sets the loadbalance extension with @name
// For example: random/round_robin/consistent_hash/least_active/...
func SetLoadbalance(name string, fcn func() loadbalance.LoadBalance) {
	loadbalances.Register(name, fcn)
}

// GetLoadbalance finds the loadbalance extension with @name
func GetLoadbalance(name string) loadbalance.LoadBalance {
	return loadbalances.MustGet(name)()
}

// UnregisterLoadbalance removes the loadbalance extension with @name
func UnregisterLoadbalance(name string) {
	loadbalances.Unregister(name)
}

// GetAllLoadbalanceNames returns all registered loadbalance names
func GetAllLoadbalanceNames() []string {
	return loadbalances.Names()
}
