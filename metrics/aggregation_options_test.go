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
	"testing"
)

import (
	"github.com/creasty/defaults"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/global"
)

// TestAggregationEnabledByDefaultAndCanBeDisabled pins the default that keeps
// the sliding-window metrics exported, as they always were, and the way to
// turn them off.
func TestAggregationEnabledByDefaultAndCanBeDisabled(t *testing.T) {
	cfg := &global.AggregateConfig{}
	require.NoError(t, defaults.Set(cfg))
	require.NotNil(t, cfg.Enabled)
	assert.True(t, *cfg.Enabled)
	assert.Equal(t, DefaultBucketNum, cfg.BucketNum)
	assert.Equal(t, DefaultTimeWindowSeconds, cfg.TimeWindowSeconds)

	opts := defaultOptions()
	WithAggregationDisabled()(opts)
	require.NotNil(t, opts.Metrics.Aggregation.Enabled)
	assert.False(t, *opts.Metrics.Aggregation.Enabled)
}

func TestWithWindowKeepsDefaultsForNonPositiveValues(t *testing.T) {
	assert.Equal(t, DefaultWindow(), windowOf([]WindowOption{WithWindow(Window{BucketNum: 0, TimeWindowSeconds: -1})}))
	assert.Equal(t, Window{BucketNum: 4, TimeWindowSeconds: 30}, windowOf([]WindowOption{WithWindow(Window{BucketNum: 4, TimeWindowSeconds: 30})}))
}

// TestWindowBucketsCoverAtLeastOneSecond guards against zero-width buckets,
// which make the sliding window divide by zero.
func TestWindowBucketsCoverAtLeastOneSecond(t *testing.T) {
	assert.Equal(t, Window{BucketNum: 1, TimeWindowSeconds: 1}, Window{BucketNum: 1001, TimeWindowSeconds: 1}.OrDefault())
	assert.Equal(t, Window{BucketNum: 5, TimeWindowSeconds: 5}, Window{BucketNum: 10, TimeWindowSeconds: 5}.OrDefault())

	for name, opt := range map[string]WindowOption{
		"WithWindow":    WithWindow(Window{BucketNum: 1001, TimeWindowSeconds: 1}),
		"custom option": func(w *Window) { *w = Window{BucketNum: 1001, TimeWindowSeconds: 1} },
	} {
		vec := NewQpsMetricVec(newMockMetricRegistry(), &MetricKey{Name: "qps"}, opt)
		assert.NotPanics(t, func() { vec.Record(map[string]string{"k": "v"}) }, name)
	}
}
