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

package constant

// metrics type
const (
	MetricsRegistry     = "dubbo.metrics.registry"
	MetricsMetadata     = "dubbo.metrics.metadata"
	MetricsApp          = "dubbo.metrics.app"
	MetricsConfigCenter = "dubbo.metrics.configCenter"
	MetricsRpc          = "dubbo.metrics.rpc"
)

const (
	TagApplicationName    = "application_name"
	TagApplicationVersion = "application_version"
	TagHostname           = "hostname"
	TagIp                 = "ip"
	TagGitCommitId        = "git_commit_id"
	TagConfigCenter       = "config_center"
	TagChangeType         = "change_type"
	TagKey                = "key"
	TagPid                = "pid"
	TagInterface          = "interface"
	TagMethod             = "method"
	TagGroup              = "group"
	TagVersion            = "version"
	TagErrorCode          = "error"
)
const (
	MetricNamespace                     = "dubbo"
	ProtocolPrometheus                  = "prometheus"
	ProtocolDefault                     = ProtocolPrometheus
	AggregationCollectorKey             = "aggregation"
	AggregationDefaultBucketNum         = 10
	AggregationDefaultTimeWindowSeconds = 120
	PrometheusDefaultMetricsPath        = "/metrics"
	PrometheusDefaultMetricsPort        = "9090"
	PrometheusDefaultPushInterval       = 30
	PrometheusDefaultJobName            = "default_dubbo_job"
	MetricFilterStartTime               = "metric_filter_start_time"
)
