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
	"io"
)

import (
	"github.com/apache/dubbo-go/metrics"
)

type UniformSnapshot struct {
	values []int64
}

func (u *UniformSnapshot) GetValue(quantile float64) (float64, error) {
	panic("implement me")
}

func (u *UniformSnapshot) GetValues() ([]int64, error) {
	panic("implement me")
}

func (u *UniformSnapshot) Size() (int, error) {
	return len(u.values), nil
}

func (u *UniformSnapshot) GetMedian() (float64, error) {
	panic("implement me")
}

func (u *UniformSnapshot) Get75thPercentile() (float64, error) {
	panic("implement me")
}

func (u *UniformSnapshot) Get95thPercentile() (float64, error) {
	panic("implement me")
}

func (u *UniformSnapshot) Get98thPercentile() (float64, error) {
	panic("implement me")
}

func (u *UniformSnapshot) Get99thPercentile() (float64, error) {
	panic("implement me")
}

func (u *UniformSnapshot) Get999thPercentile() (float64, error) {
	panic("implement me")
}

func (u *UniformSnapshot) GetMax() (int64, error) {
	panic("implement me")
}

func (u *UniformSnapshot) GetMean() (int64, error) {
	panic("implement me")
}

func (u *UniformSnapshot) GetMin() (int64, error) {
	panic("implement me")
}

func (u *UniformSnapshot) GetStdDev() (float64, error) {
	panic("implement me")
}

func (u *UniformSnapshot) Dump(writer io.Writer) error {
	panic("implement me")
}

func NewUniformSnapshot(values []int64) metrics.Snapshot {
	return &UniformSnapshot{
		values: values,
	}
}
