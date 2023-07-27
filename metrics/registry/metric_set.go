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
type DirName int8

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

const (
	dubboRegNum       = "dubbo_registry_register_metrics_num"
	dubboRegRt        = "dubbo_registry_register_metrics_rt"
	dubboRegServerNum = "dubbo_registry_register_server_metrics_num"
	dubboRegServerRt  = "dubbo_registry_register_server_metrics_rt"
	dubboNotifyRt     = "dubbo_notify_rt"
)

var (
	// register metrics key
	RegisterMetricRequests        = metrics.NewMetricKey("dubbo.registry.register.requests.total", "Total Register Requests")
	RegisterMetricRequestsSucceed = metrics.NewMetricKey("dubbo.registry.register.requests.succeed.total", "Succeed Register Requests")
	RegisterMetricRequestsFailed  = metrics.NewMetricKey("dubbo.registry.register.requests.failed.total", "Failed Register Requests")

	// subscribe metrics key
	SubscribeMetricNum        = metrics.NewMetricKey("dubbo.registry.subscribe.num.total", "Total Subscribe Num")
	SubscribeMetricNumSucceed = metrics.NewMetricKey("dubbo.registry.subscribe.num.succeed.total", "Succeed Subscribe Num")
	SubscribeMetricNumFailed  = metrics.NewMetricKey("dubbo.registry.subscribe.num.failed.total", "Failed Subscribe Num")

	// directory metrics key
	DirectoryMetricNumAll         = metrics.NewMetricKey("dubbo.registry.directory.num.all", "All Directory Urls")
	DirectoryMetricNumValid       = metrics.NewMetricKey("dubbo.registry.directory.num.valid.total", "Valid Directory Urls")
	DirectoryMetricNumToReconnect = metrics.NewMetricKey("dubbo.registry.directory.num.to_reconnect.total", "ToReconnect Directory Urls")
	DirectoryMetricNumDisable     = metrics.NewMetricKey("dubbo.registry.directory.num.disable.total", "Disable Directory Urls")

	NotifyMetricRequests = metrics.NewMetricKey("dubbo.registry.notify.requests.total", "Total Notify Requests")
	NotifyMetricNumLast  = metrics.NewMetricKey("dubbo.registry.notify.num.last", "Last Notify Nums")

	// register service metrics key
	ServiceRegisterMetricRequests        = metrics.NewMetricKey("dubbo.registry.register.service.total", "Total Service-Level Register Requests")
	ServiceRegisterMetricRequestsSucceed = metrics.NewMetricKey("dubbo.registry.register.service.succeed.total", "Succeed Service-Level Register Requests")
	ServiceRegisterMetricRequestsFailed  = metrics.NewMetricKey("dubbo.registry.register.service.failed.total", "Failed Service-Level Register Requests")

	// subscribe metrics key
	ServiceSubscribeMetricNum        = metrics.NewMetricKey("dubbo.registry.subscribe.service.num.total", "Total Service-Level Subscribe Num")
	ServiceSubscribeMetricNumSucceed = metrics.NewMetricKey("dubbo.registry.subscribe.service.num.succeed.total", "Succeed Service-Level Num")
	ServiceSubscribeMetricNumFailed  = metrics.NewMetricKey("dubbo.registry.subscribe.service.num.failed.total", "Failed Service-Level Num")

	// register metrics server rt key
	RegisterServiceRtMillisecondsAvg  = metrics.NewMetricKey("dubbo.register.service.rt.milliseconds.avg", "Average Service Register Time")
	RegisterServiceRtMillisecondsLast = metrics.NewMetricKey("dubbo.register.service.rt.milliseconds.last", "Last Service Register Time")
	RegisterServiceRtMillisecondsMax  = metrics.NewMetricKey("dubbo.register.service.rt.milliseconds.max", "Max Service Register Time")
	RegisterServiceRtMillisecondsMin  = metrics.NewMetricKey("dubbo.register.service.rt.milliseconds.min", "Min Service Register Time")
	RegisterServiceRtMillisecondsSum  = metrics.NewMetricKey("dubbo.register.service.rt.milliseconds.sum", "Sum Service Register Time")

	// register metrics rt key
	RegisterRtMillisecondsMax  = metrics.NewMetricKey("dubbo.register.rt.milliseconds.max", "Max Response Time")
	RegisterRtMillisecondsLast = metrics.NewMetricKey("dubbo.register.rt.milliseconds.last", "Last Response Time")
	RegisterRtMillisecondsAvg  = metrics.NewMetricKey("dubbo.register.rt.milliseconds.avg", "Average Response Time")
	RegisterRtMillisecondsSum  = metrics.NewMetricKey("dubbo.register.rt.milliseconds.sum", "Sum Response Time")
	RegisterRtMillisecondsMin  = metrics.NewMetricKey("dubbo.register.rt.milliseconds.min", "Min Response Time")

	// register notify rt key
	NotifyRtMillisecondsAvg  = metrics.NewMetricKey("dubbo.notify.rt.milliseconds.avg", "Average Notify Time")
	NotifyRtMillisecondsLast = metrics.NewMetricKey("dubbo.notify.rt.milliseconds.last", "Last Notify Time")
	NotifyRtMillisecondsMax  = metrics.NewMetricKey("dubbo.notify.rt.milliseconds.max", "Max Notify Time")
	NotifyRtMillisecondsMin  = metrics.NewMetricKey("dubbo.notify.rt.milliseconds.min", "Min Notify Time")
	NotifyRtMillisecondsSum  = metrics.NewMetricKey("dubbo.notify.rt.milliseconds.sum", "Sum Notify Time")
)
