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
	"dubbo.apache.org/dubbo-go/v3/common/rpcerror"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

// recordAttempt records one completed attempt.
func recordAttempt(t *testing.T, c *rpcCollector, role int, retry bool) {
	t.Helper()
	before := recordingEvent(t, c, role)
	before.retry = retry
	c.record(before)
	c.record(NewCompletionEvent(before, time.Millisecond, rpcerror.Completion{Known: true}))
}

// newCollectorOnPath returns a collector on the collection-time gauge path or,
// with legacy, on the record-time path used for custom registries.
func newCollectorOnPath(t *testing.T, legacy bool) (*rpcCollector, *prom.Registry) {
	t.Helper()
	c, reg := newRecordingCollector(t)
	if legacy {
		c.registry = legacyRegistry{c.registry}
		c.metricSet = buildMetricSet(c.registry)
	}
	return c, reg
}

func gatherFamily(t *testing.T, reg *prom.Registry, name string) *dto.MetricFamily {
	t.Helper()
	families, err := reg.Gather()
	require.NoError(t, err)
	for _, f := range families {
		assert.NotEqual(t, "dubbo_consumer_retries_total_aggregate", f.GetName())
		assert.NotContains(t, f.GetName(), "provider_retries")
		if f.GetName() == name {
			return f
		}
	}
	return nil
}

func TestConsumerAttemptsCountRetries(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		t.Run(map[bool]string{false: "collection-time gauges", true: "record-time gauges"}[legacy], func(t *testing.T) {
			c, reg := newCollectorOnPath(t, legacy)
			recordAttempt(t, c, common.CONSUMER, false)
			recordAttempt(t, c, common.CONSUMER, true)
			recordAttempt(t, c, common.CONSUMER, true)
			recordAttempt(t, c, common.PROVIDER, true)

			retries := gatherFamily(t, reg, "dubbo_consumer_retries_total")
			require.NotNil(t, retries)
			assert.Equal(t, dto.MetricType_COUNTER, retries.GetType())
			require.Len(t, retries.Metric, 1)
			assert.InDelta(t, 2.0, retries.Metric[0].GetCounter().GetValue(), 1e-9)

			labels := map[string]string{}
			for _, l := range retries.Metric[0].Label {
				labels[l.GetName()] = l.GetValue()
			}
			assert.Equal(t, "test", labels["application_name"])
			assert.Equal(t, "Fixture", labels["interface"])
			assert.Equal(t, "Call", labels["method"])
		})
	}
}

// TestRetriesExportedAtZeroBeforeFirstRetry keeps requests_total - retries_total
// defined for consumers that have never retried.
func TestRetriesExportedAtZeroBeforeFirstRetry(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		t.Run(map[bool]string{false: "collection-time gauges", true: "record-time gauges"}[legacy], func(t *testing.T) {
			c, reg := newCollectorOnPath(t, legacy)
			recordAttempt(t, c, common.CONSUMER, false)

			retries := gatherFamily(t, reg, "dubbo_consumer_retries_total")
			require.NotNil(t, retries)
			require.Len(t, retries.Metric, 1)
			assert.Zero(t, retries.Metric[0].GetCounter().GetValue())
		})
	}
}

func TestBeforeInvokeEventReadsRetryAttempt(t *testing.T) {
	url, err := common.NewURL("dubbo://127.0.0.1/Fixture?registry.role=0")
	require.NoError(t, err)
	invoker, inv := base.NewBaseInvoker(url), invocation.NewRPCInvocation("Call", nil, nil)
	assert.False(t, NewBeforeInvokeEvent(invoker, inv).(*metricsEvent).retry)
	assert.False(t, NewBeforeInvokeEventWithContext(context.Background(), invoker, inv).(*metricsEvent).retry)

	ctx := base.WithRetryAttempt(context.Background(), inv)
	assert.True(t, NewBeforeInvokeEventWithContext(ctx, invoker, inv).(*metricsEvent).retry)

	child := invocation.NewRPCInvocation("Child", nil, nil)
	assert.False(t, NewBeforeInvokeEventWithContext(ctx, invoker, child).(*metricsEvent).retry,
		"an RPC started with the retried attempt's context is not a retry")
}
