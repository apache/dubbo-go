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
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

// GetWeight gets weight for load balance strategy
func GetWeight(invoker protocol.Invoker, invocation protocol.Invocation) int64 {
	var weight int64
	url := invoker.GetURL()
	// Multiple registry scenario, load balance among multiple registries.
	isRegIvk := url.GetParamBool(constant.RegistryKey+"."+constant.RegistryLabelKey, false)
	if isRegIvk {
		weight = url.GetParamInt(constant.RegistryKey+"."+constant.WeightKey, constant.DefaultWeight)
	} else {
		weight = url.GetMethodParamInt64(invocation.MethodName(), constant.WeightKey, constant.DefaultWeight)

		if weight > 0 {
			// get service register time an do warm up time
			now := time.Now().Unix()
			timestamp := url.GetParamInt(constant.RemoteTimestampKey, now)
			if uptime := now - timestamp; uptime > 0 {
				warmup := url.GetParamInt(constant.WarmupKey, constant.DefaultWarmup)
				if uptime < warmup {
					if ww := float64(uptime) / float64(warmup) / float64(weight); ww < 1 {
						weight = 1
					} else if int64(ww) <= weight {
						weight = int64(ww)
					}
				}
			}
		}
	}

	if weight < 0 {
		weight = 0
	}

	return weight
}
