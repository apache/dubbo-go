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

	"github.com/apache/dubbo-go/common/logger"
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

// should initialize after loading configuration
func init() {
	rpt := &PrometheusReporter{
		consumerVec: newSummaryVec(consumerKey),
		providerVec: newSummaryVec(providerKey),
	}
	prometheus.MustRegister(rpt.consumerVec, rpt.providerVec)
	extension.SetMetricReporter(reporterName, rpt)
}

// it will collect the data for Prometheus
// if you want to use this, you should initialize your prometheus.
// https://prometheus.io/docs/guides/go-application/
type PrometheusReporter struct {

	// report the consumer-side's data
	consumerVec *prometheus.SummaryVec
	// report the provider-side's data
	providerVec *prometheus.SummaryVec
}

// report the duration to Prometheus
func (reporter *PrometheusReporter) Report(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation, cost time.Duration, res protocol.Result) {
	url := invoker.GetUrl()
	var sumVec *prometheus.SummaryVec
	if isProvider(url) {
		sumVec = reporter.providerVec
	} else if isConsumer(url) {
		sumVec = reporter.consumerVec
	} else {
		logger.Warnf("The url is not the consumer's or provider's, "+
			"so the invocation will be ignored. url: %s", url.String())
		return
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

// whether this url represents the application received the request as server
func isProvider(url common.URL) bool {
	role := url.GetParam(constant.ROLE_KEY, "")
	return strings.EqualFold(role, strconv.Itoa(common.PROVIDER))
}

// whether this url represents the application sent then request as client
func isConsumer(url common.URL) bool {
	role := url.GetParam(constant.ROLE_KEY, "")
	return strings.EqualFold(role, strconv.Itoa(common.CONSUMER))
}

// create SummaryVec, the Namespace is dubbo
// the objectives is from my experience.
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
