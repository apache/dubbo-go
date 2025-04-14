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

package maglevconsistenthashing

import (
	"hash/fnv"

	"dubbo.apache.org/dubbo-go/v3/protocol"
)

type Maglev struct {
	invokers    []protocol.Invoker // Instance
	lookupTable []int

	tableSize int // Prime number
}

func NewMaglev(invokers []protocol.Invoker) *Maglev {
	// Next invokers Prime number
	pn := NextPrime(len(invokers))
	m := &Maglev{
		lookupTable: make([]int, pn),
		invokers:    invokers,
		tableSize:   pn,
	}
	m.buildLookupTable()
	return m
}

// buildLookupTable 构建 Maglev 查找表
func (m *Maglev) buildLookupTable() {
	permutations := m.generatePermutations()
	next := make([]int, len(m.invokers))
	inserted := 0

	for i := range m.lookupTable {
		m.lookupTable[i] = -1
	}

	for inserted < m.tableSize {
		for i := range m.invokers {
			if inserted >= m.tableSize {
				break
			}
			offset := permutations[i][next[i]%m.tableSize]
			for m.lookupTable[offset] != -1 {
				next[i]++
				offset = permutations[i][next[i]%m.tableSize]
			}
			m.lookupTable[offset] = i
			next[i]++
			inserted++
		}
	}
}

// generatePermutations 生成每个节点的排列
func (m *Maglev) generatePermutations() [][]int {
	permutations := make([][]int, len(m.invokers))
	for i, node := range m.invokers {
		permutations[i] = m.calculatePermutation(node)
	}
	return permutations
}

// calculatePermutation 计算单个节点的排列
func (m *Maglev) calculatePermutation(node protocol.Invoker) []int {
	u := node.GetURL()
	address := u.Ip + ":" + u.Port
	offset := hash(address) % uint64(m.tableSize)
	skip := 1 + (hash(address+"skip") % uint64(m.tableSize-1))
	permutation := make([]int, m.tableSize)
	for i := 0; i < m.tableSize; i++ {
		permutation[i] = int((offset + uint64(i)*skip) % uint64(m.tableSize))
	}
	return permutation
}

func (m *Maglev) Pick(key string) protocol.Invoker {
	index := int(hash(key) % uint64(m.tableSize))
	return m.invokers[m.lookupTable[index]]
}

// hash 简单的哈希函数示例
func hash(s string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(s))
	return h.Sum64()
}

// isPrime 函数用于判断一个数是否为素数
func isPrime(num int) bool {
	if num < 2 {
		return false
	}
	for i := 2; i*i <= num; i++ {
		if num%i == 0 {
			return false
		}
	}
	return true
}

// NextPrime 函数用于找出比 n 大的第一个素数
func NextPrime(n int) int {
	num := n + 1
	for !isPrime(num) {
		num++
	}
	return num
}
