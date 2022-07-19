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
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

import (
	ocprom "contrib.go.opencensus.io/exporter/prometheus"

	"github.com/dubbogo/gost/log/logger"

	"github.com/prometheus/client_golang/prometheus"
	prom "github.com/prometheus/client_golang/prometheus"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/metrics"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

const (
	reporterName = "prometheus"
	serviceKey   = constant.ServiceKey
	groupKey     = constant.GroupKey
	versionKey   = constant.VersionKey
	methodKey    = constant.MethodKey
	timeoutKey   = constant.TimeoutKey

	// to identify side
	providerPrefix = "provider_"
	consumerPrefix = "consumer_"

	// to identify the metric's type
	rtSuffix = "_rt"
	// to identify the metric's type
	tpsSuffix = "_tps"
)

var (
	labelNames             = []string{serviceKey, groupKey, versionKey, methodKey, timeoutKey}
	reporterInstance       *PrometheusReporter
	reporterInitOnce       sync.Once
	defaultHistogramBucket = []float64{10, 50, 100, 200, 500, 1000, 10000}
)

// should initialize after loading configuration
func init() {
	//newPrometheusReporter()
	extension.SetMetricReporter(reporterName, newPrometheusReporter)
}

// PrometheusReporter will collect the data for Prometheus
// if you want to use this feature, you need to initialize your prometheus.
// https://prometheus.io/docs/guides/go-application/
type PrometheusReporter struct {
	reporterConfig *metrics.ReporterConfig
	// report the consumer-side's rt gauge data
	consumerRTSummaryVec *prometheus.SummaryVec
	// report the provider-side's rt gauge data
	providerRTSummaryVec *prometheus.SummaryVec
	// todo tps support
	// report the consumer-side's tps gauge data
	consumerTPSGaugeVec *prometheus.GaugeVec
	// report the provider-side's tps gauge data
	providerTPSGaugeVec *prometheus.GaugeVec

	userGauge      sync.Map
	userSummary    sync.Map
	userCounter    sync.Map
	userCounterVec sync.Map
	userGaugeVec   sync.Map
	userSummaryVec sync.Map

	namespace string
}

// Report reports the duration to Prometheus
// the role in url must be consumer or provider
// or it will be ignored
func (reporter *PrometheusReporter) Report(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation, cost time.Duration, res protocol.Result) {
	url := invoker.GetURL()
	var rtVec *prometheus.SummaryVec
	if isProvider(url) {
		rtVec = reporter.providerRTSummaryVec
	} else if isConsumer(url) {
		rtVec = reporter.consumerRTSummaryVec
	} else {
		logger.Warnf("The url belongs neither the consumer nor the provider, "+
			"so the invocation will be ignored. url: %s", url.String())
		return
	}

	labels := prometheus.Labels{
		serviceKey: url.Service(),
		groupKey:   url.GetParam(groupKey, ""),
		versionKey: url.GetParam(constant.AppVersionKey, ""),
		methodKey:  invocation.MethodName(),
		timeoutKey: url.GetParam(timeoutKey, ""),
	}
	costMs := cost.Nanoseconds()
	rtVec.With(labels).Observe(float64(costMs))
}

func newHistogramVec(name, namespace string, labels []string) *prometheus.HistogramVec {
	return prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      name,
			Buckets:   defaultHistogramBucket,
		},
		labels)
}

func newCounter(name, namespace string) prometheus.Counter {
	return prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      name,
		})
}

func newCounterVec(name, namespace string, labels []string) *prometheus.CounterVec {
	return prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name:      name,
			Namespace: namespace,
		}, labels)
}

func newGauge(name, namespace string) prometheus.Gauge {
	return prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:      name,
			Namespace: namespace,
		})
}

func newGaugeVec(name, namespace string, labels []string) *prometheus.GaugeVec {
	return prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      name,
			Namespace: namespace,
		}, labels)
}

func newSummary(name, namespace string) prometheus.Summary {
	return prometheus.NewSummary(
		prometheus.SummaryOpts{
			Name:      name,
			Namespace: namespace,
		})
}

// newSummaryVec create SummaryVec, the Namespace is dubbo
// the objectives is from my experience.
func newSummaryVec(name, namespace string, labels []string, maxAge int64) *prometheus.SummaryVec {
	return prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace: namespace,
			Name:      name,
			Objectives: map[float64]float64{
				0.5:   0.01,
				0.75:  0.01,
				0.90:  0.005,
				0.98:  0.002,
				0.99:  0.001,
				0.999: 0.0001,
			},
			MaxAge: time.Duration(maxAge),
		},
		labels,
	)
}

// isProvider shows whether this url represents the application received the request as server
func isProvider(url *common.URL) bool {
	role := url.GetParam(constant.RegistryRoleKey, "")
	return strings.EqualFold(role, strconv.Itoa(common.PROVIDER))
}

// isConsumer shows whether this url represents the application sent then request as client
func isConsumer(url *common.URL) bool {
	role := url.GetParam(constant.RegistryRoleKey, "")
	return strings.EqualFold(role, strconv.Itoa(common.CONSUMER))
}

// newPrometheusReporter create new prometheusReporter
// it will register the metrics into prometheus
func newPrometheusReporter(reporterConfig *metrics.ReporterConfig) metrics.Reporter {
	if reporterInstance == nil {
		reporterInitOnce.Do(func() {
			reporterInstance = &PrometheusReporter{
				reporterConfig:       reporterConfig,
				namespace:            reporterConfig.Namespace,
				consumerRTSummaryVec: newSummaryVec(consumerPrefix+serviceKey+rtSuffix, reporterConfig.Namespace, labelNames, reporterConfig.SummaryMaxAge),
				providerRTSummaryVec: newSummaryVec(providerPrefix+serviceKey+rtSuffix, reporterConfig.Namespace, labelNames, reporterConfig.SummaryMaxAge),
			}
			prom.DefaultRegisterer.MustRegister(reporterInstance.consumerRTSummaryVec, reporterInstance.providerRTSummaryVec)
			metricsExporter, err := ocprom.NewExporter(ocprom.Options{
				Registry: prom.DefaultRegisterer.(*prom.Registry),
			})
			if err != nil {
				logger.Errorf("new prometheus reporter with error = %s", err)
				return
			}

			if reporterConfig.Enable {
				if reporterConfig.Mode == metrics.ReportModePull {
					go func() {
						mux := http.NewServeMux()
						mux.Handle(reporterConfig.Path, metricsExporter)
						if err := http.ListenAndServe(":"+reporterConfig.Port, mux); err != nil {
							logger.Warnf("new prometheus reporter with error = %s", err)
						}
					}()
				}
				// todo pushgateway support
			}
		})
	}
	return reporterInstance
}

// setGauge set gauge to target value with given label, if label is not empty, set gauge vec
// if target gauge/gaugevec not exist, just create new gauge and set the value
func (reporter *PrometheusReporter) setGauge(gaugeName string, toSetValue float64, labelMap prometheus.Labels) {
	if len(labelMap) == 0 {
		// gauge
		if val, exist := reporter.userGauge.Load(gaugeName); !exist {
			newGauge := newGauge(gaugeName, reporter.namespace)
			_ = prom.DefaultRegisterer.Register(newGauge)

			reporter.userGauge.Store(gaugeName, newGauge)
			newGauge.Set(toSetValue)
		} else {
			val.(prometheus.Gauge).Set(toSetValue)
		}
		return
	}

	// gauge vec
	if val, exist := reporter.userGaugeVec.Load(gaugeName); !exist {
		keyList := make([]string, 0)
		for k, _ := range labelMap {
			keyList = append(keyList, k)
		}
		newGaugeVec := newGaugeVec(gaugeName, reporter.namespace, keyList)
		_ = prom.DefaultRegisterer.Register(newGaugeVec)
		reporter.userGaugeVec.Store(gaugeName, newGaugeVec)
		newGaugeVec.With(labelMap).Set(toSetValue)
	} else {
		val.(*prometheus.GaugeVec).With(labelMap).Set(toSetValue)
	}
}

// incCounter inc counter to inc if label is not empty, set counter vec
// if target counter/counterVec not exist, just create new counter and inc the value
func (reporter *PrometheusReporter) incCounter(counterName string, labelMap prometheus.Labels) {
	if len(labelMap) == 0 {
		// counter
		if val, exist := reporter.userCounter.Load(counterName); !exist {
			newCounter := newCounter(counterName, reporter.namespace)
			_ = prom.DefaultRegisterer.Register(newCounter)
			reporter.userCounter.Store(counterName, newCounter)
			newCounter.Inc()
		} else {
			val.(prometheus.Counter).Inc()
		}
		return
	}

	// counter vec inc
	if val, exist := reporter.userCounterVec.Load(counterName); !exist {
		keyList := make([]string, 0)
		for k, _ := range labelMap {
			keyList = append(keyList, k)
		}
		newCounterVec := newCounterVec(counterName, reporter.namespace, keyList)
		_ = prom.DefaultRegisterer.Register(newCounterVec)
		reporter.userCounterVec.Store(counterName, newCounterVec)
		newCounterVec.With(labelMap).Inc()
	} else {
		val.(*prometheus.CounterVec).With(labelMap).Inc()
	}
}

// incSummary inc summary to target value with given label, if label is not empty, set summary vec
// if target summary/summaryVec not exist, just create new summary and set the value
func (reporter *PrometheusReporter) incSummary(summaryName string, toSetValue float64, labelMap prometheus.Labels) {
	if len(labelMap) == 0 {
		// summary
		if val, exist := reporter.userSummary.Load(summaryName); !exist {
			newSummary := newSummary(summaryName, reporter.namespace)
			_ = prom.DefaultRegisterer.Register(newSummary)
			reporter.userSummary.Store(summaryName, newSummary)
			newSummary.Observe(toSetValue)
		} else {
			val.(prometheus.Summary).Observe(toSetValue)
		}
		return
	}

	// summary vec
	if val, exist := reporter.userSummaryVec.Load(summaryName); !exist {
		keyList := make([]string, 0)
		for k, _ := range labelMap {
			keyList = append(keyList, k)
		}
		newSummaryVec := newSummaryVec(summaryName, reporter.namespace, keyList, reporter.reporterConfig.SummaryMaxAge)
		_ = prom.DefaultRegisterer.Register(newSummaryVec)
		reporter.userSummaryVec.Store(summaryName, newSummaryVec)
		newSummaryVec.With(labelMap).Observe(toSetValue)
	} else {
		val.(*prometheus.SummaryVec).With(labelMap).Observe(toSetValue)
	}
}

func SetGaugeWithLabel(gaugeName string, val float64, label prometheus.Labels) {
	reporterInstance.setGauge(gaugeName, val, label)
}

func SetGauge(gaugeName string, val float64) {
	reporterInstance.setGauge(gaugeName, val, make(prometheus.Labels))
}

func IncCounterWithLabel(counterName string, label prometheus.Labels) {
	reporterInstance.incCounter(counterName, label)
}

func IncCounter(summaryName string) {
	reporterInstance.incCounter(summaryName, make(prometheus.Labels))
}

func IncSummaryWithLabel(counterName string, val float64, label prometheus.Labels) {
	reporterInstance.incSummary(counterName, val, label)
}

func IncSummary(summaryName string, val float64) {
	reporterInstance.incSummary(summaryName, val, make(prometheus.Labels))
}
