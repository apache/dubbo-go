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

package loadbalance

import (
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
)


// GetWeight returns the weight for the load‑balancing strategy.
func GetWeight(invoker protocol.Invoker, invocation protocol.Invocation) int64 {

	url := invoker.GetURL()

	// 1) Method‑level or registry‑level weight taken from URL parameters — highest priority.
	var weight = url.GetMethodParamInt64(
		invocation.MethodName(), constant.WeightKey, -1)

	if weight < 0 && url.GetParamBool(constant.RegistryKey+"."+constant.RegistryLabelKey, false) {
		weight = url.GetParamInt(constant.RegistryKey+"."+constant.WeightKey, constant.DefaultWeight)
	}

	// 2) If the URL does not specify a weight, fall back to the default value (100).
	if weight < 0 {
		weight = constant.DefaultWeight
	}

	// 3) Warm‑up adjustment (same logic as before).
	if weight > 0 {
		now := time.Now().Unix()
		ts := url.GetParamInt(constant.RemoteTimestampKey, now)
		if uptime := now - ts; uptime > 0 {
			warm := url.GetParamInt(constant.WarmupKey, constant.DefaultWarmup)
			if uptime < warm {
				calc := float64(uptime) * float64(weight) / float64(warm)
				if calc < 1 {
					weight = 1
				} else if int64(calc) <= weight {
					weight = int64(calc)
				}
			}
		}
	}

	if weight < 0 {
		weight = 0
	}
	return weight
}
