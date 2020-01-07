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
	"fmt"
	"math"
	"sort"
)

import (
	"github.com/apache/dubbo-go/metrics"
)

type UniformSnapshot struct {
	values []int64
}

func (u *UniformSnapshot) GetValue(quantile float64) (float64, error) {
	if quantile < 0.0 || quantile > 1.0 || math.IsNaN(quantile) {
		return 0, errors.New(fmt.Sprintf("The quantile must be [0, 1]: %f", quantile))
	}

	size, _ := u.Size()
	if size == 0 {
		return 0.0, nil
	}

	pos := quantile * float64(size+1)
	index := int(pos)

	if index < 1 {
		return float64(u.values[0]), nil
	}

	if index >= size {
		return float64(u.values[size-1]), nil
	}

	lower := u.values[index-1]
	upper := u.values[index]

	return float64(lower) + (pos-math.Floor(pos))*float64(upper-lower), nil
}

func (u *UniformSnapshot) GetValues() ([]int64, error) {
	size, _ := u.Size()
	result := make([]int64, size)
	copy(result, u.values)
	return result, nil
}

func (u *UniformSnapshot) Size() (int, error) {
	return len(u.values), nil
}

func (u *UniformSnapshot) GetMax() (int64, error) {
	size, _ := u.Size()
	if size == 0 {
		return 0, nil
	}
	return u.values[size-1], nil
}

func (u *UniformSnapshot) GetMean() (float64, error) {

	size, _ := u.Size()
	if size == 0 {
		return 0, nil
	}

	var sum int64 = 0
	for _, value := range u.values {
		sum += value
	}

	return float64(sum) / float64(size), nil
}

func (u *UniformSnapshot) GetMin() (int64, error) {
	size, _ := u.Size()
	if size == 0 {
		return 0, nil
	}

	return u.values[0], nil
}

func (u *UniformSnapshot) GetStdDev() (float64, error) {
	size, _ := u.Size()
	if size <= 1 {
		return 0, nil
	}

	mean, _ := u.GetMean()
	var diffSum float64 = 0
	for _, value := range u.values {
		diff := float64(value) - mean
		diffSum += diff * diff
	}

	variance := diffSum / float64(size-1)
	return math.Sqrt(variance), nil
}

// Get the median value in the distribution
func (u *UniformSnapshot) GetMedian() (float64, error) {
	return u.GetValue(0.5)
}

// Get the value at 75th percentile in the distribution
func (u *UniformSnapshot) Get75thPercentile() (float64, error) {
	return u.GetValue(0.75)
}

// Get the value at 95th percentile in the distribution
func (u *UniformSnapshot) Get95thPercentile() (float64, error) {
	return u.GetValue(0.95)
}

// Get the value at 98th percentile in the distribution
func (u *UniformSnapshot) Get98thPercentile() (float64, error) {
	return u.GetValue(0.98)
}

// Get the value at 99th percentile in the distribution
func (u *UniformSnapshot) Get99thPercentile() (float64, error) {
	return u.GetValue(0.99)
}

// Get the value at 999th percentile in the distribution
func (u *UniformSnapshot) Get999thPercentile() (float64, error) {
	return u.GetValue(0.999)
}

func NewUniformSnapshot(values []int64) metrics.Snapshot {
	// make a copy in case of `values` being changed.
	copied := make([]int64, len(values))
	copy(copied, values)
	sort.Sort(Int64Slice(copied))
	return &UniformSnapshot{
		values: copied,
	}
}

type Int64Slice []int64

func (p Int64Slice) Len() int {
	return len(p)
}

func (p Int64Slice) Less(i, j int) bool {
	return p[i] < p[j]
}

func (p Int64Slice) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}
