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

// Package integration_test exercises the real filter, synchronous recorder,
// RPC collector and HTTP exposition in a separate test process. The invoker is
// in-process; this is not a network protocol or Kubernetes integration test.
package integration_test

import (
	"context"
	"errors"
	"net/http/httptest"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

import (
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/common/rpcerror"
	"dubbo.apache.org/dubbo-go/v3/filter"
	_ "dubbo.apache.org/dubbo-go/v3/filter/metrics" // Register the metrics filter used by this test.
	"dubbo.apache.org/dubbo-go/v3/metrics"
	metricsprom "dubbo.apache.org/dubbo-go/v3/metrics/prometheus"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/dubbo"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

type fixtureInvoker struct {
	*base.BaseInvoker
	err error
}

func (i *fixtureInvoker) Invoke(context.Context, base.Invocation) result.Result {
	return &result.RPCResult{Err: i.err}
}

func TestFilterToRPCHistogramHTTP(t *testing.T) {
	// Metrics initialization is process-global. Keep the dedicated histogram
	// registry independent of other integration fixtures and their options.
	const child = "DUBBO_HISTOGRAM_INTEGRATION_CHILD"
	if os.Getenv(child) != "1" {
		runHistogramChild(t, child, "1")
		return
	}

	reg := prom.NewRegistry()
	url := common.NewURLWithOptions(
		common.WithProtocol("histogram-fixture"),
		common.WithParamsValue(constant.HistogramEnabledKey, "true"),
		common.WithParamsValue(constant.HistogramBucketsKey, "0.001,0.01,1"),
	)
	metrics.SetRegistry(url.Protocol, func(u *common.URL) metrics.MetricRegistry {
		return metricsprom.NewPromMetricRegistry(reg, u)
	})
	metrics.Init(url)
	t.Cleanup(func() { metrics.SubscribeRecorder(constant.MetricsRpc, nil) })
	f, ok := extension.GetFilter(constant.MetricsFilterKey)
	require.True(t, ok)
	providerURL := common.NewURLWithOptions(
		common.WithPath("org.example.HistogramFixture"),
		common.WithParamsValue(constant.RegistryRoleKey, strconv.Itoa(common.PROVIDER)),
		common.WithParamsValue(constant.ApplicationKey, "histogram-fixture"),
	)
	provider := &fixtureInvoker{BaseInvoker: base.NewBaseInvoker(providerURL)}
	consumerURL := common.NewURLWithOptions(
		common.WithPath("org.example.HistogramFixture"),
		common.WithParamsValue(constant.RegistryRoleKey, strconv.Itoa(common.CONSUMER)),
		common.WithParamsValue(constant.ApplicationKey, "histogram-fixture"),
	)
	consumer := &fixtureInvoker{BaseInvoker: base.NewBaseInvoker(consumerURL)}
	for _, invoker := range []*fixtureInvoker{provider, consumer} {
		for i, err := range []error{nil, errors.New("fixture failure")} {
			invoker.err = err
			res := f.Invoke(context.Background(), invoker, invocation.NewRPCInvocation("Get", nil, nil))
			if err == nil {
				require.NoError(t, res.Error())
			} else {
				require.Same(t, err, res.Error())
			}
			side := "consumer"
			if invoker == provider {
				side = "provider"
			}
			counts, gatherErr := histogramSampleCounts(reg, "dubbo_"+side+"_rt_milliseconds_histogram_seconds")
			require.NoError(t, gatherErr)
			require.Equal(t, uint64(i+1), counts["Get"])
		}
	}
	for _, side := range []string{"provider", "consumer"} {
		counts, err := histogramSampleCounts(reg, "dubbo_"+side+"_rt_milliseconds_histogram_seconds")
		require.NoError(t, err)
		require.Equal(t, uint64(2), counts["Get"])
	}

	integrationApplication = "histogram-fixture"
	checkHistogramObserverBacklog(t, reg, f)
	for _, early := range []bool{true, false} {
		checkHistogramCallback(t, reg, f, early, 1)
	}
	checkHistogramExpiredCallback(t, reg, f)

	// Exercise the same Prometheus HTTP handler used by the exporter, without
	// opening a fixed port or starting a background server.
	response := httptest.NewRecorder()
	promhttp.HandlerFor(reg, promhttp.HandlerOpts{}).ServeHTTP(response, httptest.NewRequest("GET", "/metrics", nil))
	require.Equal(t, 200, response.Code)
	body := response.Body.String()
	for _, side := range []string{"provider", "consumer"} {
		prefix := "dubbo_" + side + "_rt_milliseconds_histogram_seconds"
		assert.Contains(t, body, "# TYPE "+prefix+" histogram")
		assert.Contains(t, body, prefix+"_bucket{")
		assert.Contains(t, body, prefix+"_sum{")
		assert.Contains(t, body, prefix+"_count{")
	}
}

func histogramSampleCounts(reg prom.Gatherer, name string) (map[string]uint64, error) {
	families, err := reg.Gather()
	if err != nil {
		return nil, err
	}
	counts := make(map[string]uint64)
	for _, family := range families {
		if family.GetName() != name {
			continue
		}
		for _, m := range family.Metric {
			for _, label := range m.Label {
				if label.GetName() == "method" {
					counts[label.GetValue()] = m.GetHistogram().GetSampleCount()
				}
			}
		}
	}
	return counts, nil
}

func runHistogramChild(t *testing.T, key, value string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	name, _, _ := strings.Cut(t.Name(), "/")
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^"+name+"$", "-test.count=1")
	cmd.Env = append(os.Environ(), key+"="+value)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", output)
}

func checkHistogramObserverBacklog(t *testing.T, reg *prom.Registry, f filter.Filter) {
	t.Helper()
	observer := make(chan metrics.MetricsEvent, 1)
	metrics.Subscribe(constant.MetricsRpc, observer)
	defer metrics.Unsubscribe(constant.MetricsRpc)
	failure := errors.New("burst failure")
	invoker := &functionInvoker{BaseInvoker: base.NewBaseInvoker(fixtureURL(t, "127.0.0.1:1")),
		call: func(_ context.Context, inv base.Invocation) result.Result {
			if inv.MethodName() == "BurstFailure" {
				return &result.RPCResult{Err: failure}
			}
			return &result.RPCResult{}
		}}
	runHistogramFilterBurst(t, f, invoker, failure)
	require.Len(t, observer, 1, "observer remains full while RPCs finish")
	assertHistogramCount(t, reg, "BurstSuccess", 512)
	assertHistogramCount(t, reg, "BurstFailure", 512)
	checkMetrics(t, reg, "BurstSuccess", 512, 0, 0, 0)
	checkMetrics(t, reg, "BurstFailure", 512, 512, 0, 0)
}

func runHistogramFilterBurst(t *testing.T, f filter.Filter, invoker base.Invoker, failure error) {
	t.Helper()
	var workers sync.WaitGroup
	for range 16 {
		workers.Go(func() {
			for range 32 {
				invokeHistogramBurstCall(t, f, invoker, "BurstSuccess", nil)
				invokeHistogramBurstCall(t, f, invoker, "BurstFailure", failure)
			}
		})
	}
	workers.Wait()
}

func invokeHistogramBurstCall(t *testing.T, f filter.Filter, invoker base.Invoker, method string, want error) {
	t.Helper()
	res := f.Invoke(context.Background(), invoker, invocation.NewRPCInvocation(method, nil, nil))
	if res.Error() != want {
		t.Error("recorder changed the business result")
	}
}

func checkHistogramCallback(t *testing.T, reg *prom.Registry, f filter.Filter, early bool, wantSamples uint64) {
	t.Helper()
	method := "HistogramLateCallback"
	if early {
		method = "HistogramEarlyCallback"
	}
	url := fixtureURL(t, "127.0.0.1:1")
	transport := &controlledTransport{sent: make(chan int64, 1)}
	cause := rpcerror.Mark(errors.New("histogram callback failure"), rpcerror.Codec)
	if early {
		transport.earlyError = cause
	}
	exchange := remoting.NewExchangeClient(url, transport, time.Second, false)
	invoker := dubbo.NewDubboInvoker(url, exchange)
	called := make(chan common.CallbackResponse, 2)
	inv := invocation.NewRPCInvocationWithOptions(
		invocation.WithMethodName(method), invocation.WithReply(new(string)),
		invocation.WithAttachment(constant.AsyncKey, "true"),
		invocation.WithCallBack(func(c common.CallbackResponse) { called <- c }))
	res := f.Invoke(context.Background(), invoker, inv)
	require.NoError(t, res.Error())
	id := <-transport.sent
	if !early {
		assertHistogramCount(t, reg, method, 0)
		require.InDelta(t, 1, metricValue(t, reg, "requests_processing_total", method), 1e-9)
		(&remoting.Response{ID: id, Error: cause}).Handle()
	}
	assertCallbackCauseUnchanged(t, called)
	assertHistogramCount(t, reg, method, wantSamples)
	checkMetrics(t, reg, method, 1, 1, 1, 0)
	(&remoting.Response{ID: id, Error: cause}).Handle()
	assertHistogramCount(t, reg, method, wantSamples)
	require.Empty(t, called, "duplicate response must not call the user again")
}

func checkHistogramExpiredCallback(t *testing.T, reg *prom.Registry, f filter.Filter) {
	t.Helper()
	const method = "HistogramExpiredCallback"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	url := fixtureURL(t, "127.0.0.1:1")
	transport := &controlledTransport{sent: make(chan int64, 1)}
	exchange := remoting.NewExchangeClient(url, transport, time.Second, false)
	called := make(chan common.CallbackResponse, 2)
	inv := invocation.NewRPCInvocationWithOptions(
		invocation.WithMethodName(method), invocation.WithReply(new(string)),
		invocation.WithAttachment(constant.AsyncKey, "true"),
		invocation.WithCallBack(func(c common.CallbackResponse) { called <- c }))
	require.NoError(t, f.Invoke(ctx, dubbo.NewDubboInvoker(url, exchange), inv).Error())
	id := <-transport.sent
	cancel()
	await(t, func() bool { return metricValue(t, reg, "requests_processing_total", method) == 0 })
	assertHistogramCount(t, reg, method, 0)
	checkMetrics(t, reg, method, 0, 0, 0, 0)
	require.NotNil(t, remoting.GetPendingResponse(remoting.SequenceType(id)))
	(&remoting.Response{ID: id, Result: &result.RPCResult{}}).Handle()
	assertCallbackCauseUnchanged(t, called)
	assertHistogramCount(t, reg, method, 0)
	checkMetrics(t, reg, method, 0, 0, 0, 0)
}

func assertHistogramCount(t *testing.T, reg *prom.Registry, method string, want uint64) {
	t.Helper()
	counts, err := histogramSampleCounts(reg, "dubbo_consumer_rt_milliseconds_histogram_seconds")
	require.NoError(t, err)
	require.Equal(t, want, counts[method], method)
}

type histogramFaultRegistry struct {
	metrics.MetricRegistry
	metrics.GaugeFuncRegistry
	observe bool
}

func (r *histogramFaultRegistry) HistogramWithOptions(*metrics.MetricId, *metrics.HistogramOpts) metrics.ObservableMetric {
	if !r.observe {
		panic("fixture histogram registration failure")
	}
	return failingHistogramObservation{}
}

type failingHistogramObservation struct{}

func (failingHistogramObservation) Observe(float64) { panic("fixture histogram observation failure") }

func TestHistogramRegistryFailurePreservesRPC(t *testing.T) {
	const key = "DUBBO_HISTOGRAM_FAILURE_CHILD"
	mode := os.Getenv(key)
	if mode == "" {
		for _, mode := range []string{"cached-register", "cached-observe", "legacy-register", "legacy-observe"} {
			t.Run(mode, func(t *testing.T) { runHistogramChild(t, key, mode) })
		}
		return
	}
	reg := prom.NewRegistry()
	url := common.NewURLWithOptions(common.WithProtocol("histogram-failure"),
		common.WithParamsValue(constant.HistogramEnabledKey, "true"))
	metrics.SetRegistry(url.Protocol, func(u *common.URL) metrics.MetricRegistry {
		p := metricsprom.NewPromMetricRegistry(reg, u)
		r := &histogramFaultRegistry{MetricRegistry: p, GaugeFuncRegistry: p, observe: strings.HasSuffix(mode, "observe")}
		if strings.HasPrefix(mode, "legacy") {
			return &struct {
				metrics.MetricRegistry
				metrics.HistogramProvider
			}{p, r}
		}
		return r
	})
	metrics.Init(url)
	t.Cleanup(func() { metrics.SubscribeRecorder(constant.MetricsRpc, nil) })
	integrationApplication = "histogram-failure"
	f, ok := extension.GetFilter(constant.MetricsFilterKey)
	require.True(t, ok)
	cause := errors.New("original business failure")
	invoker := &functionInvoker{BaseInvoker: base.NewBaseInvoker(fixtureURL(t, "127.0.0.1:1")),
		call: func(context.Context, base.Invocation) result.Result { return &result.RPCResult{Err: cause} }}
	res := f.Invoke(context.Background(), invoker, invocation.NewRPCInvocation("HistogramFault", nil, nil))
	require.Same(t, cause, res.Error())
	checkMetrics(t, reg, "HistogramFault", 1, 1, 0, 0)
	for _, early := range []bool{true, false} {
		checkHistogramCallback(t, reg, f, early, 0)
	}
}
