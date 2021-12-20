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

package consistenthashing

import (
	"encoding/json"
	"hash/crc32"
	"regexp"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/loadbalance"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

const (
	// HashNodes hash nodes
	HashNodes = "hash.nodes"
	// HashArguments key of hash arguments in url
	HashArguments = "hash.arguments"
)

var (
	selectors = make(map[string]*selector)
	re        = regexp.MustCompile(constant.CommaSplitPattern)
)

func init() {
	extension.SetLoadbalance(constant.LoadBalanceKeyConsistentHashing, newConshashLoadBalance)
}

// conshashLoadBalance implementation of load balancing: using consistent hashing
type conshashLoadBalance struct{}

// newConshashLoadBalance creates NewConsistentHashLoadBalance
//
// The same parameters of the request is always sent to the same provider.
func newConshashLoadBalance() loadbalance.LoadBalance {
	return &conshashLoadBalance{}
}

// Select gets invoker based on load balancing strategy
func (lb *conshashLoadBalance) Select(invokers []protocol.Invoker, invocation protocol.Invocation) protocol.Invoker {
	methodName := invocation.MethodName()
	key := invokers[0].GetURL().ServiceKey() + "." + methodName

	// hash the invokers
	var bs []byte
	for _, invoker := range invokers {
		b, err := json.Marshal(invoker)
		if err != nil {
			return nil
		}
		bs = append(bs, b...)
	}
	hashCode := crc32.ChecksumIEEE(bs)
	selector, ok := selectors[key]
	if !ok || selector.hashCode != hashCode {
		selectors[key] = newSelector(invokers, methodName, hashCode)
		selector = selectors[key]
	}
	return selector.Select(invocation)
}
