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
	"errors"
	"math"
	"sort"
)

import (
	"github.com/Workiva/go-datastructures/common"
)

import (
	"github.com/apache/dubbo-go/metrics"
)

type WeightedSnapshot struct {
	values            []int64
	normalizedWeights []float64
	quantiles         []float64
}

func (w *WeightedSnapshot) GetValue(quantile float64) (float64, error) {
	if quantile < 0 || quantile > 1 || math.IsNaN(quantile) {
		return 0, errors.New("The quantile must be [0,1] ")
	}

	length := len(w.values)
	if length == 0 {
		return 0.0, nil
	}

	pos, found := w.searchQuantile(quantile)

	if !found {
		pos = pos - 1
	}

	if pos < 1 {
		return float64(w.values[0]), nil
	}

	// the last one
	if pos >= length {
		return float64(w.values[length-1]), nil
	}

	return float64(w.values[pos]), nil
}

// if found is true, it means the w.quantiles contains the target
// if found is false, it means the w.quantiles doesn't contains the target,
// and the pos will be the position to be inserted.
func (w *WeightedSnapshot) searchQuantile(target float64) (pos int, found bool) {
	pos = sort.SearchFloat64s(w.quantiles, target)
	found = sort.Search(len(w.quantiles), func(i int) bool {
		return w.quantiles[i] == target
	}) != len(w.quantiles)
	return pos, found
}

func (w *WeightedSnapshot) GetValues() ([]int64, error) {
	length := len(w.values)
	result := make([]int64, 0, length)
	for _, ele := range w.values {
		result = append(result, ele)
	}
	return result, nil
}

func (w *WeightedSnapshot) Size() (int, error) {
	return len(w.values), nil
}

// Get the median value in the distribution
func (w *WeightedSnapshot) GetMedian() (float64, error) {
	return w.GetValue(0.5)
}

// Get the value at 75th percentile in the distribution
func (w *WeightedSnapshot) Get75thPercentile() (float64, error) {
	return w.GetValue(0.75)
}

// Get the value at 95th percentile in the distribution
func (w *WeightedSnapshot) Get95thPercentile() (float64, error) {
	return w.GetValue(0.95)
}

// Get the value at 98th percentile in the distribution
func (w *WeightedSnapshot) Get98thPercentile() (float64, error) {
	return w.GetValue(0.98)
}

// Get the value at 99th percentile in the distribution
func (w *WeightedSnapshot) Get99thPercentile() (float64, error) {
	return w.GetValue(0.99)
}

// Get the value at 999th percentile in the distribution
func (w *WeightedSnapshot) Get999thPercentile() (float64, error) {
	return w.GetValue(0.999)
}

func (w *WeightedSnapshot) GetMax() (int64, error) {
	length := len(w.values)
	if length == 0 {
		return 0, errors.New("The values is empty. ")
	}

	return w.values[length-1], nil
}

func (w *WeightedSnapshot) GetMean() (float64, error) {
	length := len(w.values)
	if length == 0 {
		return 0, errors.New("The values is empty. ")
	}

	sum := float64(0)
	for index, ele := range w.values {
		sum += float64(ele) * w.normalizedWeights[index]
	}
	return sum, nil
}

func (w *WeightedSnapshot) GetMin() (int64, error) {
	length := len(w.values)
	if length == 0 {
		return 0, errors.New("The values is empty. ")
	}
	return w.values[0], nil
}

func (w *WeightedSnapshot) GetStdDev() (float64, error) {
	length := len(w.values)
	if length <= 1 {
		return 0, nil
	}

	mean, _ := w.GetMean()
	variance := float64(0)

	for index, ele := range w.values {
		diff := float64(ele) - mean
		variance += w.normalizedWeights[index] * diff * diff
	}
	return math.Sqrt(variance), nil
}

func NewWeightedSnapshot(samples []*WeightedSample) metrics.Snapshot {
	length := len(samples)

	result := &WeightedSnapshot{
		values:            make([]int64, 0, length),
		normalizedWeights: make([]float64, 0, length),
		quantiles:         make([]float64, length, length),
	}

	if length == 0 {
		return result
	}

	cp := make([]*WeightedSample, 0, length)
	totalWeight := float64(0)
	for _, ele := range samples {
		cp = append(cp, ele)
		totalWeight += ele.weight
	}

	// sort by value
	sort.Slice(cp, func(i, j int) bool {
		return cp[i].value < cp[j].value
	})

	for _, ele := range cp {
		result.values = append(result.values, ele.value)
		result.normalizedWeights = append(result.normalizedWeights, ele.weight/totalWeight)
	}

	for i := 1; i < length; i++ {
		result.quantiles[i] = result.quantiles[i-1] + result.normalizedWeights[i-1]
	}
	return result
}

type WeightedSample struct {
	key    float64
	value  int64
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
		key:    key,
		value:  value,
		weight: weight,
	}
}
