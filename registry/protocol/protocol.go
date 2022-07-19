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

package protocol

import (
	"context"
	"strings"
	"sync"
	"time"
)

import (
	gxset "github.com/dubbogo/gost/container/set"
	"github.com/dubbogo/gost/log/logger"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	_ "dubbo.apache.org/dubbo-go/v3/config_center/configurator"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/dubbo3/health"
	"dubbo.apache.org/dubbo-go/v3/protocol/protocolwrapper"
	"dubbo.apache.org/dubbo-go/v3/registry"
	_ "dubbo.apache.org/dubbo-go/v3/registry/directory"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

var (
	regProtocol   *registryProtocol
	once          sync.Once
	reserveParams = []string{
		"application", "codec", "exchanger", "serialization", "cluster", "connections", "deprecated", "group",
		"loadbalance", "mock", "path", "timeout", "token", "version", "warmup", "weight", "timestamp", "dubbo",
		"release", "interface", "registry.role",
	}
)

type registryProtocol struct {
	// Registry Map<RegistryAddress, Registry>
	registries *sync.Map
	// To solve the problem of RMI repeated exposure port conflicts,
	// the services that have been exposed are no longer exposed.
	// providerurl <--> exporter
	bounds                        *sync.Map
	overrideListeners             *sync.Map
	serviceConfigurationListeners *sync.Map
	providerConfigurationListener *providerConfigurationListener
	once                          sync.Once
}

func init() {
	extension.SetProtocol("registry", GetProtocol)
}

func newRegistryProtocol() *registryProtocol {
	return &registryProtocol{
		registries: &sync.Map{},
		bounds:     &sync.Map{},
	}
}

func (proto *registryProtocol) getRegistry(registryUrl *common.URL) registry.Registry {
	var err error
	reg, loaded := proto.registries.Load(registryUrl.Location)
	if !loaded {
		reg, err = extension.GetRegistry(registryUrl.Protocol, registryUrl)
		if err != nil {
			logger.Errorf("Registry can not connect success, program is going to panic.Error message is %s", err.Error())
			panic(err)
		}
		proto.registries.Store(registryUrl.Location, reg)
	}
	return reg.(registry.Registry)
}

func getCacheKey(invoker protocol.Invoker) string {
	url := getProviderUrl(invoker)
	delKeys := gxset.NewSet("dynamic", "enabled")
	return url.CloneExceptParams(delKeys).String()
}

func getUrlToRegistry(providerUrl *common.URL, registryUrl *common.URL) *common.URL {
	if registryUrl.GetParamBool("simplified", false) {
		return providerUrl.CloneWithParams(reserveParams)
	} else {
		return filterHideKey(providerUrl)
	}
}

// filterHideKey filter the parameters that do not need to be output in url(Starting with .)
func filterHideKey(url *common.URL) *common.URL {
	// be careful params maps in url is map type
	removeSet := gxset.NewSet()
	for k := range url.GetParams() {
		if strings.HasPrefix(k, ".") {
			removeSet.Add(k)
		}
	}
	return url.CloneExceptParams(removeSet)
}

func (proto *registryProtocol) initConfigurationListeners() {
	proto.overrideListeners = &sync.Map{}
	proto.serviceConfigurationListeners = &sync.Map{}
	proto.providerConfigurationListener = newProviderConfigurationListener(proto.overrideListeners)
}

// nolint
func (proto *registryProtocol) GetRegistries() []registry.Registry {
	var rs []registry.Registry
	proto.registries.Range(func(_, v interface{}) bool {
		if r, ok := v.(registry.Registry); ok {
			rs = append(rs, r)
		}
		return true
	})
	return rs
}

// Refer provider service from registry center
func (proto *registryProtocol) Refer(url *common.URL) protocol.Invoker {
	registryUrl := url
	serviceUrl := registryUrl.SubURL
	if registryUrl.Protocol == constant.RegistryProtocol {
		registryUrl.Protocol = registryUrl.GetParam(constant.RegistryKey, "")
	}

	reg := proto.getRegistry(url)

	// new registry directory for store service url from registry
	directory, err := extension.GetDefaultRegistryDirectory(registryUrl, reg)
	if err != nil {
		logger.Errorf("consumer service %v create registry directory error, error message is %s, and will return nil invoker!",
			serviceUrl.String(), err.Error())
		return nil
	}

	err = reg.Register(serviceUrl)
	if err != nil {
		logger.Errorf("consumer service %v register registry %v error, error message is %s",
			serviceUrl.String(), registryUrl.String(), err.Error())
	}

	// new cluster invoker
	clusterKey := serviceUrl.GetParam(constant.ClusterKey, constant.DefaultCluster)
	cluster, err := extension.GetCluster(clusterKey)
	if err != nil {
		panic(err)
	}
	invoker := cluster.Join(directory)
	return invoker
}

// Export provider service to registry center
func (proto *registryProtocol) Export(originInvoker protocol.Invoker) protocol.Exporter {
	proto.once.Do(func() {
		proto.initConfigurationListeners()
	})
	registryUrl := getRegistryUrl(originInvoker)
	providerUrl := getProviderUrl(originInvoker)

	overriderUrl := getSubscribedOverrideUrl(providerUrl)
	// Deprecated! subscribe to override rules in 2.6.x or before.
	overrideSubscribeListener := newOverrideSubscribeListener(overriderUrl, originInvoker, proto)
	proto.overrideListeners.Store(overriderUrl, overrideSubscribeListener)
	proto.providerConfigurationListener.OverrideUrl(providerUrl)
	serviceConfigurationListener := newServiceConfigurationListener(overrideSubscribeListener, providerUrl)
	proto.serviceConfigurationListeners.Store(providerUrl.ServiceKey(), serviceConfigurationListener)
	serviceConfigurationListener.OverrideUrl(providerUrl)

	// export invoker
	exporter := proto.doLocalExport(originInvoker, providerUrl)

	// update health status
	health.SetServingStatusServing(registryUrl.Service())

	if len(registryUrl.Protocol) > 0 {
		// url to registry
		reg := proto.getRegistry(registryUrl)
		registeredProviderUrl := getUrlToRegistry(providerUrl, registryUrl)

		err := reg.Register(registeredProviderUrl)
		if err != nil {
			logger.Errorf("provider service %v register registry %v error, error message is %s",
				providerUrl.Key(), registryUrl.Key(), err.Error())
			return nil
		}

		go func() {
			if err := reg.Subscribe(overriderUrl, overrideSubscribeListener); err != nil {
				logger.Warnf("reg.subscribe(overriderUrl:%v) = error:%v", overriderUrl, err)
			}
		}()

		exporter.SetRegisterUrl(registeredProviderUrl)
		exporter.SetSubscribeUrl(overriderUrl)

	} else {
		logger.Warnf("provider service %v do not regist to registry %v. possible direct connection provider",
			providerUrl.Key(), registryUrl.Key())
	}

	return exporter
}

func (proto *registryProtocol) doLocalExport(originInvoker protocol.Invoker, providerUrl *common.URL) *exporterChangeableWrapper {
	key := getCacheKey(originInvoker)
	cachedExporter, loaded := proto.bounds.Load(key)
	if !loaded {
		// new Exporter
		invokerDelegate := newInvokerDelegate(originInvoker, providerUrl)
		cachedExporter = newExporterChangeableWrapper(originInvoker,
			extension.GetProtocol(protocolwrapper.FILTER).Export(invokerDelegate))
		proto.bounds.Store(key, cachedExporter)
	}
	return cachedExporter.(*exporterChangeableWrapper)
}

func (proto *registryProtocol) reExport(invoker protocol.Invoker, newUrl *common.URL) {
	key := getCacheKey(invoker)
	if oldExporter, loaded := proto.bounds.Load(key); loaded {
		wrappedNewInvoker := newInvokerDelegate(invoker, newUrl)
		oldExporter.(protocol.Exporter).Unexport()
		proto.bounds.Delete(key)
		// oldExporter Unexport function unRegister rpcService from the serviceMap, so need register it again as far as possible
		if err := registerServiceMap(invoker); err != nil {
			logger.Error(err.Error())
		}
		proto.Export(wrappedNewInvoker)
		// TODO:  unregister & unsubscribe
	}
}

func registerServiceMap(invoker protocol.Invoker) error {
	providerUrl := getProviderUrl(invoker)
	// the bean.name param of providerUrl is the ServiceConfig id property
	// such as dubbo://:20000/org.apache.dubbo.UserProvider?bean.name=UserProvider&cluster=failfast...
	id := providerUrl.GetParam(constant.BeanNameKey, "")

	serviceConfig := config.GetProviderConfig().Services[id]
	if serviceConfig == nil {
		s := "reExport can not get serviceConfig"
		return perrors.New(s)
	}
	rpcService := config.GetProviderService(id)
	if rpcService == nil {
		s := "reExport can not get RPCService"
		return perrors.New(s)
	}

	_, err := common.ServiceMap.Register(serviceConfig.Interface,
		// FIXME
		serviceConfig.ProtocolIDs[0], serviceConfig.Group,
		serviceConfig.Version, rpcService)
	if err != nil {
		s := "reExport can not re register ServiceMap. Error message is " + err.Error()
		return perrors.New(s)
	}
	return nil
}

type overrideSubscribeListener struct {
	url           *common.URL
	originInvoker protocol.Invoker
	protocol      *registryProtocol
	configurator  config_center.Configurator
}

func newOverrideSubscribeListener(overriderUrl *common.URL, invoker protocol.Invoker, proto *registryProtocol) *overrideSubscribeListener {
	return &overrideSubscribeListener{url: overriderUrl, originInvoker: invoker, protocol: proto}
}

// Notify will be triggered when a service change notification is received.
func (nl *overrideSubscribeListener) Notify(event *registry.ServiceEvent) {
	if isMatched(event.Service, nl.url) && event.Action == remoting.EventTypeAdd {
		nl.configurator = extension.GetDefaultConfigurator(event.Service)
		nl.doOverrideIfNecessary()
	}
}

func (nl *overrideSubscribeListener) NotifyAll(events []*registry.ServiceEvent, callback func()) {
	defer callback()
	if len(events) == 0 {
		return
	}
	for _, e := range events {
		nl.Notify(e)
	}
}

func (nl *overrideSubscribeListener) doOverrideIfNecessary() {
	providerUrl := getProviderUrl(nl.originInvoker)
	key := getCacheKey(nl.originInvoker)
	if exporter, ok := nl.protocol.bounds.Load(key); ok {
		currentUrl := exporter.(protocol.Exporter).GetInvoker().GetURL()
		// Compatible with the 2.6.x
		if nl.configurator != nil {
			nl.configurator.Configure(providerUrl)
		}
		// provider application level  management in 2.7.x
		for _, v := range nl.protocol.providerConfigurationListener.Configurators() {
			v.Configure(providerUrl)
		}
		// provider service level  management in 2.7.x
		if serviceListener, ok := nl.protocol.serviceConfigurationListeners.Load(providerUrl.ServiceKey()); ok {
			listener := serviceListener.(*serviceConfigurationListener)
			for _, v := range listener.Configurators() {
				v.Configure(providerUrl)
			}
		}

		if currentUrl.String() != providerUrl.String() {
			newRegUrl := nl.originInvoker.GetURL().Clone()
			setProviderUrl(newRegUrl, providerUrl)
			nl.protocol.reExport(nl.originInvoker, newRegUrl)
		}
	}
}

func isMatched(providerUrl *common.URL, consumerUrl *common.URL) bool {
	// Compatible with the 2.6.x
	if len(providerUrl.GetParam(constant.CategoryKey, "")) == 0 &&
		providerUrl.Protocol == constant.OverrideProtocol {
		providerUrl.AddParam(constant.CategoryKey, constant.ConfiguratorsCategory)
	}
	consumerInterface := consumerUrl.GetParam(constant.InterfaceKey, consumerUrl.Path)
	providerInterface := providerUrl.GetParam(constant.InterfaceKey, providerUrl.Path)

	if !(constant.AnyValue == consumerInterface ||
		constant.AnyValue == providerInterface ||
		providerInterface == consumerInterface) {
		return false
	}

	if !isMatchCategory(providerUrl.GetParam(constant.CategoryKey, constant.DefaultCategory),
		consumerUrl.GetParam(constant.CategoryKey, constant.DefaultCategory)) {
		return false
	}

	if !providerUrl.GetParamBool(constant.EnabledKey, true) &&
		consumerUrl.GetParam(constant.EnabledKey, "") != constant.AnyValue {
		return false
	}
	consumerGroup := consumerUrl.GetParam(constant.GroupKey, "")
	consumerVersion := consumerUrl.GetParam(constant.VersionKey, "")
	consumerClassifier := consumerUrl.GetParam(constant.ClassifierKey, "")

	providerGroup := providerUrl.GetParam(constant.GroupKey, "")
	providerVersion := providerUrl.GetParam(constant.VersionKey, "")
	providerClassifier := providerUrl.GetParam(constant.ClassifierKey, "")
	// todo: public static boolean isContains(String values, String value) {
	//        return isNotEmpty(values) && isContains(CommaSplitPattern.split(values), value);
	//    }
	return (consumerGroup == constant.AnyValue || consumerGroup == providerGroup ||
		strings.Contains(consumerGroup, providerGroup)) && (consumerVersion == constant.AnyValue ||
		consumerVersion == providerVersion) && (len(consumerClassifier) == 0 ||
		consumerClassifier == constant.AnyValue || consumerClassifier == providerClassifier)
}

func isMatchCategory(category string, categories string) bool {
	if len(categories) == 0 {
		return category == constant.DefaultCategory
	} else if strings.Contains(categories, constant.AnyValue) {
		return true
	} else if strings.Contains(categories, constant.RemoveValuePrefix) {
		return !strings.Contains(categories, constant.RemoveValuePrefix+category)
	} else {
		return strings.Contains(categories, category)
	}
}

func getSubscribedOverrideUrl(providerUrl *common.URL) *common.URL {
	newUrl := providerUrl.Clone()
	newUrl.Protocol = constant.ProviderProtocol
	newUrl.SetParam(constant.CategoryKey, constant.ConfiguratorsCategory)
	newUrl.SetParam(constant.CheckKey, "false")
	return newUrl
}

// Destroy registry protocol
func (proto *registryProtocol) Destroy() {
	proto.bounds.Range(func(key, value interface{}) bool {
		// protocol holds the exporters actually, instead, registry holds them in order to avoid export repeatedly, so
		// the work for unexport should be finished in protocol.Unexport(), see also config.destroyProviderProtocols().
		exporter := value.(*exporterChangeableWrapper)
		reg := proto.getRegistry(getRegistryUrl(exporter.originInvoker))
		if err := reg.UnRegister(exporter.registerUrl); err != nil {
			panic(err)
		}
		// TODO unsubscribeUrl

		// close all protocol server after consumerUpdateWait + stepTimeout(max time wait during
		// waitAndAcceptNewRequests procedure)
		go func() {
			select {
			case <-time.After(config.GetShutDown().GetStepTimeout() + config.GetShutDown().GetConsumerUpdateWaitTime()):
				exporter.Unexport()
				proto.bounds.Delete(key)
			}
		}()
		return true
	})

	proto.registries.Range(func(key, value interface{}) bool {
		proto.registries.Delete(key)
		return true
	})
}

func getRegistryUrl(invoker protocol.Invoker) *common.URL {
	// here add * for return a new url
	url := invoker.GetURL()
	// if the protocol == registry, set protocol the registry value in url.params
	if url.Protocol == constant.RegistryProtocol {
		url.Protocol = url.GetParam(constant.RegistryKey, "")
	}
	return url
}

func getProviderUrl(invoker protocol.Invoker) *common.URL {
	url := invoker.GetURL()
	// be careful params maps in url is map type
	return url.SubURL.Clone()
}

func setProviderUrl(regURL *common.URL, providerURL *common.URL) {
	regURL.SubURL = providerURL
}

// GetProtocol return the singleton registryProtocol
func GetProtocol() protocol.Protocol {
	once.Do(func() {
		regProtocol = newRegistryProtocol()
	})
	return regProtocol
}

type invokerDelegate struct {
	invoker protocol.Invoker
	protocol.BaseInvoker
}

func newInvokerDelegate(invoker protocol.Invoker, url *common.URL) *invokerDelegate {
	return &invokerDelegate{
		invoker:     invoker,
		BaseInvoker: *protocol.NewBaseInvoker(url),
	}
}

// Invoke remote service base on URL of wrappedInvoker
func (ivk *invokerDelegate) Invoke(ctx context.Context, invocation protocol.Invocation) protocol.Result {
	return ivk.invoker.Invoke(ctx, invocation)
}

type exporterChangeableWrapper struct {
	protocol.Exporter
	originInvoker protocol.Invoker
	exporter      protocol.Exporter
	registerUrl   *common.URL
	subscribeUrl  *common.URL
}

func (e *exporterChangeableWrapper) Unexport() {
	e.exporter.Unexport()
}

func (e *exporterChangeableWrapper) SetRegisterUrl(registerUrl *common.URL) {
	e.registerUrl = registerUrl
}

func (e *exporterChangeableWrapper) SetSubscribeUrl(subscribeUrl *common.URL) {
	e.subscribeUrl = subscribeUrl
}

func (e *exporterChangeableWrapper) GetInvoker() protocol.Invoker {
	return e.exporter.GetInvoker()
}

func newExporterChangeableWrapper(originInvoker protocol.Invoker, exporter protocol.Exporter) *exporterChangeableWrapper {
	return &exporterChangeableWrapper{
		originInvoker: originInvoker,
		exporter:      exporter,
	}
}

type providerConfigurationListener struct {
	registry.BaseConfigurationListener
	overrideListeners *sync.Map
}

func newProviderConfigurationListener(overrideListeners *sync.Map) *providerConfigurationListener {
	listener := &providerConfigurationListener{}
	listener.overrideListeners = overrideListeners
	listener.InitWith(
		config.GetRootConfig().Application.Name+constant.ConfiguratorSuffix,
		listener,
		extension.GetDefaultConfiguratorFunc(),
	)
	return listener
}

// Process notified once there's any change happens on the provider config
func (listener *providerConfigurationListener) Process(event *config_center.ConfigChangeEvent) {
	listener.BaseConfigurationListener.Process(event)
	listener.overrideListeners.Range(func(key, value interface{}) bool {
		value.(*overrideSubscribeListener).doOverrideIfNecessary()
		return true
	})
}

type serviceConfigurationListener struct {
	registry.BaseConfigurationListener
	overrideListener *overrideSubscribeListener
	providerUrl      *common.URL
}

func newServiceConfigurationListener(overrideListener *overrideSubscribeListener, providerUrl *common.URL) *serviceConfigurationListener {
	listener := &serviceConfigurationListener{overrideListener: overrideListener, providerUrl: providerUrl}
	listener.InitWith(
		providerUrl.EncodedServiceKey()+constant.ConfiguratorSuffix,
		listener,
		extension.GetDefaultConfiguratorFunc(),
	)
	return listener
}

// Process notified once there's any change happens on the service config
func (listener *serviceConfigurationListener) Process(event *config_center.ConfigChangeEvent) {
	listener.BaseConfigurationListener.Process(event)
	listener.overrideListener.doOverrideIfNecessary()
}
