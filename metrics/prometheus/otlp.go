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
	"errors"
	"fmt"
	"time"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"github.com/google/uuid"

	prom "github.com/prometheus/client_golang/prometheus"

	prombridge "go.opentelemetry.io/contrib/bridges/prometheus"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"

	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
)

// newOtlpMeterProvider builds a MeterProvider that periodically exports
// everything gathered from gatherer over OTLP. The metrics keep the names and
// labels they have on the Prometheus endpoint.
func newOtlpMeterProvider(ctx context.Context, url *common.URL, gatherer prom.Gatherer) (*sdkmetric.MeterProvider, error) {
	endpoint := url.GetParam(constant.OtlpMetricsEndpointKey, "")
	if endpoint == "" {
		return nil, errors.New("metrics.otlp.endpoint is required when the OTLP metrics exporter is enabled")
	}
	insecure := url.GetParamBool(constant.OtlpMetricsInsecureKey, false)

	var exporter sdkmetric.Exporter
	var err error
	switch protocol := url.GetParam(constant.OtlpMetricsProtocolKey, constant.OtlpMetricsProtocolHTTP); protocol {
	case constant.OtlpMetricsProtocolHTTP:
		opts := []otlpmetrichttp.Option{otlpmetrichttp.WithEndpoint(endpoint)}
		if insecure {
			opts = append(opts, otlpmetrichttp.WithInsecure())
		}
		exporter, err = otlpmetrichttp.New(ctx, opts...)
	case constant.OtlpMetricsProtocolGrpc:
		opts := []otlpmetricgrpc.Option{otlpmetricgrpc.WithEndpoint(endpoint)}
		if insecure {
			opts = append(opts, otlpmetricgrpc.WithInsecure())
		}
		exporter, err = otlpmetricgrpc.New(ctx, opts...)
	default:
		return nil, fmt.Errorf("unsupported metrics.otlp.protocol %q, want %q or %q",
			protocol, constant.OtlpMetricsProtocolHTTP, constant.OtlpMetricsProtocolGrpc)
	}
	if err != nil {
		return nil, fmt.Errorf("create OTLP metrics exporter: %w", err)
	}

	interval := url.GetParamByIntValue(constant.OtlpMetricsExportIntervalKey, constant.OtlpMetricsDefaultExportInterval)
	if interval <= 0 {
		interval = constant.OtlpMetricsDefaultExportInterval
	}
	reader := sdkmetric.NewPeriodicReader(exporter,
		sdkmetric.WithInterval(time.Duration(interval)*time.Second),
		sdkmetric.WithProducer(prombridge.NewMetricProducer(prombridge.WithGatherer(gatherer))))
	return sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(reader),
		sdkmetric.WithResource(otlpResource(url)),
	), nil
}

// otlpResource identifies the exporting process. Unlike a Prometheus scrape,
// OTLP has no target to tell replicas apart, so each process gets its own
// service.instance.id unless OTEL_RESOURCE_ATTRIBUTES sets one; otherwise the
// unlabeled runtime metrics of replicas would merge into one series.
func otlpResource(url *common.URL) *resource.Resource {
	env := resource.Environment()
	attrs := []attribute.KeyValue{
		semconv.ServiceName(url.GetParam(constant.ApplicationKey, "")),
		semconv.ServiceVersion(url.GetParam(constant.AppVersionKey, "")),
	}
	if _, ok := env.Set().Value(semconv.ServiceInstanceIDKey); !ok {
		attrs = append(attrs, semconv.ServiceInstanceID(uuid.NewString()))
	}
	// Later attributes win, as in the SDK's merge with the environment.
	return resource.NewSchemaless(append(env.Attributes(), attrs...)...)
}

// exportOtlp starts the periodic OTLP export and stops it on shutdown, after a
// final export.
func (p *promMetricRegistry) exportOtlp() {
	provider, err := newOtlpMeterProvider(context.Background(), p.url, p.gather)
	if err != nil {
		logger.Errorf("[Metrics][OTLP] OTLP metrics exporter not started, err=%v", err)
		return
	}
	extension.AddCustomShutdownCallback(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := provider.Shutdown(ctx); err != nil {
			logger.Warnf("[Metrics][OTLP] OTLP metrics exporter shutdown failed, err=%v", err)
		}
	})
	logger.Infof("[Metrics][OTLP] exporting metrics over OTLP/%s to %s",
		p.url.GetParam(constant.OtlpMetricsProtocolKey, constant.OtlpMetricsProtocolHTTP),
		p.url.GetParam(constant.OtlpMetricsEndpointKey, ""))
}
