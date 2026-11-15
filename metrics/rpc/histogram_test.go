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

package rpc

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

import (
	prom "github.com/prometheus/client_golang/prometheus"

	dto "github.com/prometheus/client_model/go"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/rpcerror"
	"dubbo.apache.org/dubbo-go/v3/metrics"
	metricsprom "dubbo.apache.org/dubbo-go/v3/metrics/prometheus"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

func TestRPCHistogramOpts(t *testing.T) {
	defaults := []float64{0.1, 0.3, 0.5, 1, 3, 5, 10}
	for _, tt := range []struct {
		name string
		raw  string
		want []float64
	}{
		{"missing", "", defaults},
		{"custom", "0.0001, 0.001,0.01", []float64{0.0001, 0.001, 0.01}},
		{"malformed", "0.1,no", defaults},
		{"empty element", "0.1,", defaults},
		{"zero", "0,1", defaults},
		{"negative", "-1,1", defaults},
		{"duplicate", "0.1,0.1", defaults},
		{"descending", "1,0.1", defaults},
		{"nan", "NaN", defaults},
		{"positive infinity", "+Inf", defaults},
		{"negative infinity", "-Inf", defaults},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, rpcHistogramOpts(tt.raw).Buckets)
		})
	}
	first := rpcHistogramOpts("")
	first.Buckets[0] = 99
	assert.Equal(t, defaults, rpcHistogramOpts("").Buckets)
}

func histogramTestURL(role int) *common.URL {
	return common.NewURLWithOptions(
		common.WithProtocol("tri"),
		common.WithPath("org.example.UserService"),
		common.WithParamsValue(constant.RegistryRoleKey, strconv.Itoa(role)),
		common.WithParamsValue(constant.ApplicationKey, "histogram-test"),
		common.WithParamsValue(constant.AppVersionKey, "app-v1"),
		common.WithParamsValue(constant.GroupKey, "test-group"),
		common.WithParamsValue(constant.VersionKey, "service-v1"),
		common.WithParamsValue(constant.HistogramEnabledKey, "true"),
	)
}

func metricFamily(t *testing.T, reg *prom.Registry, name string) *dto.MetricFamily {
	t.Helper()
	families, err := reg.Gather()
	require.NoError(t, err)
	for _, family := range families {
		if family.GetName() == name {
			return family
		}
	}
	t.Fatalf("metric family %s not found", name)
	return nil
}

func TestRPCHistogramRecordsCompletedCalls(t *testing.T) {
	for _, tt := range []struct {
		name   string
		role   int
		legacy bool
	}{
		{"provider", common.PROVIDER, false},
		{"consumer", common.CONSUMER, false},
		{"provider", common.PROVIDER, true},
		{"consumer", common.CONSUMER, true},
	} {
		t.Run(fmt.Sprintf("%s/legacy=%t", tt.name, tt.legacy), func(t *testing.T) {
			reg := prom.NewRegistry()
			url := histogramTestURL(tt.role)
			p := metricsprom.NewPromMetricRegistry(reg, url)
			var registry metrics.MetricRegistry = p
			if tt.legacy {
				registry = &histogramLegacyRegistry{MetricRegistry: p, HistogramProvider: p}
			}
			c := newRPCCollector(registry, url)
			invoker := base.NewBaseInvoker(url)
			inv := invocation.NewRPCInvocation("Get", nil, nil)
			for _, call := range []struct {
				cost time.Duration
				err  error
			}{
				{500 * time.Microsecond, nil},
				{200 * time.Millisecond, errors.New("RPC failed")},
				{11 * time.Second, nil},
			} {
				before := NewBeforeInvokeEvent(invoker, inv).(*metricsEvent)
				before.collector = c
				c.record(before)
				after := NewAfterInvokeEvent(invoker, inv, call.cost, &result.RPCResult{Err: call.err}).(*metricsEvent)
				after.collector = c
				c.record(after)
			}
			prefix := "dubbo_" + tt.name
			family := metricFamily(t, reg, prefix+"_rt_milliseconds_histogram_seconds")
			require.Equal(t, dto.MetricType_HISTOGRAM, family.GetType())
			require.Len(t, family.Metric, 1)
			m := family.Metric[0]
			labels := make(map[string]string)
			for _, label := range m.Label {
				labels[label.GetName()] = label.GetValue()
			}
			assert.Equal(t, buildLabels(url, inv), labels, "histogram must use exactly the existing RT identity")
			h := m.GetHistogram()
			assert.Equal(t, uint64(3), h.GetSampleCount())
			assert.InDelta(t, 11.2005, h.GetSampleSum(), 1e-12)
			require.Len(t, h.Bucket, 7)
			for i, bound := range []float64{0.1, 0.3, 0.5, 1, 3, 5, 10} {
				assert.InDelta(t, bound, h.Bucket[i].GetUpperBound(), 1e-12)
			}
			assert.Equal(t, uint64(1), h.Bucket[0].GetCumulativeCount())
			assert.Equal(t, uint64(2), h.Bucket[6].GetCumulativeCount())
			// The old RT still truncates milliseconds, rather than adopting new units.
			assert.InDelta(t, 11200, metricFamily(t, reg, prefix+"_rt_milliseconds_sum").Metric[0].GetGauge().GetValue(), 1e-9)
			assert.InDelta(t, 3, metricFamily(t, reg, prefix+"_requests_total").Metric[0].GetCounter().GetValue(), 1e-9)
			assert.InDelta(t, 1, metricFamily(t, reg, prefix+"_requests_failed_total").Metric[0].GetCounter().GetValue(), 1e-9)
			assert.Zero(t, metricFamily(t, reg, prefix+"_requests_processing_total").Metric[0].GetGauge().GetValue())

			text, err := p.Scrape()
			require.NoError(t, err)
			assert.Contains(t, text, `le="+Inf"} 3`)
			// Repeated scrapes without new events leave cumulative histograms intact.
			assert.Equal(t, h, metricFamily(t, reg, family.GetName()).Metric[0].GetHistogram())
		})
	}
}

// Only the original MetricRegistry methods are exposed by this wrapper.
type legacyHistogramRegistry struct {
	metrics.MetricRegistry
}

func TestRPCHistogramDisabledOrUnsupported(t *testing.T) {
	for _, mode := range []string{"missing", "disabled", "unsupported", "unsupported-cached"} {
		t.Run(mode, func(t *testing.T) {
			reg := prom.NewRegistry()
			url := histogramTestURL(common.PROVIDER)
			switch mode {
			case "missing":
				url.SetParam(constant.HistogramEnabledKey, "")
			case "disabled":
				url.SetParam(constant.HistogramEnabledKey, "false")
			}
			p := metricsprom.NewPromMetricRegistry(reg, url)
			var registry metrics.MetricRegistry = p
			if mode == "unsupported" {
				registry = &legacyHistogramRegistry{MetricRegistry: p}
			}
			if mode == "unsupported-cached" {
				registry = &gaugeOnlyHistogramRegistry{MetricRegistry: p, GaugeFuncRegistry: p}
			}
			c := newRPCCollector(registry, url)
			require.Nil(t, c.histogramRegistry)
			invoker := base.NewBaseInvoker(url)
			inv := invocation.NewRPCInvocation("Get", nil, nil)
			before := NewBeforeInvokeEvent(invoker, inv).(*metricsEvent)
			before.collector = c
			c.record(before)
			after := NewAfterInvokeEvent(invoker, inv, time.Millisecond, &result.RPCResult{}).(*metricsEvent)
			after.collector = c
			c.record(after)
			families, err := reg.Gather()
			require.NoError(t, err)
			for _, family := range families {
				assert.NotContains(t, family.GetName(), "histogram")
			}
			assert.InDelta(t, 1, metricFamily(t, reg, "dubbo_provider_requests_total").Metric[0].GetCounter().GetValue(), 1e-9)
		})
	}
}

func TestRPCHistogramCustomAndInvalidBuckets(t *testing.T) {
	for _, raw := range []string{"0.0001,0.001", "1,0.1"} {
		t.Run(raw, func(t *testing.T) {
			reg := prom.NewRegistry()
			url := histogramTestURL(common.CONSUMER)
			url.SetParam(constant.HistogramBucketsKey, raw)
			p := metricsprom.NewPromMetricRegistry(reg, url)
			c := newRPCCollector(p, url)
			invoker := base.NewBaseInvoker(url)
			for _, method := range []string{"Get", "Create"} {
				inv := invocation.NewRPCInvocation(method, nil, nil)
				before := NewBeforeInvokeEvent(invoker, inv).(*metricsEvent)
				before.collector = c
				c.record(before)
				after := NewAfterInvokeEvent(invoker, inv, 500*time.Microsecond, &result.RPCResult{}).(*metricsEvent)
				after.collector = c
				c.record(after)
			}
			family := metricFamily(t, reg, consumerRTHistogram.Name)
			require.Len(t, family.Metric, 2)
			for _, m := range family.Metric {
				if strings.HasPrefix(raw, "0.0001") {
					require.Len(t, m.GetHistogram().Bucket, 2)
					assert.Zero(t, m.GetHistogram().Bucket[0].GetCumulativeCount())
					assert.Equal(t, uint64(1), m.GetHistogram().Bucket[1].GetCumulativeCount())
				} else {
					require.Len(t, m.GetHistogram().Bucket, 7)
					assert.InDelta(t, 0.1, m.GetHistogram().Bucket[0].GetUpperBound(), 1e-12)
				}
			}
		})
	}
}

// Exposes configurable histograms without collection-time gauges.
type histogramLegacyRegistry struct {
	metrics.MetricRegistry
	metrics.HistogramProvider
}

type gaugeOnlyHistogramRegistry struct {
	metrics.MetricRegistry
	metrics.GaugeFuncRegistry
}

type countingHistogramRegistry struct {
	metrics.MetricRegistry
	metrics.GaugeFuncRegistry
	provider      metrics.HistogramProvider
	registrations atomic.Int64
}

func (r *countingHistogramRegistry) HistogramWithOptions(id *metrics.MetricId, opts *metrics.HistogramOpts) metrics.ObservableMetric {
	r.registrations.Add(1)
	return r.provider.HistogramWithOptions(id, opts)
}

func TestRPCHistogramConcurrentRecording(t *testing.T) {
	for _, role := range []int{common.PROVIDER, common.CONSUMER} {
		t.Run(strconv.Itoa(role), func(t *testing.T) {
			reg := prom.NewRegistry()
			url := histogramTestURL(role)
			p := metricsprom.NewPromMetricRegistry(reg, url)
			r := &countingHistogramRegistry{MetricRegistry: p, GaugeFuncRegistry: p, provider: p}
			c := newRPCCollector(r, url)
			events, identities := concurrentHistogramIdentities(c, url)
			stopGathering := startConcurrentGathering(t, reg)
			recordHistogramBurst(c, events)
			stopGathering()
			family := metricFamily(t, reg, rpcHistogramID(events[0].role, nil).Name)
			assertHistogramIdentities(t, family, identities)
			require.Equal(t, int64(16), r.registrations.Load(), "bind once per full identity")
		})
	}
}

func concurrentHistogramIdentities(c *rpcCollector, url *common.URL) ([]*metricsEvent, []map[string]string) {
	var events []*metricsEvent
	var identities []map[string]string
	for i := range 16 {
		u := histogramTestURL(common.CONSUMER)
		u.SetParam(constant.RegistryRoleKey, url.GetParam(constant.RegistryRoleKey, ""))
		u.SetParam(constant.GroupKey, strconv.Itoa(i%4))
		u.SetParam(constant.VersionKey, strconv.Itoa((i/4)%2))
		before := NewBeforeInvokeEvent(base.NewBaseInvoker(u), invocation.NewRPCInvocation("Call"+strconv.Itoa(i/8), nil, nil)).(*metricsEvent)
		before.collector = c
		events = append(events, before)
		identities = append(identities, maps.Clone(before.labels))
	}
	return events, identities
}

func recordHistogramBurst(c *rpcCollector, events []*metricsEvent) {
	start := make(chan struct{})
	var workers sync.WaitGroup
	failure := errors.New("histogram failure")
	for worker := range 32 {
		workers.Go(func() {
			<-start
			before := events[worker%len(events)]
			for i := range 64 {
				c.record(before)
				var err error
				if i%2 != 0 {
					err = failure
				}
				c.record(NewCompletionEvent(before, 500*time.Microsecond, rpcerror.Completion{Known: true, Err: err}))
			}
		})
	}
	close(start)
	workers.Wait()
}

func assertHistogramIdentities(t *testing.T, family *dto.MetricFamily, identities []map[string]string) {
	t.Helper()
	require.Len(t, family.Metric, len(identities))
	for _, sample := range family.Metric {
		labels := make(map[string]string)
		for _, pair := range sample.Label {
			labels[pair.GetName()] = pair.GetValue()
		}
		require.Contains(t, identities, labels)
		h := sample.GetHistogram()
		require.Equal(t, uint64(128), h.GetSampleCount())
		require.InDelta(t, 0.064, h.GetSampleSum(), 1e-12)
		for _, bucket := range h.Bucket {
			require.Equal(t, uint64(128), bucket.GetCumulativeCount())
		}
	}
}

func TestRPCHistogramObservationTerminals(t *testing.T) {
	for _, expire := range []bool{false, true} {
		t.Run(fmt.Sprintf("expired=%t", expire), func(t *testing.T) {
			reg := prom.NewRegistry()
			url := histogramTestURL(common.CONSUMER)
			p := metricsprom.NewPromMetricRegistry(reg, url)
			c := newRPCCollector(p, url)
			before := recordingEvent(t, c, common.CONSUMER)
			c.record(before)
			assertNoHistogramSamples(t, reg)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			_, observation := rpcerror.WithObservation(ctx)
			done := make(chan struct{})
			observation.OnComplete(func(completion rpcerror.Completion) {
				c.record(NewCompletionEvent(before, 500*time.Microsecond, completion))
				close(done)
			})
			observation.BeginAsync(ctx, time.Minute)
			observation.FinishInvoke(nil, true)
			if expire {
				cancel()
			} else {
				observation.Complete(nil)
			}
			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("observation did not terminate")
			}
			observation.Complete(nil)
			observation.Complete(errors.New("late failure"))
			assert.Zero(t, scalarSnapshot(t, reg)["dubbo_consumer_requests_processing_total"])
			if expire {
				assertNoHistogramSamples(t, reg)
				assert.Zero(t, scalarSnapshot(t, reg)["dubbo_consumer_requests_total"])
				return
			}
			h := metricFamily(t, reg, consumerRTHistogram.Name).Metric[0].GetHistogram()
			require.Equal(t, uint64(1), h.GetSampleCount())
			require.InDelta(t, 0.0005, h.GetSampleSum(), 1e-12)
		})
	}
}

func assertNoHistogramSamples(t *testing.T, reg *prom.Registry) {
	t.Helper()
	families, err := reg.Gather()
	require.NoError(t, err)
	for _, family := range families {
		require.NotContains(t, family.GetName(), "histogram")
	}
}

func TestRPCHistogramCompletionExpiryRace(t *testing.T) {
	reg := prom.NewRegistry()
	url := histogramTestURL(common.CONSUMER)
	c := newRPCCollector(metricsprom.NewPromMetricRegistry(reg, url), url)
	before := recordingEvent(t, c, common.CONSUMER)
	var completed atomic.Uint64
	for range 64 {
		raceHistogramTerminal(c, before, &completed)
	}
	assert.Zero(t, scalarSnapshot(t, reg)["dubbo_consumer_requests_processing_total"])
	families, err := reg.Gather()
	require.NoError(t, err)
	var count uint64
	var sum float64
	for _, family := range families {
		if family.GetName() == consumerRTHistogram.Name {
			count = family.Metric[0].GetHistogram().GetSampleCount()
			sum = family.Metric[0].GetHistogram().GetSampleSum()
		}
	}
	require.Equal(t, completed.Load(), count)
	require.InDelta(t, float64(completed.Load())*0.0005, sum, 1e-12)
	require.InDelta(t, float64(completed.Load()), scalarSnapshot(t, reg)["dubbo_consumer_requests_total"], 1e-9)
}

func raceHistogramTerminal(c *rpcCollector, before *metricsEvent, completed *atomic.Uint64) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, observation := rpcerror.WithObservation(ctx)
	done := make(chan struct{})
	observation.OnComplete(func(completion rpcerror.Completion) {
		if completion.Known {
			completed.Add(1)
		}
		c.record(NewCompletionEvent(before, 500*time.Microsecond, completion))
		close(done)
	})
	c.record(before)
	observation.BeginAsync(ctx, time.Minute)
	observation.FinishInvoke(nil, true)
	start := make(chan struct{})
	var contenders sync.WaitGroup
	contenders.Go(func() { <-start; cancel() })
	contenders.Go(func() { <-start; observation.Complete(nil) })
	close(start)
	contenders.Wait()
	<-done
	observation.Complete(errors.New("late result"))
}
