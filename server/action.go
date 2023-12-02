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
	"github.com/dubbogo/gost/log/logger"
	gxnet "github.com/dubbogo/gost/net"

	perrors "github.com/pkg/errors"

	"go.uber.org/atomic"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/graceful_shutdown"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/protocolwrapper"
)

// Prefix returns dubbo.service.${InterfaceName}.
func (svcOpts *ServiceOptions) Prefix() string {
	return strings.Join([]string{constant.ServiceConfigPrefix, svcOpts.Id}, ".")
}

func (svcOpts *ServiceOptions) check() error {
	srv := svcOpts.Service
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
			return fmt.Errorf("[ServiceConfig] Cannot parse the configuration tps.limit.interval for service %svcOpts, please check your configuration", srv.Interface)
		}
		if tpsLimitInterval < 0 {
			return fmt.Errorf("[ServiceConfig] The configuration tps.limit.interval for service %svcOpts must be positive, please check your configuration", srv.Interface)
		}
	}

	if srv.TpsLimitRate != "" {
		tpsLimitRate, err := strconv.ParseInt(srv.TpsLimitRate, 0, 0)
		if err != nil {
			return fmt.Errorf("[ServiceConfig] Cannot parse the configuration tps.limit.rate for service %svcOpts, please check your configuration", srv.Interface)
		}
		if tpsLimitRate < 0 {
			return fmt.Errorf("[ServiceConfig] The configuration tps.limit.rate for service %svcOpts must be positive, please check your configuration", srv.Interface)
		}
	}
	return nil
}

// InitExported will set exported as false atom bool
func (svcOpts *ServiceOptions) InitExported() {
	svcOpts.exported = atomic.NewBool(false)
}

// IsExport will return whether the service config is exported or not
func (svcOpts *ServiceOptions) IsExport() bool {
	return svcOpts.exported.Load()
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

func (svcOpts *ServiceOptions) ExportWithoutInfo() error {
	return svcOpts.export(nil)
}

func (svcOpts *ServiceOptions) ExportWithInfo(info *ServiceInfo) error {
	return svcOpts.export(info)
}

func (svcOpts *ServiceOptions) export(info *ServiceInfo) error {
	svc := svcOpts.Service

	if info != nil {
		if svc.Interface == "" {
			svc.Interface = info.InterfaceName
		}
		svcOpts.Id = info.InterfaceName
		svcOpts.info = info
	}
	// TODO: delay needExport
	if svcOpts.unexported != nil && svcOpts.unexported.Load() {
		err := perrors.Errorf("The service %v has already unexported!", svc.Interface)
		logger.Errorf(err.Error())
		return err
	}
	if svcOpts.exported != nil && svcOpts.exported.Load() {
		logger.Warnf("The service %v has already exported!", svc.Interface)
		return nil
	}

	regUrls := make([]*common.URL, 0)
	if !svc.NotRegister {
		regUrls = config.LoadRegistries(svc.RegistryIDs, svcOpts.registriesCompat, common.PROVIDER)
	}

	urlMap := svcOpts.getUrlMap()
	protocolConfigs := loadProtocol(svc.ProtocolIDs, svcOpts.protocolsCompat)
	if len(protocolConfigs) == 0 {
		logger.Warnf("The service %v'svcOpts '%v' protocols don't has right protocolConfigs, Please check your configuration center and transfer protocol ", svc.Interface, svc.ProtocolIDs)
		return nil
	}

	var invoker protocol.Invoker
	ports := getRandomPort(protocolConfigs)
	nextPort := ports.Front()
	proxyFactory := extension.GetProxyFactory(svcOpts.ProxyFactoryKey)
	for _, proto := range protocolConfigs {
		// *important* Register should have been replaced by processing of ServiceInfo.
		// but many modules like metadata need to make use of information from ServiceMap.
		// todo(DMwangnimg): finish replacing procedure

		// registry the service reflect
		methods, err := common.ServiceMap.Register(svc.Interface, proto.Name, svc.Group, svc.Version, svcOpts.rpcService)
		if err != nil {
			formatErr := perrors.Errorf("The service %v needExport the protocol %v error! Error message is %v.",
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
			common.WithParamsValue(constant.BeanNameKey, svcOpts.Id),
			//common.WithParamsValue(constant.SslEnabledKey, strconv.FormatBool(config.GetSslEnabled())),
			common.WithMethods(strings.Split(methods, ",")),
			// todo(DMwangnima): remove this
			common.WithAttribute(constant.ServiceInfoKey, info),
			common.WithToken(svc.Token),
			common.WithParamsValue(constant.MetadataTypeKey, svcOpts.metadataType),
			// fix https://github.com/apache/dubbo-go/issues/2176
			common.WithParamsValue(constant.MaxServerSendMsgSize, proto.MaxServerSendMsgSize),
			common.WithParamsValue(constant.MaxServerRecvMsgSize, proto.MaxServerRecvMsgSize),
		)
		if len(svc.Tag) > 0 {
			ivkURL.AddParam(constant.Tagkey, svc.Tag)
		}

		// post process the URL to be exported
		svcOpts.postProcessConfig(ivkURL)
		// config post processor may set "needExport" to false
		if !ivkURL.GetParamBool(constant.ExportKey, true) {
			return nil
		}

		if len(regUrls) > 0 {
			svcOpts.cacheMutex.Lock()
			if svcOpts.cacheProtocol == nil {
				logger.Debugf(fmt.Sprintf("First load the registry protocol, url is {%v}!", ivkURL))
				svcOpts.cacheProtocol = extension.GetProtocol(constant.RegistryProtocol)
			}
			svcOpts.cacheMutex.Unlock()

			for _, regUrl := range regUrls {
				setRegistrySubURL(ivkURL, regUrl)
				if info == nil {
					invoker = proxyFactory.GetInvoker(regUrl)
				} else {
					invoker = newInfoInvoker(regUrl, info, svcOpts.rpcService)
				}
				exporter := svcOpts.cacheProtocol.Export(invoker)
				if exporter == nil {
					return perrors.New(fmt.Sprintf("Registry protocol new exporter error, registry is {%v}, url is {%v}", regUrl, ivkURL))
				}
				svcOpts.exporters = append(svcOpts.exporters, exporter)
			}
		} else {
			if ivkURL.GetParam(constant.InterfaceKey, "") == constant.MetadataServiceName {
				ms, err := extension.GetLocalMetadataService("")
				if err != nil {
					logger.Warnf("needExport org.apache.dubbo.metadata.MetadataService failed beacause of %svcOpts ! pls check if you import _ \"dubbo.apache.org/dubbo-go/v3/metadata/service/local\"", err)
					return nil
				}
				if err := ms.SetMetadataServiceURL(ivkURL); err != nil {
					logger.Warnf("SetMetadataServiceURL error = %svcOpts", err)
				}
			}
			if info == nil {
				invoker = proxyFactory.GetInvoker(ivkURL)
			} else {
				invoker = newInfoInvoker(ivkURL, info, svcOpts.rpcService)
			}
			exporter := extension.GetProtocol(protocolwrapper.FILTER).Export(invoker)
			if exporter == nil {
				return perrors.New(fmt.Sprintf("Filter protocol without registry new exporter error, url is {%v}", ivkURL))
			}
			svcOpts.exporters = append(svcOpts.exporters, exporter)
		}
		publishServiceDefinition(ivkURL)
		// this protocol would be destroyed in graceful_shutdown
		// please refer to (https://github.com/apache/dubbo-go/issues/2429)
		graceful_shutdown.RegisterProtocol(proto.Name)
	}
	svcOpts.exported.Store(true)
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
func (svcOpts *ServiceOptions) Unexport() {
	if !svcOpts.exported.Load() {
		return
	}
	if svcOpts.unexported.Load() {
		return
	}

	func() {
		svcOpts.exportersLock.Lock()
		defer svcOpts.exportersLock.Unlock()
		for _, exporter := range svcOpts.exporters {
			exporter.UnExport()
		}
		svcOpts.exporters = nil
	}()

	svcOpts.exported.Store(false)
	svcOpts.unexported.Store(true)
}

// Implement only store the @s and return
func (svcOpts *ServiceOptions) Implement(rpcService common.RPCService) {
	svcOpts.rpcService = rpcService
}

func (svcOpts *ServiceOptions) getUrlMap() url.Values {
	srv := svcOpts.Service
	app := svcOpts.applicationCompat
	metrics := svcOpts.srvOpts.Metrics
	tracing := svcOpts.srvOpts.Otel.TracingConfig

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

	//filter
	var filters string
	if srv.Filter == "" {
		filters = constant.DefaultServiceFilters
	} else {
		filters = srv.Filter
	}
	if svcOpts.adaptiveService {
		filters += fmt.Sprintf(",%s", constant.AdaptiveServiceProviderFilterKey)
	}
	if metrics.Enable != nil && *metrics.Enable {
		filters += fmt.Sprintf(",%s", constant.MetricsFilterKey)
	}
	if tracing.Enable != nil && *tracing.Enable {
		filters += fmt.Sprintf(",%s", constant.OTELServerTraceKey)
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

	// whether to needExport or not
	urlMap.Set(constant.ExportKey, strconv.FormatBool(svcOpts.needExport))
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
func (svcOpts *ServiceOptions) GetExportedUrls() []*common.URL {
	if svcOpts.exported.Load() {
		var urls []*common.URL
		for _, exporter := range svcOpts.exporters {
			urls = append(urls, exporter.GetInvoker().GetURL())
		}
		return urls
	}
	return nil
}

// postProcessConfig asks registered ConfigPostProcessor to post-process the current ServiceConfig.
func (svcOpts *ServiceOptions) postProcessConfig(url *common.URL) {
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

// todo(DMwangnima): think about moving this function to a common place(e.g. /common/config)
func getRegistryIds(registries map[string]*global.RegistryConfig) []string {
	ids := make([]string, 0)
	for key := range registries {
		ids = append(ids, key)
	}
	return removeDuplicateElement(ids)
}

// removeDuplicateElement remove duplicate element
func removeDuplicateElement(items []string) []string {
	result := make([]string, 0, len(items))
	temp := map[string]struct{}{}
	for _, item := range items {
		if _, ok := temp[item]; !ok && item != "" {
			temp[item] = struct{}{}
			result = append(result, item)
		}
	}
	return result
}
