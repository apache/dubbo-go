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
	ClientNameKey = "remote-client-name"
)

const (
	GroupKey               = "group"
	VersionKey             = "version"
	InterfaceKey           = "interface"
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
	DotSeparator           = "."
	CommaSeparator         = ","
	SslEnabledKey          = "ssl-enabled"
	// ParamsTypeKey key used in pass through invoker factory, to define param type
	ParamsTypeKey        = "parameter-type-names"
	MetadataTypeKey      = "metadata-type"
	MaxCallSendMsgSize   = "max-call-send-msg-size"
	MaxServerSendMsgSize = "max-server-send-msg-size"
	MaxCallRecvMsgSize   = "max-call-recv-msg-size"
	MaxServerRecvMsgSize = "max-server-recv-msg-size"
)

const (
	ServiceFilterKey   = "service.filter"
	ReferenceFilterKey = "reference.filter"
)

// Filter Keys
const (
	AccessLogFilterKey                   = "accesslog"
	ActiveFilterKey                      = "active"
	AdaptiveServiceProviderFilterKey     = "padasvc"
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
	DefaultTPSLimitRate                = -1
	TPSLimitIntervalKey                = "tps.limit.interval"
	DefaultTPSLimitInterval            = -1
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

// metadata report keys
const (
	MetadataReportNamespaceKey = "metadata-report.namespace"
	MetadataReportGroupKey     = "metadata-report.group"
	MetadataReportUsernameKey  = "metadata-report.username"
	MetadataReportPasswordKey  = "metadata-report.password"
	MetadataReportProtocolKey  = "metadata-report.protocol"
)

// registry keys
const (
	RegistryKey             = "registry"
	RegistryProtocol        = "registry"
	ServiceRegistryProtocol = "service-discovery-registry"
	RegistryRoleKey         = "registry.role"
	RegistryDefaultKey      = "registry.default"
	RegistryAccessKey       = "registry.accesskey"
	RegistrySecretKey       = "registry.secretkey"
	RegistryTimeoutKey      = "registry.timeout"
	RegistryLabelKey        = "label"
	PreferredKey            = "preferred"
	RegistryZoneKey         = "zone"
	RegistryZoneForceKey    = "zone.force"
	RegistryTTLKey          = "registry.ttl"
	RegistrySimplifiedKey   = "simplified"
	RegistryNamespaceKey    = "registry.namespace"
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

// config center keys
const (
	ConfigNamespaceKey        = "config-center.namespace"
	ConfigGroupKey            = "config-center.group"
	ConfigAppIDKey            = "config-center.appId"
	ConfigClusterKey          = "config-center.cluster"
	ConfigTimeoutKey          = "config-center.timeout"
	ConfigUsernameKey         = "config-center.username"
	ConfigAccessKey           = "config-center.access"
	ConfigPasswordKey         = "config-center.password"
	ConfigLogDirKey           = "config-center.logDir"
	ConfigVersionKey          = "config-center.configVersion"
	CompatibleConfigKey       = "config-center.compatible_config"
	ConfigSecretKey           = "config-center.secret"
	ConfigBackupConfigKey     = "config-center.isBackupConfig"
	ConfigBackupConfigPathKey = "config-center.backupConfigPath"
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
	TracingConfigPrefix        = "dubbo.tracing"
	LoggerConfigPrefix         = "dubbo.logger"
	CustomConfigPrefix         = "dubbo.custom"
)

const (
	ConfiguratorSuffix = ".configurators"
)

const (
	NacosKey                  = "nacos"
	NacosGroupKey             = "nacos.group"
	NacosDefaultRoleType      = 3
	NacosCacheDirKey          = "nacos.cacheDir"
	NacosLogDirKey            = "nacos.logDir"
	NacosBeatIntervalKey      = "nacos.beatInterval"
	NacosEndpoint             = "endpoint"
	NacosServiceNameSeparator = ":"
	NacosCategoryKey          = "nacos.category"
	NacosProtocolKey          = "protocol"
	NacosPathKey              = "path"
	NacosNamespaceID          = "nacos.namespaceId"
	NacosNotLoadLocalCache    = "nacos.not.load.cache"
	NacosAppNameKey           = "appName"
	NacosRegionIDKey          = "nacos.regionId"
	NacosAccessKey            = "nacos.access"
	NacosSecretKey            = "nacos.secret"
	NacosOpenKmsKey           = "kms"
	NacosUpdateThreadNumKey   = "updateThreadNum"
	NacosLogLevelKey          = "nacos.logLevel"
	NacosUsername             = "nacos.username"
	NacosPassword             = "nacos.password"
	NacosTimeout              = "nacos.timeout"
)

const (
	PolarisKey                  = "polaris"
	PolarisDefaultRoleType      = 3
	PolarisConfigFilePath       = "configPath"
	PolarisNamespace            = "namespace"
	PolarisServiceToken         = "token"
	PolarisServiceNameSeparator = ":"
	PolarisDubboPath            = "DUBBOPATH"
	PolarisInstanceID           = "polaris.instanceID"
	PolarisDefaultNamespace     = "default"
	PolarisDubboGroup           = "dubbo.group"
	PolarisClientName           = "polaris-client"
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
	TracingConfigKey     = "config.tracing"
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

// AdaptiveService Filter
// goland:noinspection ALL
const (
	// attribute keys
	AdaptiveServiceUpdaterKey = "adaptive-service.updater"
	// attachment keys
	AdaptiveServiceRemainingKey = "adaptive-service.remaining"
	AdaptiveServiceInflightKey  = "adaptive-service.inflight"
	AdaptiveServiceEnabledKey   = "adaptive-service.enabled"
	// enabled value
	AdaptiveServiceIsEnabled = "1"
)
