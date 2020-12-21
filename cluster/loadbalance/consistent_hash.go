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
	"crypto/md5"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"regexp"
	"sort"
	"strconv"
	"strings"
)
import (
	gxsort "github.com/dubbogo/gost/sort"
)
import (
	"github.com/apache/dubbo-go/cluster"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/protocol"
)

const (
	// ConsistentHash consistent hash
	ConsistentHash = "consistenthash"
	// HashNodes hash nodes
	HashNodes = "hash.nodes"
	// HashArguments key of hash arguments in url
	HashArguments = "hash.arguments"
)

var (
	selectors = make(map[string]*ConsistentHashSelector)
	re        = regexp.MustCompile(constant.COMMA_SPLIT_PATTERN)
)

func init() {
	extension.SetLoadbalance(ConsistentHash, NewConsistentHashLoadBalance)
}

// ConsistentHashLoadBalance implementation of load balancing: using consistent hashing
type ConsistentHashLoadBalance struct {
}

// NewConsistentHashLoadBalance creates NewConsistentHashLoadBalance
//
// The same parameters of the request is always sent to the same provider.
func NewConsistentHashLoadBalance() cluster.LoadBalance {
	return &ConsistentHashLoadBalance{}
}

// Select gets invoker based on load balancing strategy
func (lb *ConsistentHashLoadBalance) Select(invokers []protocol.Invoker, invocation protocol.Invocation) protocol.Invoker {
	methodName := invocation.MethodName()
	key := invokers[0].GetUrl().ServiceKey() + "." + methodName

	// hash the invokers
	bs := make([]byte, 0)
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
		selectors[key] = newConsistentHashSelector(invokers, methodName, hashCode)
		selector = selectors[key]
	}
	return selector.Select(invocation)
}

// ConsistentHashSelector implementation of Selector:get invoker based on load balancing strategy
type ConsistentHashSelector struct {
	hashCode        uint32
	replicaNum      int
	virtualInvokers map[uint32]protocol.Invoker
	keys            gxsort.Uint32Slice
	argumentIndex   []int
}

func newConsistentHashSelector(invokers []protocol.Invoker, methodName string,
	hashCode uint32) *ConsistentHashSelector {

	selector := &ConsistentHashSelector{}
	selector.virtualInvokers = make(map[uint32]protocol.Invoker)
	selector.hashCode = hashCode
	url := invokers[0].GetUrl()
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
		u := invoker.GetUrl()
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
func (c *ConsistentHashSelector) Select(invocation protocol.Invocation) protocol.Invoker {
	key := c.toKey(invocation.Arguments())
	digest := md5.Sum([]byte(key))
	return c.selectForKey(c.hash(digest, 0))
}

func (c *ConsistentHashSelector) toKey(args []interface{}) string {
	var sb strings.Builder
	for i := range c.argumentIndex {
		if i >= 0 && i < len(args) {
			fmt.Fprint(&sb, args[i].(string))
		}
	}
	return sb.String()
}

func (c *ConsistentHashSelector) selectForKey(hash uint32) protocol.Invoker {
	idx := sort.Search(len(c.keys), func(i int) bool {
		return c.keys[i] >= hash
	})
	if idx == len(c.keys) {
		idx = 0
	}
	return c.virtualInvokers[c.keys[idx]]
}

// nolint
func (c *ConsistentHashSelector) hash(digest [16]byte, i int) uint32 {
	return uint32((digest[3+i*4]&0xFF)<<24) | uint32((digest[2+i*4]&0xFF)<<16) |
		uint32((digest[1+i*4]&0xFF)<<8) | uint32(digest[i*4]&0xFF)&0xFFFFFFF
}
