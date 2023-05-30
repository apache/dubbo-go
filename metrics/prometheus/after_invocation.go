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
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

import (
	"github.com/dubbogo/gost/log/logger"
	"github.com/prometheus/client_golang/prometheus"
)

func (reporter *PrometheusReporter) ReportAfterInvocation(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation, cost time.Duration, res protocol.Result) {
	if !reporter.reporterConfig.Enable {
		return
	}

	url := invoker.GetURL()

	var role string // provider or consumer
	if isProvider(url) {
		role = providerField
	} else if isConsumer(url) {
		role = consumerField
	} else {
		logger.Warnf("The url belongs neither the consumer nor the provider, "+
			"so the invocation will be ignored. url: %s", url.String())
		return
	}
	labels := prometheus.Labels{
		applicationNameKey: url.GetParam(constant.ApplicationKey, ""),
		groupKey:           url.Group(),
		hostnameKey:        "",
		interfaceKey:       url.Service(),
		ipKey:              common.GetLocalIp(),
		versionKey:         url.GetParam(constant.AppVersionKey, ""),
		methodKey:          invocation.MethodName(),
	}

	reporter.reportRTSummaryVec(role, &labels, cost.Milliseconds())
	reporter.reportRequestTotalCounterVec(role, &labels)
}

func (r *PrometheusReporter) reportRTSummaryVec(role string, labels *prometheus.Labels, costMs int64) {
	switch role {
	case providerField:
		r.providerRTSummaryVec.With(*labels).Observe(float64(costMs))
	case consumerField:
		r.consumerRTSummaryVec.With(*labels).Observe(float64(costMs))
	}
}

func (r *PrometheusReporter) reportRequestTotalCounterVec(role string, labels *prometheus.Labels) {
	switch role {
	case providerField:
		r.providerRequestTotalCounterVec.With(*labels).Inc()
	case consumerField:
		r.consumerRequestTotalCounterVec.With(*labels).Inc()
	}
}
