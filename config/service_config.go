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
	ProtocolIDs                 []string          `default:"[\"dubbo\"]"  validate:"required"  yaml:"protocolIDs"  json:"protocolIDs,omitempty" property:"protocolIDs"` // multi protocolIDs support, split by ','
	Interface                   string            `validate:"required"  yaml:"interface"  json:"interface,omitempty" property:"interface"`
	RegistryIDs                 []string          `yaml:"registryIDs"  json:"registryIDs,omitempty"  property:"registryIDs"`
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

	RCProtocolsMap  map[string]*ProtocolConfig
	RCRegistriesMap map[string]*RegistryConfig
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
	svc.RCRegistriesMap = rc.Registries
	svc.RCProtocolsMap = rc.Protocols
	if rc.Provider != nil {
		svc.ProxyFactoryKey = rc.Provider.ProxyFactory
	}
	svc.RegistryIDs = translateRegistryIds(svc.RegistryIDs)
	if len(svc.RegistryIDs) <= 0 {
		svc.RegistryIDs = rc.Provider.RegistryIDs
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

	regUrls := loadRegistries(svc.RegistryIDs, svc.RCRegistriesMap, common.PROVIDER)
	urlMap := svc.getUrlMap()
	protocolConfigs := loadProtocol(svc.ProtocolIDs, svc.RCProtocolsMap)
	if len(protocolConfigs) == 0 {
		logger.Warnf("The service %v's '%v' protocols don't has right protocolConfigs, Please check your configuration center and transfer protocol ", svc.Interface, svc.ProtocolIDs)
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
					logger.Warnf("export org.apache.dubbo.metadata.MetadataService failed beacause of %s ! pls check if you import _ \"dubbo.apache.org/dubbo-go/v3/metadata/service/local\"", err)
					return nil
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
	ac := GetApplicationConfig()
	urlMap.Set(constant.APPLICATION_KEY, ac.Name)
	urlMap.Set(constant.ORGANIZATION_KEY, ac.Organization)
	urlMap.Set(constant.NAME_KEY, ac.Name)
	urlMap.Set(constant.MODULE_KEY, ac.Module)
	urlMap.Set(constant.APP_VERSION_KEY, ac.Version)
	urlMap.Set(constant.OWNER_KEY, ac.Owner)
	urlMap.Set(constant.ENVIRONMENT_KEY, ac.Environment)

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

// postProcessConfig asks registered ConfigPostProcessor to post-process the current ServiceConfig.
func (svc *ServiceConfig) postProcessConfig(url *common.URL) {
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

func (pcb *ServiceConfigBuilder) Build() *ServiceConfig {
	return pcb.serviceConfig
}
