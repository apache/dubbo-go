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
	"strconv"
	"strings"
	"sync"
	"time"
)
import (
	"github.com/prometheus/client_golang/prometheus"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/metrics"
	"github.com/apache/dubbo-go/protocol"
)

const (
	reporterName = "prometheus"
	serviceKey   = constant.SERVICE_KEY
	groupKey     = constant.GROUP_KEY
	versionKey   = constant.VERSION_KEY
	methodKey    = constant.METHOD_KEY
	timeoutKey   = constant.TIMEOUT_KEY

	providerKey = "provider"
	consumerKey = "consumer"

	// to identify the metric's type
	histogramSuffix = "_histogram"
	// to identify the metric's type
	summarySuffix = "_summary"
)

var (
	labelNames       = []string{serviceKey, groupKey, versionKey, methodKey, timeoutKey}
	namespace        = config.GetApplicationConfig().Name
	reporterInstance *PrometheusReporter
	reporterInitOnce sync.Once
)

// should initialize after loading configuration
func init() {

	extension.SetMetricReporter(reporterName, newPrometheusReporter)
}

// PrometheusReporter will collect the data for Prometheus
// if you want to use this feature, you need to initialize your prometheus.
// https://prometheus.io/docs/guides/go-application/
type PrometheusReporter struct {

	// report the consumer-side's summary data
	consumerSummaryVec *prometheus.SummaryVec
	// report the provider-side's summary data
	providerSummaryVec *prometheus.SummaryVec

	// report the provider-side's histogram data
	providerHistogramVec *prometheus.HistogramVec
	// report the consumer-side's histogram data
	consumerHistogramVec *prometheus.HistogramVec
}

// Report reports the duration to Prometheus
// the role in url must be consumer or provider
// or it will be ignored
func (reporter *PrometheusReporter) Report(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation, cost time.Duration, res protocol.Result) {
	url := invoker.GetUrl()
	var sumVec *prometheus.SummaryVec
	var hisVec *prometheus.HistogramVec
	if isProvider(url) {
		sumVec = reporter.providerSummaryVec
		hisVec = reporter.providerHistogramVec
	} else if isConsumer(url) {
		sumVec = reporter.consumerSummaryVec
		hisVec = reporter.consumerHistogramVec
	} else {
		logger.Warnf("The url belongs neither the consumer nor the provider, "+
			"so the invocation will be ignored. url: %s", url.String())
		return
	}

	labels := prometheus.Labels{
		serviceKey: url.Service(),
		groupKey:   url.GetParam(groupKey, ""),
		versionKey: url.GetParam(versionKey, ""),
		methodKey:  invocation.MethodName(),
		timeoutKey: url.GetParam(timeoutKey, ""),
	}

	costMs := float64(cost.Nanoseconds() / constant.MsToNanoRate)
	sumVec.With(labels).Observe(costMs)
	hisVec.With(labels).Observe(costMs)
}

func newHistogramVec(side string) *prometheus.HistogramVec {
	mc := config.GetMetricConfig()
	return prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: side,
			Name:      serviceKey + histogramSuffix,
			Help:      "This is the dubbo's histogram metrics",
			Buckets:   mc.GetHistogramBucket(),
		},
		labelNames)
}

// whether this url represents the application received the request as server
func isProvider(url *common.URL) bool {
	role := url.GetParam(constant.ROLE_KEY, "")
	return strings.EqualFold(role, strconv.Itoa(common.PROVIDER))
}

// whether this url represents the application sent then request as client
func isConsumer(url *common.URL) bool {
	role := url.GetParam(constant.ROLE_KEY, "")
	return strings.EqualFold(role, strconv.Itoa(common.CONSUMER))
}

// newSummaryVec create SummaryVec, the Namespace is dubbo
// the objectives is from my experience.
func newSummaryVec(side string) *prometheus.SummaryVec {
	return prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace: namespace,
			Help:      "This is the dubbo's summary metrics",
			Subsystem: side,
			Name:      serviceKey + summarySuffix,
			Objectives: map[float64]float64{
				0.5:   0.01,
				0.75:  0.01,
				0.90:  0.005,
				0.98:  0.002,
				0.99:  0.001,
				0.999: 0.0001,
			},
		},
		labelNames,
	)
}

// newPrometheusReporter create new prometheusReporter
// it will register the metrics into prometheus
func newPrometheusReporter() metrics.Reporter {
	if reporterInstance == nil {
		reporterInitOnce.Do(func() {
			reporterInstance = &PrometheusReporter{
				consumerSummaryVec: newSummaryVec(consumerKey),
				providerSummaryVec: newSummaryVec(providerKey),

				consumerHistogramVec: newHistogramVec(consumerKey),
				providerHistogramVec: newHistogramVec(providerKey),
			}
			prometheus.MustRegister(reporterInstance.consumerSummaryVec, reporterInstance.providerSummaryVec,
				reporterInstance.consumerHistogramVec, reporterInstance.providerHistogramVec)
		})
	}
	return reporterInstance
}
