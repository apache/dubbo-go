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

package server

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

func observabilityTestServiceOptions(filter string, internal bool) *ServiceOptions {
	enabled := true
	return &ServiceOptions{
		Service:     &global.ServiceConfig{Interface: "com.example.Service", Filter: filter, Params: map[string]string{}},
		Provider:    &global.ProviderConfig{},
		Application: &global.ApplicationConfig{},
		srvOpts: &ServerOptions{
			Metrics: &global.MetricsConfig{Enable: &enabled},
			Otel:    &global.OtelConfig{TracingConfig: &global.OtelTraceConfig{Enable: &enabled}},
		},
		internal: internal,
	}
}

func serviceFilters(svcOpts *ServiceOptions) []string {
	return strings.Split(svcOpts.getUrlMap().Get(constant.ServiceFilterKey), ",")
}

func TestGetUrlMapObservabilityFilters(t *testing.T) {
	filters := serviceFilters(observabilityTestServiceOptions("", false))
	assert.Contains(t, filters, constant.MetricsFilterKey)
	assert.Contains(t, filters, constant.OTELServerTraceKey)

	for _, configured := range []string{"", "echo,metrics,otelServerTrace,token"} {
		filters = serviceFilters(observabilityTestServiceOptions(configured, true))
		assert.NotContains(t, filters, constant.MetricsFilterKey, "an internal service is not counted, filter=%q", configured)
		assert.NotContains(t, filters, constant.OTELServerTraceKey, "an internal service is not traced, filter=%q", configured)
		assert.Contains(t, filters, constant.EchoFilterKey, "other provider filters still apply, filter=%q", configured)
	}
}

func TestInternalServicesExportWithoutObservabilityFilters(t *testing.T) {
	resetInternalProviderServicesForTest(t)
	var exportCount atomic.Int32
	registerCountingServeTestProtocols(t, &exportCount, nil, nil, nil, nil)
	internal := &InternalService{
		Name: "observability-test",
		Init: func(*ServiceOptions) (*ServiceDefinition, bool) {
			return &ServiceDefinition{
				Handler: &MockServerRPCService{},
				Opts:    []ServiceOption{WithNotRegister(), WithInterface("observability.test.Internal")},
			}, true
		},
	}
	SetProviderServices(internal)
	enabled := true
	srv, err := NewServer(
		WithServerProtocol(protocol.WithTriple(), protocol.WithPort(20999)),
		// A filter list configured for the application's services is inherited
		// by internal services too.
		WithServerFilter(constant.EchoFilterKey+","+constant.MetricsFilterKey+","+constant.OTELServerTraceKey),
		SetServerMetrics(&global.MetricsConfig{Enable: &enabled}),
		SetServerOtel(&global.OtelConfig{TracingConfig: &global.OtelTraceConfig{Enable: &enabled}}),
	)
	require.NoError(t, err)

	require.NoError(t, srv.exportInternalServices(context.Background()))
	t.Cleanup(internal.svcOpts.Unexport)
	require.EqualValues(t, 1, exportCount.Load())
	urls := internal.svcOpts.GetExportedUrls()
	require.Len(t, urls, 1)
	filters := strings.Split(urls[0].GetParam(constant.ServiceFilterKey, ""), ",")
	assert.Contains(t, filters, constant.EchoFilterKey)
	assert.NotContains(t, filters, constant.MetricsFilterKey)
	assert.NotContains(t, filters, constant.OTELServerTraceKey)
}
