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
	"dubbo.apache.org/dubbo-go/v3/protocol/protocolwrapper"
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

	Protocols     map[string]*ProtocolConfig
	unexported    *atomic.Bool
	exported      *atomic.Bool
	export        bool // a flag to control whether the current service should export or not
	rpcService    common.RPCService
	cacheMutex    sync.Mutex
	cacheProtocol protocol.Protocol

	exportersLock sync.Mutex
	exporters     []protocol.Exporter

	rootConfig *RootConfig
}

// Prefix returns dubbo.service.${InterfaceName}.
func (c *ServiceConfig) Prefix() string {
	return constant.ServiceConfigPrefix + c.id
}

// UnmarshalYAML unmarshal the ServiceConfig by @unmarshal function
func (c *ServiceConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	if err := defaults.Set(c); err != nil {
		return err
	}
	type plain ServiceConfig
	if err := unmarshal((*plain)(c)); err != nil {
		return err
	}
	c.exported = atomic.NewBool(false)
	c.unexported = atomic.NewBool(false)
	c.export = true
	return nil
}

func (c *ServiceConfig) CheckConfig() error {
	// todo check
	defaults.MustSet(c)
	return verify(c)
}

func (c *ServiceConfig) Validate(rootConfig *RootConfig) {
	c.rootConfig = rootConfig
	c.exported = atomic.NewBool(false)
	c.unexported = atomic.NewBool(false)
	c.export = true
	// todo set default application
}

//getRegistryServices get registry services
func getRegistryServices(side int, services map[string]*ServiceConfig, registryIds []string) map[string]*ServiceConfig {
	var (
		svc              *ServiceConfig
		exist            bool
		initService      map[string]common.RPCService
		registryServices map[string]*ServiceConfig
	)
	if side == common.PROVIDER {
		initService = proServices
	} else if side == common.CONSUMER {
		initService = conServices
	}
	registryServices = make(map[string]*ServiceConfig, len(initService))
	for key := range initService {
		//存在配置了使用用户的配置
		if svc, exist = services[key]; !exist {
			svc = new(ServiceConfig)
		}
		defaults.MustSet(svc)
		if len(svc.Registry) <= 0 {
			svc.Registry = registryIds
		}
		svc.id = key
		svc.export = true
		svc.unexported = atomic.NewBool(false)
		svc.exported = atomic.NewBool(false)
		svc.Registry = translateRegistryIds(svc.Registry)
		if err := verify(svc); err != nil {
			return nil
		}
		registryServices[key] = svc
	}
	return registryServices
}

// InitExported will set exported as false atom bool
func (c *ServiceConfig) InitExported() {
	c.exported = atomic.NewBool(false)
}

// IsExport will return whether the service config is exported or not
func (c *ServiceConfig) IsExport() bool {
	return c.exported.Load()
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
func (c *ServiceConfig) Export() error {
	// TODO: config center start here

	// TODO: delay export
	if c.unexported != nil && c.unexported.Load() {
		err := perrors.Errorf("The service %v has already unexported!", c.Interface)
		logger.Errorf(err.Error())
		return err
	}
	if c.unexported != nil && c.exported.Load() {
		logger.Warnf("The service %v has already exported!", c.Interface)
		return nil
	}

	regUrls := loadRegistries(c.Registry, c.rootConfig.Registries, common.PROVIDER)
	urlMap := c.getUrlMap()
	protocolConfigs := loadProtocol(c.Protocol, c.rootConfig.Protocols)
	if len(protocolConfigs) == 0 {
		logger.Warnf("The service %v's '%v' protocols don't has right protocolConfigs", c.Interface, c.Protocol)
		return nil
	}

	ports := getRandomPort(protocolConfigs)
	nextPort := ports.Front()
	proxyFactory := extension.GetProxyFactory(c.rootConfig.Provider.ProxyFactory)
	for _, proto := range protocolConfigs {
		// registry the service reflect
		methods, err := common.ServiceMap.Register(c.Interface, proto.Name, c.Group, c.Version, c.rpcService)
		if err != nil {
			formatErr := perrors.Errorf("The service %v export the protocol %v error! Error message is %v.",
				c.Interface, proto.Name, err.Error())
			logger.Errorf(formatErr.Error())
			return formatErr
		}

		port := proto.Port
		if len(proto.Port) == 0 {
			port = nextPort.Value.(string)
			nextPort = nextPort.Next()
		}
		ivkURL := common.NewURLWithOptions(
			common.WithPath(c.Interface),
			common.WithProtocol(proto.Name),
			common.WithIp(proto.Ip),
			common.WithPort(port),
			common.WithParams(urlMap),
			common.WithParamsValue(constant.BEAN_NAME_KEY, c.id),
			//common.WithParamsValue(constant.SSL_ENABLED_KEY, strconv.FormatBool(config.GetSslEnabled())),
			common.WithMethods(strings.Split(methods, ",")),
			common.WithToken(c.Token),
		)
		if len(c.Tag) > 0 {
			ivkURL.AddParam(constant.Tagkey, c.Tag)
		}

		// post process the URL to be exported
		c.postProcessConfig(ivkURL)
		// config post processor may set "export" to false
		if !ivkURL.GetParamBool(constant.EXPORT_KEY, true) {
			return nil
		}

		if len(regUrls) > 0 {
			c.cacheMutex.Lock()
			if c.cacheProtocol == nil {
				logger.Infof(fmt.Sprintf("First load the registry protocol, url is {%v}!", ivkURL))
				c.cacheProtocol = extension.GetProtocol("registry")
			}
			c.cacheMutex.Unlock()

			for _, regUrl := range regUrls {
				regUrl.SubURL = ivkURL
				invoker := proxyFactory.GetInvoker(regUrl)
				exporter := c.cacheProtocol.Export(invoker)
				if exporter == nil {
					return perrors.New(fmt.Sprintf("Registry protocol new exporter error, registry is {%v}, url is {%v}", regUrl, ivkURL))
				}
				c.exporters = append(c.exporters, exporter)
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
			c.exporters = append(c.exporters, exporter)
		}
		publishServiceDefinition(ivkURL)
	}
	c.exported.Store(true)
	return nil
}

//loadProtocol filter protocols by ids
func loadProtocol(protocolIds []string, protocols map[string]*ProtocolConfig) []*ProtocolConfig {
	returnProtocols := make([]*ProtocolConfig, 0, len(protocols))
	for _, v := range protocolIds {
		for k, protocol := range protocols {
			if v == k {
				returnProtocols = append(returnProtocols, protocol)
			}
		}
	}
	return returnProtocols
}

func loadRegistries(registryIds []string, registries map[string]*RegistryConfig, roleType common.RoleType) []*common.URL {
	var urls []*common.URL
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
			addresses := strings.Split(registryConf.Address, ",")
			address := addresses[0]
			address = registryConf.translateRegistryAddress()
			url, err := common.NewURL(constant.REGISTRY_PROTOCOL+"://"+address,
				common.WithParams(registryConf.getUrlMap(roleType)),
				common.WithParamsValue("simplified", strconv.FormatBool(registryConf.Simplified)),
				common.WithUsername(registryConf.Username),
				common.WithPassword(registryConf.Password),
				common.WithLocation(registryConf.Address),
			)

			if err != nil {
				logger.Errorf("The registry id: %s url is invalid, error: %#v", k, err)
				panic(err)
			} else {
				urls = append(urls, url)
			}
		}
	}

	return urls
}

// Unexport will call unexport of all exporters service config exported
func (c *ServiceConfig) Unexport() {
	if !c.exported.Load() {
		return
	}
	if c.unexported.Load() {
		return
	}

	func() {
		c.exportersLock.Lock()
		defer c.exportersLock.Unlock()
		for _, exporter := range c.exporters {
			exporter.Unexport()
		}
		c.exporters = nil
	}()

	c.exported.Store(false)
	c.unexported.Store(true)
}

// Implement only store the @s and return
func (c *ServiceConfig) Implement(s common.RPCService) {
	c.rpcService = s
}

func (c *ServiceConfig) getUrlMap() url.Values {
	urlMap := url.Values{}
	// first set user params
	for k, v := range c.Params {
		urlMap.Set(k, v)
	}
	urlMap.Set(constant.INTERFACE_KEY, c.Interface)
	urlMap.Set(constant.TIMESTAMP_KEY, strconv.FormatInt(time.Now().Unix(), 10))
	urlMap.Set(constant.CLUSTER_KEY, c.Cluster)
	urlMap.Set(constant.LOADBALANCE_KEY, c.Loadbalance)
	urlMap.Set(constant.WARMUP_KEY, c.Warmup)
	urlMap.Set(constant.RETRIES_KEY, c.Retries)
	urlMap.Set(constant.GROUP_KEY, c.Group)
	urlMap.Set(constant.VERSION_KEY, c.Version)
	urlMap.Set(constant.ROLE_KEY, strconv.Itoa(common.PROVIDER))
	urlMap.Set(constant.RELEASE_KEY, "dubbo-golang-"+constant.Version)
	urlMap.Set(constant.SIDE_KEY, (common.RoleType(common.PROVIDER)).Role())
	urlMap.Set(constant.MESSAGE_SIZE_KEY, strconv.Itoa(c.GrpcMaxMessageSize))
	// todo: move
	urlMap.Set(constant.SERIALIZATION_KEY, c.Serialization)
	// application config info
	//urlMap.Set(constant.APPLICATION_KEY, applicationConfig.Name)
	//urlMap.Set(constant.ORGANIZATION_KEY, applicationConfig.Organization)
	//urlMap.Set(constant.NAME_KEY, applicationConfig.Name)
	//urlMap.Set(constant.MODULE_KEY, applicationConfig.Module)
	//urlMap.Set(constant.APP_VERSION_KEY, applicationConfig.Version)
	//urlMap.Set(constant.OWNER_KEY, applicationConfig.Owner)
	//urlMap.Set(constant.ENVIRONMENT_KEY, applicationConfig.Environment)

	// filter
	urlMap.Set(constant.SERVICE_FILTER_KEY, mergeValue(c.rootConfig.Provider.Filter, c.Filter, constant.DEFAULT_SERVICE_FILTERS))

	// filter special config
	urlMap.Set(constant.AccessLogFilterKey, c.AccessLog)
	// tps limiter
	urlMap.Set(constant.TPS_LIMIT_STRATEGY_KEY, c.TpsLimitStrategy)
	urlMap.Set(constant.TPS_LIMIT_INTERVAL_KEY, c.TpsLimitInterval)
	urlMap.Set(constant.TPS_LIMIT_RATE_KEY, c.TpsLimitRate)
	urlMap.Set(constant.TPS_LIMITER_KEY, c.TpsLimiter)
	urlMap.Set(constant.TPS_REJECTED_EXECUTION_HANDLER_KEY, c.TpsLimitRejectedHandler)

	// execute limit filter
	urlMap.Set(constant.EXECUTE_LIMIT_KEY, c.ExecuteLimit)
	urlMap.Set(constant.EXECUTE_REJECTED_EXECUTION_HANDLER_KEY, c.ExecuteLimitRejectedHandler)

	// auth filter
	urlMap.Set(constant.SERVICE_AUTH_KEY, c.Auth)
	urlMap.Set(constant.PARAMETER_SIGNATURE_ENABLE_KEY, c.ParamSign)

	// whether to export or not
	urlMap.Set(constant.EXPORT_KEY, strconv.FormatBool(c.export))

	for _, v := range c.Methods {
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
func (c *ServiceConfig) GetExportedUrls() []*common.URL {
	if c.exported.Load() {
		var urls []*common.URL
		for _, exporter := range c.exporters {
			urls = append(urls, exporter.GetInvoker().GetURL())
		}
		return urls
	}
	return nil
}

func publishServiceDefinition(url *common.URL) {
	if remoteMetadataService, err := extension.GetRemoteMetadataService(); err == nil && remoteMetadataService != nil {
		remoteMetadataService.PublishServiceDefinition(url)

	}
}

// postProcessConfig asks registered ConfigPostProcessor to post-process the current ServiceConfig.
func (c *ServiceConfig) postProcessConfig(url *common.URL) {
	for _, p := range extension.GetConfigPostProcessors() {
		p.PostProcessServiceConfig(url)
	}
}

// NewServiceConfig The only way to get a new ServiceConfig
func NewServiceConfig(id string) *ServiceConfig {
	return &ServiceConfig{
		id:         id,
		unexported: atomic.NewBool(false),
		exported:   atomic.NewBool(false),
		export:     true,
	}
}

// ServiceConfigOpt is the option to init ServiceConfig
type ServiceConfigOpt func(config *ServiceConfig) *ServiceConfig

// NewDefaultServiceConfig returns default ServiceConfig
func NewDefaultServiceConfig() *ServiceConfig {
	newServiceConfig := NewServiceConfig("")
	newServiceConfig.Params = make(map[string]string)
	newServiceConfig.Methods = make([]*MethodConfig, 0, 8)
	return newServiceConfig
}

// NewServiceConfigByAPI is named as api, because there is NewServiceConfig func already declared
// NewServiceConfigByAPI returns ServiceConfig with given @opts
func NewServiceConfigByAPI(opts ...ServiceConfigOpt) *ServiceConfig {
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

// WithServiceProtocol returns ServiceConfigOpt with given protocolKey @protocol
func WithServiceProtocol(protocol string) ServiceConfigOpt {
	return func(config *ServiceConfig) *ServiceConfig {
		config.Protocol = append(config.Protocol, protocol)
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
