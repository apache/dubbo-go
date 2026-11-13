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
	_ "dubbo.apache.org/dubbo-go/v3/filter/metrics" // Register the metrics filter used by this test.
	"dubbo.apache.org/dubbo-go/v3/metrics"
	metricsprom "dubbo.apache.org/dubbo-go/v3/metrics/prometheus"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
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
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestFilterToRPCHistogramHTTP$", "-test.count=1")
		cmd.Env = append(os.Environ(), child+"=1")
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "%s", output)
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
	t.Cleanup(func() { metrics.Unsubscribe(constant.MetricsRpc) })
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
			require.Equal(t, err, res.Error())
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
