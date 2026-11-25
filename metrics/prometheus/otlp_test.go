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

package prometheus

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	neturl "net/url"
	"strings"
	"sync"
	"testing"
)

import (
	prom "github.com/prometheus/client_golang/prometheus"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"

	collectormetrics "go.opentelemetry.io/proto/otlp/collector/metrics/v1"

	"google.golang.org/grpc"

	"google.golang.org/protobuf/proto"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

// otlpReceiver is a minimal OTLP/HTTP metrics receiver.
type otlpReceiver struct {
	mu       sync.Mutex
	requests []*collectormetrics.ExportMetricsServiceRequest
}

func (r *otlpReceiver) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	body, err := io.ReadAll(req.Body)
	if err != nil || req.URL.Path != "/v1/metrics" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	msg := &collectormetrics.ExportMetricsServiceRequest{}
	if err := proto.Unmarshal(body, msg); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	r.mu.Lock()
	r.requests = append(r.requests, msg)
	r.mu.Unlock()
	w.Header().Set("Content-Type", "application/x-protobuf")
	_, _ = w.Write([]byte{})
}

func otlpURL(t *testing.T, params map[string]string) *common.URL {
	t.Helper()
	values := neturl.Values{}
	for k, v := range params {
		values.Set(k, v)
	}
	u, err := common.NewURL("prometheus://localhost?" + values.Encode())
	require.NoError(t, err)
	return u
}

func TestOtlpMeterProviderExportsGatheredMetrics(t *testing.T) {
	receiver := &otlpReceiver{}
	server := httptest.NewServer(receiver)
	t.Cleanup(server.Close)

	reg := prom.NewRegistry()
	requests := prom.NewCounterVec(prom.CounterOpts{Name: "dubbo_consumer_requests_total", Help: "test"}, []string{"interface"})
	reg.MustRegister(requests)
	requests.WithLabelValues("greet.GreetService").Add(3)

	provider, err := newOtlpMeterProvider(context.Background(), otlpURL(t, map[string]string{
		constant.OtlpMetricsEndpointKey: strings.TrimPrefix(server.URL, "http://"),
		constant.OtlpMetricsInsecureKey: "true",
		constant.ApplicationKey:         "otlp-app",
	}), reg)
	require.NoError(t, err)
	require.NoError(t, provider.ForceFlush(context.Background()))
	// Shutdown exports once more, so an update after the flush still arrives.
	requests.WithLabelValues("greet.GreetService").Add(2)
	require.NoError(t, provider.Shutdown(context.Background()))

	receiver.mu.Lock()
	defer receiver.mu.Unlock()
	require.Len(t, receiver.requests, 2)
	assert.InDelta(t, 5.0, sumValue(receiver.requests[1], "dubbo_consumer_requests_total"), 1e-9)
	rm := receiver.requests[0].GetResourceMetrics()
	require.NotEmpty(t, rm)
	var serviceName string
	for _, attr := range rm[0].GetResource().GetAttributes() {
		if attr.GetKey() == "service.name" {
			serviceName = attr.GetValue().GetStringValue()
		}
	}
	assert.Equal(t, "otlp-app", serviceName)

	var found bool
	for _, sm := range rm[0].GetScopeMetrics() {
		for _, m := range sm.GetMetrics() {
			if m.GetName() != "dubbo_consumer_requests_total" {
				continue
			}
			found = true
			points := m.GetSum().GetDataPoints()
			require.Len(t, points, 1)
			assert.InDelta(t, 3.0, points[0].GetAsDouble(), 1e-9)
			require.Len(t, points[0].GetAttributes(), 1)
			assert.Equal(t, "interface", points[0].GetAttributes()[0].GetKey())
			assert.Equal(t, "greet.GreetService", points[0].GetAttributes()[0].GetValue().GetStringValue())
		}
	}
	assert.True(t, found, "the Prometheus counter is exported with the same name")
}

func sumValue(req *collectormetrics.ExportMetricsServiceRequest, name string) float64 {
	var total float64
	for _, rm := range req.GetResourceMetrics() {
		for _, sm := range rm.GetScopeMetrics() {
			for _, m := range sm.GetMetrics() {
				if m.GetName() == name {
					for _, p := range m.GetSum().GetDataPoints() {
						total += p.GetAsDouble()
					}
				}
			}
		}
	}
	return total
}

func TestOtlpResourceIdentifiesTheProcess(t *testing.T) {
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "")
	u := otlpURL(t, map[string]string{constant.ApplicationKey: "orders"})
	first, _ := otlpResource(u).Set().Value(semconv.ServiceInstanceIDKey)
	second, _ := otlpResource(u).Set().Value(semconv.ServiceInstanceIDKey)
	assert.NotEmpty(t, first.AsString())
	assert.NotEqual(t, first.AsString(), second.AsString(), "replicas of one application get distinct instance IDs")

	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "service.instance.id=orders-0")
	provided, ok := otlpResource(u).Set().Value(semconv.ServiceInstanceIDKey)
	require.True(t, ok)
	assert.Equal(t, "orders-0", provided.AsString(), "a deployment-provided instance ID is kept")
}

func TestOtlpMeterProviderConfigErrors(t *testing.T) {
	reg := prom.NewRegistry()
	_, err := newOtlpMeterProvider(context.Background(), otlpURL(t, nil), reg)
	require.ErrorContains(t, err, "metrics.otlp.endpoint")

	_, err = newOtlpMeterProvider(context.Background(), otlpURL(t, map[string]string{
		constant.OtlpMetricsEndpointKey: "127.0.0.1:4318",
		constant.OtlpMetricsProtocolKey: "udp",
	}), reg)
	require.ErrorContains(t, err, "unsupported metrics.otlp.protocol")
}

// grpcReceiver is a minimal OTLP/gRPC metrics receiver.
type grpcReceiver struct {
	collectormetrics.UnimplementedMetricsServiceServer
	mu       sync.Mutex
	received int
}

func (r *grpcReceiver) Export(_ context.Context, req *collectormetrics.ExportMetricsServiceRequest) (*collectormetrics.ExportMetricsServiceResponse, error) {
	r.mu.Lock()
	r.received += len(req.GetResourceMetrics())
	r.mu.Unlock()
	return &collectormetrics.ExportMetricsServiceResponse{}, nil
}

func TestOtlpMeterProviderGrpc(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := grpc.NewServer()
	receiver := &grpcReceiver{}
	collectormetrics.RegisterMetricsServiceServer(server, receiver)
	go func() { _ = server.Serve(lis) }()
	t.Cleanup(server.Stop)

	reg := prom.NewRegistry()
	counter := prom.NewCounter(prom.CounterOpts{Name: "dubbo_provider_requests_total", Help: "test"})
	reg.MustRegister(counter)
	counter.Inc()

	provider, err := newOtlpMeterProvider(context.Background(), otlpURL(t, map[string]string{
		constant.OtlpMetricsEndpointKey: lis.Addr().String(),
		constant.OtlpMetricsProtocolKey: constant.OtlpMetricsProtocolGrpc,
		constant.OtlpMetricsInsecureKey: "true",
	}), reg)
	require.NoError(t, err)
	require.NoError(t, provider.ForceFlush(context.Background()))
	require.NoError(t, provider.Shutdown(context.Background()))

	receiver.mu.Lock()
	defer receiver.mu.Unlock()
	assert.Positive(t, receiver.received)
}
