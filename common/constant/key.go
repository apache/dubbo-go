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

type DubboCtxKey string

const (
	ASYNC_KEY = "async" // it's value should be "true" or "false" of string type
)

const (
	GROUP_KEY                = "group"
	VERSION_KEY              = "version"
	INTERFACE_KEY            = "interface"
	MESSAGE_SIZE_KEY         = "message_size"
	PATH_KEY                 = "path"
	SERVICE_KEY              = "service"
	METHODS_KEY              = "methods"
	TIMEOUT_KEY              = "timeout"
	CATEGORY_KEY             = "category"
	CHECK_KEY                = "check"
	ENABLED_KEY              = "enabled"
	SIDE_KEY                 = "side"
	OVERRIDE_PROVIDERS_KEY   = "providerAddresses"
	BEAN_NAME_KEY            = "bean.name"
	GENERIC_KEY              = "generic"
	CLASSIFIER_KEY           = "classifier"
	TOKEN_KEY                = "token"
	LOCAL_ADDR               = "local-addr"
	REMOTE_ADDR              = "remote-addr"
	DEFAULT_REMOTING_TIMEOUT = 3000
	RELEASE_KEY              = "release"
	ANYHOST_KEY              = "anyhost"
	PORT_KEY                 = "port"
	PROTOCOL_KEY             = "protocol"
	PATH_SEPARATOR           = "/"
	// DUBBO_KEY                = "dubbo"
	SSL_ENABLED_KEY = "ssl-enabled"
	// PARAMS_TYPE_Key key used in pass through invoker factory, to define param type
	PARAMS_TYPE_Key = "parameter-type-names"
)

const (
	SERVICE_FILTER_KEY   = "service.filter"
	REFERENCE_FILTER_KEY = "reference.filter"
)

// Filter Keys
const (
	AccessLogFilterKey                = "accesslog"
	ActiveFilterKey                   = "active"
	AuthConsumerFilterKey             = "sign"
	AuthProviderFilterKey             = "auth"
	EchoFilterKey                     = "echo"
	ExecuteLimitFilterKey             = "execute"
	GenericFilterKey                  = "generic"
	GenericServiceFilterKey           = "generic_service"
	GracefulShutdownProviderFilterKey = "pshutdown"
	GracefulShutdownConsumerFilterKey = "cshutdown"
	HystrixConsumerFilterKey          = "hystrix_consumer"
	HystrixProviderFilterKey          = "hystrix_provider"
	MetricsFilterKey                  = "metrics"
	SeataFilterKey                    = "seata"
	SentinelProviderFilterKey         = "sentinel-provider"
	SentinelConsumerFilterKey         = "sentinel-consumer"
	TokenFilterKey                    = "token"
	TpsLimitFilterKey                 = "tps"
	TracingFilterKey                  = "tracing"
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
	SERIALIZATION_KEY                      = "serialization"
	PID_KEY                                = "pid"
	SYNC_REPORT_KEY                        = "sync.report"
	RETRY_PERIOD_KEY                       = "retry.period"
	RETRY_TIMES_KEY                        = "retry.times"
	CYCLE_REPORT_KEY                       = "cycle.report"
	DEFAULT_BLACK_LIST_RECOVER_BLOCK       = 16
)

const (
	DUBBOGO_CTX_KEY = DubboCtxKey("dubbogo-ctx")
)

const (
	REGISTRY_KEY         = "registry"
	REGISTRY_PROTOCOL    = "registry"
	ROLE_KEY             = "registry.role"
	REGISTRY_DEFAULT_KEY = "registry.default"
	// Deprecated use CONFIG_TIMEOUT_KEY key
	REGISTRY_TIMEOUT_KEY = "registry.timeout"
	REGISTRY_LABEL_KEY   = "label"
	PREFERRED_KEY        = "preferred"
	ZONE_KEY             = "zone"
	ZONE_FORCE_KEY       = "zone.force"
	REGISTRY_TTL_KEY     = "registry.ttl"
)

const (
	APPLICATION_KEY          = "application"
	ORGANIZATION_KEY         = "organization"
	NAME_KEY                 = "name"
	MODULE_KEY               = "module"
	APP_VERSION_KEY          = "app.version"
	OWNER_KEY                = "owner"
	ENVIRONMENT_KEY          = "environment"
	METHOD_KEY               = "method"
	METHOD_KEYS              = "methods"
	RULE_KEY                 = "rule"
	RUNTIME_KEY              = "runtime"
	BACKUP_KEY               = "backup"
	ROUTERS_CATEGORY         = "routers"
	ROUTE_PROTOCOL           = "route"
	CONDITION_ROUTE_PROTOCOL = "condition"
	TAG_ROUTE_PROTOCOL       = "tag"
	PROVIDERS_CATEGORY       = "providers"
	ROUTER_KEY               = "router"
	EXPORT_KEY               = "export"
)

const (
	CONFIG_NAMESPACE_KEY  = "namespace"
	CONFIG_GROUP_KEY      = "group"
	CONFIG_APP_ID_KEY     = "appId"
	CONFIG_CLUSTER_KEY    = "cluster"
	CONFIG_TIMEOUT_KEY    = "timeout"
	CONFIG_USERNAME_KEY   = "username"
	CONFIG_PASSWORD_KEY   = "password"
	CONFIG_LOG_DIR_KEY    = "logDir"
	CONFIG_VERSION_KEY    = "configVersion"
	COMPATIBLE_CONFIG_KEY = "compatible_config"
)

const (
	RegistryConfigPrefix       = "dubbo.registries."
	SingleRegistryConfigPrefix = "dubbo.registry."
	ReferenceConfigPrefix      = "dubbo.reference."
	ServiceConfigPrefix        = "dubbo.service."
	ConfigBasePrefix           = "dubbo.base."
	RemotePrefix               = "dubbo.remote."
	ServiceDiscPrefix          = "dubbo.service-discovery."
	ProtocolConfigPrefix       = "dubbo.protocols."
	ProviderConfigPrefix       = "dubbo.provider."
	ConsumerConfigPrefix       = "dubbo.consumer."
	ShutdownConfigPrefix       = "dubbo.shutdown."
	MetadataReportPrefix       = "dubbo.metadata-report."
	RouterConfigPrefix         = "dubbo.router."
)

const (
	CONFIGURATORS_SUFFIX = ".configurators"
)

const (
	NACOS_KEY                    = "nacos"
	NACOS_DEFAULT_ROLETYPE       = 3
	NACOS_CACHE_DIR_KEY          = "cacheDir"
	NACOS_LOG_DIR_KEY            = "logDir"
	NACOS_BEAT_INTERVAL_KEY      = "beatInterval"
	NACOS_ENDPOINT               = "endpoint"
	NACOS_SERVICE_NAME_SEPARATOR = ":"
	NACOS_CATEGORY_KEY           = "category"
	NACOS_PROTOCOL_KEY           = "protocol"
	NACOS_PATH_KEY               = "path"
	NACOS_NAMESPACE_ID           = "namespaceId"
	NACOS_PASSWORD               = "password"
	NACOS_USERNAME               = "username"
	NACOS_NOT_LOAD_LOCAL_CACHE   = "nacos.not.load.cache"
	NACOS_APP_NAME_KEY           = "appName"
	NACOS_REGION_ID_KEY          = "regionId"
	NACOS_ACCESS_KEY             = "access"
	NACOS_SECRET_KEY             = "secret"
	NACOS_OPEN_KMS_KEY           = "kms"
	NACOS_UPDATE_THREAD_NUM_KEY  = "updateThreadNum"
	NACOS_LOG_LEVEL_KEY          = "logLevel"
)

const (
	FILE_KEY = "file"
)

const (
	ZOOKEEPER_KEY = "zookeeper"
)

const (
	ETCDV3_KEY = "etcdv3"
)

const (
	// PassThroughProxyFactoryKey is key of proxy factory with raw data input service
	PassThroughProxyFactoryKey = "dubbo-raw"
)

const (
	TRACING_REMOTE_SPAN_CTX = DubboCtxKey("tracing.remote.span.ctx")
)

// Use for router module
const (
	// TagRouterRuleSuffix Specify tag router suffix
	TagRouterRuleSuffix = ".tag-router"
	// ConditionRouterRuleSuffix Specify condition router suffix
	ConditionRouterRuleSuffix = ".condition-router"
	// ForceUseTag is the tag in attachment
	ForceUseTag = "dubbo.force.tag"
	Tagkey      = "dubbo.tag"
	// AttachmentKey in context in invoker
	AttachmentKey = DubboCtxKey("attachment")
)

// Auth filter
const (
	// name of service filter
	SERVICE_AUTH_KEY = "auth"
	// key of authenticator
	AUTHENTICATOR_KEY = "authenticator"
	// name of default authenticator
	DEFAULT_AUTHENTICATOR = "accesskeys"
	// name of default url storage
	DEFAULT_ACCESS_KEY_STORAGE = "urlstorage"
	// key of storage
	ACCESS_KEY_STORAGE_KEY = "accessKey.storage"
	// key of request timestamp
	REQUEST_TIMESTAMP_KEY = "timestamp"
	// key of request signature
	REQUEST_SIGNATURE_KEY = "signature"
	// AK key
	AK_KEY = "ak"
	// signature format
	SIGNATURE_STRING_FORMAT = "%s#%s#%s#%s"
	// key whether enable signature
	PARAMETER_SIGNATURE_ENABLE_KEY = "param.sign"
	// consumer
	CONSUMER = "consumer"
	// key of access key id
	ACCESS_KEY_ID_KEY = ".accessKeyId"
	// key of secret access key
	SECRET_ACCESS_KEY_KEY = ".secretAccessKey"
)

// metadata report

const (
	METACONFIG_REMOTE  = "remote"
	METACONFIG_LOCAL   = "local"
	KEY_SEPARATOR      = ":"
	DEFAULT_PATH_TAG   = "metadata"
	KEY_REVISON_PREFIX = "revision"

	// metadata service
	METADATA_SERVICE_NAME = "org.apache.dubbo.metadata.MetadataService"
)

// service discovery
const (
	SUBSCRIBED_SERVICE_NAMES_KEY               = "subscribed-services"
	PROVIDED_BY                                = "provided-by"
	EXPORTED_SERVICES_REVISION_PROPERTY_NAME   = "dubbo.metadata.revision"
	SUBSCRIBED_SERVICES_REVISION_PROPERTY_NAME = "dubbo.subscribed-services.revision"
	SERVICE_INSTANCE_SELECTOR                  = "service-instance-selector"
	METADATA_STORAGE_TYPE_PROPERTY_NAME        = "dubbo.metadata.storage-type"
	DEFAULT_METADATA_STORAGE_TYPE              = "local"
	REMOTE_METADATA_STORAGE_TYPE               = "remote"
	SERVICE_INSTANCE_ENDPOINTS                 = "dubbo.endpoints"
	METADATA_SERVICE_PREFIX                    = "dubbo.metadata-service."
	METADATA_SERVICE_URL_PARAMS_PROPERTY_NAME  = METADATA_SERVICE_PREFIX + "url-params"
	METADATA_SERVICE_URLS_PROPERTY_NAME        = METADATA_SERVICE_PREFIX + "urls"

	// SERVICE_DISCOVERY_KEY indicate which service discovery instance will be used
	SERVICE_DISCOVERY_KEY = "service_discovery"
)

// Generic Filter

const (
	GenericSerializationDefault = "true"
	// disable "protobuf-json" temporarily
	//GenericSerializationProtobuf = "protobuf-json"
	GenericSerializationGson = "gson"
)
