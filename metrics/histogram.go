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

package metrics

import (
	"fmt"
	"math"
)

// HistogramProvider is an optional MetricRegistry capability.
// Existing registries do not need to implement it. HistogramWithOptions must
// preserve the default behavior of Histogram when opts or Buckets is empty.
type HistogramProvider interface {
	HistogramWithOptions(*MetricId, *HistogramOpts) ObservableMetric
}

// HistogramOpts configures finite bucket upper bounds, in the same unit as
// observed values. Bounds must be positive, finite and strictly increasing.
// Empty Buckets selects the registry's defaults. The first registration of a
// metric name determines its buckets for all label sets; they cannot be changed
// by subsequent calls. Registries must not retain the caller's slice.
type HistogramOpts struct {
	Buckets []float64
}

// Validate checks bucket boundaries. Nil options and empty buckets are valid.
func (o *HistogramOpts) Validate() error {
	if o == nil {
		return nil
	}
	for i, bound := range o.Buckets {
		if math.IsNaN(bound) || math.IsInf(bound, 0) || bound <= 0 {
			return fmt.Errorf("histogram bucket %d must be positive and finite", i)
		}
		if i > 0 && bound <= o.Buckets[i-1] {
			return fmt.Errorf("histogram buckets must be strictly increasing")
		}
	}
	return nil
}
