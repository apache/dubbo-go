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
	"errors"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"
)

import (
	prom "github.com/prometheus/client_golang/prometheus"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/metrics"
	configMetrics "dubbo.apache.org/dubbo-go/v3/metrics/config_center"
	metadataMetrics "dubbo.apache.org/dubbo-go/v3/metrics/metadata"
	metricsProm "dubbo.apache.org/dubbo-go/v3/metrics/prometheus"
	registryMetrics "dubbo.apache.org/dubbo-go/v3/metrics/registry"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/dubbo"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

// Each subprocess exercises the real one-time Metrics.Init lifecycle.
func TestSynchronousRPCRecording(t *testing.T) {
	for _, mode := range []string{"true", "false", "late", "panic", "slow"} {
		t.Run(mode, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestSynchronousRPCFixture$")
			cmd.Env = append(os.Environ(), "DUBBO_RPC_RECORDING_FIXTURE="+mode)
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("%v\n%s", err, out)
			}
		})
	}
}

func TestSynchronousRPCFixture(t *testing.T) {
	mode, ok := os.LookupEnv("DUBBO_RPC_RECORDING_FIXTURE")
	if !ok {
		t.Skip("subprocess fixture")
	}
	fixture := newSynchronousRecordingFixture(t, mode)
	fixture.initialize(t, mode)
	if mode == "panic" || mode == "slow" {
		fixture.assertRegistryFailure(t, mode)
		return
	}
	enabled := mode != "false"
	fixture.assertFirstCall(t, enabled)
	fixture.assertBurst(t, enabled)
	if enabled {
		assertOtherCollectors(t, fixture.registry)
	}
}

type synchronousRecordingFixture struct {
	registry *prom.Registry
	filter   filter.Filter
	invoker  *functionInvoker
	url      *common.URL
}

func newSynchronousRecordingFixture(t *testing.T, mode string) *synchronousRecordingFixture {
	t.Helper()
	reg := prom.NewRegistry()
	integrationApplication = "synchronous-recording"
	metrics.SetRegistry("synchronous-test", func(url *common.URL) metrics.MetricRegistry {
		r := metricsProm.NewPromMetricRegistry(reg, url)
		if mode == "panic" || mode == "slow" {
			return &faultyRegistry{MetricRegistry: r, mode: mode}
		}
		return r
	})
	urlMode := "true"
	if mode == "false" {
		urlMode = "false"
	}
	url, _ := common.NewURL("synchronous-test://127.0.0.1?metrics.rpc.enabled=" + urlMode)
	mf, _ := extension.GetFilter(constant.MetricsFilterKey)
	invoker := &functionInvoker{BaseInvoker: base.NewBaseInvoker(fixtureURL(t, "127.0.0.1:1")), call: func(context.Context, base.Invocation) result.Result { return &result.RPCResult{} }}
	return &synchronousRecordingFixture{registry: reg, filter: mf, invoker: invoker, url: url}
}

func (f *synchronousRecordingFixture) initialize(t *testing.T, mode string) {
	t.Helper()
	if mode == "late" {
		f.invoker.call = func(context.Context, base.Invocation) result.Result { metrics.Init(f.url); return &result.RPCResult{} }
		f.filter.Invoke(context.Background(), f.invoker, invocation.NewRPCInvocation("BeforeInit", nil, nil))
		if got := metricValue(t, f.registry, "requests_total", "BeforeInit"); got != 0 {
			t.Fatalf("pre-init completion recorded: %v", got)
		}
		f.invoker.call = func(context.Context, base.Invocation) result.Result { return &result.RPCResult{} }
	} else {
		metrics.Init(f.url)
	}
	metrics.Init(f.url) // Process-wide initialization must not install a second recorder.
}

func (f *synchronousRecordingFixture) assertRegistryFailure(t *testing.T, mode string) {
	t.Helper()
	cause := errors.New("user-visible business error")
	f.invoker.call = func(context.Context, base.Invocation) result.Result { return &result.RPCResult{Err: cause} }
	start := time.Now()
	if got := f.filter.Invoke(context.Background(), f.invoker, invocation.NewRPCInvocation("RegistryFault", nil, nil)).Error(); got != cause {
		t.Fatalf("registry changed result: %v", got)
	}
	if mode == "slow" && time.Since(start) < 10*time.Millisecond {
		t.Fatal("slow synchronous registry did not exercise caller cost")
	}
	// Even an inline/late real Exchange response must reach the user's
	// callback when synchronous Registry recording panics or is slow.
	for _, early := range []bool{true, false} {
		f.assertRegistryFailureCallback(t, cause, early)
	}
}

func (f *synchronousRecordingFixture) assertRegistryFailureCallback(t *testing.T, cause error, early bool) {
	t.Helper()
	transport := &controlledTransport{sent: make(chan int64, 1)}
	if early {
		transport.earlyError = cause
	}
	callbackInvoker := dubbo.NewDubboInvoker(f.invoker.GetURL(), remoting.NewExchangeClient(f.invoker.GetURL(), transport, time.Second, false))
	called := make(chan common.CallbackResponse, 1)
	call := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("RegistryFaultCallback"), invocation.WithReply(new(string)), invocation.WithAttachment(constant.AsyncKey, "true"), invocation.WithCallBack(func(c common.CallbackResponse) { called <- c }))
	if res := f.filter.Invoke(context.Background(), callbackInvoker, call); res.Error() != nil {
		t.Fatal("registry changed callback submission", res.Error())
	}
	id := <-transport.sent
	if !early {
		(&remoting.Response{ID: id, Error: cause}).Handle()
	}
	assertCallbackCauseUnchanged(t, called)
	(&remoting.Response{ID: id, Error: cause}).Handle()
	if len(called) != 0 {
		t.Fatal("registry fault duplicated user callback")
	}
}

func (f *synchronousRecordingFixture) assertFirstCall(t *testing.T, enabled bool) {
	t.Helper()
	// Recording must be ready on the very first call after Init, without a
	// scheduler sleep or a warmup loop. Disabled RPC metrics stay absent.
	f.filter.Invoke(context.Background(), f.invoker, invocation.NewRPCInvocation("First", nil, nil))
	want := float64(0)
	if enabled {
		want = 1
	}
	if got := metricValue(t, f.registry, "requests_total", "First"); got != want {
		t.Errorf("first call: got %v want %v", got, want)
	}
}

func (f *synchronousRecordingFixture) assertBurst(t *testing.T, enabled bool) {
	t.Helper()
	// A full observer queue must not be able to interrupt built-in recording.
	observer := make(chan metrics.MetricsEvent, 1)
	metrics.Subscribe(constant.MetricsRpc, observer)
	defer metrics.Unsubscribe(constant.MetricsRpc)
	const workers = 32
	const perWorker = 128
	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() { f.invokeBurstCalls(t, perWorker) })
	}
	wg.Wait()
	want := float64(workers * perWorker)
	if !enabled {
		want = 0
	}
	// No sleep or queue drain: recording is complete when Invoke returns.
	for _, suffix := range []string{"requests_total", "requests_succeed_total", "requests_total_aggregate", "requests_succeed_total_aggregate"} {
		if got := metricValue(t, f.registry, suffix, "Burst"); got != want {
			t.Errorf("%s: got %v want %v", suffix, got, want)
		}
	}
	if got := metricValue(t, f.registry, "requests_processing_total", "Burst"); got != 0 {
		t.Errorf("processing=%v", got)
	}
	if len(observer) != 1 {
		t.Error("best-effort observer delivery was removed")
	}
}

func (f *synchronousRecordingFixture) invokeBurstCalls(t *testing.T, count int) {
	t.Helper()
	for range count {
		res := f.filter.Invoke(context.Background(), f.invoker, invocation.NewRPCInvocation("Burst", nil, nil))
		if res.Error() != nil {
			t.Error(res.Error())
		}
	}
}

func assertOtherCollectors(t *testing.T, reg *prom.Registry) {
	t.Helper()
	e := metadataMetrics.NewMetadataMetricTimeEvent(metadataMetrics.MetadataPush)
	e.Succ = true
	e.End = time.Now()
	metrics.Publish(e)
	metrics.Publish(registryMetrics.NewRegisterEvent(true, time.Now()))
	metrics.Publish(configMetrics.NewIncMetricEvent("fixture", "group", remoting.EventTypeUpdate, configMetrics.Nacos))
	await(t, func() bool { return otherCollectorsRecorded(t, reg) })
}

func otherCollectorsRecorded(t *testing.T, reg *prom.Registry) bool {
	t.Helper()
	families, err := reg.Gather()
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	for _, f := range families {
		for _, m := range f.Metric {
			if m.Counter != nil && m.Counter.GetValue() > 0 {
				found[f.GetName()] = true
			}
		}
	}
	metadataOK, registryOK := false, false
	for name := range found {
		metadataOK = metadataOK || strings.HasPrefix(name, "dubbo_metadata_")
		registryOK = registryOK || strings.HasPrefix(name, "dubbo_registry_")
	}
	return metadataOK && registryOK && found["dubbo_configcenter_total"]
}

type faultyRegistry struct {
	metrics.MetricRegistry
	mode string
}

func (r *faultyRegistry) Gauge(id *metrics.MetricId) metrics.GaugeMetric {
	if strings.HasPrefix(id.Name, "dubbo_consumer_") {
		if r.mode == "panic" {
			panic("diagnostic Registry failure")
		}
		time.Sleep(10 * time.Millisecond)
	}
	return r.MetricRegistry.Gauge(id)
}
