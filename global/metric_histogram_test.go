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

package global

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHistogramConfigClone(t *testing.T) {
	enabled := true
	c := &MetricsConfig{Histogram: &HistogramConfig{Enabled: &enabled, Buckets: []float64{0.001, 1}}}
	clone := c.Clone()
	require.Equal(t, c, clone)
	*clone.Histogram.Enabled = false
	clone.Histogram.Buckets[0] = 0.1
	assert.True(t, *c.Histogram.Enabled)
	assert.InDelta(t, 0.001, c.Histogram.Buckets[0], 1e-12)
	assert.Nil(t, (&MetricsConfig{}).Clone().Histogram)
	assert.Equal(t, &HistogramConfig{}, (&HistogramConfig{}).Clone())
}
