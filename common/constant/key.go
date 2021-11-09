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
	AsyncKey = "async" // it's value should be "true" or "false" of string type
)

const (
	GroupKey               = "group"
	VersionKey             = "version"
	InterfaceKey           = "interface"
	InstanceKey            = "instance"
	MessageSizeKey         = "message_size"
	PathKey                = "path"
	ServiceKey             = "service"
	MethodsKey             = "methods"
	TimeoutKey             = "timeout"
	CategoryKey            = "category"
	CheckKey               = "check"
	EnabledKey             = "enabled"
	SideKey                = "side"
	OverrideProvidersKey   = "providerAddresses"
	BeanNameKey            = "bean.name"
	GenericKey             = "generic"
	ClassifierKey          = "classifier"
	TokenKey               = "token"
	LocalAddr              = "local-addr"
	RemoteAddr             = "remote-addr"
	DefaultRemotingTimeout = 3000
	ReleaseKey             = "release"
	AnyhostKey             = "anyhost"
	PortKey                = "port"
	ProtocolKey            = "protocol"
	PathSeparator          = "/"
	CommaSeparator         = ","
	SslEnabledKey          = "ssl-enabled"
	// ParamsTypeKey key used in pass through invoker factory, to define param type
	ParamsTypeKey   = "parameter-type-names"
	MetadataTypeKey = "metadata-type"
)

const (
	ServiceFilterKey   = "service.filter"
	ReferenceFilterKey = "reference.filter"
)

// Filter Keys
const (
	AccessLogFilterKey                   = "accesslog"
	ActiveFilterKey                      = "active"
	AuthConsumerFilterKey                = "sign"
	AuthProviderFilterKey                = "auth"
	EchoFilterKey                        = "echo"
	ExecuteLimitFilterKey                = "execute"
	GenericFilterKey                     = "generic"
	GenericServiceFilterKey              = "generic_service"
	GracefulShutdownProviderFilterKey    = "pshutdown"
	GracefulShutdownConsumerFilterKey    = "cshutdown"
	GracefulShutdownFilterShutdownConfig = "GracefulShutdownFilterShutdownConfig"
	HystrixConsumerFilterKey             = "hystrix_consumer"
	HystrixProviderFilterKey             = "hystrix_provider"
	MetricsFilterKey                     = "metrics"
	SeataFilterKey                       = "seata"
	SentinelProviderFilterKey            = "sentinel-provider"
	SentinelConsumerFilterKey            = "sentinel-consumer"
	TokenFilterKey                       = "token"
	TpsLimitFilterKey                    = "tps"
	TracingFilterKey                     = "tracing"
)

const (
	TimestampKey                       = "timestamp"
	RemoteTimestampKey                 = "remote.timestamp"
	ClusterKey                         = "cluster"
	LoadbalanceKey                     = "loadbalance"
	WeightKey                          = "weight"
	WarmupKey                          = "warmup"
	RetriesKey                         = "retries"
	StickyKey                          = "sticky"
	BeanName                           = "bean.name"
	FailBackTasksKey                   = "failbacktasks"
	ForksKey                           = "forks"
	DefaultForks                       = 2
	DefaultTimeout                     = 1000
	TPSLimiterKey                      = "tps.limiter"
	TPSRejectedExecutionHandlerKey     = "tps.limit.rejected.handler"
	TPSLimitRateKey                    = "tps.limit.rate"
	DefaultTPSLimitRate                = "-1"
	TPSLimitIntervalKey                = "tps.limit.interval"
	DefaultTPSLimitInterval            = "60000"
	TPSLimitStrategyKey                = "tps.limit.strategy"
	ExecuteLimitKey                    = "execute.limit"
	DefaultExecuteLimit                = "-1"
	ExecuteRejectedExecutionHandlerKey = "execute.limit.rejected.handler"
	SerializationKey                   = "serialization"
	PIDKey                             = "pid"
	SyncReportKey                      = "sync.report"
	RetryPeriodKey                     = "retry.period"
	RetryTimesKey                      = "retry.times"
	CycleReportKey                     = "cycle.report"
	DefaultBlackListRecoverBlock       = 16
)

const (
	DubboGoCtxKey = DubboCtxKey("dubbogo-ctx")
)

const (
	RegistryKey             = "registry"
	RegistryProtocol        = "registry"
	ServiceRegistryProtocol = "service-discovery-registry"
	RoleKey                 = "registry.role"
	RegistryDefaultKey      = "registry.default"
	RegistryTimeoutKey      = "registry.timeout"
	RegistryLabelKey        = "label"
	PreferredKey            = "preferred"
	ZoneKey                 = "zone"
	ZoneForceKey            = "zone.force"
	RegistryTTLKey          = "registry.ttl"
	SimplifiedKey           = "simplified"
	NamespaceKey            = "namespace"
	RegistryGroupKey        = "registry.group"
)

const (
	ApplicationKey         = "application"
	OrganizationKey        = "organization"
	NameKey                = "name"
	ModuleKey              = "module"
	AppVersionKey          = "app.version"
	OwnerKey               = "owner"
	EnvironmentKey         = "environment"
	MethodKey              = "method"
	MethodKeys             = "methods"
	RuleKey                = "rule"
	RuntimeKey             = "runtime"
	BackupKey              = "backup"
	RoutersCategory        = "routers"
	RouteProtocol          = "route"
	ConditionRouteProtocol = "condition"
	TagRouteProtocol       = "tag"
	ProvidersCategory      = "providers"
	RouterKey              = "router"
	ExportKey              = "export"
)

const (
	ConfigNamespaceKey        = "namespace"
	ConfigGroupKey            = "group"
	ConfigAppIDKey            = "appId"
	ConfigClusterKey          = "cluster"
	ConfigTimeoutKey          = "timeout"
	ConfigUsernameKey         = "username"
	ConfigPasswordKey         = "password"
	ConfigLogDirKey           = "logDir"
	ConfigVersionKey          = "configVersion"
	CompatibleConfigKey       = "compatible_config"
	ConfigSecretKey           = "secret"
	ConfigBackupConfigKey     = "isBackupConfig"
	ConfigBackupConfigPathKey = "backupConfigPath"
)

const (
	RegistryConfigPrefix       = "dubbo.registries"
	ApplicationConfigPrefix    = "dubbo.application"
	ConfigCenterPrefix         = "dubbo.config-center"
	SingleRegistryConfigPrefix = "dubbo.registry"
	ReferenceConfigPrefix      = "dubbo.reference"
	ServiceConfigPrefix        = "dubbo.service"
	ConfigBasePrefix           = "dubbo.base"
	RemotePrefix               = "dubbo.remote"
	ServiceDiscPrefix          = "dubbo.service-discovery"
	ProtocolConfigPrefix       = "dubbo.protocols"
	ProviderConfigPrefix       = "dubbo.provider"
	ConsumerConfigPrefix       = "dubbo.consumer"
	ShutdownConfigPrefix       = "dubbo.shutdown"
	MetadataReportPrefix       = "dubbo.metadata-report"
	RouterConfigPrefix         = "dubbo.router"
	LoggerConfigPrefix         = "dubbo.logger"
)

const (
	ConfiguratorSuffix = ".configurators"
)

const (
	NacosKey                  = "nacos"
	NacosDefaultRoleType      = 3
	NacosCacheDirKey          = "cacheDir"
	NacosLogDirKey            = "logDir"
	NacosBeatIntervalKey      = "beatInterval"
	NacosEndpoint             = "endpoint"
	NacosServiceNameSeparator = ":"
	NacosCategoryKey          = "category"
	NacosProtocolKey          = "protocol"
	NacosPathKey              = "path"
	NacosNamespaceID          = "namespaceId"
	NacosPassword             = "password"
	NacosUsername             = "username"
	NacosNotLoadLocalCache    = "nacos.not.load.cache"
	NacosAppNameKey           = "appName"
	NacosRegionIDKey          = "regionId"
	NacosAccessKey            = "access"
	NacosSecretKey            = "secret"
	NacosOpenKmsKey           = "kms"
	NacosUpdateThreadNumKey   = "updateThreadNum"
	NacosLogLevelKey          = "logLevel"
)

const (
	FileKey = "file"
)

const (
	ZookeeperKey = "zookeeper"
)

const (
	EtcdV3Key = "etcdv3"
)

const (
	// PassThroughProxyFactoryKey is key of proxy factory with raw data input service
	PassThroughProxyFactoryKey = "dubbo-raw"
)

const (
	TracingRemoteSpanCtx = DubboCtxKey("tracing.remote.span.ctx")
)

// Use for router module
const (
	// TagRouterRuleSuffix Specify tag router suffix
	TagRouterRuleSuffix = ".tag-router"
	// ConditionRouterRuleSuffix Specify condition router suffix
	ConditionRouterRuleSuffix = ".condition-router"
	// MeshRouteSuffix Specify mesh router suffix
	MeshRouteSuffix = ".MESHAPPRULE"
	// ForceUseTag is the tag in attachment
	ForceUseTag = "dubbo.force.tag"
	Tagkey      = "dubbo.tag"
	// AttachmentKey in context in invoker
	AttachmentKey = DubboCtxKey("attachment")
)

// Auth filter
const (
	// name of service filter
	ServiceAuthKey = "auth"
	// key of authenticator
	AuthenticatorKey = "authenticator"
	// name of default authenticator
	DefaultAuthenticator = "accesskeys"
	// name of default url storage
	DefaultAccessKeyStorage = "urlstorage"
	// key of storage
	AccessKeyStorageKey = "accessKey.storage"
	// key of request timestamp
	RequestTimestampKey = "timestamp"
	// key of request signature
	RequestSignatureKey = "signature"
	// AK key
	AKKey = "ak"
	// signature format
	SignatureStringFormat = "%s#%s#%s#%s"
	// key whether enable signature
	ParameterSignatureEnableKey = "param.sign"
	// consumer
	Consumer = "consumer"
	// key of access key id
	AccessKeyIDKey = ".accessKeyId"
	// key of secret access key
	SecretAccessKeyKey = ".secretAccessKey"
)

// metadata report

const (
	MetaConfigRemote  = "remote"
	MetaConfigLocal   = "local"
	KeySeparator      = ":"
	DefaultPathTag    = "metadata"
	KeyRevisionPrefix = "revision"

	// metadata service
	MetadataServiceName = "org.apache.dubbo.metadata.MetadataService"
)

// service discovery
const (
	SubscribedServiceNamesKey              = "subscribed-services"
	ProvidedBy                             = "provided-by"
	ExportedServicesRevisionPropertyName   = "dubbo.metadata.revision"
	SubscribedServicesRevisionPropertyName = "dubbo.subscribed-services.revision"
	ServiceInstanceSelector                = "service-instance-selector"
	MetadataStorageTypePropertyName        = "dubbo.metadata.storage-type"
	DefaultMetadataStorageType             = "local"
	RemoteMetadataStorageType              = "remote"
	ServiceInstanceEndpoints               = "dubbo.endpoints"
	MetadataServicePrefix                  = "dubbo.metadata-service."
	MetadataServiceURLParamsPropertyName   = MetadataServicePrefix + "url-params"
	MetadataServiceURLsPropertyName        = MetadataServicePrefix + "urls"

	// ServiceDiscoveryKey indicate which service discovery instance will be used
	ServiceDiscoveryKey = "service_discovery"
)

// Generic Filter

const (
	GenericSerializationDefault = "true"
	// disable "protobuf-json" temporarily
	//GenericSerializationProtobuf = "protobuf-json"
	GenericSerializationGson = "gson"
)
