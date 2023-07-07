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

package app_info

import (
	"dubbo.apache.org/dubbo-go/v3/metrics"
	"dubbo.apache.org/dubbo-go/v3/metrics/common"
)

/*
 * # HELP dubbo_application_info_total Total Application Info
 * # TYPE dubbo_application_info_total counter
 * dubbo_application_info_total{application_name="metrics-provider",application_version="3.2.1",git_commit_id="20de8b22ffb2a23531f6d9494a4963fcabd52561",hostname="localhost",ip="127.0.0.1",} 1.0
 */
var info = common.NewMetricKey("dubbo_application_info_total", "Total Application Info") // Total Application Info	include application name、version etc

func init() {
	metrics.AddCollector("application_info", func(mr metrics.MetricRegistry, config *metrics.ReporterConfig) {
		mr.Counter(&metrics.MetricId{Name: info.Name, Desc: info.Desc, Tags: common.NewApplicationLevel().Tags()}).Inc()
	})
}
