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
	"dubbo.apache.org/dubbo-go/v3/metrics/util/aggregate"

	prom "github.com/prometheus/client_golang/prometheus"

	"github.com/stretchr/testify/assert"
)

func TestRtVecCollect(t *testing.T) {
	opts := &RtOpts{
		Name:              "request_num",
		bucketNum:         10,
		timeWindowSeconds: 120,
		Help:              "Request cost",
	}
	labels := []string{"app", "version"}
	vecs := [2]*RtVec{NewRtVec(opts, labels), NewAggRtVec(opts, labels)}
	for _, r := range vecs {
		r.With(map[string]string{"app": "dubbo", "version": "1.0.0"}).Observe(100)
		ch := make(chan prom.Metric, len(r.metrics))
		r.Collect(ch)
		close(ch)
		assert.Equal(t, len(ch), len(r.metrics))
		for _, m := range r.metrics {
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
}

func TestRtVecDescribe(t *testing.T) {
	opts := &RtOpts{
		Name:              "request_num",
		bucketNum:         10,
		timeWindowSeconds: 120,
		Help:              "Request cost",
	}
	labels := []string{"app", "version"}
	vecs := [2]*RtVec{NewRtVec(opts, labels), NewAggRtVec(opts, labels)}
	for _, r := range vecs {
		ch := make(chan *prom.Desc, len(r.metrics))
		r.Describe(ch)
		close(ch)
		assert.Equal(t, len(ch), len(r.metrics))
		for _, m := range r.metrics {
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
}

func TestValueObserve(t *testing.T) {
	rts := []*Rt{
		{
			tags: map[string]string{},
			obs:  &aggResult{agg: aggregate.NewTimeWindowAggregator(10, 10)},
		},
		{
			tags: map[string]string{},
			obs:  &valueResult{val: aggregate.NewResult()},
		},
	}
	want := &aggregate.Result{
		Avg:   1,
		Min:   1,
		Max:   1,
		Count: 1,
		Total: 1,
		Last:  1,
	}
	for i, v := range rts {
		v.Observe(float64(1))
		r := v.obs.result()
		if i == 0 {
			r.Last = 1 // agg result no Last, value is NaN
		}
		if !reflect.DeepEqual(r, want) {
			t.Errorf("Result() = %v, want %v", r, want)
		}
	}
}

func TestRtVecWith(t *testing.T) {
	opts := &RtOpts{
		Name:              "request_num",
		bucketNum:         10,
		timeWindowSeconds: 120,
		Help:              "Request cost",
	}
	labels := []string{"app", "version"}
	vecs := [2]*RtVec{NewRtVec(opts, labels), NewAggRtVec(opts, labels)}
	tags := map[string]string{"app": "dubbo", "version": "1.0.0"}
	for _, r := range vecs {
		first := r.With(tags)
		second := r.With(tags)
		assert.True(t, first == second) // init once
	}
}

func TestRtVecWithConcurrent(t *testing.T) {
	opts := &RtOpts{
		Name:              "request_num",
		bucketNum:         10,
		timeWindowSeconds: 120,
		Help:              "Request cost",
	}
	labels := []string{"app", "version"}
	labelValues := map[string]string{"app": "dubbo", "version": "1.0.0"}
	vecs := [2]*RtVec{NewRtVec(opts, labels), NewAggRtVec(opts, labels)}
	for _, r := range vecs {
		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				r.With(labelValues).Observe(100)
				wg.Done()
			}()
		}
		wg.Wait()
		res := r.With(labelValues).(*Rt).obs.result()
		assert.True(t, res.Count == uint64(10))
	}
}
