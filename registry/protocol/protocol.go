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
)

import (
	gxset "github.com/dubbogo/gost/container/set"
	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	_ "dubbo.apache.org/dubbo-go/v3/config_center/configurator"
	"dubbo.apache.org/dubbo-go/v3/protocol"
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
	invokers []protocol.Invoker
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

func getCacheKey(invoker protocol.Invoker) string {
	url := getProviderUrl(invoker)
	delKeys := gxset.NewSet("dynamic", "enabled")
	return url.CloneExceptParams(delKeys).String()
}

func newRegistryProtocol() *registryProtocol {
	return &registryProtocol{
		registries: &sync.Map{},
		bounds:     &sync.Map{},
	}
}

func getRegistry(regUrl *common.URL) registry.Registry {
	reg, err := extension.GetRegistry(regUrl.Protocol, regUrl)
	if err != nil {
		logger.Errorf("Registry can not connect success, program is going to panic.Error message is %s", err.Error())
		panic(err.Error())
	}
	return reg
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
	if registryUrl.Protocol == constant.REGISTRY_PROTOCOL {
		registryUrl.Protocol = registryUrl.GetParam(constant.REGISTRY_KEY, "")
	}

	var reg registry.Registry
	if regI, loaded := proto.registries.Load(registryUrl.Key()); !loaded {
		reg = getRegistry(registryUrl)
		proto.registries.Store(registryUrl.Key(), reg)
	} else {
		reg = regI.(registry.Registry)
	}

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
	cluster := extension.GetCluster(serviceUrl.GetParam(constant.CLUSTER_KEY, constant.DEFAULT_CLUSTER))
	invoker := cluster.Join(directory)
	proto.invokers = append(proto.invokers, invoker)
	return invoker
}

// Export provider service to registry center
func (proto *registryProtocol) Export(invoker protocol.Invoker) protocol.Exporter {
	proto.once.Do(func() {
		proto.initConfigurationListeners()
	})
	registryUrl := getRegistryUrl(invoker)
	providerUrl := getProviderUrl(invoker)

	overriderUrl := getSubscribedOverrideUrl(providerUrl)
	// Deprecated! subscribe to override rules in 2.6.x or before.
	overrideSubscribeListener := newOverrideSubscribeListener(overriderUrl, invoker, proto)
	proto.overrideListeners.Store(overriderUrl, overrideSubscribeListener)
	proto.providerConfigurationListener.OverrideUrl(providerUrl)
	serviceConfigurationListener := newServiceConfigurationListener(overrideSubscribeListener, providerUrl)
	proto.serviceConfigurationListeners.Store(providerUrl.ServiceKey(), serviceConfigurationListener)
	serviceConfigurationListener.OverrideUrl(providerUrl)

	var reg registry.Registry
	if regI, loaded := proto.registries.Load(registryUrl.Key()); !loaded {
		reg = getRegistry(registryUrl)
		proto.registries.Store(registryUrl.Key(), reg)
		logger.Infof("Export proto:%p registries address:%p", proto, proto.registries)
	} else {
		reg = regI.(registry.Registry)
	}

	registeredProviderUrl := getUrlToRegistry(providerUrl, registryUrl)
	err := reg.Register(registeredProviderUrl)
	if err != nil {
		logger.Errorf("provider service %v register registry %v error, error message is %s",
			providerUrl.Key(), registryUrl.Key(), err.Error())
		return nil
	}

	key := getCacheKey(invoker)
	logger.Infof("The cached exporter keys is %v!", key)
	cachedExporter, loaded := proto.bounds.Load(key)
	if loaded {
		logger.Infof("The exporter has been cached, and will return cached exporter!")
	} else {
		wrappedInvoker := newWrappedInvoker(invoker, providerUrl)
		cachedExporter = extension.GetProtocol(protocolwrapper.FILTER).Export(wrappedInvoker)
		proto.bounds.Store(key, cachedExporter)
		logger.Infof("The exporter has not been cached, and will return a new exporter!")
	}

	go func() {
		if err = reg.Subscribe(overriderUrl, overrideSubscribeListener); err != nil {
			logger.Warnf("reg.subscribe(overriderUrl:%v) = error:%v", overriderUrl, err)
		}
	}()
	return cachedExporter.(protocol.Exporter)
}

func (proto *registryProtocol) reExport(invoker protocol.Invoker, newUrl *common.URL) {
	key := getCacheKey(invoker)
	if oldExporter, loaded := proto.bounds.Load(key); loaded {
		wrappedNewInvoker := newWrappedInvoker(invoker, newUrl)
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
	id := providerUrl.GetParam(constant.BEAN_NAME_KEY, "")

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

	_, err := common.ServiceMap.Register(serviceConfig.InterfaceName,
		serviceConfig.Protocol, serviceConfig.Group,
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
	if len(providerUrl.GetParam(constant.CATEGORY_KEY, "")) == 0 &&
		providerUrl.Protocol == constant.OVERRIDE_PROTOCOL {
		providerUrl.AddParam(constant.CATEGORY_KEY, constant.CONFIGURATORS_CATEGORY)
	}
	consumerInterface := consumerUrl.GetParam(constant.INTERFACE_KEY, consumerUrl.Path)
	providerInterface := providerUrl.GetParam(constant.INTERFACE_KEY, providerUrl.Path)

	if !(constant.ANY_VALUE == consumerInterface ||
		constant.ANY_VALUE == providerInterface ||
		providerInterface == consumerInterface) {
		return false
	}

	if !isMatchCategory(providerUrl.GetParam(constant.CATEGORY_KEY, constant.DEFAULT_CATEGORY),
		consumerUrl.GetParam(constant.CATEGORY_KEY, constant.DEFAULT_CATEGORY)) {
		return false
	}

	if !providerUrl.GetParamBool(constant.ENABLED_KEY, true) &&
		consumerUrl.GetParam(constant.ENABLED_KEY, "") != constant.ANY_VALUE {
		return false
	}
	consumerGroup := consumerUrl.GetParam(constant.GROUP_KEY, "")
	consumerVersion := consumerUrl.GetParam(constant.VERSION_KEY, "")
	consumerClassifier := consumerUrl.GetParam(constant.CLASSIFIER_KEY, "")

	providerGroup := providerUrl.GetParam(constant.GROUP_KEY, "")
	providerVersion := providerUrl.GetParam(constant.VERSION_KEY, "")
	providerClassifier := providerUrl.GetParam(constant.CLASSIFIER_KEY, "")
	// todo: public static boolean isContains(String values, String value) {
	//        return isNotEmpty(values) && isContains(COMMA_SPLIT_PATTERN.split(values), value);
	//    }
	return (consumerGroup == constant.ANY_VALUE || consumerGroup == providerGroup ||
		strings.Contains(consumerGroup, providerGroup)) && (consumerVersion == constant.ANY_VALUE ||
		consumerVersion == providerVersion) && (len(consumerClassifier) == 0 ||
		consumerClassifier == constant.ANY_VALUE || consumerClassifier == providerClassifier)
}

func isMatchCategory(category string, categories string) bool {
	if len(categories) == 0 {
		return category == constant.DEFAULT_CATEGORY
	} else if strings.Contains(categories, constant.ANY_VALUE) {
		return true
	} else if strings.Contains(categories, constant.REMOVE_VALUE_PREFIX) {
		return !strings.Contains(categories, constant.REMOVE_VALUE_PREFIX+category)
	} else {
		return strings.Contains(categories, category)
	}
}

func getSubscribedOverrideUrl(providerUrl *common.URL) *common.URL {
	newUrl := providerUrl.Clone()
	newUrl.Protocol = constant.PROVIDER_PROTOCOL
	newUrl.SetParam(constant.CATEGORY_KEY, constant.CONFIGURATORS_CATEGORY)
	newUrl.SetParam(constant.CHECK_KEY, "false")
	return newUrl
}

// Destroy registry protocol
func (proto *registryProtocol) Destroy() {
	// invoker.Destroy() should be performed in config.destroyConsumerProtocols().
	proto.invokers = []protocol.Invoker{}
	proto.bounds.Range(func(key, value interface{}) bool {
		// protocol holds the exporters actually, instead, registry holds them in order to avoid export repeatedly, so
		// the work for unexport should be finished in protocol.Unexport(), see also config.destroyProviderProtocols().
		proto.bounds.Delete(key)
		return true
	})
	proto.registries.Range(func(key, value interface{}) bool {
		reg := value.(registry.Registry)
		if reg.IsAvailable() {
			reg.Destroy()
		}
		proto.registries.Delete(key)
		return true
	})
}

func getRegistryUrl(invoker protocol.Invoker) *common.URL {
	// here add * for return a new url
	url := invoker.GetURL()
	// if the protocol == registry, set protocol the registry value in url.params
	if url.Protocol == constant.REGISTRY_PROTOCOL {
		url.Protocol = url.GetParam(constant.REGISTRY_KEY, "")
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

type wrappedInvoker struct {
	invoker protocol.Invoker
	protocol.BaseInvoker
}

func newWrappedInvoker(invoker protocol.Invoker, url *common.URL) *wrappedInvoker {
	return &wrappedInvoker{
		invoker:     invoker,
		BaseInvoker: *protocol.NewBaseInvoker(url),
	}
}

// Invoke remote service base on URL of wrappedInvoker
func (ivk *wrappedInvoker) Invoke(ctx context.Context, invocation protocol.Invocation) protocol.Result {
	return ivk.invoker.Invoke(ctx, invocation)
}

type providerConfigurationListener struct {
	registry.BaseConfigurationListener
	overrideListeners *sync.Map
}

func newProviderConfigurationListener(overrideListeners *sync.Map) *providerConfigurationListener {
	listener := &providerConfigurationListener{}
	listener.overrideListeners = overrideListeners
	listener.InitWith(
		config.GetProviderConfig().ApplicationConfig.Name+constant.CONFIGURATORS_SUFFIX,
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
		providerUrl.EncodedServiceKey()+constant.CONFIGURATORS_SUFFIX,
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
