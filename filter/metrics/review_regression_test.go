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
	"context"
	"testing"
	"time"
)

import (
	prom "github.com/prometheus/client_golang/prometheus"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/rpcerror"
	metricapi "dubbo.apache.org/dubbo-go/v3/metrics"
	metricsProm "dubbo.apache.org/dubbo-go/v3/metrics/prometheus"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

type panicInvoker struct {
	base.Invoker
	observation *rpcerror.Observation
	completed   []rpcerror.Completion
}

func (i *panicInvoker) Invoke(ctx context.Context, _ base.Invocation) result.Result {
	i.observation = rpcerror.ObservationFromContext(ctx)
	i.observation.BeginAsync(ctx, time.Minute)
	i.observation.OnComplete(func(c rpcerror.Completion) { i.completed = append(i.completed, c) })
	panic("downstream")
}
func TestMetricsPanicClosesObservationAndProcessing(t *testing.T) {
	// An earlier legacy filter test leaves a closed best-effort subscriber.
	// Replace it before publishing this test's independent metrics series.
	metricapi.Subscribe(constant.MetricsRpc, make(chan metricapi.MetricsEvent, 10))
	defer metricapi.Unsubscribe(constant.MetricsRpc)
	registry := prom.NewRegistry()
	metricapi.SetRegistry("review-panic", func(u *common.URL) metricapi.MetricRegistry { return metricsProm.NewPromMetricRegistry(registry, u) })
	metricsURL, _ := common.NewURL("review-panic://127.0.0.1?application=panic-test")
	metricapi.Init(metricsURL)
	url, _ := common.NewURL("dubbo://127.0.0.1/Fixture?registry.role=0")
	invoker := &panicInvoker{Invoker: base.NewBaseInvoker(url)}
	func() {
		defer func() {
			if recover() != "downstream" {
				t.Error("panic changed")
			}
		}()
		(&metricsFilter{}).Invoke(context.Background(), invoker, invocation.NewRPCInvocation("Call", nil, nil))
	}()
	if len(invoker.completed) != 1 || invoker.completed[0].Known {
		t.Fatal("panic did not close unknown observation")
	}
	invoker.observation.Complete(nil)
	if len(invoker.completed) != 1 {
		t.Fatal("late async response completed twice")
	}
	families, err := registry.Gather()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, family := range families {
		if family.GetName() == "dubbo_consumer_requests_processing_total" {
			for _, metric := range family.Metric {
				found = true
				if metric.Gauge.GetValue() != 0 {
					t.Fatal("processing leaked")
				}
			}
		}
	}
	if !found {
		t.Fatal("processing gauge was not recorded")
	}
}
