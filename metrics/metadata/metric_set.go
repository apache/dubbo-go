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

const (
	dubboMetadataPush             = "dubbo_metadata_push_num"
	dubboPushRt                   = "dubbo_push_rt_milliseconds"
	dubboMetadataSubscribe        = "dubbo_metadata_subscribe_num"
	dubboSubscribeRt              = "dubbo_subscribe_rt_milliseconds"
	dubboMetadataStoreProvider    = "dubbo_metadata_store_provider"
	dubboStoreProviderInterfaceRt = "dubbo_store_provider_interface_rt_milliseconds"
	dubboSubscribeServiceRt       = "dubbo_subscribe_service_rt_milliseconds"
)

const (
	totalSuffix  = "_total"
	succSuffix   = "_succeed_total"
	failedSuffix = "_failed_total"
	sumSuffix    = "_sum"
	lastSuffix   = "_last"
	minSuffix    = "_min"
	maxSuffix    = "_max"
	avgSuffix    = "_avg"
)

var (
	// app level
	metadataPushNum        = metrics.NewMetricKey(dubboMetadataPush+totalSuffix, "Total Num")
	metadataPushNumSucceed = metrics.NewMetricKey(dubboMetadataPush+succSuffix, "Succeed Push Num")
	metadataPushNumFailed  = metrics.NewMetricKey(dubboMetadataPush+failedSuffix, "Failed Push Num")
	// app level
	metadataSubNum        = metrics.NewMetricKey(dubboMetadataSubscribe+totalSuffix, "Total Metadata Subscribe Num")
	metadataSubNumSucceed = metrics.NewMetricKey(dubboMetadataSubscribe+succSuffix, "Succeed Metadata Subscribe Num")
	metadataSubNumFailed  = metrics.NewMetricKey(dubboMetadataSubscribe+failedSuffix, "Failed Metadata Subscribe Num")
	// app level
	pushRtSum  = metrics.NewMetricKey(dubboPushRt+sumSuffix, "Sum Response Time")
	pushRtLast = metrics.NewMetricKey(dubboPushRt+lastSuffix, "Last Response Time")
	pushRtMin  = metrics.NewMetricKey(dubboPushRt+minSuffix, "Min Response Time")
	pushRtMax  = metrics.NewMetricKey(dubboPushRt+maxSuffix, "Max Response Time")
	pushRtAvg  = metrics.NewMetricKey(dubboPushRt+avgSuffix, "Average Response Time")
	// app level
	subscribeRtSum  = metrics.NewMetricKey(dubboSubscribeRt+sumSuffix, "Sum Response Time")
	subscribeRtLast = metrics.NewMetricKey(dubboSubscribeRt+lastSuffix, "Last Response Time")
	subscribeRtMin  = metrics.NewMetricKey(dubboSubscribeRt+minSuffix, "Min Response Time")
	subscribeRtMax  = metrics.NewMetricKey(dubboSubscribeRt+maxSuffix, "Max Response Time")
	subscribeRtAvg  = metrics.NewMetricKey(dubboSubscribeRt+avgSuffix, "Average Response Time")

	/*
	   # HELP dubbo_metadata_store_provider_succeed_total Succeed Store Provider Metadata
	   # TYPE dubbo_metadata_store_provider_succeed_total gauge
	   dubbo_metadata_store_provider_succeed_total{application_name="metrics-provider",hostname="localhost",interface="org.apache.dubbo.samples.metrics.prometheus.api.DemoService2",ip="10.252.156.213",} 1.0
	   dubbo_metadata_store_provider_succeed_total{application_name="metrics-provider",hostname="localhost",interface="org.apache.dubbo.samples.metrics.prometheus.api.DemoService",ip="10.252.156.213",} 1.0
	*/
	// service level
	metadataStoreProviderFailed  = metrics.NewMetricKey(dubboMetadataStoreProvider+failedSuffix, "Total Failed Provider Metadata Store")
	metadataStoreProviderSucceed = metrics.NewMetricKey(dubboMetadataStoreProvider+succSuffix, "Total Succeed Provider Metadata Store")
	metadataStoreProvider        = metrics.NewMetricKey(dubboMetadataStoreProvider+totalSuffix, "Total Provider Metadata Store")

	/*
	   # HELP dubbo_store_provider_interface_rt_milliseconds_avg Average Response Time
	   # TYPE dubbo_store_provider_interface_rt_milliseconds_avg gauge
	   dubbo_store_provider_interface_rt_milliseconds_avg{application_name="metrics-provider",application_version="3.2.1",git_commit_id="20de8b22ffb2a23531f6d9494a4963fcabd52561",hostname="localhost",interface="org.apache.dubbo.samples.metrics.prometheus.api.DemoService",ip="10.252.156.213",} 504.0
	   dubbo_store_provider_interface_rt_milliseconds_avg{application_name="metrics-provider",application_version="3.2.1",git_commit_id="20de8b22ffb2a23531f6d9494a4963fcabd52561",hostname="localhost",interface="org.apache.dubbo.samples.metrics.prometheus.api.DemoService2",ip="10.252.156.213",} 10837.0
	*/
	// service level
	storeProviderInterfaceRtAvg  = metrics.NewMetricKey(dubboStoreProviderInterfaceRt+avgSuffix, "Average Store Provider Interface Time")
	storeProviderInterfaceRtLast = metrics.NewMetricKey(dubboStoreProviderInterfaceRt+lastSuffix, "Last Store Provider Interface Time")
	storeProviderInterfaceRtMax  = metrics.NewMetricKey(dubboStoreProviderInterfaceRt+maxSuffix, "Max Store Provider Interface Time")
	storeProviderInterfaceRtMin  = metrics.NewMetricKey(dubboStoreProviderInterfaceRt+minSuffix, "Min Store Provider Interface Time")
	storeProviderInterfaceRtSum  = metrics.NewMetricKey(dubboStoreProviderInterfaceRt+sumSuffix, "Sum Store Provider Interface Time")

	subscribeServiceRtLast = metrics.NewMetricKey(dubboSubscribeServiceRt+lastSuffix, "Last Subscribe Service Time")
	subscribeServiceRtMax  = metrics.NewMetricKey(dubboSubscribeServiceRt+maxSuffix, "Max Subscribe Service Time")
	subscribeServiceRtMin  = metrics.NewMetricKey(dubboSubscribeServiceRt+minSuffix, "Min Subscribe Service Time")
	subscribeServiceRtSum  = metrics.NewMetricKey(dubboSubscribeServiceRt+sumSuffix, "Sum Subscribe Service Time")
	subscribeServiceRtAvg  = metrics.NewMetricKey(dubboSubscribeServiceRt+avgSuffix, "Average Subscribe Service Time")
)
