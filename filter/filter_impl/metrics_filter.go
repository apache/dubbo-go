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

package filter_impl

import (
	"strings"
	"time"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/filter"
	"github.com/apache/dubbo-go/metrics"
	_ "github.com/apache/dubbo-go/metrics/impl"
	"github.com/apache/dubbo-go/protocol"
)

const (
	metricsFilterName = "metrics"
	successKey        = "success"
	errorKey          = "error"

	providerSide = "provider"
	groupName    = "dubbo"
)

func init() {
	extension.SetFilter(metricsFilterName, newMetricsFilter)
}

type metricsFilter struct {
	metricManger metrics.MetricManager
}

func (mf *metricsFilter) Invoke(invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	start := time.Now()
	result := invoker.Invoke(invocation)
	end := time.Now()

	duration := end.Sub(start)

	status := successKey
	if result.Error() != nil {
		status = errorKey
	}

	// using goroutine to avoid blocking the main process
	go mf.report(invoker, invocation, duration, status)

	return result
}

func isProvider(url common.URL) bool {
	side := url.GetParam(constant.SIDE_KEY, "")
	return strings.EqualFold(side, providerSide)
}

func (mf *metricsFilter) OnResponse(result protocol.Result, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	return result
}

func (mf *metricsFilter) report(invoker protocol.Invoker, invocation protocol.Invocation, duration time.Duration, result string) {
	serviceName := invoker.GetUrl().Service()
	methodName := invocation.MethodName()
	tags := make(map[string]string, 4)
	tags[constant.SERVICE_KEY] = serviceName
	tags[constant.METHOD_KEY] = methodName
	var global, method *metrics.MetricName
	if isProvider(invoker.GetUrl()) {
		global = metrics.NewMetricName(constant.DubboProvider, nil, metrics.Major)
		method = metrics.NewMetricName(constant.DubboProviderMethod, tags, metrics.Normal)
	} else {
		global = metrics.NewMetricName(constant.DubboConsumer, nil, metrics.Major)
		method = metrics.NewMetricName(constant.DubboConsumer, tags, metrics.Normal)
	}
	mf.setCompassQuantity(result, duration, global, method)
}

func (mf *metricsFilter) setCompassQuantity(result string, duration time.Duration, metricsNames ...*metrics.MetricName) {
	for _, metricName := range metricsNames {
		compass := mf.metricManger.GetFastCompass(groupName, metricName)
		compass.Record(duration, result)
	}
}

func newMetricsFilter() filter.Filter {
	return &metricsFilter{
		metricManger: extension.GetMetricManager(config.GetMetricConfig().GetMetricManagerName()),
	}
}
