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

type NopSnapshot struct{}

func (n *NopSnapshot) GetValue(quantile float64) (float64, error) {
	return 0, nil
}

func (n *NopSnapshot) GetValues() ([]int64, error) {
	return make([]int64, 0, 0), nil
}

func (n *NopSnapshot) Size() (int, error) {
	return 0, nil
}

func (n *NopSnapshot) GetMedian() (float64, error) {
	return 0, nil
}

func (n *NopSnapshot) Get75thPercentile() (float64, error) {
	return 0, nil
}

func (n *NopSnapshot) Get95thPercentile() (float64, error) {
	return 0, nil
}

func (n *NopSnapshot) Get98thPercentile() (float64, error) {
	return 0, nil
}

func (n *NopSnapshot) Get99thPercentile() (float64, error) {
	return 0, nil
}

func (n *NopSnapshot) Get999thPercentile() (float64, error) {
	return 0, nil
}

func (n *NopSnapshot) GetMax() (int64, error) {
	return 0, nil
}

func (n *NopSnapshot) GetMean() (float64, error) {
	return 0, nil
}

func (n *NopSnapshot) GetMin() (int64, error) {
	return 0, nil
}

func (n *NopSnapshot) GetStdDev() (float64, error) {
	return 0, nil
}

var nopSnapshot = &NopSnapshot{}
