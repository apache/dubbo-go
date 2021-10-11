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
	"crypto/md5"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

import (
	gxsort "github.com/dubbogo/gost/sort"
)

import (
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

// selector implementation of Selector:get invoker based on load balancing strategy
type selector struct {
	hashCode        uint32
	replicaNum      int
	virtualInvokers map[uint32]protocol.Invoker
	keys            gxsort.Uint32Slice
	argumentIndex   []int
}

func newSelector(invokers []protocol.Invoker, methodName string,
	hashCode uint32) *selector {

	selector := &selector{}
	selector.virtualInvokers = make(map[uint32]protocol.Invoker)
	selector.hashCode = hashCode
	url := invokers[0].GetURL()
	selector.replicaNum = url.GetMethodParamIntValue(methodName, HashNodes, 160)
	indices := re.Split(url.GetMethodParam(methodName, HashArguments, "0"), -1)
	for _, index := range indices {
		i, err := strconv.Atoi(index)
		if err != nil {
			return nil
		}
		selector.argumentIndex = append(selector.argumentIndex, i)
	}
	for _, invoker := range invokers {
		u := invoker.GetURL()
		address := u.Ip + ":" + u.Port
		for i := 0; i < selector.replicaNum/4; i++ {
			digest := md5.Sum([]byte(address + strconv.Itoa(i)))
			for j := 0; j < 4; j++ {
				key := selector.hash(digest, j)
				selector.keys = append(selector.keys, key)
				selector.virtualInvokers[key] = invoker
			}
		}
	}
	sort.Sort(selector.keys)
	return selector
}

// Select gets invoker based on load balancing strategy
func (c *selector) Select(invocation protocol.Invocation) protocol.Invoker {
	key := c.toKey(invocation.Arguments())
	digest := md5.Sum([]byte(key))
	return c.selectForKey(c.hash(digest, 0))
}

func (c *selector) toKey(args []interface{}) string {
	var sb strings.Builder
	for i := range c.argumentIndex {
		if i >= 0 && i < len(args) {
			_, _ = fmt.Fprint(&sb, args[i].(string))
		}
	}
	return sb.String()
}

func (c *selector) selectForKey(hash uint32) protocol.Invoker {
	idx := sort.Search(len(c.keys), func(i int) bool {
		return c.keys[i] >= hash
	})
	if idx == len(c.keys) {
		idx = 0
	}
	return c.virtualInvokers[c.keys[idx]]
}

func (c *selector) hash(digest [16]byte, i int) uint32 {
	return (uint32(digest[3+i*4]&0xFF) << 24) | (uint32(digest[2+i*4]&0xFF) << 16) |
		(uint32(digest[1+i*4]&0xFF) << 8) | uint32(digest[i*4]&0xFF)&0xFFFFFFF
}
