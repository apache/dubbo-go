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

package server

import (
	"container/list"

	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
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
	commonCfg "dubbo.apache.org/dubbo-go/v3/common/config"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/protocol/protocolwrapper"
)

// Prefix returns dubbo.service.${InterfaceName}.
func (s *ServerOptions) Prefix() string {
	return strings.Join([]string{constant.ServiceConfigPrefix, s.Id}, ".")
}

func (s *ServerOptions) Init(opts ...ServerOption) error {
	for _, opt := range opts {
		opt(s)
	}
	if err := defaults.Set(s); err != nil {
		return err
	}
	for _, opt := range opts {
		opt(s)
	}

	srv := s.Server

	s.exported = atomic.NewBool(false)

	application := s.Application
	if application != nil {
		s.applicationCompat = compatApplicationConfig(application)
		if err := s.applicationCompat.Init(); err != nil {
			return err
		}
		s.metadataType = s.applicationCompat.MetadataType
		if srv.Group == "" {
			srv.Group = s.applicationCompat.Group
		}
		if srv.Version == "" {
			srv.Version = s.applicationCompat.Version
		}
	}
	s.unexported = atomic.NewBool(false)
	err := s.check()
	if err != nil {
		panic(err)
	}
	s.Export = true
	return commonCfg.Verify(s)
}

func (s *ServerOptions) check() error {
	srv := s.Server
	// check if the limiter has been imported
	if srv.TpsLimiter != "" {
		_, err := extension.GetTpsLimiter(srv.TpsLimiter)
		if err != nil {
			panic(err)
		}
	}
	if srv.TpsLimitStrategy != "" {
		_, err := extension.GetTpsLimitStrategyCreator(srv.TpsLimitStrategy)
		if err != nil {
			panic(err)
		}
	}
	if srv.TpsLimitRejectedHandler != "" {
		_, err := extension.GetRejectedExecutionHandler(srv.TpsLimitRejectedHandler)
		if err != nil {
			panic(err)
		}
	}

	if srv.TpsLimitInterval != "" {
		tpsLimitInterval, err := strconv.ParseInt(srv.TpsLimitInterval, 0, 0)
		if err != nil {
			return fmt.Errorf("[ServiceConfig] Cannot parse the configuration tps.limit.interval for service %s, please check your configuration", srv.Interface)
		}
		if tpsLimitInterval < 0 {
			return fmt.Errorf("[ServiceConfig] The configuration tps.limit.interval for service %s must be positive, please check your configuration", srv.Interface)
		}
	}

	if srv.TpsLimitRate != "" {
		tpsLimitRate, err := strconv.ParseInt(srv.TpsLimitRate, 0, 0)
		if err != nil {
			return fmt.Errorf("[ServiceConfig] Cannot parse the configuration tps.limit.rate for service %s, please check your configuration", srv.Interface)
		}
		if tpsLimitRate < 0 {
			return fmt.Errorf("[ServiceConfig] The configuration tps.limit.rate for service %s must be positive, please check your configuration", srv.Interface)
		}
	}
	return nil
}

// InitExported will set exported as false atom bool
func (s *ServerOptions) InitExported() {
	s.exported = atomic.NewBool(false)
}

// IsExport will return whether the service config is exported or not
func (s *ServerOptions) IsExport() bool {
	return s.exported.Load()
}

// Get Random Port
func getRandomPort(protocolConfigs []*config.ProtocolConfig) *list.List {
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

func (s *ServerOptions) ExportWithoutInfo() error {
	return s.export(nil)
}

func (s *ServerOptions) ExportWithInfo(info *ServiceInfo) error {
	return s.export(info)
}

func (s *ServerOptions) export(info *ServiceInfo) error {
	srv := s.Server

	var methodInfos []MethodInfo
	var methods string
	if info != nil {
		srv.Interface = info.InterfaceName
		methodInfos = info.Methods
		s.Id = info.InterfaceName
		s.info = info
	}
	// TODO: delay export
	if s.unexported != nil && s.unexported.Load() {
		err := perrors.Errorf("The service %v has already unexported!", srv.Interface)
		logger.Errorf(err.Error())
		return err
	}
	if s.exported != nil && s.exported.Load() {
		logger.Warnf("The service %v has already exported!", srv.Interface)
		return nil
	}

	regUrls := make([]*common.URL, 0)
	if !srv.NotRegister {
		regUrls = config.LoadRegistries(srv.RegistryIDs, s.registriesCompat, common.PROVIDER)
	}

	urlMap := s.getUrlMap()
	protocolConfigs := loadProtocol(srv.ProtocolIDs, s.protocolCompat)
	if len(protocolConfigs) == 0 {
		logger.Warnf("The service %v's '%v' protocols don't has right protocolConfigs, Please check your configuration center and transfer protocol ", srv.Interface, srv.ProtocolIDs)
		return nil
	}

	ports := getRandomPort(protocolConfigs)
	nextPort := ports.Front()
	proxyFactory := extension.GetProxyFactory(s.ProxyFactoryKey)
	for _, proto := range protocolConfigs {
		if info != nil {
			for _, info := range methodInfos {
				methods += info.Name + ","
			}
		} else {
			// registry the service reflect
			var err error
			methods, err = common.ServiceMap.Register(srv.Interface, proto.Name, srv.Group, srv.Version, s.rpcService)
			if err != nil {
				formatErr := perrors.Errorf("The service %v export the protocol %v error! Error message is %v.",
					srv.Interface, proto.Name, err.Error())
				logger.Errorf(formatErr.Error())
				return formatErr
			}
		}

		port := proto.Port
		if len(proto.Port) == 0 {
			port = nextPort.Value.(string)
			nextPort = nextPort.Next()
		}
		ivkURL := common.NewURLWithOptions(
			common.WithPath(srv.Interface),
			common.WithProtocol(proto.Name),
			common.WithIp(proto.Ip),
			common.WithPort(port),
			common.WithParams(urlMap),
			common.WithParamsValue(constant.BeanNameKey, s.Id),
			//common.WithParamsValue(constant.SslEnabledKey, strconv.FormatBool(config.GetSslEnabled())),
			common.WithMethods(strings.Split(methods, ",")),
			common.WithMethodInfos(methodInfos),
			common.WithToken(srv.Token),
			common.WithParamsValue(constant.MetadataTypeKey, s.metadataType),
			// fix https://github.com/apache/dubbo-go/issues/2176
			common.WithParamsValue(constant.MaxServerSendMsgSize, proto.MaxServerSendMsgSize),
			common.WithParamsValue(constant.MaxServerRecvMsgSize, proto.MaxServerRecvMsgSize),
		)
		if len(srv.Tag) > 0 {
			ivkURL.AddParam(constant.Tagkey, srv.Tag)
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
				s.cacheProtocol = extension.GetProtocol(constant.RegistryProtocol)
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

// setRegistrySubURL set registry sub url is ivkURl
func setRegistrySubURL(ivkURL *common.URL, regUrl *common.URL) {
	ivkURL.AddParam(constant.RegistryKey, regUrl.GetParam(constant.RegistryKey, ""))
	regUrl.SubURL = ivkURL
}

// loadProtocol filter protocols by ids
func loadProtocol(protocolIds []string, protocols map[string]*config.ProtocolConfig) []*config.ProtocolConfig {
	returnProtocols := make([]*config.ProtocolConfig, 0, len(protocols))
	for _, v := range protocolIds {
		for k, config := range protocols {
			if v == k {
				returnProtocols = append(returnProtocols, config)
			}
		}
	}
	return returnProtocols
}

// Unexport will call unexport of all exporters service config exported
func (s *ServerOptions) Unexport() {
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
			exporter.UnExport()
		}
		s.exporters = nil
	}()

	s.exported.Store(false)
	s.unexported.Store(true)
}

// Implement only store the @s and return
func (s *ServerOptions) Implement(rpcService common.RPCService) {
	s.rpcService = rpcService
}

func (s *ServerOptions) getUrlMap() url.Values {
	srv := s.Server
	app := s.applicationCompat

	urlMap := url.Values{}
	// first set user params
	for k, v := range srv.Params {
		urlMap.Set(k, v)
	}
	urlMap.Set(constant.InterfaceKey, srv.Interface)
	urlMap.Set(constant.TimestampKey, strconv.FormatInt(time.Now().Unix(), 10))
	urlMap.Set(constant.ClusterKey, srv.Cluster)
	urlMap.Set(constant.LoadbalanceKey, srv.Loadbalance)
	urlMap.Set(constant.WarmupKey, srv.Warmup)
	urlMap.Set(constant.RetriesKey, srv.Retries)
	if srv.Group != "" {
		urlMap.Set(constant.GroupKey, srv.Group)
	}
	if srv.Version != "" {
		urlMap.Set(constant.VersionKey, srv.Version)
	}
	urlMap.Set(constant.RegistryRoleKey, strconv.Itoa(common.PROVIDER))
	urlMap.Set(constant.ReleaseKey, "dubbo-golang-"+constant.Version)
	urlMap.Set(constant.SideKey, (common.RoleType(common.PROVIDER)).Role())
	// todo: move
	urlMap.Set(constant.SerializationKey, srv.Serialization)
	// application config info
	urlMap.Set(constant.ApplicationKey, app.Name)
	urlMap.Set(constant.OrganizationKey, app.Organization)
	urlMap.Set(constant.NameKey, app.Name)
	urlMap.Set(constant.ModuleKey, app.Module)
	urlMap.Set(constant.AppVersionKey, app.Version)
	urlMap.Set(constant.OwnerKey, app.Owner)
	urlMap.Set(constant.EnvironmentKey, app.Environment)

	// filter
	var filters string
	if srv.Filter == "" {
		filters = constant.DefaultServiceFilters
	} else {
		filters = srv.Filter
	}
	if s.adaptiveService {
		filters += fmt.Sprintf(",%s", constant.AdaptiveServiceProviderFilterKey)
	}
	urlMap.Set(constant.ServiceFilterKey, filters)

	// filter special config
	urlMap.Set(constant.AccessLogFilterKey, srv.AccessLog)
	// tps limiter
	urlMap.Set(constant.TPSLimitStrategyKey, srv.TpsLimitStrategy)
	urlMap.Set(constant.TPSLimitIntervalKey, srv.TpsLimitInterval)
	urlMap.Set(constant.TPSLimitRateKey, srv.TpsLimitRate)
	urlMap.Set(constant.TPSLimiterKey, srv.TpsLimiter)
	urlMap.Set(constant.TPSRejectedExecutionHandlerKey, srv.TpsLimitRejectedHandler)
	urlMap.Set(constant.TracingConfigKey, srv.TracingKey)

	// execute limit filter
	urlMap.Set(constant.ExecuteLimitKey, srv.ExecuteLimit)
	urlMap.Set(constant.ExecuteRejectedExecutionHandlerKey, srv.ExecuteLimitRejectedHandler)

	// auth filter
	urlMap.Set(constant.ServiceAuthKey, srv.Auth)
	urlMap.Set(constant.ParameterSignatureEnableKey, srv.ParamSign)

	// whether to export or not
	urlMap.Set(constant.ExportKey, strconv.FormatBool(s.Export))
	urlMap.Set(constant.PIDKey, fmt.Sprintf("%d", os.Getpid()))

	for _, v := range srv.Methods {
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
func (s *ServerOptions) GetExportedUrls() []*common.URL {
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
func (s *ServerOptions) postProcessConfig(url *common.URL) {
	for _, p := range extension.GetConfigPostProcessors() {
		p.PostProcessServiceConfig(url)
	}
}

func publishServiceDefinition(url *common.URL) {
	localService, err := extension.GetLocalMetadataService(constant.DefaultKey)
	if err != nil {
		logger.Warnf("get local metadata service failed, please check if you have imported _ \"dubbo.apache.org/dubbo-go/v3/metadata/service/local\"")
		return
	}
	localService.PublishServiceDefinition(url)
	if url.GetParam(constant.MetadataTypeKey, "") != constant.RemoteMetadataStorageType {
		return
	}
	if remoteMetadataService, err := extension.GetRemoteMetadataService(); err == nil && remoteMetadataService != nil {
		remoteMetadataService.PublishServiceDefinition(url)
	}
}
