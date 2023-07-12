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

package metadata

import (
	"dubbo.apache.org/dubbo-go/v3/metrics"
)

type MetricName int8

const (
	MetadataPush MetricName = iota
	MetadataSub
	StoreProvider
	// PushRt
	// SubscribeRt
	// StoreProviderInterfaceRt
	SubscribeServiceRt
)

var (
	// app level
	metadataPushNum        = metrics.NewMetricKey("dubbo_metadata_push_num_total", "Total Num")
	metadataPushNumSucceed = metrics.NewMetricKey("dubbo_metadata_push_num_succeed_total", "Succeed Push Num")
	metadataPushNumFailed  = metrics.NewMetricKey("dubbo_metadata_push_num_failed_total", "Failed Push Num")
	// app level
	metadataSubNum        = metrics.NewMetricKey("dubbo_metadata_subscribe_num_total", "Total Metadata Subscribe Num")
	metadataSubNumSucceed = metrics.NewMetricKey("dubbo_metadata_subscribe_num_succeed_total", "Succeed Metadata Subscribe Num")
	metadataSubNumFailed  = metrics.NewMetricKey("dubbo_metadata_subscribe_num_failed_total", "Failed Metadata Subscribe Num")
	// app level
	pushRtSum  = metrics.NewMetricKey("dubbo_push_rt_milliseconds_sum", "Sum Response Time")
	pushRtLast = metrics.NewMetricKey("dubbo_push_rt_milliseconds_last", "Last Response Time")
	pushRtMin  = metrics.NewMetricKey("dubbo_push_rt_milliseconds_min", "Min Response Time")
	pushRtMax  = metrics.NewMetricKey("dubbo_push_rt_milliseconds_max", "Max Response Time")
	pushRtAvg  = metrics.NewMetricKey("dubbo_push_rt_milliseconds_avg", "Average Response Time")
	// app level
	subscribeRtSum  = metrics.NewMetricKey("dubbo_subscribe_rt_milliseconds_sum", "Sum Response Time")
	subscribeRtLast = metrics.NewMetricKey("dubbo_subscribe_rt_milliseconds_last", "Last Response Time")
	subscribeRtMin  = metrics.NewMetricKey("dubbo_subscribe_rt_milliseconds_min", "Min Response Time")
	subscribeRtMax  = metrics.NewMetricKey("dubbo_subscribe_rt_milliseconds_max", "Max Response Time")
	subscribeRtAvg  = metrics.NewMetricKey("dubbo_subscribe_rt_milliseconds_avg", "Average Response Time")

	/*
	   # HELP dubbo_metadata_store_provider_succeed_total Succeed Store Provider Metadata
	   # TYPE dubbo_metadata_store_provider_succeed_total gauge
	   dubbo_metadata_store_provider_succeed_total{application_name="metrics-provider",hostname="foghostdeMacBook-Pro.local",interface="org.apache.dubbo.samples.metrics.prometheus.api.DemoService2",ip="10.252.156.213",} 1.0
	   dubbo_metadata_store_provider_succeed_total{application_name="metrics-provider",hostname="foghostdeMacBook-Pro.local",interface="org.apache.dubbo.samples.metrics.prometheus.api.DemoService",ip="10.252.156.213",} 1.0
	*/
	// service level
	metadataStoreProviderFailed  = metrics.NewMetricKey("dubbo_metadata_store_provider_failed_total", "Total Failed Provider Metadata Store")
	metadataStoreProviderSucceed = metrics.NewMetricKey("dubbo_metadata_store_provider_succeed_total", "Total Succeed Provider Metadata Store")
	metadataStoreProvider        = metrics.NewMetricKey("dubbo_metadata_store_provider_total", "Total Provider Metadata Store")

	/*
	   # HELP dubbo_store_provider_interface_rt_milliseconds_avg Average Response Time
	   # TYPE dubbo_store_provider_interface_rt_milliseconds_avg gauge
	   dubbo_store_provider_interface_rt_milliseconds_avg{application_name="metrics-provider",application_version="3.2.1",git_commit_id="20de8b22ffb2a23531f6d9494a4963fcabd52561",hostname="foghostdeMacBook-Pro.local",interface="org.apache.dubbo.samples.metrics.prometheus.api.DemoService",ip="10.252.156.213",} 504.0
	   dubbo_store_provider_interface_rt_milliseconds_avg{application_name="metrics-provider",application_version="3.2.1",git_commit_id="20de8b22ffb2a23531f6d9494a4963fcabd52561",hostname="foghostdeMacBook-Pro.local",interface="org.apache.dubbo.samples.metrics.prometheus.api.DemoService2",ip="10.252.156.213",} 10837.0
	*/
	// service level
	storeProviderInterfaceRtAvg  = metrics.NewMetricKey("dubbo_store_provider_interface_rt_milliseconds_avg", "Average Store Provider Interface Time")
	storeProviderInterfaceRtLast = metrics.NewMetricKey("dubbo_store_provider_interface_rt_milliseconds_last", "Last Store Provider Interface Time")
	storeProviderInterfaceRtMax  = metrics.NewMetricKey("dubbo_store_provider_interface_rt_milliseconds_max", "Max Store Provider Interface Time")
	storeProviderInterfaceRtMin  = metrics.NewMetricKey("dubbo_store_provider_interface_rt_milliseconds_min", "Min Store Provider Interface Time")
	storeProviderInterfaceRtSum  = metrics.NewMetricKey("dubbo_store_provider_interface_rt_milliseconds_sum", "Sum Store Provider Interface Time")

	subscribeServiceRtLast = metrics.NewMetricKey("dubbo_subscribe_service_rt_milliseconds_last", "Last Subscribe Service Time")
	subscribeServiceRtMax  = metrics.NewMetricKey("dubbo_subscribe_service_rt_milliseconds_max", "Max Subscribe Service Time")
	subscribeServiceRtMin  = metrics.NewMetricKey("dubbo_subscribe_service_rt_milliseconds_min", "Min Subscribe Service Time")
	subscribeServiceRtSum  = metrics.NewMetricKey("dubbo_subscribe_service_rt_milliseconds_sum", "Sum Subscribe Service Time")
	subscribeServiceRtAvg  = metrics.NewMetricKey("dubbo_subscribe_service_rt_milliseconds_avg", "Average Subscribe Service Time")
)
