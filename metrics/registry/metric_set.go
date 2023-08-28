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

package registry

import (
	"dubbo.apache.org/dubbo-go/v3/metrics"
)

type MetricName int8

const (
	Reg MetricName = iota
	Sub
	Notify
	Directory
	ServerReg
	ServerSub
)

const (
	NumAllInc           = "numAllInc"
	NumAllDec           = "numAllDec"
	NumDisableTotal     = "numDisableTotal"
	NumToReconnectTotal = "numToReconnectTotal"
	NumValidTotal       = "numValidTotal"
)

var (
	// register metrics key
	RegisterMetricRequests        = metrics.NewMetricKey("dubbo_registry_register_requests_total", "Total Register Requests")
	RegisterMetricRequestsSucceed = metrics.NewMetricKey("dubbo_registry_register_requests_succeed_total", "Succeed Register Requests")
	RegisterMetricRequestsFailed  = metrics.NewMetricKey("dubbo_registry_register_requests_failed_total", "Failed Register Requests")

	// subscribe metrics key
	SubscribeMetricNum        = metrics.NewMetricKey("dubbo_registry_subscribe_num_total", "Total Subscribe Num")
	SubscribeMetricNumSucceed = metrics.NewMetricKey("dubbo_registry_subscribe_num_succeed_total", "Succeed Subscribe Num")
	SubscribeMetricNumFailed  = metrics.NewMetricKey("dubbo_registry_subscribe_num_failed_total", "Failed Subscribe Num")

	// directory metrics key
	DirectoryMetricNumAll         = metrics.NewMetricKey("dubbo_registry_directory_num_all", "All Directory Urls")
	DirectoryMetricNumValid       = metrics.NewMetricKey("dubbo_registry_directory_num_valid_total", "Valid Directory Urls")
	DirectoryMetricNumToReconnect = metrics.NewMetricKey("dubbo_registry_directory_num_to_reconnect_total", "ToReconnect Directory Urls")
	DirectoryMetricNumDisable     = metrics.NewMetricKey("dubbo_registry_directory_num_disable_total", "Disable Directory Urls")

	NotifyMetricRequests = metrics.NewMetricKey("dubbo_registry_notify_requests_total", "Total Notify Requests")
	NotifyMetricNumLast  = metrics.NewMetricKey("dubbo_registry_notify_num_last", "Last Notify Nums")

	// register service metrics key
	ServiceRegisterMetricRequests        = metrics.NewMetricKey("dubbo_registry_register_service_total", "Total Service-Level Register Requests")
	ServiceRegisterMetricRequestsSucceed = metrics.NewMetricKey("dubbo_registry_register_service_succeed_total", "Succeed Service-Level Register Requests")
	ServiceRegisterMetricRequestsFailed  = metrics.NewMetricKey("dubbo_registry_register_service_failed_total", "Failed Service-Level Register Requests")

	// subscribe metrics key
	ServiceSubscribeMetricNum        = metrics.NewMetricKey("dubbo_registry_subscribe_service_num_total", "Total Service-Level Subscribe Num")
	ServiceSubscribeMetricNumSucceed = metrics.NewMetricKey("dubbo_registry_subscribe_service_num_succeed_total", "Succeed Service-Level Num")
	ServiceSubscribeMetricNumFailed  = metrics.NewMetricKey("dubbo_registry_subscribe_service_num_failed_total", "Failed Service-Level Num")

	// register metrics server rt key
	RegisterServiceRt = metrics.NewMetricKey("dubbo_register_service_rt_milliseconds", "Service Register Time")

	// register metrics rt key
	RegisterRt = metrics.NewMetricKey("dubbo_register_rt_milliseconds", "Response Time")

	// notify rt key
	NotifyRt = metrics.NewMetricKey("dubbo_notify_rt_milliseconds", "Notify Time")
)
