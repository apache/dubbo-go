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

package integration_test

import (
	"context"
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
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/filter"
	_ "dubbo.apache.org/dubbo-go/v3/filter/metrics"
	"dubbo.apache.org/dubbo-go/v3/metrics"
	metricsProm "dubbo.apache.org/dubbo-go/v3/metrics/prometheus"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

var (
	integrationRegistry    = prom.NewRegistry()
	integrationOnce        sync.Once
	integrationRun         atomic.Int32
	integrationApplication string
)

func initializeMetrics(t *testing.T) (*prom.Registry, filter.Filter) {
	t.Helper()
	reg := integrationRegistry
	integrationApplication = "failure-fixture-" + strconv.Itoa(int(integrationRun.Add(1)))
	integrationOnce.Do(func() {
		metrics.SetRegistry("failure-test", func(url *common.URL) metrics.MetricRegistry {
			return metricsProm.NewPromMetricRegistry(reg, url)
		})
		metricURL, _ := common.NewURL("failure-test://127.0.0.1?rpc.enabled=true")
		metrics.Init(metricURL)
	})
	mf, ok := extension.GetFilter(constant.MetricsFilterKey)
	if !ok {
		t.Fatal("metrics filter not registered")
	}
	// Collector subscription starts asynchronously. A separate warmup identity
	// makes readiness explicit rather than relying on a scheduler sleep.
	warmURL := fixtureURL(t, "127.0.0.1:1")
	warm := &functionInvoker{BaseInvoker: base.NewBaseInvoker(warmURL),
		call: func(context.Context, base.Invocation) result.Result { return &result.RPCResult{} }}
	await(t, func() bool {
		mf.Invoke(context.Background(), warm, invocation.NewRPCInvocation("Warmup", nil, nil))
		return metricValue(t, reg, "requests_total", "Warmup") > 0
	})

	return reg, mf
}

type functionInvoker struct {
	*base.BaseInvoker
	call func(context.Context, base.Invocation) result.Result
}

func (i *functionInvoker) Invoke(ctx context.Context, inv base.Invocation) result.Result {
	return i.call(ctx, inv)
}

func fixtureURL(t *testing.T, address string) *common.URL {
	t.Helper()
	url, err := common.NewURL("dubbo://" + address + "/fixture.Service?application=" + integrationApplication + "&interface=fixture.Service&group=fixture&version=1&timeout=300ms&registry.role=" + strconv.Itoa(common.CONSUMER))
	if err != nil {
		t.Fatal(err)
	}
	return url
}
func await(t *testing.T, ready func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if ready() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("condition not observed before deadline")
}
func metricValue(t *testing.T, reg *prom.Registry, suffix, method string) float64 {
	t.Helper()
	families, err := reg.Gather()
	if err != nil {
		t.Fatal(err)
	}
	var value float64
	for _, family := range families {
		if family.GetName() != "dubbo_consumer_"+suffix {
			continue
		}
		for _, m := range family.Metric {
			matched := false
			applicationMatched := false
			labels := make(map[string]string)
			for _, l := range m.Label {
				labels[l.GetName()] = l.GetValue()
				if l.GetName() == "method" && l.GetValue() == method {
					matched = true
				}
				if l.GetName() == "application_name" && l.GetValue() == integrationApplication {
					applicationMatched = true
				}
			}
			if matched && applicationMatched {
				if labels["interface"] != "fixture.Service" || labels["group"] != "fixture" || labels["version"] != "1" || labels["ip"] == "" {
					t.Fatalf("incorrect exported labels: %v", labels)
				}
				if len(labels) != 8 {
					t.Fatalf("unexpected new label dimensions: %v", labels)
				}
				wantType := dto.MetricType_COUNTER
				if strings.HasSuffix(suffix, "_aggregate") || suffix == "requests_processing_total" || suffix == "qps_total" {
					wantType = dto.MetricType_GAUGE
				}
				if family.GetType() != wantType {
					t.Fatalf("incorrect metric type for %s: %v", suffix, family.GetType())
				}
				if family.GetType() == dto.MetricType_COUNTER {
					value += m.GetCounter().GetValue()
				} else {
					value += m.GetGauge().GetValue()
				}
			}
		}
	}
	return value
}
func checkMetrics(t *testing.T, reg *prom.Registry, method string, total, failed, codec, network float64) {
	t.Helper()
	want := map[string]float64{
		"requests_total": total, "requests_failed_total": failed,
		"requests_failed_codec_total": codec, "requests_failed_network_total": network,
		"requests_failed_codec_total_aggregate": codec, "requests_failed_network_total_aggregate": network,
		"requests_processing_total": 0, "requests_unknown_failed_total": failed - codec - network,
	}
	await(t, func() bool {
		for suffix, expected := range want {
			if metricValue(t, reg, suffix, method) != expected {
				return false
			}
		}
		return true
	})
}
