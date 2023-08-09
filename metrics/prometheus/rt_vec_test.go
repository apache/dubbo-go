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
	"reflect"
	"sync"
	"testing"
)

import (
	prom "github.com/prometheus/client_golang/prometheus"

	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/metrics/util/aggregate"
)

func TestRtVecCollect(t *testing.T) {
	r := &RtVec{
		opts: &RtOpts{
			Name:              "request_num",
			bucketNum:         10,
			timeWindowSeconds: 120,
			Help:              "Request cost",
		},
		labelNames: []string{"app", "version"},
	}
	labels := map[string]string{"app": "dubbo", "version": "1.0.0"}
	r.With(labels).Observe(100)
	ch := make(chan prom.Metric, len(rtMetrics))
	r.Collect(ch)
	close(ch)
	assert.Equal(t, len(ch), len(rtMetrics))
	for _, m := range rtMetrics {
		metric, ok := <-ch
		if !ok {
			t.Error("not enough metrics")
		} else {
			str := metric.Desc().String()
			assert.Contains(t, str, m.nameSuffix)
			assert.Contains(t, str, m.helpPrefix)
			assert.Contains(t, str, "app")
			assert.Contains(t, str, "version")
		}
	}
}

func TestRtVecDescribe(t *testing.T) {
	r := &RtVec{
		opts: &RtOpts{
			Name:              "request_num",
			bucketNum:         10,
			timeWindowSeconds: 120,
			Help:              "Request cost",
		},
		labelNames: []string{"app", "version"},
	}
	ch := make(chan *prom.Desc, len(rtMetrics))
	r.Describe(ch)
	close(ch)
	assert.Equal(t, len(ch), len(rtMetrics))
	for _, m := range rtMetrics {
		desc, ok := <-ch
		if !ok {
			t.Error(t, "not enough desc")
		} else {
			str := desc.String()
			assert.Contains(t, str, m.nameSuffix)
			assert.Contains(t, str, m.helpPrefix)
			assert.Contains(t, str, "app")
			assert.Contains(t, str, "version")
		}
	}
}

func TestValueAggObserve(t *testing.T) {
	v := &valueAgg{
		tags: map[string]string{},
		agg:  aggregate.NewTimeWindowAggregator(10, 10),
	}
	v.Observe(float64(1))
	r := v.agg.Result()
	want := &aggregate.Result{
		Avg:   1,
		Min:   1,
		Max:   1,
		Count: 1,
		Total: 1,
		Last:  1,
	}
	if !reflect.DeepEqual(r, want) {
		t.Errorf("Result() = %v, want %v", r, want)
	}
}

func TestRtVecWith(t *testing.T) {
	r := &RtVec{
		opts: &RtOpts{
			Name:              "request_num",
			bucketNum:         10,
			timeWindowSeconds: 120,
			Help:              "Request cost",
		},
		labelNames: []string{"app", "version"},
	}
	labels := map[string]string{"app": "dubbo", "version": "1.0.0"}
	first := r.With(labels)
	second := r.With(labels)
	assert.True(t, first == second) // init once
}

func TestRtVecWithConcurrent(t *testing.T) {
	r := &RtVec{
		opts: &RtOpts{
			Name:              "request_num",
			bucketNum:         10,
			timeWindowSeconds: 120,
			Help:              "Request cost",
		},
		labelNames: []string{"app", "version"},
	}
	labels := map[string]string{"app": "dubbo", "version": "1.0.0"}
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			r.With(labels).Observe(100)
			wg.Done()
		}()
	}
	wg.Wait()
	res := r.With(labels).(*valueAgg).agg.Result()
	assert.True(t, res.Count == uint64(10))
}
