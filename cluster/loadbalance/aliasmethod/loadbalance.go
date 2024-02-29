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

// Package aliasmethod implements alias-method algorithm load balance strategy.
package aliasmethod // weighted random with alias-method algorithm

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/loadbalance"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

func init() {
	extension.SetLoadbalance(constant.LoadBalanceKeyAliasMethod, newWeightedRandomWithAliasMethodBalance)
}

type weightedRandomWithAliasMethodBalance struct{}

// newWeightedRandomWithAliasMethodBalance returns a loadbalancer using alias-method algorithm..
func newWeightedRandomWithAliasMethodBalance() loadbalance.LoadBalance {
	return &weightedRandomWithAliasMethodBalance{}
}

// Select gets invoker based on interleaved weighted round robine load balancing strategy
func (lb *weightedRandomWithAliasMethodBalance) Select(invokers []protocol.Invoker, invocation protocol.Invocation) protocol.Invoker {
	count := len(invokers)
	if count == 0 {
		return nil
	}
	if count == 1 {
		return invokers[0]
	}

	wramp := NewAliasMethodPicker(invokers, invocation)
	return wramp.Pick()
}
