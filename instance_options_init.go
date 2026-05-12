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

package dubbo

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

import (
	getty "github.com/apache/dubbo-getty"

	"github.com/creasty/defaults"

	"github.com/dubbogo/gost/log/logger"

	"github.com/knadh/koanf"

	"go.opentelemetry.io/otel"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	commonCfg "dubbo.apache.org/dubbo-go/v3/common/config"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	aslimiter "dubbo.apache.org/dubbo-go/v3/filter/adaptivesvc/limiter"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/internal"
	"dubbo.apache.org/dubbo-go/v3/metrics"
	metricsConfigCenter "dubbo.apache.org/dubbo-go/v3/metrics/config_center"
	"dubbo.apache.org/dubbo-go/v3/otel/trace"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

func (rc *InstanceOptions) finalizeGlobalOptionsWithRuntimeActivation(activateRuntime bool) error {
	if err := rc.initGlobalApplication(); err != nil {
		return err
	}
	if err := rc.initGlobalCustom(); err != nil {
		return err
	}
	if err := rc.initGlobalProtocols(); err != nil {
		return err
	}
	if err := rc.initGlobalRegistries(); err != nil {
		return err
	}
	if err := rc.initGlobalMetrics(activateRuntime); err != nil {
		return err
	}
	if err := rc.initGlobalOtel(activateRuntime); err != nil {
		return err
	}
	if err := rc.initGlobalRouters(); err != nil {
		return err
	}
	for _, tracingConfig := range rc.Tracing {
		if tracingConfig == nil {
			continue
		}
		if err := defaults.Set(tracingConfig); err != nil {
			return err
		}
		if err := commonCfg.Verify(tracingConfig); err != nil {
			return err
		}
	}
	if err := rc.initGlobalProvider(activateRuntime); err != nil {
		return err
	}
	if err := rc.initGlobalConsumer(); err != nil {
		return err
	}
	return rc.initGlobalShutdown()
}

func (rc *InstanceOptions) initGlobalApplication() error {
	if rc.Application == nil {
		rc.Application = global.DefaultApplicationConfig()
	}
	if err := defaults.Set(rc.Application); err != nil {
		return err
	}
	if rc.Application.Name == "" {
		rc.Application.Name = constant.DefaultDubboApp
	}
	return commonCfg.Verify(rc.Application)
}

func (rc *InstanceOptions) initGlobalCustom() error {
	if rc.Custom == nil {
		rc.Custom = global.DefaultCustomConfig()
	}
	if rc.Custom.ConfigMap == nil {
		rc.Custom.ConfigMap = make(map[string]any)
	}
	if err := defaults.Set(rc.Custom); err != nil {
		return err
	}
	return commonCfg.Verify(rc.Custom)
}

func (rc *InstanceOptions) initGlobalProtocols() error {
	if len(rc.Protocols) <= 0 {
		rc.Protocols = map[string]*global.ProtocolConfig{
			constant.TriProtocol: {},
		}
	}
	for key, protocolConfig := range rc.Protocols {
		if protocolConfig == nil {
			protocolConfig = &global.ProtocolConfig{}
			rc.Protocols[key] = protocolConfig
		}
		if err := initGlobalProtocol(protocolConfig); err != nil {
			return err
		}
	}
	return nil
}

func initGlobalProtocol(protocolConfig *global.ProtocolConfig) error {
	if err := defaults.Set(protocolConfig); err != nil {
		return err
	}
	if protocolConfig.Name == "" {
		protocolConfig.Name = constant.TriProtocol
	}
	if protocolConfig.Port == "" {
		protocolConfig.Port = constant.DefaultTripleProtocolPort
	}
	if protocolConfig.TripleConfig == nil {
		protocolConfig.TripleConfig = global.DefaultTripleConfig()
	} else {
		if protocolConfig.TripleConfig.Http3 == nil {
			protocolConfig.TripleConfig.Http3 = global.DefaultHttp3Config()
		}
		if protocolConfig.TripleConfig.Cors == nil {
			protocolConfig.TripleConfig.Cors = global.DefaultCorsConfig()
		}
		if protocolConfig.TripleConfig.OpenAPI == nil {
			protocolConfig.TripleConfig.OpenAPI = global.DefaultOpenAPIConfig()
		}
		protocolConfig.TripleConfig.OpenAPI.Init()
	}
	return commonCfg.Verify(protocolConfig)
}

func (rc *InstanceOptions) initGlobalRegistries() error {
	if rc.Registries == nil {
		rc.Registries = global.DefaultRegistriesConfig()
	}
	for _, reg := range rc.Registries {
		if reg == nil {
			continue
		}
		if reg.Params == nil {
			reg.Params = make(map[string]string)
		}
		if err := defaults.Set(reg); err != nil {
			return err
		}
		var err error
		reg.Protocol, reg.Address, err = translateGlobalAddress(reg.Protocol, reg.Address, false)
		if err != nil {
			return err
		}
		if err := commonCfg.Verify(reg); err != nil {
			return err
		}
	}
	return validateGlobalRegistryAddresses(rc.Registries)
}

func translateGlobalAddress(protocol, address string, preserveQuery bool) (string, string, error) {
	if !strings.Contains(address, "://") {
		return protocol, address, nil
	}
	u, err := url.Parse(address)
	if err != nil {
		return "", "", err
	}
	if preserveQuery {
		return u.Scheme, strings.TrimPrefix(address, u.Scheme+"://"), nil
	}
	return u.Scheme, u.Host + u.Path, nil
}

func validateGlobalRegistryAddresses(registries map[string]*global.RegistryConfig) error {
	cacheKeyMap := make(map[string]string, len(registries))
	for id, reg := range registries {
		if reg == nil {
			continue
		}
		cacheKey := reg.Address
		if reg.Namespace != "" {
			cacheKey = cacheKey + "?" + constant.NacosNamespaceID + "=" + reg.Namespace
		}
		if existingID, exists := cacheKeyMap[cacheKey]; exists {
			return fmt.Errorf("duplicate registry address: [%s] used by both [%s] and [%s]", cacheKey, existingID, id)
		}
		cacheKeyMap[cacheKey] = id
	}
	return nil
}

func (rc *InstanceOptions) initGlobalMetrics(activateRuntime bool) error {
	if rc.Metrics == nil {
		rc.Metrics = global.DefaultMetricsConfig()
	}
	if rc.Metrics.Prometheus == nil {
		rc.Metrics.Prometheus = &global.PrometheusConfig{}
	}
	if rc.Metrics.Prometheus.Exporter == nil {
		rc.Metrics.Prometheus.Exporter = &global.Exporter{}
	}
	if rc.Metrics.Prometheus.Pushgateway == nil {
		rc.Metrics.Prometheus.Pushgateway = &global.PushgatewayConfig{}
	}
	if rc.Metrics.Aggregation == nil {
		rc.Metrics.Aggregation = &global.AggregateConfig{}
	}
	if rc.Metrics.Probe == nil {
		rc.Metrics.Probe = &global.ProbeConfig{}
	}
	if err := defaults.Set(rc.Metrics); err != nil {
		return err
	}
	if err := commonCfg.Verify(rc.Metrics); err != nil {
		return err
	}
	if activateRuntime && rc.Metrics.Enable != nil && *rc.Metrics.Enable {
		metrics.Init(metricsURL(rc))
	}
	return nil
}

func metricsURL(rc *InstanceOptions) *common.URL {
	mc := rc.Metrics
	u, _ := common.NewURL("localhost", common.WithProtocol(mc.Protocol))
	u.SetParam(constant.PrometheusExporterMetricsPortKey, mc.Port)
	u.SetParam(constant.PrometheusExporterMetricsPathKey, mc.Path)
	if rc.Application != nil {
		u.SetParam(constant.ApplicationKey, rc.Application.Name)
		u.SetParam(constant.AppVersionKey, rc.Application.Version)
	}
	u.SetParam(constant.RpcEnabledKey, strconv.FormatBool(*mc.Enable))
	u.SetParam(constant.MetadataEnabledKey, strconv.FormatBool(*mc.EnableMetadata))
	u.SetParam(constant.RegistryEnabledKey, strconv.FormatBool(*mc.EnableRegistry))
	u.SetParam(constant.ConfigCenterEnabledKey, strconv.FormatBool(*mc.EnableConfigCenter))
	if mc.Aggregation != nil {
		u.SetParam(constant.AggregationEnabledKey, strconv.FormatBool(*mc.Aggregation.Enabled))
		u.SetParam(constant.AggregationBucketNumKey, strconv.Itoa(mc.Aggregation.BucketNum))
		u.SetParam(constant.AggregationTimeWindowSecondsKey, strconv.Itoa(mc.Aggregation.TimeWindowSeconds))
	}
	if mc.Prometheus != nil {
		if mc.Prometheus.Exporter != nil {
			u.SetParam(constant.PrometheusExporterEnabledKey, strconv.FormatBool(*mc.Prometheus.Exporter.Enabled))
		}
		if mc.Prometheus.Pushgateway != nil {
			pushGateway := mc.Prometheus.Pushgateway
			u.SetParam(constant.PrometheusPushgatewayEnabledKey, strconv.FormatBool(*pushGateway.Enabled))
			u.SetParam(constant.PrometheusPushgatewayBaseUrlKey, pushGateway.BaseUrl)
			u.SetParam(constant.PrometheusPushgatewayUsernameKey, pushGateway.Username)
			u.SetParam(constant.PrometheusPushgatewayPasswordKey, pushGateway.Password)
			u.SetParam(constant.PrometheusPushgatewayPushIntervalKey, strconv.Itoa(pushGateway.PushInterval))
			u.SetParam(constant.PrometheusPushgatewayJobKey, pushGateway.Job)
		}
	}
	return u
}

func (rc *InstanceOptions) initGlobalOtel(activateRuntime bool) error {
	if rc.Otel == nil {
		rc.Otel = global.DefaultOtelConfig()
	}
	if rc.Otel.TracingConfig == nil {
		rc.Otel.TracingConfig = &global.OtelTraceConfig{}
	}
	if err := defaults.Set(rc.Otel); err != nil {
		return err
	}
	if err := commonCfg.Verify(rc.Otel); err != nil {
		return err
	}
	if rc.Otel.TracingConfig.Enable != nil && *rc.Otel.TracingConfig.Enable {
		if !activateRuntime {
			return nil
		}
		extension.AddCustomShutdownCallback(extension.GetTraceShutdownCallback())
		c := rc.Otel.TracingConfig
		serviceNamespace, serviceName, serviceVersion := "", "", ""
		if rc.Application != nil {
			serviceNamespace = rc.Application.Organization
			serviceName = rc.Application.Name
			serviceVersion = rc.Application.Version
		}
		exporter, err := extension.GetTraceExporter(c.Exporter, &trace.ExporterConfig{
			Exporter:         c.Exporter,
			Endpoint:         c.Endpoint,
			SampleMode:       c.SampleMode,
			SampleRatio:      c.SampleRatio,
			Propagator:       c.Propagator,
			Insecure:         c.Insecure,
			ServiceNamespace: serviceNamespace,
			ServiceName:      serviceName,
			ServiceVersion:   serviceVersion,
		})
		if err != nil {
			return err
		}
		otel.SetTracerProvider(exporter.GetTracerProvider())
		otel.SetTextMapPropagator(exporter.GetPropagator())

		if c.Exporter == "stdout" {
			logger.Infof("enable %s trace provider with propagator: %s", c.Exporter, c.Propagator)
		} else {
			logger.Infof("enable %s trace provider with endpoint: %s, propagator: %s", c.Exporter, c.Endpoint, c.Propagator)
		}
		logger.Infof("sample mode: %s", c.SampleMode)
		if c.SampleMode == "ratio" {
			logger.Infof("sample ratio: %.2f", c.SampleRatio)
		}
	}
	return nil
}

func (rc *InstanceOptions) initGlobalRouters() error {
	for _, routerConfig := range rc.Router {
		if routerConfig == nil {
			continue
		}
		if err := defaults.Set(routerConfig); err != nil {
			return err
		}
		if err := commonCfg.Verify(routerConfig); err != nil {
			return err
		}
	}
	return nil
}

func (rc *InstanceOptions) initGlobalProvider(activateRuntime bool) error {
	if rc.Provider == nil {
		rc.Provider = global.DefaultProviderConfig()
	}
	if rc.Provider.Services == nil {
		rc.Provider.Services = make(map[string]*global.ServiceConfig)
	}
	if err := defaults.Set(rc.Provider); err != nil {
		return err
	}
	rc.Provider.RegistryIDs = commonCfg.TranslateIds(rc.Provider.RegistryIDs)
	if len(rc.Provider.RegistryIDs) <= 0 {
		rc.Provider.RegistryIDs = globalRegistryIDs(rc.Registries)
	}
	rc.Provider.ProtocolIDs = commonCfg.TranslateIds(rc.Provider.ProtocolIDs)
	if rc.Provider.TracingKey == "" && len(rc.Tracing) > 0 {
		for key := range rc.Tracing {
			rc.Provider.TracingKey = key
			break
		}
	}

	for _, serviceConfig := range rc.Provider.Services {
		if serviceConfig == nil {
			continue
		}
		if err := rc.initGlobalService(serviceConfig); err != nil {
			return err
		}
	}
	if err := commonCfg.Verify(rc.Provider); err != nil {
		return err
	}
	if rc.Provider.AdaptiveServiceVerbose {
		if !rc.Provider.AdaptiveService {
			return fmt.Errorf("the adaptive service is disabled, adaptive service verbose should be disabled either")
		}
		if activateRuntime {
			logger.Infof("adaptive service verbose is enabled.")
			logger.Debugf("debug-level info could be shown.")
			aslimiter.Verbose = true
		}
	}
	return nil
}

func (rc *InstanceOptions) initGlobalService(serviceConfig *global.ServiceConfig) error {
	if err := defaults.Set(serviceConfig); err != nil {
		return err
	}
	if serviceConfig.Filter == "" {
		serviceConfig.Filter = rc.Provider.Filter
	}
	if rc.Application != nil {
		if serviceConfig.Version == "" {
			serviceConfig.Version = rc.Application.Version
		}
		if serviceConfig.Group == "" {
			serviceConfig.Group = rc.Application.Group
		}
	}
	if len(serviceConfig.RCRegistriesMap) == 0 {
		serviceConfig.RCRegistriesMap = rc.Registries
	}
	if len(serviceConfig.RCProtocolsMap) == 0 {
		serviceConfig.RCProtocolsMap = rc.Protocols
	}
	serviceConfig.ProxyFactoryKey = rc.Provider.ProxyFactory
	serviceConfig.RegistryIDs = commonCfg.TranslateIds(serviceConfig.RegistryIDs)
	if len(serviceConfig.RegistryIDs) <= 0 {
		serviceConfig.RegistryIDs = rc.Provider.RegistryIDs
	}
	serviceConfig.ProtocolIDs = commonCfg.TranslateIds(serviceConfig.ProtocolIDs)
	if len(serviceConfig.ProtocolIDs) <= 0 {
		serviceConfig.ProtocolIDs = rc.Provider.ProtocolIDs
	}
	if len(serviceConfig.ProtocolIDs) <= 0 {
		serviceConfig.ProtocolIDs = make([]string, 0, len(rc.Protocols))
		for key := range rc.Protocols {
			serviceConfig.ProtocolIDs = append(serviceConfig.ProtocolIDs, key)
		}
	}
	if serviceConfig.TracingKey == "" {
		serviceConfig.TracingKey = rc.Provider.TracingKey
	}
	for _, method := range serviceConfig.Methods {
		if err := internal.ValidateMethodConfig(method); err != nil {
			return err
		}
	}
	return commonCfg.Verify(serviceConfig)
}

func (rc *InstanceOptions) initGlobalConsumer() error {
	if rc.Consumer == nil {
		rc.Consumer = global.DefaultConsumerConfig()
	}
	if rc.Consumer.References == nil {
		rc.Consumer.References = make(map[string]*global.ReferenceConfig)
	}
	if err := defaults.Set(rc.Consumer); err != nil {
		return err
	}
	rc.Consumer.RegistryIDs = commonCfg.TranslateIds(rc.Consumer.RegistryIDs)
	if len(rc.Consumer.RegistryIDs) <= 0 {
		rc.Consumer.RegistryIDs = globalRegistryIDs(rc.Registries)
	}
	if rc.Consumer.TracingKey == "" && len(rc.Tracing) > 0 {
		for key := range rc.Tracing {
			rc.Consumer.TracingKey = key
			break
		}
	}
	for _, referenceConfig := range rc.Consumer.References {
		if referenceConfig == nil {
			continue
		}
		if err := rc.initGlobalReference(referenceConfig); err != nil {
			return err
		}
	}
	return commonCfg.Verify(rc.Consumer)
}

func (rc *InstanceOptions) initGlobalReference(referenceConfig *global.ReferenceConfig) error {
	if referenceConfig.ProtocolClientConfig == nil {
		referenceConfig.ProtocolClientConfig = global.DefaultClientProtocolConfig()
	}
	if err := defaults.Set(referenceConfig); err != nil {
		return err
	}
	for _, method := range referenceConfig.MethodsConfig {
		if err := internal.ValidateMethodConfig(method); err != nil {
			return err
		}
	}
	if rc.Application != nil {
		if referenceConfig.Group == "" {
			referenceConfig.Group = rc.Application.Group
		}
		if referenceConfig.Version == "" {
			referenceConfig.Version = rc.Application.Version
		}
	}
	referenceConfig.RegistryIDs = commonCfg.TranslateIds(referenceConfig.RegistryIDs)
	if referenceConfig.Filter == "" {
		referenceConfig.Filter = rc.Consumer.Filter
	}
	if len(referenceConfig.RegistryIDs) <= 0 {
		referenceConfig.RegistryIDs = rc.Consumer.RegistryIDs
	}
	if referenceConfig.Protocol == "" {
		referenceConfig.Protocol = rc.Consumer.Protocol
	}
	if referenceConfig.Protocol == "" {
		referenceConfig.Protocol = constant.TriProtocol
	}
	if referenceConfig.TracingKey == "" {
		referenceConfig.TracingKey = rc.Consumer.TracingKey
	}
	if referenceConfig.Check == nil {
		referenceConfig.Check = &rc.Consumer.Check
	}
	if referenceConfig.Cluster == "" {
		referenceConfig.Cluster = constant.ClusterKeyFailover
	}
	return commonCfg.Verify(referenceConfig)
}

func (rc *InstanceOptions) initGlobalShutdown() error {
	if rc.Shutdown == nil {
		rc.Shutdown = global.DefaultShutdownConfig()
	}
	return defaults.Set(rc.Shutdown)
}

func globalRegistryIDs(regs map[string]*global.RegistryConfig) []string {
	ids := make([]string, 0, len(regs))
	for key := range regs {
		ids = append(ids, key)
	}
	return ids
}

func (rc *InstanceOptions) initGlobalLogger() error {
	if rc.Logger == nil {
		rc.Logger = global.DefaultLoggerConfig()
	}
	if rc.Logger.File == nil {
		rc.Logger.File = &global.File{}
	}
	if err := defaults.Set(rc.Logger); err != nil {
		return err
	}
	if err := commonCfg.Verify(rc.Logger); err != nil {
		return err
	}
	log, err := extension.GetLogger(rc.Logger.Driver, loggerURL(rc.Logger))
	if err != nil {
		return err
	}
	logger.SetLogger(log)
	getty.SetLogger(log)
	return nil
}

func loggerURL(l *global.LoggerConfig) *common.URL {
	address := fmt.Sprintf("%s://%s", l.Driver, l.Level)
	u, _ := common.NewURL(address,
		common.WithParamsValue(constant.LoggerLevelKey, l.Level),
		common.WithParamsValue(constant.LoggerDriverKey, l.Driver),
		common.WithParamsValue(constant.LoggerFormatKey, l.Format),
		common.WithParamsValue(constant.LoggerAppenderKey, l.Appender),
		common.WithParamsValue(constant.LoggerFileNameKey, l.File.Name),
		common.WithParamsValue(constant.LoggerFileNaxSizeKey, strconv.Itoa(l.File.MaxSize)),
		common.WithParamsValue(constant.LoggerFileMaxBackupsKey, strconv.Itoa(l.File.MaxBackups)),
		common.WithParamsValue(constant.LoggerFileMaxAgeKey, strconv.Itoa(l.File.MaxAge)),
		common.WithParamsValue(constant.LoggerFileCompressKey, strconv.FormatBool(*l.File.Compress)),
	)

	if l.TraceIntegration != nil {
		if l.TraceIntegration.Enabled != nil {
			u.AddParam(constant.LoggerTraceEnabledKey, strconv.FormatBool(*l.TraceIntegration.Enabled))
		}
		if l.TraceIntegration.RecordErrorToSpan != nil {
			u.AddParam(constant.LoggerTraceRecordErrorKey, strconv.FormatBool(*l.TraceIntegration.RecordErrorToSpan))
		}
	}

	return u
}

func (rc *InstanceOptions) initGlobalConfigCenter() (bool, error) {
	if rc.ConfigCenter == nil {
		return false, nil
	}
	cc := rc.ConfigCenter
	if cc.Params == nil {
		cc.Params = make(map[string]string)
	}
	if err := defaults.Set(cc); err != nil {
		return false, err
	}
	var err error
	cc.Protocol, cc.Address, err = translateGlobalAddress(cc.Protocol, cc.Address, true)
	if err != nil {
		return false, err
	}
	if err := commonCfg.Verify(cc); err != nil {
		return false, err
	}
	return rc.startGlobalConfigCenter()
}

func (rc *InstanceOptions) startGlobalConfigCenter() (bool, error) {
	cc := rc.ConfigCenter
	dynamicConfig, err := getGlobalDynamicConfiguration(cc)
	if err != nil {
		logger.Errorf("[Config Center] Start dynamic configuration center error, error message is %v", err)
		return false, err
	}

	strConf, err := dynamicConfig.GetProperties(cc.DataId, config_center.WithGroup(cc.Group))
	if err != nil {
		logger.Warnf("[Config Center] Dynamic config center has started, but config may not be initialized, because: %s", err)
		return false, nil
	}
	defer metrics.Publish(metricsConfigCenter.NewIncMetricEvent(cc.DataId, cc.Group, remoting.EventTypeAdd, cc.Protocol))
	if len(strConf) == 0 {
		logger.Warnf("[Config Center] Dynamic config center has started, but got empty config with config-center configuration %+v\n"+
			"Please check if your config-center config is correct.", cc)
		return false, nil
	}
	conf := NewLoaderConf(WithDelim("."), WithGenre(cc.FileExtension), WithBytes([]byte(strConf)))
	koan := GetConfigResolver(conf)
	if err = koan.UnmarshalWithConf(rc.Prefix(), rc, koanf.UnmarshalConf{Tag: "yaml"}); err != nil {
		return false, err
	}

	dynamicConfig.AddListener(cc.DataId, rc, config_center.WithGroup(cc.Group))
	return true, nil
}

func getGlobalDynamicConfiguration(cc *global.CenterConfig) (config_center.DynamicConfiguration, error) {
	envInstance := commonCfg.GetEnvInstance()
	if envInstance.GetDynamicConfiguration() != nil {
		return envInstance.GetDynamicConfiguration(), nil
	}
	configCenterURL, err := globalConfigCenterURL(cc)
	if err != nil {
		return nil, err
	}
	factory, err := extension.GetConfigCenterFactory(configCenterURL.Protocol)
	if err != nil {
		return nil, err
	}
	dynamicConfig, err := factory.GetDynamicConfiguration(configCenterURL)
	if err != nil {
		return nil, err
	}
	envInstance.SetDynamicConfiguration(dynamicConfig)
	return dynamicConfig, nil
}

func globalConfigCenterURL(cc *global.CenterConfig) (*common.URL, error) {
	return common.NewURL(cc.Address,
		common.WithProtocol(cc.Protocol),
		common.WithParams(globalConfigCenterURLMap(cc)),
		common.WithUsername(cc.Username),
		common.WithPassword(cc.Password),
	)
}

func globalConfigCenterURLMap(cc *global.CenterConfig) url.Values {
	urlMap := url.Values{}
	urlMap.Set(constant.ConfigNamespaceKey, cc.Namespace)
	urlMap.Set(constant.ConfigGroupKey, cc.Group)
	urlMap.Set(constant.ConfigClusterKey, cc.Cluster)
	urlMap.Set(constant.ConfigAppIDKey, cc.AppID)
	urlMap.Set(constant.ConfigTimeoutKey, cc.Timeout)
	urlMap.Set(constant.ClientNameKey, strings.Join([]string{constant.ConfigCenterPrefix, cc.Protocol, cc.Address}, "-"))

	for key, val := range cc.Params {
		urlMap.Set(key, val)
	}
	return urlMap
}

func (rc *InstanceOptions) Process(event *config_center.ConfigChangeEvent) {
	logger.Infof("CenterConfig process event:\n%+v", event)
	if event == nil {
		return
	}
	value, ok := event.Value.(string)
	if !ok {
		logger.Errorf("CenterConfig process event value is not string")
		return
	}
	conf := NewLoaderConf(WithBytes([]byte(value)))
	koan := GetConfigResolver(conf)
	update := &InstanceOptions{}
	if err := koan.UnmarshalWithConf(rc.Prefix(), update, koanf.UnmarshalConf{Tag: "yaml"}); err != nil {
		logger.Errorf("CenterConfig process unmarshalConf failed, got error %#v", err)
		return
	}

	for registryID, updateRegistry := range update.Registries {
		if updateRegistry == nil {
			continue
		}
		if registryConfig := rc.Registries[registryID]; registryConfig != nil && updateRegistry.Timeout != registryConfig.Timeout {
			registryConfig.Timeout = updateRegistry.Timeout
			logger.Infof("RegistryConfigs Timeout was dynamically updated, new value:%v", registryConfig.Timeout)
		}
	}
	if rc.Consumer != nil && update.Consumer != nil && update.Consumer.RequestTimeout != rc.Consumer.RequestTimeout {
		rc.Consumer.RequestTimeout = update.Consumer.RequestTimeout
		logger.Infof("ConsumerConfig's RequestTimeout was dynamically updated, new value:%v", rc.Consumer.RequestTimeout)
	}

	setCompatRootConfig(compatRootConfig(rc))
}
