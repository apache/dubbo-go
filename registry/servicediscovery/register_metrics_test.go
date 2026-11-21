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

package servicediscovery

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/metrics"
	metricsRegistry "dubbo.apache.org/dubbo-go/v3/metrics/registry"
)

// registerMetricsFixture builds a service discovery registry with one
// exported provider URL and subscribes to registry metrics events.
func registerMetricsFixture(t *testing.T) (*serviceDiscoveryRegistry, *mockServiceDiscovery, *mockServiceNameMapping, *common.URL, chan metrics.MetricsEvent) {
	t.Helper()
	mockSD, mockMapping := setupEnvironment(t)
	regID := fmt.Sprintf("mock-reg-%s-%d", t.Name(), time.Now().UnixNano())
	registryURL, err := common.NewURL(testRegistryURL,
		common.WithParamsValue(constant.RegistryKey, "mock"),
		common.WithParamsValue(constant.RegistryIdKey, regID))
	require.NoError(t, err)
	reg, err := newServiceDiscoveryRegistry(registryURL)
	require.NoError(t, err)
	providerURL, err := common.NewURL("dubbo://127.0.0.1:20880/",
		common.WithParamsValue(constant.ApplicationKey, testApp),
		common.WithInterface(testInterface),
		common.WithParamsValue(constant.SideKey, constant.SideProvider),
	)
	require.NoError(t, err)

	ch := make(chan metrics.MetricsEvent, 10)
	metrics.Subscribe(constant.MetricsRegistry, ch)
	t.Cleanup(func() { metrics.Unsubscribe(constant.MetricsRegistry) })
	return reg.(*serviceDiscoveryRegistry), mockSD, mockMapping, providerURL, ch
}

func drainRegistryEvents(ch chan metrics.MetricsEvent) []*metricsRegistry.RegistryMetricsEvent {
	var events []*metricsRegistry.RegistryMetricsEvent
	for {
		select {
		case e := <-ch:
			if re, ok := e.(*metricsRegistry.RegistryMetricsEvent); ok {
				events = append(events, re)
			}
		default:
			return events
		}
	}
}

func eventsNamed(events []*metricsRegistry.RegistryMetricsEvent, name metricsRegistry.MetricName) []*metricsRegistry.RegistryMetricsEvent {
	var out []*metricsRegistry.RegistryMetricsEvent
	for _, e := range events {
		if e.Name == name {
			out = append(out, e)
		}
	}
	return out
}

// TestRegisterServicePublishesInstanceRegisterMetrics covers the
// application-level path: instance registration must be counted as a
// register request, like Dubbo Java's AbstractServiceDiscovery#register.
func TestRegisterServicePublishesInstanceRegisterMetrics(t *testing.T) {
	sdReg, mockSD, _, providerURL, ch := registerMetricsFixture(t)
	require.NoError(t, sdReg.Register(providerURL))
	drainRegistryEvents(ch)

	require.NoError(t, sdReg.RegisterService())
	require.True(t, mockSD.registerCalled)

	reg := eventsNamed(drainRegistryEvents(ch), metricsRegistry.Reg)
	require.Len(t, reg, 1)
	assert.True(t, reg[0].Succ)
	assert.False(t, reg[0].End.Before(reg[0].Start))
}

func TestRegisterServicePublishesFailedInstanceRegisterMetrics(t *testing.T) {
	sdReg, mockSD, _, providerURL, ch := registerMetricsFixture(t)
	require.NoError(t, sdReg.Register(providerURL))
	drainRegistryEvents(ch)
	mockSD.registerErr = errors.New("registry unavailable")

	require.Error(t, sdReg.RegisterService())

	reg := eventsNamed(drainRegistryEvents(ch), metricsRegistry.Reg)
	require.Len(t, reg, 1)
	assert.False(t, reg[0].Succ)
}

// TestRegisterPublishesServiceRegisterMetricsWithMappingResult checks that
// the service-level event reflects the mapping result instead of always
// reporting success.
func TestRegisterPublishesServiceRegisterMetricsWithMappingResult(t *testing.T) {
	sdReg, _, mockMapping, providerURL, ch := registerMetricsFixture(t)
	require.NoError(t, sdReg.Register(providerURL))
	serverReg := eventsNamed(drainRegistryEvents(ch), metricsRegistry.ServerReg)
	require.Len(t, serverReg, 1)
	assert.True(t, serverReg[0].Succ)

	mockMapping.mapErr = errors.New("mapping unavailable")
	require.Error(t, sdReg.Register(providerURL))
	serverReg = eventsNamed(drainRegistryEvents(ch), metricsRegistry.ServerReg)
	require.Len(t, serverReg, 1)
	assert.False(t, serverReg[0].Succ)
}
