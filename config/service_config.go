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
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

import (
	"github.com/creasty/defaults"

	"github.com/dubbogo/gost/log/logger"
	gxnet "github.com/dubbogo/gost/net"

	perrors "github.com/pkg/errors"

	"go.uber.org/atomic"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/protocolwrapper"
)

// ServiceConfig is the configuration of the service provider
type ServiceConfig struct {
	id                          string
	Filter                      string            `yaml:"filter" json:"filter,omitempty" property:"filter"`
	ProtocolIDs                 []string          `yaml:"protocol-ids"  json:"protocol-ids,omitempty" property:"protocol-ids"` // multi protocolIDs support, split by ','
	Interface                   string            `yaml:"interface"  json:"interface,omitempty" property:"interface"`
	RegistryIDs                 []string          `yaml:"registry-ids"  json:"registry-ids,omitempty"  property:"registry-ids"`
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
	NotRegister                 bool              `yaml:"not_register" json:"not_register,omitempty" property:"not_register"`
	ParamSign                   string            `yaml:"param.sign" json:"param.sign,omitempty" property:"param.sign"`
	Tag                         string            `yaml:"tag" json:"tag,omitempty" property:"tag"`
	GrpcMaxMessageSize          int               `default:"4" yaml:"max_message_size" json:"max_message_size,omitempty"`
	TracingKey                  string            `yaml:"tracing-key" json:"tracing-key,omitempty" propertiy:"tracing-key"`

	RCProtocolsMap  map[string]*ProtocolConfig
	RCRegistriesMap map[string]*RegistryConfig
	ProxyFactoryKey string
	adaptiveService bool
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
func (s *ServiceConfig) Prefix() string {
	return strings.Join([]string{constant.ServiceConfigPrefix, s.id}, ".")
}

func (s *ServiceConfig) Init(rc *RootConfig) error {
	if err := initProviderMethodConfig(s); err != nil {
		return err
	}
	if err := defaults.Set(s); err != nil {
		return err
	}
	s.exported = atomic.NewBool(false)
	s.metadataType = rc.Application.MetadataType
	if s.Filter == "" {
		s.Filter = rc.Provider.Filter
	}
	if s.Version == "" {
		s.Version = rc.Application.Version
	}
	if s.Group == "" {
		s.Group = rc.Application.Group
	}
	s.unexported = atomic.NewBool(false)
	if len(s.RCRegistriesMap) == 0 {
		s.RCRegistriesMap = rc.Registries
	}
	if len(s.RCProtocolsMap) == 0 {
		s.RCProtocolsMap = rc.Protocols
	}
	if rc.Provider != nil {
		s.ProxyFactoryKey = rc.Provider.ProxyFactory
	}
	s.RegistryIDs = translateIds(s.RegistryIDs)
	if len(s.RegistryIDs) <= 0 {
		s.RegistryIDs = rc.Provider.RegistryIDs
	}

	s.ProtocolIDs = translateIds(s.ProtocolIDs)
	if len(s.ProtocolIDs) <= 0 {
		s.ProtocolIDs = rc.Provider.ProtocolIDs
	}
	if len(s.ProtocolIDs) <= 0 {
		for k, _ := range rc.Protocols {
			s.ProtocolIDs = append(s.ProtocolIDs, k)
		}
	}
	if s.TracingKey == "" {
		s.TracingKey = rc.Provider.TracingKey
	}
	err := s.check()
	if err != nil {
		panic(err)
	}
	s.export = true
	return verify(s)
}

func (s *ServiceConfig) check() error {
	// check if the limiter has been imported
	if s.TpsLimiter != "" {
		_, err := extension.GetTpsLimiter(s.TpsLimiter)
		if err != nil {
			panic(err)
		}
	}
	if s.TpsLimitStrategy != "" {
		_, err := extension.GetTpsLimitStrategyCreator(s.TpsLimitStrategy)
		if err != nil {
			panic(err)
		}
	}
	if s.TpsLimitRejectedHandler != "" {
		_, err := extension.GetRejectedExecutionHandler(s.TpsLimitRejectedHandler)
		if err != nil {
			panic(err)
		}
	}

	if s.TpsLimitInterval != "" {
		tpsLimitInterval, err := strconv.ParseInt(s.TpsLimitInterval, 0, 0)
		if err != nil {
			return fmt.Errorf("[ServiceConfig] Cannot parse the configuration tps.limit.interval for service %s, please check your configuration", s.Interface)
		}
		if tpsLimitInterval < 0 {
			return fmt.Errorf("[ServiceConfig] The configuration tps.limit.interval for service %s must be positive, please check your configuration", s.Interface)
		}
	}

	if s.TpsLimitRate != "" {
		tpsLimitRate, err := strconv.ParseInt(s.TpsLimitRate, 0, 0)
		if err != nil {
			return fmt.Errorf("[ServiceConfig] Cannot parse the configuration tps.limit.rate for service %s, please check your configuration", s.Interface)
		}
		if tpsLimitRate < 0 {
			return fmt.Errorf("[ServiceConfig] The configuration tps.limit.rate for service %s must be positive, please check your configuration", s.Interface)
		}
	}
	return nil
}

// InitExported will set exported as false atom bool
func (s *ServiceConfig) InitExported() {
	s.exported = atomic.NewBool(false)
}

// IsExport will return whether the service config is exported or not
func (s *ServiceConfig) IsExport() bool {
	return s.exported.Load()
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
func (s *ServiceConfig) Export() error {
	// TODO: delay export
	if s.unexported != nil && s.unexported.Load() {
		err := perrors.Errorf("The service %v has already unexported!", s.Interface)
		logger.Errorf(err.Error())
		return err
	}
	if s.exported != nil && s.exported.Load() {
		logger.Warnf("The service %v has already exported!", s.Interface)
		return nil
	}

	regUrls := make([]*common.URL, 0)
	if !s.NotRegister {
		regUrls = loadRegistries(s.RegistryIDs, s.RCRegistriesMap, common.PROVIDER)
	}

	urlMap := s.getUrlMap()
	protocolConfigs := loadProtocol(s.ProtocolIDs, s.RCProtocolsMap)
	if len(protocolConfigs) == 0 {
		logger.Warnf("The service %v's '%v' protocols don't has right protocolConfigs, Please check your configuration center and transfer protocol ", s.Interface, s.ProtocolIDs)
		return nil
	}

	ports := getRandomPort(protocolConfigs)
	nextPort := ports.Front()
	proxyFactory := extension.GetProxyFactory(s.ProxyFactoryKey)
	for _, proto := range protocolConfigs {
		// registry the service reflect
		methods, err := common.ServiceMap.Register(s.Interface, proto.Name, s.Group, s.Version, s.rpcService)
		if err != nil {
			formatErr := perrors.Errorf("The service %v export the protocol %v error! Error message is %v.",
				s.Interface, proto.Name, err.Error())
			logger.Errorf(formatErr.Error())
			return formatErr
		}

		port := proto.Port
		if len(proto.Port) == 0 {
			port = nextPort.Value.(string)
			nextPort = nextPort.Next()
		}
		ivkURL := common.NewURLWithOptions(
			common.WithPath(s.Interface),
			common.WithProtocol(proto.Name),
			common.WithIp(proto.Ip),
			common.WithPort(port),
			common.WithParams(urlMap),
			common.WithParamsValue(constant.BeanNameKey, s.id),
			//common.WithParamsValue(constant.SslEnabledKey, strconv.FormatBool(config.GetSslEnabled())),
			common.WithMethods(strings.Split(methods, ",")),
			common.WithToken(s.Token),
			common.WithParamsValue(constant.MetadataTypeKey, s.metadataType),
		)
		if len(s.Tag) > 0 {
			ivkURL.AddParam(constant.Tagkey, s.Tag)
		}

		// post process the URL to be exported
		s.postProcessConfig(ivkURL)
		// config post processor may set "export" to false
		if !ivkURL.GetParamBool(constant.ExportKey, true) {
			return nil
		}

		if len(regUrls) > 0 {
			s.cacheMutex.Lock()
			if s.cacheProtocol == nil {
				logger.Debugf(fmt.Sprintf("First load the registry protocol, url is {%v}!", ivkURL))
				s.cacheProtocol = extension.GetProtocol("registry")
			}
			s.cacheMutex.Unlock()

			for _, regUrl := range regUrls {
				setRegistrySubURL(ivkURL, regUrl)
				invoker := proxyFactory.GetInvoker(regUrl)
				exporter := s.cacheProtocol.Export(invoker)
				if exporter == nil {
					return perrors.New(fmt.Sprintf("Registry protocol new exporter error, registry is {%v}, url is {%v}", regUrl, ivkURL))
				}
				s.exporters = append(s.exporters, exporter)
			}
		} else {
			if ivkURL.GetParam(constant.InterfaceKey, "") == constant.MetadataServiceName {
				ms, err := extension.GetLocalMetadataService("")
				if err != nil {
					logger.Warnf("export org.apache.dubbo.metadata.MetadataService failed beacause of %s ! pls check if you import _ \"dubbo.apache.org/dubbo-go/v3/metadata/service/local\"", err)
					return nil
				}
				if err := ms.SetMetadataServiceURL(ivkURL); err != nil {
					logger.Warnf("SetMetadataServiceURL error = %s", err)
				}
			}
			invoker := proxyFactory.GetInvoker(ivkURL)
			exporter := extension.GetProtocol(protocolwrapper.FILTER).Export(invoker)
			if exporter == nil {
				return perrors.New(fmt.Sprintf("Filter protocol without registry new exporter error, url is {%v}", ivkURL))
			}
			s.exporters = append(s.exporters, exporter)
		}
		publishServiceDefinition(ivkURL)
	}
	s.exported.Store(true)
	return nil
}

//setRegistrySubURL set registry sub url is ivkURl
func setRegistrySubURL(ivkURL *common.URL, regUrl *common.URL) {
	ivkURL.AddParam(constant.RegistryKey, regUrl.GetParam(constant.RegistryKey, ""))
	regUrl.SubURL = ivkURL
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
func (s *ServiceConfig) Unexport() {
	if !s.exported.Load() {
		return
	}
	if s.unexported.Load() {
		return
	}

	func() {
		s.exportersLock.Lock()
		defer s.exportersLock.Unlock()
		for _, exporter := range s.exporters {
			exporter.Unexport()
		}
		s.exporters = nil
	}()

	s.exported.Store(false)
	s.unexported.Store(true)
}

// Implement only store the @s and return
func (s *ServiceConfig) Implement(rpcService common.RPCService) {
	s.rpcService = rpcService
}

func (s *ServiceConfig) getUrlMap() url.Values {
	urlMap := url.Values{}
	// first set user params
	for k, v := range s.Params {
		urlMap.Set(k, v)
	}
	urlMap.Set(constant.InterfaceKey, s.Interface)
	urlMap.Set(constant.TimestampKey, strconv.FormatInt(time.Now().Unix(), 10))
	urlMap.Set(constant.ClusterKey, s.Cluster)
	urlMap.Set(constant.LoadbalanceKey, s.Loadbalance)
	urlMap.Set(constant.WarmupKey, s.Warmup)
	urlMap.Set(constant.RetriesKey, s.Retries)
	if s.Group != "" {
		urlMap.Set(constant.GroupKey, s.Group)
	}
	if s.Version != "" {
		urlMap.Set(constant.VersionKey, s.Version)
	}
	urlMap.Set(constant.RegistryRoleKey, strconv.Itoa(common.PROVIDER))
	urlMap.Set(constant.ReleaseKey, "dubbo-golang-"+constant.Version)
	urlMap.Set(constant.SideKey, (common.RoleType(common.PROVIDER)).Role())
	urlMap.Set(constant.MessageSizeKey, strconv.Itoa(s.GrpcMaxMessageSize))
	// todo: move
	urlMap.Set(constant.SerializationKey, s.Serialization)
	// application config info
	ac := GetApplicationConfig()
	urlMap.Set(constant.ApplicationKey, ac.Name)
	urlMap.Set(constant.OrganizationKey, ac.Organization)
	urlMap.Set(constant.NameKey, ac.Name)
	urlMap.Set(constant.ModuleKey, ac.Module)
	urlMap.Set(constant.AppVersionKey, ac.Version)
	urlMap.Set(constant.OwnerKey, ac.Owner)
	urlMap.Set(constant.EnvironmentKey, ac.Environment)

	// filter
	var filters string
	if s.Filter == "" {
		filters = constant.DefaultServiceFilters
	} else {
		filters = s.Filter
	}
	if s.adaptiveService {
		filters += fmt.Sprintf(",%s", constant.AdaptiveServiceProviderFilterKey)
	}
	urlMap.Set(constant.ServiceFilterKey, filters)

	// filter special config
	urlMap.Set(constant.AccessLogFilterKey, s.AccessLog)
	// tps limiter
	urlMap.Set(constant.TPSLimitStrategyKey, s.TpsLimitStrategy)
	urlMap.Set(constant.TPSLimitIntervalKey, s.TpsLimitInterval)
	urlMap.Set(constant.TPSLimitRateKey, s.TpsLimitRate)
	urlMap.Set(constant.TPSLimiterKey, s.TpsLimiter)
	urlMap.Set(constant.TPSRejectedExecutionHandlerKey, s.TpsLimitRejectedHandler)
	urlMap.Set(constant.TracingConfigKey, s.TracingKey)

	// execute limit filter
	urlMap.Set(constant.ExecuteLimitKey, s.ExecuteLimit)
	urlMap.Set(constant.ExecuteRejectedExecutionHandlerKey, s.ExecuteLimitRejectedHandler)

	// auth filter
	urlMap.Set(constant.ServiceAuthKey, s.Auth)
	urlMap.Set(constant.ParameterSignatureEnableKey, s.ParamSign)

	// whether to export or not
	urlMap.Set(constant.ExportKey, strconv.FormatBool(s.export))
	urlMap.Set(constant.PIDKey, fmt.Sprintf("%d", os.Getpid()))

	for _, v := range s.Methods {
		prefix := "methods." + v.Name + "."
		urlMap.Set(prefix+constant.LoadbalanceKey, v.LoadBalance)
		urlMap.Set(prefix+constant.RetriesKey, v.Retries)
		urlMap.Set(prefix+constant.WeightKey, strconv.FormatInt(v.Weight, 10))

		urlMap.Set(prefix+constant.TPSLimitStrategyKey, v.TpsLimitStrategy)
		urlMap.Set(prefix+constant.TPSLimitIntervalKey, v.TpsLimitInterval)
		urlMap.Set(prefix+constant.TPSLimitRateKey, v.TpsLimitRate)

		urlMap.Set(constant.ExecuteLimitKey, v.ExecuteLimit)
		urlMap.Set(constant.ExecuteRejectedExecutionHandlerKey, v.ExecuteLimitRejectedHandler)
	}

	return urlMap
}

// GetExportedUrls will return the url in service config's exporter
func (s *ServiceConfig) GetExportedUrls() []*common.URL {
	if s.exported.Load() {
		var urls []*common.URL
		for _, exporter := range s.exporters {
			urls = append(urls, exporter.GetInvoker().GetURL())
		}
		return urls
	}
	return nil
}

// postProcessConfig asks registered ConfigPostProcessor to post-process the current ServiceConfig.
func (s *ServiceConfig) postProcessConfig(url *common.URL) {
	for _, p := range extension.GetConfigPostProcessors() {
		p.PostProcessServiceConfig(url)
	}
}

// newEmptyServiceConfig returns default ServiceConfig
func newEmptyServiceConfig() *ServiceConfig {
	newServiceConfig := &ServiceConfig{
		unexported:      atomic.NewBool(false),
		exported:        atomic.NewBool(false),
		export:          true,
		RCProtocolsMap:  make(map[string]*ProtocolConfig),
		RCRegistriesMap: make(map[string]*RegistryConfig),
	}
	newServiceConfig.Params = make(map[string]string)
	newServiceConfig.Methods = make([]*MethodConfig, 0, 8)
	return newServiceConfig
}

type ServiceConfigBuilder struct {
	serviceConfig *ServiceConfig
}

func NewServiceConfigBuilder() *ServiceConfigBuilder {
	return &ServiceConfigBuilder{serviceConfig: newEmptyServiceConfig()}
}

func (pcb *ServiceConfigBuilder) SetRegistryIDs(registryIDs ...string) *ServiceConfigBuilder {
	pcb.serviceConfig.RegistryIDs = registryIDs
	return pcb
}

func (pcb *ServiceConfigBuilder) SetProtocolIDs(protocolIDs ...string) *ServiceConfigBuilder {
	pcb.serviceConfig.ProtocolIDs = protocolIDs
	return pcb
}

func (pcb *ServiceConfigBuilder) SetInterface(interfaceName string) *ServiceConfigBuilder {
	pcb.serviceConfig.Interface = interfaceName
	return pcb
}

func (pcb *ServiceConfigBuilder) SetMetadataType(setMetadataType string) *ServiceConfigBuilder {
	pcb.serviceConfig.metadataType = setMetadataType
	return pcb
}

func (pcb *ServiceConfigBuilder) SetLoadBalancce(lb string) *ServiceConfigBuilder {
	pcb.serviceConfig.Loadbalance = lb
	return pcb
}

func (pcb *ServiceConfigBuilder) SetWarmUpTie(warmUp string) *ServiceConfigBuilder {
	pcb.serviceConfig.Warmup = warmUp
	return pcb
}

func (pcb *ServiceConfigBuilder) SetCluster(cluster string) *ServiceConfigBuilder {
	pcb.serviceConfig.Cluster = cluster
	return pcb
}

func (pcb *ServiceConfigBuilder) AddRCProtocol(protocolName string, protocolConfig *ProtocolConfig) *ServiceConfigBuilder {
	pcb.serviceConfig.RCProtocolsMap[protocolName] = protocolConfig
	return pcb
}

func (pcb *ServiceConfigBuilder) AddRCRegistry(registryName string, registryConfig *RegistryConfig) *ServiceConfigBuilder {
	pcb.serviceConfig.RCRegistriesMap[registryName] = registryConfig
	return pcb
}

func (pcb *ServiceConfigBuilder) SetGroup(group string) *ServiceConfigBuilder {
	pcb.serviceConfig.Group = group
	return pcb
}
func (pcb *ServiceConfigBuilder) SetVersion(version string) *ServiceConfigBuilder {
	pcb.serviceConfig.Version = version
	return pcb
}

func (pcb *ServiceConfigBuilder) SetProxyFactoryKey(proxyFactoryKey string) *ServiceConfigBuilder {
	pcb.serviceConfig.ProxyFactoryKey = proxyFactoryKey
	return pcb
}

func (pcb *ServiceConfigBuilder) SetRPCService(service common.RPCService) *ServiceConfigBuilder {
	pcb.serviceConfig.rpcService = service
	return pcb
}

func (pcb *ServiceConfigBuilder) SetSerialization(serialization string) *ServiceConfigBuilder {
	pcb.serviceConfig.Serialization = serialization
	return pcb
}

func (pcb *ServiceConfigBuilder) SetServiceID(id string) *ServiceConfigBuilder {
	pcb.serviceConfig.id = id
	return pcb
}

func (pcb *ServiceConfigBuilder) SetNotRegister(notRegister bool) *ServiceConfigBuilder {
	pcb.serviceConfig.NotRegister = notRegister
	return pcb
}

func (pcb *ServiceConfigBuilder) Build() *ServiceConfig {
	return pcb.serviceConfig
}
