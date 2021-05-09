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

package extension

import (
	"sort"
)

import (
	"dubbo.apache.org/dubbo-go/v3/registry"
)

var customizers = make([]registry.ServiceInstanceCustomizer, 0, 8)

// AddCustomizers will put the customizer into slices and then sort them;
// this method will be invoked several time, so we sort them here.
func AddCustomizers(cus registry.ServiceInstanceCustomizer) {
	customizers = append(customizers, cus)
	sort.Stable(customizerSlice(customizers))
}

// GetCustomizers will return the sorted customizer
// the result won't be nil
func GetCustomizers() []registry.ServiceInstanceCustomizer {
	return customizers
}

type customizerSlice []registry.ServiceInstanceCustomizer

// nolint
func (c customizerSlice) Len() int {
	return len(c)
}

// nolint
func (c customizerSlice) Swap(i, j int) { c[i], c[j] = c[j], c[i] }

// nolint
func (c customizerSlice) Less(i, j int) bool { return c[i].GetPriority() < c[j].GetPriority() }
