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

const (
	Dubbo            = "dubbo"
	ProviderProtocol = "provider"
	// OverrideProtocol is compatible with 2.6.x
	OverrideProtocol = "override"
	EmptyProtocol    = "empty"
	RouterProtocol   = "router"
)

const (
	DefaultWeight           = 100     //
	DefaultWarmup           = 10 * 60 // in java here is 10*60*1000 because of System.currentTimeMillis() is measured in milliseconds & in go time.Unix() is second
	DefaultLoadBalance      = "random"
	DefaultRetries          = "2"
	DefaultRetriesInt       = 2
	DefaultProtocol         = "dubbo"
	DefaultRegTimeout       = "5s"
	DefaultRegTTL           = "15m"
	DefaultCluster          = "failover"
	DefaultFailbackTimes    = "3"
	DefaultFailbackTimesInt = 3
	DefaultFailbackTasks    = 100
	DefaultRestClient       = "resty"
	DefaultRestServer       = "go-restful"
	DefaultPort             = 20000
)

const (
	DefaultKey   = "default"
	Generic      = "$invoke"
	GenericAsync = "$invokeAsync"
	Echo         = "$echo"
)

// default filters
const (
	// DefaultServiceFilters defines default service filters, it is highly recommended
	// that put the AdaptiveServiceProviderFilterKey at the end.
	DefaultServiceFilters = EchoFilterKey + "," +
		MetricsFilterKey + "," + TokenFilterKey + "," + AccessLogFilterKey + "," + TpsLimitFilterKey + "," +
		GenericServiceFilterKey + "," + ExecuteLimitFilterKey + "," + GracefulShutdownProviderFilterKey

	DefaultReferenceFilters = GracefulShutdownConsumerFilterKey
)

const (
	AnyValue          = "*"
	AnyHostValue      = "0.0.0.0"
	LocalHostValue    = "192.168.1.1"
	RemoveValuePrefix = "-"
)

const (
	ConfiguratorsCategory           = "configurators"
	RouterCategory                  = "category"
	DefaultCategory                 = ProviderCategory
	DynamicConfiguratorsCategory    = "dynamicconfigurators"
	AppDynamicConfiguratorsCategory = "appdynamicconfigurators"
	ProviderCategory                = "providers"
	ConsumerCategory                = "consumers"
)

const (
	CommaSplitPattern = "\\s*[,]+\\s*"
)

const (
	SimpleMetadataServiceName = "MetadataService"
	DefaultRevision           = "N/A"
)

const (
	ServiceDiscoveryDefaultGroup = "DEFAULT_GROUP"
)
