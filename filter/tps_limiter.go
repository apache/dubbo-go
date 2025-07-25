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

package filter

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
)

// TpsLimiter is the interface which defines the Limiter that judge if the TPS overs the threshold
//
// # IsAllowable method will check whether this invocation should be enabled for further process
//
// please register your implementation by invoking SetTpsLimiter
// The usage, for example:
// "UserProvider":
//
//	registry: "hangzhouzk"
//	protocol : "dubbo"
//	interface : "com.ikurento.user.UserProvider"
//	... # other configuration
//	tps.limiter: "the name of limiter",
/*
 * please register your implementation by invoking SetTpsLimitStrategy
 * "UserProvider":
 *   registry: "hangzhouzk"
 *   protocol : "dubbo"
 *   interface : "com.ikurento.user.UserProvider"
 *   ... # other configuration
 *   tps.limiter: "method-service" # the name of limiter
 *   tps.limit.strategy: "name of implementation" # service-level
 *   methods:
 *    - name: "GetUser"
 *      tps.interval: 3000
 *      tps.limit.strategy: "name of implementation" # method-level
 */
type TpsLimiter interface {
	IsAllowable(*common.URL, base.Invocation) bool
}
