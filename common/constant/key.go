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
	DUBBO_KEY                = "dubbo"
	SSL_ENABLED_KEY          = "ssl-enabled"
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
	SERIALIZATION_KEY                      = "serialization"
	PID_KEY                                = "pid"
	SYNC_REPORT_KEY                        = "sync.report"
	RETRY_PERIOD_KEY                       = "retry.period"
	RETRY_TIMES_KEY                        = "retry.times"
	CYCLE_REPORT_KEY                       = "cycle.report"
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
)

const (
	CONFIG_NAMESPACE_KEY  = "config.namespace"
	CONFIG_GROUP_KEY      = "config.group"
	CONFIG_APP_ID_KEY     = "config.appId"
	CONFIG_CLUSTER_KEY    = "config.cluster"
	CONFIG_CHECK_KEY      = "config.check"
	CONFIG_TIMEOUT_KET    = "config.timeout"
	CONFIG_LOG_DIR_KEY    = "config.logDir"
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
	NACOS_ENDPOINT               = "endpoint"
	NACOS_SERVICE_NAME_SEPARATOR = ":"
	NACOS_CATEGORY_KEY           = "category"
	NACOS_PROTOCOL_KEY           = "protocol"
	NACOS_PATH_KEY               = "path"
	NACOS_NAMESPACE_ID           = "namespaceId"
	NACOS_PASSWORD               = "password"
	NACOS_USERNAME               = "username"
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
	CONSUL_KEY          = "consul"
	CHECK_PASS_INTERVAL = "consul-check-pass-interval"
	// default time-to-live in millisecond
	DEFAULT_CHECK_PASS_INTERVAL = 16000
	QUERY_TAG                   = "consul_query_tag"
	ACL_TOKEN                   = "acl-token"
	// default deregister critical server after
	DEFAULT_DEREGISTER_TIME = "20s"
	DEREGISTER_AFTER        = "consul-deregister-critical-service-after"
)

const (
	TRACING_REMOTE_SPAN_CTX = "tracing.remote.span.ctx"
)

// Use for router module
const (
	// ConditionRouterName Specify file condition router name
	ConditionRouterName = "condition"
	// ConditionAppRouterName Specify listenable application router name
	ConditionAppRouterName = "app"
	// ListenableRouterName Specify listenable router name
	ListenableRouterName = "listenable"
	// HealthCheckRouterName Specify the name of HealthCheckRouter
	HealthCheckRouterName = "health_check"
	// TagRouterName Specify the name of TagRouter
	TagRouterName = "tag"
	// TagRouterRuleSuffix Specify tag router suffix
	TagRouterRuleSuffix  = ".tag-router"
	RemoteApplicationKey = "remote.application"
	// ConditionRouterRuleSuffix Specify condition router suffix
	ConditionRouterRuleSuffix = ".condition-router"

	// Force Force key in router module
	RouterForce = "force"
	// Enabled Enabled key in router module
	RouterEnabled = "enabled"
	// Priority Priority key in router module
	RouterPriority = "priority"
	// RouterScope Scope key in router module
	RouterScope = "scope"
	// RouterApplicationScope Scope key in router module
	RouterApplicationScope = "application"
	// RouterServiceScope Scope key in router module
	RouterServiceScope = "service"
	// RouterRuleKey defines the key of the router, service's/application's name
	RouterRuleKey = "key"
	// ForceUseTag is the tag in attachment
	ForceUseTag = "dubbo.force.tag"
	Tagkey      = "dubbo.tag"

	// Attachment key in context in invoker
	AttachmentKey = "attachment"
)

const (
	// name of consumer sign filter
	CONSUMER_SIGN_FILTER = "sign"
	// name of consumer sign filter
	PROVIDER_AUTH_FILTER = "auth"
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
	PARAMTER_SIGNATURE_ENABLE_KEY = "param.sign"
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

// HealthCheck Router
const (
	// The key of HealthCheck SPI
	HEALTH_CHECKER = "health.checker"
	// The name of the default implementation of HealthChecker
	DEFAULT_HEALTH_CHECKER = "default"
	// The key of oustanding-request-limit
	OUTSTANDING_REQUEST_COUNT_LIMIT_KEY = "outstanding.request.limit"
	// The key of successive-failed-request's threshold
	SUCCESSIVE_FAILED_REQUEST_THRESHOLD_KEY = "successive.failed.threshold"
	// The key of circuit-tripped timeout factor
	CIRCUIT_TRIPPED_TIMEOUT_FACTOR_KEY = "circuit.tripped.timeout.factor"
	// The default threshold of  successive-failed-request if not specfied
	DEFAULT_SUCCESSIVE_FAILED_THRESHOLD = 5
	// The default maximum diff between successive-failed-request's threshold and actual successive-failed-request's count
	DEFAULT_SUCCESSIVE_FAILED_REQUEST_MAX_DIFF = 5
	// The default factor of  circuit-tripped timeout if not specfied
	DEFAULT_CIRCUIT_TRIPPED_TIMEOUT_FACTOR = 1000
	// The default time window of circuit-tripped  in millisecond if not specfied
	MAX_CIRCUIT_TRIPPED_TIMEOUT_IN_MS = 30000
)

// service discovery
const (
	SUBSCRIBED_SERVICE_NAMES_KEY               = "subscribed-services"
	PROVIDER_BY                                = "provided-by"
	EXPORTED_SERVICES_REVISION_PROPERTY_NAME   = "dubbo.exported-services.revision"
	SUBSCRIBED_SERVICES_REVISION_PROPERTY_NAME = "dubbo.subscribed-services.revision"
	SERVICE_INSTANCE_SELECTOR                  = "service-instance-selector"
	METADATA_STORAGE_TYPE_PROPERTY_NAME        = "dubbo.metadata.storage-type"
	DEFAULT_METADATA_STORAGE_TYPE              = "local"
	SERVICE_INSTANCE_ENDPOINTS                 = "dubbo.endpoints"
	METADATA_SERVICE_PREFIX                    = "dubbo.metadata-service."
	METADATA_SERVICE_URL_PARAMS_PROPERTY_NAME  = METADATA_SERVICE_PREFIX + "url-params"
	METADATA_SERVICE_URLS_PROPERTY_NAME        = METADATA_SERVICE_PREFIX + "urls"

	// SERVICE_DISCOVERY_KEY indicate which service discovery instance will be used
	SERVICE_DISCOVERY_KEY = "service_discovery"
)
