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
	"bytes"
	"context"
	"net/http"
	"sync"
	"time"
)

import (
	"github.com/dubbogo/gost/log/logger"

	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/client_golang/prometheus/push"

	"github.com/prometheus/common/expfmt"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/metrics"
)

func init() {
	metrics.SetRegistry(constant.ProtocolPrometheus, func(url *common.URL) metrics.MetricRegistry {
		return &promMetricRegistry{r: prom.DefaultRegisterer, gather: prom.DefaultGatherer, url: url}
	})
}

type promMetricRegistry struct {
	r      prom.Registerer
	gather prom.Gatherer
	vecs   sync.Map
	url    *common.URL
}

func NewPromMetricRegistry(reg *prom.Registry, url *common.URL) *promMetricRegistry {
	return &promMetricRegistry{r: reg, gather: reg, url: url}
}

func (p *promMetricRegistry) getOrComputeVec(key string, supplier func() prom.Collector) interface{} {
	v, ok := p.vecs.Load(key)
	if !ok {
		v, ok = p.vecs.LoadOrStore(key, supplier())
		if !ok {
			p.r.MustRegister(v.(prom.Collector)) // only registe collector which stored success
		}
	}
	return v
}

func (p *promMetricRegistry) Counter(m *metrics.MetricId) metrics.CounterMetric {
	vec := p.getOrComputeVec(m.Name, func() prom.Collector {
		return prom.NewCounterVec(prom.CounterOpts{
			Name: m.Name,
			Help: m.Desc,
		}, m.TagKeys())
	}).(*prom.CounterVec)
	return vec.With(m.Tags)
}

func (p *promMetricRegistry) Gauge(m *metrics.MetricId) metrics.GaugeMetric {
	vec := p.getOrComputeVec(m.Name, func() prom.Collector {
		return prom.NewGaugeVec(prom.GaugeOpts{
			Name: m.Name,
			Help: m.Desc,
		}, m.TagKeys())
	}).(*prom.GaugeVec)
	return vec.With(m.Tags)
}

func (p *promMetricRegistry) Histogram(m *metrics.MetricId) metrics.ObservableMetric {
	vec := p.getOrComputeVec(m.Name, func() prom.Collector {
		return prom.NewHistogramVec(prom.HistogramOpts{
			Name: m.Name,
			Help: m.Desc,
		}, m.TagKeys())
	}).(*prom.HistogramVec)
	return vec.With(m.Tags)
}

func (p *promMetricRegistry) Summary(m *metrics.MetricId) metrics.ObservableMetric {
	vec := p.getOrComputeVec(m.Name, func() prom.Collector {
		return prom.NewSummaryVec(prom.SummaryOpts{
			Name: m.Name,
			Help: m.Desc,
		}, m.TagKeys())
	}).(*prom.SummaryVec)
	return vec.With(m.Tags)
}

func (p *promMetricRegistry) Rt(m *metrics.MetricId, opts *metrics.RtOpts) metrics.ObservableMetric {
	key := m.Name
	var supplier func() prom.Collector
	if opts != nil && opts.Aggregate {
		key += "_aggregate"
		if opts.BucketNum == 0 {
			opts.BucketNum = p.url.GetParamByIntValue(constant.AggregationBucketNumKey, constant.AggregationDefaultBucketNum)
		}
		if opts.TimeWindowSeconds == 0 {
			opts.TimeWindowSeconds = p.url.GetParamInt(constant.AggregationTimeWindowSecondsKey, constant.AggregationDefaultTimeWindowSeconds)
		}
		supplier = func() prom.Collector {
			return NewAggRtVec(&RtOpts{
				Name:              m.Name,
				Help:              m.Desc,
				bucketNum:         opts.BucketNum,
				timeWindowSeconds: opts.TimeWindowSeconds,
			}, m.TagKeys())
		}
	} else {
		supplier = func() prom.Collector {
			return NewRtVec(&RtOpts{
				Name: m.Name,
				Help: m.Desc,
			}, m.TagKeys())
		}
	}
	vec := p.getOrComputeVec(key, supplier).(*RtVec)
	return vec.With(m.Tags)
}

func (p *promMetricRegistry) Export() {
	if p.url.GetParamBool(constant.PrometheusExporterEnabledKey, false) {
		go func() {
			mux := http.NewServeMux()
			path := p.url.GetParam(constant.PrometheusDefaultMetricsPath, constant.PrometheusDefaultMetricsPath)
			port := p.url.GetParam(constant.PrometheusExporterMetricsPortKey, constant.PrometheusDefaultMetricsPort)
			mux.Handle(path, promhttp.InstrumentMetricHandler(p.r, promhttp.HandlerFor(p.gather, promhttp.HandlerOpts{})))
			srv := &http.Server{Addr: ":" + port, Handler: mux}
			extension.AddCustomShutdownCallback(func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := srv.Shutdown(ctx); nil != err {
					logger.Fatalf("prometheus server shutdown failed, err: %v", err)
				} else {
					logger.Info("prometheus server gracefully shutdown success")
				}
			})
			logger.Infof("prometheus endpoint :%s%s", port, path)
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed { // except Shutdown or Close
				logger.Errorf("new prometheus server with error = %v", err)
			}
		}()
	}
	if p.url.GetParamBool(constant.PrometheusPushgatewayEnabledKey, false) {
		baseUrl, exist := p.url.GetNonDefaultParam(constant.PrometheusPushgatewayBaseUrlKey)
		if !exist {
			logger.Error("no pushgateway url found in config path: metrics.prometheus.pushgateway.bash-url, please check your config file")
			return
		}
		username := p.url.GetParam(constant.PrometheusPushgatewayBaseUrlKey, "")
		password := p.url.GetParam(constant.PrometheusPushgatewayBaseUrlKey, "")
		job := p.url.GetParam(constant.PrometheusPushgatewayJobKey, constant.PrometheusDefaultJobName)
		pushInterval := p.url.GetParamByIntValue(constant.PrometheusPushgatewayPushIntervalKey, constant.PrometheusDefaultPushInterval)
		pusher := push.New(baseUrl, job).Gatherer(p.gather)
		if len(username) != 0 {
			pusher.BasicAuth(username, password)
		}
		logger.Infof("prometheus pushgateway will push to %s every %d seconds", baseUrl, pushInterval)
		ticker := time.NewTicker(time.Duration(pushInterval) * time.Second)
		go func() {
			for range ticker.C {
				err := pusher.Add()
				if err != nil {
					logger.Errorf("push metric data to prometheus push gateway error", err)
				} else {
					logger.Debugf("prometheus pushgateway push to %s success", baseUrl)
				}
			}
		}()
	}
}

func (p *promMetricRegistry) Scrape() (string, error) {
	gathering, err := p.gather.Gather()
	if err != nil {
		return "", err
	}
	out := &bytes.Buffer{}
	for _, mf := range gathering {
		if _, err := expfmt.MetricFamilyToText(out, mf); err != nil {
			return "", err
		}
	}
	return out.String(), nil
}
