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
	ASYNC_KEY = "async" // it's value should be "true" or "false" of string type
)

const (
	GROUP_KEY              = "group"
	VERSION_KEY            = "version"
	INTERFACE_KEY          = "interface"
	PATH_KEY               = "path"
	SERVICE_KEY            = "service"
	METHODS_KEY            = "methods"
	TIMEOUT_KEY            = "timeout"
	CATEGORY_KEY           = "category"
	CHECK_KEY              = "check"
	ENABLED_KEY            = "enabled"
	SIDE_KEY               = "side"
	OVERRIDE_PROVIDERS_KEY = "providerAddresses"
	BEAN_NAME_KEY          = "bean.name"
	GENERIC_KEY            = "generic"
	CLASSIFIER_KEY         = "classifier"
	TOKEN_KEY              = "token"
	LOCAL_ADDR             = "local-addr"
	REMOTE_ADDR            = "remote-addr"
)

const (
	SERVICE_FILTER_KEY   = "service.filter"
	REFERENCE_FILTER_KEY = "reference.filter"
)

const (
	TIMESTAMP_KEY                          = "timestamp"
	REMOTE_TIMESTAMP_KEY                   = "remote.timestamp"
	CLUSTER_KEY                            = "cluster"
	LOADBALANCE_KEY                        = "loadbalance"
	WEIGHT_KEY                             = "weight"
	WARMUP_KEY                             = "warmup"
	RETRIES_KEY                            = "retries"
	STICKY_KEY                             = "sticky"
	BEAN_NAME                              = "bean.name"
	FAIL_BACK_TASKS_KEY                    = "failbacktasks"
	FORKS_KEY                              = "forks"
	DEFAULT_FORKS                          = 2
	DEFAULT_TIMEOUT                        = 1000
	ACCESS_LOG_KEY                         = "accesslog"
	TPS_LIMITER_KEY                        = "tps.limiter"
	TPS_REJECTED_EXECUTION_HANDLER_KEY     = "tps.limit.rejected.handler"
	TPS_LIMIT_RATE_KEY                     = "tps.limit.rate"
	DEFAULT_TPS_LIMIT_RATE                 = "-1"
	TPS_LIMIT_INTERVAL_KEY                 = "tps.limit.interval"
	DEFAULT_TPS_LIMIT_INTERVAL             = "60000"
	TPS_LIMIT_STRATEGY_KEY                 = "tps.limit.strategy"
	EXECUTE_LIMIT_KEY                      = "execute.limit"
	DEFAULT_EXECUTE_LIMIT                  = "-1"
	EXECUTE_REJECTED_EXECUTION_HANDLER_KEY = "execute.limit.rejected.handler"
	PROVIDER_SHUTDOWN_FILTER               = "pshutdown"
	CONSUMER_SHUTDOWN_FILTER               = "cshutdown"
)

const (
	DUBBOGO_CTX_KEY = "dubbogo-ctx"
)

const (
	REGISTRY_KEY         = "registry"
	REGISTRY_PROTOCOL    = "registry"
	ROLE_KEY             = "registry.role"
	REGISTRY_DEFAULT_KEY = "registry.default"
	REGISTRY_TIMEOUT_KEY = "registry.timeout"
)

const (
	APPLICATION_KEY  = "application"
	ORGANIZATION_KEY = "organization"
	NAME_KEY         = "name"
	MODULE_KEY       = "module"
	APP_VERSION_KEY  = "app.version"
	OWNER_KEY        = "owner"
	ENVIRONMENT_KEY  = "environment"
	METHOD_KEY       = "method"
	METHOD_KEYS      = "methods"
	RULE_KEY         = "rule"
)

const (
	CONFIG_NAMESPACE_KEY  = "config.namespace"
	CONFIG_GROUP_KEY      = "config.group"
	CONFIG_APP_ID_KEY     = "config.appId"
	CONFIG_CLUSTER_KEY    = "config.cluster"
	CONFIG_CHECK_KEY      = "config.check"
	CONFIG_TIMEOUT_KET    = "config.timeout"
	CONFIG_VERSION_KEY    = "configVersion"
	COMPATIBLE_CONFIG_KEY = "compatible_config"
)
const (
	RegistryConfigPrefix       = "dubbo.registries."
	SingleRegistryConfigPrefix = "dubbo.registry."
	ReferenceConfigPrefix      = "dubbo.reference."
	ServiceConfigPrefix        = "dubbo.service."
	ProtocolConfigPrefix       = "dubbo.protocols."
	ProviderConfigPrefix       = "dubbo.provider."
	ConsumerConfigPrefix       = "dubbo.consumer."
	ShutdownConfigPrefix       = "dubbo.shutdown."
)

const (
	CONFIGURATORS_SUFFIX = ".configurators"
)

const (
	NACOS_KEY                    = "nacos"
	NACOS_DEFAULT_ROLETYPE       = 3
	NACOS_CACHE_DIR_KEY          = "cacheDir"
	NACOS_LOG_DIR_KEY            = "logDir"
	NACOS_ENDPOINT               = "endpoint"
	NACOS_SERVICE_NAME_SEPARATOR = ":"
	NACOS_CATEGORY_KEY           = "category"
	NACOS_PROTOCOL_KEY           = "protocol"
	NACOS_PATH_KEY               = "path"
)

const (
	TRACING_REMOTE_SPAN_CTX = "tracing.remote.span.ctx"
)
