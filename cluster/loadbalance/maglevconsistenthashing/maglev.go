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
	"math"

	"github.com/cespare/xxhash/v2"

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
			idx := next[i]
			offset := permutations[i][idx%m.tableSize]
			for m.lookupTable[offset] != -1 {
				idx++
				next[i] = idx
				offset = permutations[i][idx%m.tableSize]
			}
			m.lookupTable[offset] = i
			next[i] = idx + 1
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
	tableSize := uint64(m.tableSize)
	offset := hash(address) % tableSize
	skip := 1 + (hash(address+"skip") % (tableSize - 1))
	permutation := make([]int, m.tableSize)
	for i := 0; i < m.tableSize; i++ {
		permutation[i] = int((offset + uint64(i)*skip) % tableSize)
	}
	return permutation
}

func (m *Maglev) Pick(key string) protocol.Invoker {
	index := int(hash(key) % uint64(m.tableSize))
	return m.invokers[m.lookupTable[index]]
}

// hash.Hash64
func hash(s string) uint64 {
	return xxhash.Sum64String(s)
}

// isPrime 函数用于判断一个数是否为素数
func isPrime(num int) bool {
	if num < 2 {
		return false
	}
	if num == 2 {
		return true
	}
	if num%2 == 0 {
		return false
	}
	sqrt := int(math.Sqrt(float64(num)))
	for i := 3; i <= sqrt; i += 2 {
		if num%i == 0 {
			return false
		}
	}
	return true
}

// NextPrime 函数用于找出比 n 大的第一个素数
func NextPrime(n int) int {
	for num := n + 1; ; num++ {
		if isPrime(num) {
			return num
		}
	}
}
