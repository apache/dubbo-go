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

package prometheus

import (
	"strings"
	"sync"
)

import (
	"github.com/prometheus/client_golang/prometheus"
)

func convertLabelsToMapKey(labels prometheus.Labels) string {
	return strings.Join([]string{
		labels[applicationNameKey],
		labels[groupKey],
		labels[hostnameKey],
		labels[interfaceKey],
		labels[ipKey],
		labels[versionKey],
		labels[methodKey],
	}, "_")
}

func updateMin(m *sync.Map, labels *prometheus.Labels, gaugeVec *prometheus.GaugeVec, costMs int64) {
	key := convertLabelsToMapKey(*labels)
	for {
		if actual, loaded := m.LoadOrStore(key, costMs); loaded {
			if costMs < actual.(int64) {
				// need to update
				if m.CompareAndSwap(key, actual, costMs) {
					// value is not changed, update success
					gaugeVec.With(*labels).Set(float64(costMs))
					break
				}
			} else {
				// no need to update
				break
			}
		} else {
			// store current costMs as this labels' init value
			gaugeVec.With(*labels).Set(float64(costMs))
			break
		}
	}
}
