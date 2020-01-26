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
	"time"
)
import (
	"github.com/prometheus/client_golang/prometheus"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
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
	remoteKey    = "remote"
	localKey     = "local"

	providerKey = "provider"
	consumerKey = "consumer"
)

func init() {
	extension.SetMetricReporter(reporterName, newPrometheus())
}

// it will collect the data for Prometheus
type PrometheusReporter struct {

	// report the consumer-side's data
	consumerVec *prometheus.SummaryVec
	// report the provider-side's data
	providerVec *prometheus.SummaryVec
}

// report the duration to Prometheus
func (reporter *PrometheusReporter) Report(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation, cost time.Duration) {
	url := invoker.GetUrl()
	var sumVec *prometheus.SummaryVec
	if isProvider(url) {
		sumVec = reporter.providerVec
	} else {
		sumVec = reporter.consumerVec
	}

	sumVec.With(prometheus.Labels{
		serviceKey: url.Service(),
		groupKey:   url.GetParam(groupKey, ""),
		versionKey: url.GetParam(versionKey, ""),
		methodKey:  invocation.MethodName(),
		timeoutKey: url.GetParam(timeoutKey, ""),
		remoteKey:  invocation.AttachmentsByKey(constant.REMOTE_ADDR, ""),
		localKey:   invocation.AttachmentsByKey(constant.REMOTE_ADDR, ""),
	}).Observe(float64(cost.Nanoseconds() / constant.MsToNanoRate))
}

func isProvider(url common.URL) bool {
	side := url.GetParam(constant.ROLE_KEY, "")
	return strings.EqualFold(side, strconv.Itoa(common.PROVIDER))
}

func newPrometheus() metrics.Reporter {
	// cfg := *config.GetMetricConfig().GetPrometheusConfig()
	result := &PrometheusReporter{
		consumerVec: newSummaryVec(consumerKey),
		providerVec: newSummaryVec(providerKey),
	}
	prometheus.MustRegister(result.consumerVec, result.providerVec)
	return result
}

func newSummaryVec(side string) *prometheus.SummaryVec {

	return prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace: metrics.NameSpace,
			Help:      "this is the dubbo's metrics",
			Subsystem: side,
			Name:      serviceKey,
			Objectives: map[float64]float64{
				0.5:   0.01,
				0.75:  0.01,
				0.90:  0.005,
				0.98:  0.002,
				0.99:  0.001,
				0.999: 0.0001,
			},
		},
		[]string{serviceKey, groupKey, versionKey, methodKey, timeoutKey, remoteKey, localKey},
	)
}
