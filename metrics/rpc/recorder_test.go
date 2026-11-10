/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements. See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License. You may obtain a copy of the License at
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
	"errors"
	"maps"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

import (
	prom "github.com/prometheus/client_golang/prometheus"

	dto "github.com/prometheus/client_model/go"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/rpcerror"
	"dubbo.apache.org/dubbo-go/v3/metrics"
	metricsProm "dubbo.apache.org/dubbo-go/v3/metrics/prometheus"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	tri "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
)

func newRecordingCollector(t *testing.T) (*rpcCollector, *prom.Registry) {
	t.Helper()
	reg := prom.NewRegistry()
	url, _ := common.NewURL("prometheus://127.0.0.1")
	r := metricsProm.NewPromMetricRegistry(reg, url)
	return &rpcCollector{registry: r, metricSet: buildMetricSet(r)}, reg
}
func recordingEvent(t *testing.T, c *rpcCollector, role int) *metricsEvent {
	t.Helper()
	url, _ := common.NewURL("dubbo://127.0.0.1/Fixture?application=test&app.version=1&group=g&version=v&registry.role=" + strconv.Itoa(role))
	e := NewBeforeInvokeEvent(base.NewBaseInvoker(url), invocation.NewRPCInvocation("Call", nil, nil)).(*metricsEvent)
	e.collector = c
	return e
}

func TestConcurrentRecorderMatchesLegacyMetrics(t *testing.T) {
	for _, role := range []int{common.PROVIDER, common.CONSUMER} {
		t.Run(strconv.Itoa(role), func(t *testing.T) {
			c, reg := newRecordingCollector(t)
			old, oldReg := newRecordingCollector(t)
			errs := []error{nil, errors.New("unknown"), tri.NewError(tri.CodeDeadlineExceeded, errors.New("timeout")), tri.NewError(tri.CodeResourceExhausted, errors.New("limit")), tri.NewError(tri.CodeUnavailable, errors.New("unavailable")), tri.NewError(tri.CodeBizError, errors.New("business")), rpcerror.Mark(errors.New("network"), rpcerror.Network), rpcerror.Mark(errors.New("codec"), rpcerror.Codec)}
			var events []*metricsEvent
			for range 32 {
				for i := range 72 {
					before := recordingEvent(t, c, role)
					end := *before
					end.name = AfterInvoke
					end.hasResult = true
					end.costTime = 7 * time.Millisecond
					if i%9 == 8 {
						end.name = ObservationExpired
						end.hasResult = false
					} else {
						end.err = errs[i%9]
					}
					events = append(events, before, &end)
					old.beforeInvokeHandler(before)
					if end.name == ObservationExpired {
						old.observationExpiredHandler(&end)
					} else {
						old.afterInvokeHandler(&end)
					}
				}
			}
			stop := make(chan struct{})
			var gathering sync.WaitGroup
			for range 2 {
				gathering.Go(func() {
					for {
						select {
						case <-stop:
							return
						default:
							if _, err := reg.Gather(); err != nil {
								t.Error(err)
								return
							}
						}
					}
				})
			}
			var wg sync.WaitGroup
			for w := range 32 {
				wg.Go(func() {
					for i := w * 144; i < (w+1)*144; i++ {
						c.record(events[i])
					}
				})
			}
			wg.Wait()
			close(stop)
			gathering.Wait()
			got := scalarSnapshot(t, reg)
			want := scalarSnapshot(t, oldReg)
			delete(got, "dubbo_"+events[0].role+"_qps_total")
			delete(want, "dubbo_"+events[0].role+"_qps_total")
			if !maps.Equal(got, want) {
				t.Fatalf("metrics differ\ngot %v\nwant %v", got, want)
			}
		})
	}
}

func scalarSnapshot(t *testing.T, r *prom.Registry) map[string]float64 {
	t.Helper()
	families, err := r.Gather()
	if err != nil {
		t.Fatal(err)
	}
	values := map[string]float64{}
	for _, f := range families {
		for _, m := range f.Metric {
			if f.GetType() == dto.MetricType_COUNTER {
				values[f.GetName()] += m.GetCounter().GetValue()
			} else {
				values[f.GetName()] += m.GetGauge().GetValue()
			}
		}
	}
	return values
}

func TestRecorderLazyResultsAndExpiry(t *testing.T) {
	c, r := newRecordingCollector(t)
	before := recordingEvent(t, c, common.CONSUMER)
	c.record(before)
	end := *before
	end.name = ObservationExpired
	c.record(&end)
	values := scalarSnapshot(t, r)
	if len(values) != 2 || values["dubbo_consumer_requests_processing_total"] != 0 {
		t.Fatalf("expiry created result metrics: %v", values)
	}
	c.record(before)
	end.name = AfterInvoke
	end.hasResult = true
	c.record(&end)
	values = scalarSnapshot(t, r)
	if len(values) != 18 {
		t.Fatalf("success-only identity created %d scalars, want 18", len(values))
	}
	if c.getSeries(before).failed.Load() != nil {
		t.Fatal("unused failed counter bound")
	}
}

func TestRecorderUsesCompleteIdentity(t *testing.T) {
	c, r := newRecordingCollector(t)
	first := recordingEvent(t, c, common.CONSUMER)
	for _, label := range []string{"", constant.TagApplicationName, constant.TagApplicationVersion, constant.TagHostname, constant.TagIp, constant.TagInterface, constant.TagMethod, constant.TagGroup, constant.TagVersion} {
		e := *first
		e.labels = maps.Clone(first.labels)
		if label != "" {
			e.labels[label] += "-other"
		}
		c.record(&e)
		end := e
		end.name = AfterInvoke
		end.hasResult = true
		c.record(&end)
	}
	families, err := r.Gather()
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range families {
		if f.GetName() == "dubbo_consumer_requests_total" {
			if len(f.Metric) != 9 {
				t.Fatalf("collapsed identities: %d", len(f.Metric))
			}
			for _, m := range f.Metric {
				if m.GetCounter().GetValue() != 1 || len(m.Label) != 8 {
					t.Fatal("identity or labels changed")
				}
			}
			return
		}
	}
	t.Fatal("missing completed metrics")
}

func TestCompletionKeepsReadinessSnapshot(t *testing.T) {
	saved := activeCollector.Load()
	defer activeCollector.Store(saved)
	activeCollector.Store(nil)
	c, r := newRecordingCollector(t)
	before := recordingEvent(t, c, common.CONSUMER)
	before.collector = nil
	activeCollector.Store(c)
	end := NewCompletionEvent(before, time.Millisecond, rpcerror.Completion{Known: true})
	c.record(end)
	if len(scalarSnapshot(t, r)) != 0 {
		t.Fatal("pre-init observation acquired completion-only recording")
	}
}

type legacyRegistry struct{ metrics.MetricRegistry }

func TestLegacyRegistryRetainsSerializedWindowGauges(t *testing.T) {
	c, r := newRecordingCollector(t)
	c.registry = legacyRegistry{c.registry}
	c.metricSet = buildMetricSet(c.registry)
	before := recordingEvent(t, c, common.CONSUMER)
	var wg sync.WaitGroup
	for range 16 {
		wg.Go(func() {
			for range 32 {
				e := *before
				c.record(&e)
				e.name = AfterInvoke
				e.hasResult = true
				c.record(&e)
			}
		})
	}
	wg.Wait()
	v := scalarSnapshot(t, r)
	if v["dubbo_consumer_requests_total"] != 512 || v["dubbo_consumer_requests_total_aggregate"] != 512 || v["dubbo_consumer_requests_processing_total"] != 0 {
		t.Fatalf("legacy registry lost updates: %v", v)
	}
}

type panickingRegistry struct{ metrics.MetricRegistry }

func (panickingRegistry) Gauge(*metrics.MetricId) metrics.GaugeMetric {
	panic("broken custom registry")
}
func TestRecordingFailureIsIsolated(t *testing.T) {
	c, _ := newRecordingCollector(t)
	c.registry = panickingRegistry{c.registry}
	c.metricSet = buildMetricSet(c.registry)
	e := recordingEvent(t, c, common.CONSUMER)
	c.record(e) // Must recover before Publish returns to the business path.
	c.record(e)
}

type countingObservable struct {
	metrics.ObservableMetric
	count *atomic.Int64
}

func (m countingObservable) Observe(v float64) {
	m.ObservableMetric.Observe(v)
	m.count.Add(1)
}

type countingRegistry struct {
	metrics.MetricRegistry
	metrics.GaugeFuncRegistry
	observations sync.Map
}

func (r *countingRegistry) Rt(id *metrics.MetricId, opts *metrics.RtOpts) metrics.ObservableMetric {
	n, _ := r.observations.LoadOrStore(id.Name, &atomic.Int64{})
	return countingObservable{r.MetricRegistry.Rt(id, opts), n.(*atomic.Int64)}
}

func TestRecorderPreservesEveryLatencyObservation(t *testing.T) {
	c, reg := newRecordingCollector(t)
	r := &countingRegistry{MetricRegistry: c.registry, GaugeFuncRegistry: c.registry.(metrics.GaugeFuncRegistry)}
	c.registry = r
	c.metricSet = buildMetricSet(r)
	var wg sync.WaitGroup
	for range 16 {
		wg.Go(func() {
			for i := range 128 {
				before := recordingEvent(t, c, common.CONSUMER)
				c.record(before)
				end := *before
				end.name = AfterInvoke
				end.hasResult = true
				end.costTime = 7 * time.Millisecond
				switch i % 4 {
				case 0:
					end.name = ObservationExpired
				case 1:
					end.err = errors.New("failure")
				}
				c.record(&end)
			}
		})
	}
	wg.Wait()
	for _, name := range []string{"dubbo_consumer_rt_milliseconds", "dubbo_consumer_rt"} {
		n, ok := r.observations.Load(name)
		if !ok || n.(*atomic.Int64).Load() != 1536 {
			t.Errorf("%s: observations=%v, want 1536", name, n)
		}
	}
	values := scalarSnapshot(t, reg)
	if values["dubbo_consumer_requests_total"] != 1536 || values["dubbo_consumer_requests_processing_total"] != 0 {
		t.Fatalf("incorrect completion accounting: %v", values)
	}
	for name, want := range map[string]float64{"dubbo_consumer_rt_milliseconds_sum": 1536 * 7, "dubbo_consumer_rt_milliseconds_avg": 7, "dubbo_consumer_rt_avg_milliseconds_aggregate": 7, "dubbo_consumer_rt_milliseconds_p99": 7} {
		if got := values[name]; got != want {
			t.Errorf("%s: got %v want %v", name, got, want)
		}
	}
}
