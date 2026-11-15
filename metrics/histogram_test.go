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
	"math"
	"testing"
)

import (
	"github.com/stretchr/testify/require"
)

func TestHistogramOptsValidate(t *testing.T) {
	for _, tt := range []struct {
		name  string
		opts  *HistogramOpts
		valid bool
	}{
		{"nil", nil, true},
		{"empty", &HistogramOpts{}, true},
		{"valid", &HistogramOpts{Buckets: []float64{0.001, 0.1, 1}}, true},
		{"zero", &HistogramOpts{Buckets: []float64{0, 1}}, false},
		{"negative", &HistogramOpts{Buckets: []float64{-1, 1}}, false},
		{"duplicate", &HistogramOpts{Buckets: []float64{1, 1}}, false},
		{"descending", &HistogramOpts{Buckets: []float64{1, 0.1}}, false},
		{"nan", &HistogramOpts{Buckets: []float64{math.NaN()}}, false},
		{"positive infinity", &HistogramOpts{Buckets: []float64{math.Inf(1)}}, false},
		{"negative infinity", &HistogramOpts{Buckets: []float64{math.Inf(-1)}}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if tt.valid {
				require.NoError(t, tt.opts.Validate())
			} else {
				require.Error(t, tt.opts.Validate())
			}
		})
	}
}
