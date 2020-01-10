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

package impl

import (
	"github.com/Workiva/go-datastructures/common"
)

type WeightedSnapshot struct {
	
}

type WeightedSample struct {
	key float64
	value int64
	weight float64
}

func (w *WeightedSample) Compare(other common.Comparator) int {
	sample := other.(*WeightedSample)
	if w.key < sample.key {
		return -1
	}

	if w.key > sample.key {
		return 1
	}
	return 0
}

func NewWeightSample(key float64, value int64, weight float64) *WeightedSample {
	return &WeightedSample{
		key:    0,
		value:  0,
		weight: 0,
	}
}


