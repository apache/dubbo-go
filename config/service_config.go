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

package config

import (
	"container/list"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

import (
	"github.com/creasty/defaults"

	gxnet "github.com/dubbogo/gost/net"

	perrors "github.com/pkg/errors"

	"go.uber.org/atomic"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/protocolwrapper"
)

// ServiceConfig is the configuration of the service provider
type ServiceConfig struct {
	id                          string
	Filter                      string            `yaml:"filter" json:"filter,omitempty" property:"filter"`
	Protocol                    []string          `default:"[\"dubbo\"]"  validate:"required"  yaml:"protocol"  json:"protocol,omitempty" property:"protocol"` // multi protocol support, split by ','
	Interface                   string            `validate:"required"  yaml:"interface"  json:"interface,omitempty" property:"interface"`
	Registry                    []string          `yaml:"registry"  json:"registry,omitempty"  property:"registry"`
	Cluster                     string            `default:"failover" yaml:"cluster"  json:"cluster,omitempty" property:"cluster"`
	Loadbalance                 string            `default:"random" yaml:"loadbalance"  json:"loadbalance,omitempty"  property:"loadbalance"`
	Group                       string            `yaml:"group"  json:"group,omitempty" property:"group"`
	Version                     string            `yaml:"version"  json:"version,omitempty" property:"version" `
	Methods                     []*MethodConfig   `yaml:"methods"  json:"methods,omitempty" property:"methods"`
	Warmup                      string            `yaml:"warmup"  json:"warmup,omitempty"  property:"warmup"`
	Retries                     string            `yaml:"retries"  json:"retries,omitempty" property:"retries"`
	Serialization               string            `yaml:"serialization" json:"serialization" property:"serialization"`
	Params                      map[string]string `yaml:"params"  json:"params,omitempty" property:"params"`
	Token                       string            `yaml:"token" json:"token,omitempty" property:"token"`
	AccessLog                   string            `yaml:"accesslog" json:"accesslog,omitempty" property:"accesslog"`
	TpsLimiter                  string            `yaml:"tps.limiter" json:"tps.limiter,omitempty" property:"tps.limiter"`
	TpsLimitInterval            string            `yaml:"tps.limit.interval" json:"tps.limit.interval,omitempty" property:"tps.limit.interval"`
	TpsLimitRate                string            `yaml:"tps.limit.rate" json:"tps.limit.rate,omitempty" property:"tps.limit.rate"`
	TpsLimitStrategy            string            `yaml:"tps.limit.strategy" json:"tps.limit.strategy,omitempty" property:"tps.limit.strategy"`
	TpsLimitRejectedHandler     string            `yaml:"tps.limit.rejected.handler" json:"tps.limit.rejected.handler,omitempty" property:"tps.limit.rejected.handler"`
	ExecuteLimit                string            `yaml:"execute.limit" json:"execute.limit,omitempty" property:"execute.limit"`
	ExecuteLimitRejectedHandler string            `yaml:"execute.limit.rejected.handler" json:"execute.limit.rejected.handler,omitempty" property:"execute.limit.rejected.handler"`
	Auth                        string            `yaml:"auth" json:"auth,omitempty" property:"auth"`
	ParamSign                   string            `yaml:"param.sign" json:"param.sign,omitempty" property:"param.sign"`
	Tag                         string            `yaml:"tag" json:"tag,omitempty" property:"tag"`
	GrpcMaxMessageSize          int               `default:"4" yaml:"max_message_size" json:"max_message_size,omitempty"`

	Protocols       map[string]*ProtocolConfig
	Registries      map[string]*RegistryConfig
	ProxyFactoryKey string
	unexported      *atomic.Bool
	exported        *atomic.Bool
	export          bool // a flag to control whether the current service should export or not
	rpcService      common.RPCService
	cacheMutex      sync.Mutex
	cacheProtocol   protocol.Protocol
	exportersLock   sync.Mutex
	exporters       []protocol.Exporter

	metadataType string
}

// Prefix returns dubbo.service.${InterfaceName}.
func (svc *ServiceConfig) Prefix() string {
	return strings.Join([]string{constant.ServiceConfigPrefix, svc.id}, ".")
}

func (svc *ServiceConfig) Init(rc *RootConfig) error {
	if err := initProviderMethodConfig(svc); err != nil {
		return err
	}
	if err := defaults.Set(svc); err != nil {
		return err
	}
	svc.exported = atomic.NewBool(false)
	svc.metadataType = rc.Application.MetadataType
	svc.unexported = atomic.NewBool(false)
	svc.Registries = rc.Registries
	svc.Protocols = rc.Protocols
	if rc.Provider != nil {
		svc.ProxyFactoryKey = rc.Provider.ProxyFactory
	}
	svc.Registry = translateRegistryIds(svc.Registry)
	if len(svc.Registry) <= 0 {
		svc.Registry = rc.Provider.Registry
	}
	svc.export = true
	return verify(svc)
}

// InitExported will set exported as false atom bool
func (svc *ServiceConfig) InitExported() {
	svc.exported = atomic.NewBool(false)
}

// IsExport will return whether the service config is exported or not
func (svc *ServiceConfig) IsExport() bool {
	return svc.exported.Load()
}

// Get Random Port
func getRandomPort(protocolConfigs []*ProtocolConfig) *list.List {
	ports := list.New()
	for _, proto := range protocolConfigs {
		if len(proto.Port) > 0 {
			continue
		}

		tcp, err := gxnet.ListenOnTCPRandomPort(proto.Ip)
		if err != nil {
			panic(perrors.New(fmt.Sprintf("Get tcp port error, err is {%v}", err)))
		}
		defer tcp.Close()
		ports.PushBack(strings.Split(tcp.Addr().String(), ":")[1])
	}
	return ports
}

// Export exports the service
func (svc *ServiceConfig) Export() error {
	// TODO: delay export
	if svc.unexported != nil && svc.unexported.Load() {
		err := perrors.Errorf("The service %v has already unexported!", svc.Interface)
		logger.Errorf(err.Error())
		return err
	}
	if svc.unexported != nil && svc.exported.Load() {
		logger.Warnf("The service %v has already exported!", svc.Interface)
		return nil
	}

	regUrls := loadRegistries(svc.Registry, svc.Registries, common.PROVIDER)
	urlMap := svc.getUrlMap()
	protocolConfigs := loadProtocol(svc.Protocol, svc.Protocols)
	if len(protocolConfigs) == 0 {
		logger.Warnf("The service %v's '%v' protocols don't has right protocolConfigs, Please check your configuration center and transfer protocol ", svc.Interface, svc.Protocol)
		return nil
	}

	ports := getRandomPort(protocolConfigs)
	nextPort := ports.Front()
	proxyFactory := extension.GetProxyFactory(svc.ProxyFactoryKey)
	for _, proto := range protocolConfigs {
		// registry the service reflect
		methods, err := common.ServiceMap.Register(svc.Interface, proto.Name, svc.Group, svc.Version, svc.rpcService)
		if err != nil {
			formatErr := perrors.Errorf("The service %v export the protocol %v error! Error message is %v.",
				svc.Interface, proto.Name, err.Error())
			logger.Errorf(formatErr.Error())
			return formatErr
		}

		port := proto.Port
		if len(proto.Port) == 0 {
			port = nextPort.Value.(string)
			nextPort = nextPort.Next()
		}
		ivkURL := common.NewURLWithOptions(
			common.WithPath(svc.Interface),
			common.WithProtocol(proto.Name),
			common.WithIp(proto.Ip),
			common.WithPort(port),
			common.WithParams(urlMap),
			common.WithParamsValue(constant.BEAN_NAME_KEY, svc.id),
			//common.WithParamsValue(constant.SSL_ENABLED_KEY, strconv.FormatBool(config.GetSslEnabled())),
			common.WithMethods(strings.Split(methods, ",")),
			common.WithToken(svc.Token),
			common.WithParamsValue(constant.METADATATYPE_KEY, svc.metadataType),
		)
		if len(svc.Tag) > 0 {
			ivkURL.AddParam(constant.Tagkey, svc.Tag)
		}

		// post process the URL to be exported
		svc.postProcessConfig(ivkURL)
		// config post processor may set "export" to false
		if !ivkURL.GetParamBool(constant.EXPORT_KEY, true) {
			return nil
		}

		if len(regUrls) > 0 {
			svc.cacheMutex.Lock()
			if svc.cacheProtocol == nil {
				logger.Infof(fmt.Sprintf("First load the registry protocol, url is {%v}!", ivkURL))
				svc.cacheProtocol = extension.GetProtocol("registry")
			}
			svc.cacheMutex.Unlock()

			for _, regUrl := range regUrls {
				regUrl.SubURL = ivkURL
				invoker := proxyFactory.GetInvoker(regUrl)
				exporter := svc.cacheProtocol.Export(invoker)
				if exporter == nil {
					return perrors.New(fmt.Sprintf("Registry protocol new exporter error, registry is {%v}, url is {%v}", regUrl, ivkURL))
				}
				svc.exporters = append(svc.exporters, exporter)
			}
		} else {
			if ivkURL.GetParam(constant.INTERFACE_KEY, "") == constant.METADATA_SERVICE_NAME {
				ms, err := extension.GetLocalMetadataService("")
				if err != nil {
					return err
				}
				ms.SetMetadataServiceURL(ivkURL)
			}
			invoker := proxyFactory.GetInvoker(ivkURL)
			exporter := extension.GetProtocol(protocolwrapper.FILTER).Export(invoker)
			if exporter == nil {
				return perrors.New(fmt.Sprintf("Filter protocol without registry new exporter error, url is {%v}", ivkURL))
			}
			svc.exporters = append(svc.exporters, exporter)
		}
		publishServiceDefinition(ivkURL)
	}
	svc.exported.Store(true)
	return nil
}

//loadProtocol filter protocols by ids
func loadProtocol(protocolIds []string, protocols map[string]*ProtocolConfig) []*ProtocolConfig {
	returnProtocols := make([]*ProtocolConfig, 0, len(protocols))
	for _, v := range protocolIds {
		for k, config := range protocols {
			if v == k {
				returnProtocols = append(returnProtocols, config)
			}
		}
	}
	return returnProtocols
}

func loadRegistries(registryIds []string, registries map[string]*RegistryConfig, roleType common.RoleType) []*common.URL {
	var registryURLs []*common.URL
	//trSlice := strings.Split(targetRegistries, ",")

	for k, registryConf := range registries {
		target := false

		// if user not config targetRegistries, default load all
		// Notice: in func "func Split(s, sep string) []string" comment:
		// if s does not contain sep and sep is not empty, SplitAfter returns
		// a slice of length 1 whose only element is s. So we have to add the
		// condition when targetRegistries string is not set (it will be "" when not set)
		if len(registryIds) == 0 || (len(registryIds) == 1 && registryIds[0] == "") {
			target = true
		} else {
			// else if user config targetRegistries
			for _, tr := range registryIds {
				if tr == k {
					target = true
					break
				}
			}
		}

		if target {
			if registryURL, err := registryConf.toURL(roleType); err != nil {
				logger.Errorf("The registry id: %s url is invalid, error: %#v", k, err)
				panic(err)
			} else {
				registryURLs = append(registryURLs, registryURL)
			}
		}
	}

	return registryURLs
}

// Unexport will call unexport of all exporters service config exported
func (svc *ServiceConfig) Unexport() {
	if !svc.exported.Load() {
		return
	}
	if svc.unexported.Load() {
		return
	}

	func() {
		svc.exportersLock.Lock()
		defer svc.exportersLock.Unlock()
		for _, exporter := range svc.exporters {
			exporter.Unexport()
		}
		svc.exporters = nil
	}()

	svc.exported.Store(false)
	svc.unexported.Store(true)
}

// Implement only store the @s and return
func (svc *ServiceConfig) Implement(s common.RPCService) {
	svc.rpcService = s
}

func (svc *ServiceConfig) getUrlMap() url.Values {
	urlMap := url.Values{}
	// first set user params
	for k, v := range svc.Params {
		urlMap.Set(k, v)
	}
	urlMap.Set(constant.INTERFACE_KEY, svc.Interface)
	urlMap.Set(constant.TIMESTAMP_KEY, strconv.FormatInt(time.Now().Unix(), 10))
	urlMap.Set(constant.CLUSTER_KEY, svc.Cluster)
	urlMap.Set(constant.LOADBALANCE_KEY, svc.Loadbalance)
	urlMap.Set(constant.WARMUP_KEY, svc.Warmup)
	urlMap.Set(constant.RETRIES_KEY, svc.Retries)
	urlMap.Set(constant.GROUP_KEY, svc.Group)
	urlMap.Set(constant.VERSION_KEY, svc.Version)
	urlMap.Set(constant.ROLE_KEY, strconv.Itoa(common.PROVIDER))
	urlMap.Set(constant.RELEASE_KEY, "dubbo-golang-"+constant.Version)
	urlMap.Set(constant.SIDE_KEY, (common.RoleType(common.PROVIDER)).Role())
	urlMap.Set(constant.MESSAGE_SIZE_KEY, strconv.Itoa(svc.GrpcMaxMessageSize))
	// todo: move
	urlMap.Set(constant.SERIALIZATION_KEY, svc.Serialization)
	// application config info
	//urlMap.Set(constant.APPLICATION_KEY, applicationConfig.Name)
	//urlMap.Set(constant.ORGANIZATION_KEY, applicationConfig.Organization)
	//urlMap.Set(constant.NAME_KEY, applicationConfig.Name)
	//urlMap.Set(constant.MODULE_KEY, applicationConfig.Module)
	//urlMap.Set(constant.APP_VERSION_KEY, applicationConfig.Version)
	//urlMap.Set(constant.OWNER_KEY, applicationConfig.Owner)
	//urlMap.Set(constant.ENVIRONMENT_KEY, applicationConfig.Environment)

	// filter
	if svc.Filter == "" {
		urlMap.Set(constant.SERVICE_FILTER_KEY, constant.DEFAULT_SERVICE_FILTERS)
	} else {
		urlMap.Set(constant.SERVICE_FILTER_KEY, svc.Filter)
	}

	// filter special config
	urlMap.Set(constant.AccessLogFilterKey, svc.AccessLog)
	// tps limiter
	urlMap.Set(constant.TPS_LIMIT_STRATEGY_KEY, svc.TpsLimitStrategy)
	urlMap.Set(constant.TPS_LIMIT_INTERVAL_KEY, svc.TpsLimitInterval)
	urlMap.Set(constant.TPS_LIMIT_RATE_KEY, svc.TpsLimitRate)
	urlMap.Set(constant.TPS_LIMITER_KEY, svc.TpsLimiter)
	urlMap.Set(constant.TPS_REJECTED_EXECUTION_HANDLER_KEY, svc.TpsLimitRejectedHandler)

	// execute limit filter
	urlMap.Set(constant.EXECUTE_LIMIT_KEY, svc.ExecuteLimit)
	urlMap.Set(constant.EXECUTE_REJECTED_EXECUTION_HANDLER_KEY, svc.ExecuteLimitRejectedHandler)

	// auth filter
	urlMap.Set(constant.SERVICE_AUTH_KEY, svc.Auth)
	urlMap.Set(constant.PARAMETER_SIGNATURE_ENABLE_KEY, svc.ParamSign)

	// whether to export or not
	urlMap.Set(constant.EXPORT_KEY, strconv.FormatBool(svc.export))

	for _, v := range svc.Methods {
		prefix := "methods." + v.Name + "."
		urlMap.Set(prefix+constant.LOADBALANCE_KEY, v.LoadBalance)
		urlMap.Set(prefix+constant.RETRIES_KEY, v.Retries)
		urlMap.Set(prefix+constant.WEIGHT_KEY, strconv.FormatInt(v.Weight, 10))

		urlMap.Set(prefix+constant.TPS_LIMIT_STRATEGY_KEY, v.TpsLimitStrategy)
		urlMap.Set(prefix+constant.TPS_LIMIT_INTERVAL_KEY, v.TpsLimitInterval)
		urlMap.Set(prefix+constant.TPS_LIMIT_RATE_KEY, v.TpsLimitRate)

		urlMap.Set(constant.EXECUTE_LIMIT_KEY, v.ExecuteLimit)
		urlMap.Set(constant.EXECUTE_REJECTED_EXECUTION_HANDLER_KEY, v.ExecuteLimitRejectedHandler)
	}

	return urlMap
}

// GetExportedUrls will return the url in service config's exporter
func (svc *ServiceConfig) GetExportedUrls() []*common.URL {
	if svc.exported.Load() {
		var urls []*common.URL
		for _, exporter := range svc.exporters {
			urls = append(urls, exporter.GetInvoker().GetURL())
		}
		return urls
	}
	return nil
}

func (svc *ServiceConfig) publishServiceDefinition(url *common.URL) {
	//svc.rootConfig.MetadataReportConfig.
	if remoteMetadataService, err := extension.GetRemoteMetadataService(); err == nil && remoteMetadataService != nil {
		remoteMetadataService.PublishServiceDefinition(url)
	}
}

// postProcessConfig asks registered ConfigPostProcessor to post-process the current ServiceConfig.
func (svc *ServiceConfig) postProcessConfig(url *common.URL) {
	for _, p := range extension.GetConfigPostProcessors() {
		p.PostProcessServiceConfig(url)
	}
}

// ServiceConfigOpt is the option to init ServiceConfig
type ServiceConfigOpt func(config *ServiceConfig) *ServiceConfig

// NewDefaultServiceConfig returns default ServiceConfig
func NewDefaultServiceConfig() *ServiceConfig {
	newServiceConfig := &ServiceConfig{
		unexported: atomic.NewBool(false),
		exported:   atomic.NewBool(false),
		export:     true,
		Protocols:  make(map[string]*ProtocolConfig),
		Registries: make(map[string]*RegistryConfig),
	}
	newServiceConfig.Params = make(map[string]string)
	newServiceConfig.Methods = make([]*MethodConfig, 0, 8)
	return newServiceConfig
}

// NewServiceConfig returns ServiceConfig with given @opts
func NewServiceConfig(opts ...ServiceConfigOpt) *ServiceConfig {
	defaultServiceConfig := NewDefaultServiceConfig()
	for _, v := range opts {
		v(defaultServiceConfig)
	}
	return defaultServiceConfig
}

// WithServiceRegistry returns ServiceConfigOpt with given registryKey @registry
func WithServiceRegistry(registry string) ServiceConfigOpt {
	return func(config *ServiceConfig) *ServiceConfig {
		config.Registry = append(config.Registry, registry)
		return config
	}
}

// WithServiceProtocolKeys returns ServiceConfigOpt with given protocolKey @protocol
func WithServiceProtocolKeys(protocolKeys ...string) ServiceConfigOpt {
	return func(config *ServiceConfig) *ServiceConfig {
		config.Protocol = protocolKeys
		return config
	}
}

// WithServiceInterface returns ServiceConfigOpt with given @interfaceName
func WithServiceInterface(interfaceName string) ServiceConfigOpt {
	return func(config *ServiceConfig) *ServiceConfig {
		config.Interface = interfaceName
		return config
	}
}

// WithServiceMetadataType returns ServiceConfigOpt with given @metadataType
func WithServiceMetadataType(metadataType string) ServiceConfigOpt {
	return func(config *ServiceConfig) *ServiceConfig {
		config.metadataType = metadataType
		return config
	}
}

// WithServiceLoadBalance returns ServiceConfigOpt with given load balance @lb
func WithServiceLoadBalance(lb string) ServiceConfigOpt {
	return func(config *ServiceConfig) *ServiceConfig {
		config.Loadbalance = lb
		return config
	}
}

// WithServiceWarmUpTime returns ServiceConfigOpt with given @warmUp time
func WithServiceWarmUpTime(warmUp string) ServiceConfigOpt {
	return func(config *ServiceConfig) *ServiceConfig {
		config.Warmup = warmUp
		return config
	}
}

// WithServiceCluster returns ServiceConfigOpt with given cluster name @cluster
func WithServiceCluster(cluster string) ServiceConfigOpt {
	return func(config *ServiceConfig) *ServiceConfig {
		config.Cluster = cluster
		return config
	}
}

// WithServiceMethod returns ServiceConfigOpt with given @name, @retries and load balance @lb
func WithServiceMethod(name, retries, lb string) ServiceConfigOpt {
	return func(config *ServiceConfig) *ServiceConfig {
		config.Methods = append(config.Methods, &MethodConfig{
			Name:        name,
			Retries:     retries,
			LoadBalance: lb,
		})
		return config
	}
}

func WithServiceProtocol(protocolName string, protocolConfig *ProtocolConfig) ServiceConfigOpt {
	return func(config *ServiceConfig) *ServiceConfig {
		config.Protocols[protocolName] = protocolConfig
		return config
	}
}

func WithServiceRegistries(registryName string, registryConfig *RegistryConfig) ServiceConfigOpt {
	return func(config *ServiceConfig) *ServiceConfig {
		config.Registries[registryName] = registryConfig
		return config
	}
}

func WithServiceGroup(group string) ServiceConfigOpt {
	return func(config *ServiceConfig) *ServiceConfig {
		config.Group = group
		return config
	}
}

func WithServiceVersion(version string) ServiceConfigOpt {
	return func(config *ServiceConfig) *ServiceConfig {
		config.Version = version
		return config
	}
}

func WithProxyFactoryKey(proxyFactoryKey string) ServiceConfigOpt {
	return func(config *ServiceConfig) *ServiceConfig {
		config.ProxyFactoryKey = proxyFactoryKey
		return config
	}
}

func WithRPCService(service common.RPCService) ServiceConfigOpt {
	return func(config *ServiceConfig) *ServiceConfig {
		config.rpcService = service
		return config
	}
}

func WithServiceID(id string) ServiceConfigOpt {
	return func(config *ServiceConfig) *ServiceConfig {
		config.id = id
		return config
	}
}
