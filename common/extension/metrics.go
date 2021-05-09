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

package extension

import (
	"dubbo.apache.org/dubbo-go/v3/metrics"
)

// we couldn't store the instance because the some instance may initialize before loading configuration
// so lazy initialization will be better.
var metricReporterMap = make(map[string]func() metrics.Reporter, 4)

// SetMetricReporter sets a reporter with the @name
func SetMetricReporter(name string, reporterFunc func() metrics.Reporter) {
	metricReporterMap[name] = reporterFunc
}

// GetMetricReporter finds the reporter with @name.
// if not found, it will panic.
// we should know that this method usually is called when system starts, so we should panic
func GetMetricReporter(name string) metrics.Reporter {
	reporterFunc, found := metricReporterMap[name]
	if !found {
		panic("Cannot find the reporter with name: " + name)
	}
	return reporterFunc()
}
