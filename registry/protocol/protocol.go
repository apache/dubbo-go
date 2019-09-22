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
	"strings"
	"sync"
)

import (
	"github.com/dubbogo/gost/container"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/common/proxy/proxy_factory"
	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/config_center"
	_ "github.com/apache/dubbo-go/config_center/configurator"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/protocol/protocolwrapper"
	"github.com/apache/dubbo-go/registry"
	directory2 "github.com/apache/dubbo-go/registry/directory"
	"github.com/apache/dubbo-go/remoting"
)

var (
	regProtocol *registryProtocol
)

type registryProtocol struct {
	invokers []protocol.Invoker
	// Registry  Map<RegistryAddress, Registry>
	registries *sync.Map
	//To solve the problem of RMI repeated exposure port conflicts, the services that have been exposed are no longer exposed.
	//providerurl <--> exporter
	bounds                        *sync.Map
	overrideListeners             *sync.Map
	serviceConfigurationListeners *sync.Map
	providerConfigurationListener *providerConfigurationListener
	once                          sync.Once
}

func init() {
	extension.SetProtocol("registry", GetProtocol)
}

func getCacheKey(url *common.URL) string {
	newUrl := url.Clone()
	delKeys := container.NewSet("dynamic", "enabled")
	newUrl.RemoveParams(delKeys)
	return newUrl.String()
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
func (proto *registryProtocol) initConfigurationListeners() {
	proto.overrideListeners = &sync.Map{}
	proto.serviceConfigurationListeners = &sync.Map{}
	proto.providerConfigurationListener = newProviderConfigurationListener(proto.overrideListeners)
}
func (proto *registryProtocol) Refer(url common.URL) protocol.Invoker {

	var registryUrl = url
	var serviceUrl = registryUrl.SubURL
	if registryUrl.Protocol == constant.REGISTRY_PROTOCOL {
		protocol := registryUrl.GetParam(constant.REGISTRY_KEY, "")
		registryUrl.Protocol = protocol
	}

	var reg registry.Registry

	if regI, loaded := proto.registries.Load(registryUrl.Key()); !loaded {
		reg = getRegistry(&registryUrl)
		proto.registries.Store(registryUrl.Key(), reg)
	} else {
		reg = regI.(registry.Registry)
	}

	//new registry directory for store service url from registry
	directory, err := directory2.NewRegistryDirectory(&registryUrl, reg)
	if err != nil {
		logger.Errorf("consumer service %v  create registry directory  error, error message is %s, and will return nil invoker!",
			serviceUrl.String(), err.Error())
		return nil
	}
	err = reg.Register(*serviceUrl)
	if err != nil {
		logger.Errorf("consumer service %v register registry %v error, error message is %s",
			serviceUrl.String(), registryUrl.String(), err.Error())
	}
	go directory.Subscribe(serviceUrl)

	//new cluster invoker
	cluster := extension.GetCluster(serviceUrl.GetParam(constant.CLUSTER_KEY, constant.DEFAULT_CLUSTER))

	invoker := cluster.Join(directory)
	proto.invokers = append(proto.invokers, invoker)
	return invoker
}

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
	} else {
		reg = regI.(registry.Registry)
	}

	err := reg.Register(*providerUrl)
	if err != nil {
		logger.Errorf("provider service %v register registry %v error, error message is %s",
			providerUrl.Key(), registryUrl.Key(), err.Error())
		return nil
	}

	key := getCacheKey(providerUrl)
	logger.Infof("The cached exporter keys is %v !", key)
	cachedExporter, loaded := proto.bounds.Load(key)
	if loaded {
		logger.Infof("The exporter has been cached, and will return cached exporter!")
	} else {
		wrappedInvoker := newWrappedInvoker(invoker, providerUrl)
		cachedExporter = extension.GetProtocol(protocolwrapper.FILTER).Export(wrappedInvoker)
		proto.bounds.Store(key, cachedExporter)
		logger.Infof("The exporter has not been cached, and will return a new  exporter!")
	}

	go reg.Subscribe(overriderUrl, overrideSubscribeListener)
	return cachedExporter.(protocol.Exporter)

}
func (proto *registryProtocol) reExport(invoker protocol.Invoker, newUrl *common.URL) {
	url := getProviderUrl(invoker)
	key := getCacheKey(url)
	if oldExporter, loaded := proto.bounds.Load(key); loaded {
		wrappedNewInvoker := newWrappedInvoker(invoker, newUrl)
		oldExporter.(protocol.Exporter).Unexport()
		proto.bounds.Delete(key)
		proto.Export(wrappedNewInvoker)
		//TODO:  unregister & unsubscribe

	}
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
func (nl *overrideSubscribeListener) Notify(event *registry.ServiceEvent) {
	if isMatched(&(event.Service), nl.url) && event.Action == remoting.EventTypeAdd {
		nl.configurator = extension.GetDefaultConfigurator(&(event.Service))
		nl.doOverrideIfNecessary()
	}
}
func (nl *overrideSubscribeListener) doOverrideIfNecessary() {
	providerUrl := getProviderUrl(nl.originInvoker)
	key := getCacheKey(providerUrl)
	if exporter, ok := nl.protocol.bounds.Load(key); ok {
		currentUrl := exporter.(protocol.Exporter).GetInvoker().GetUrl()
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
			newRegUrl := nl.originInvoker.GetUrl()
			setProviderUrl(&newRegUrl, providerUrl)
			nl.protocol.reExport(nl.originInvoker, &newRegUrl)
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
	//todo: public static boolean isContains(String values, String value) {
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

func (proto *registryProtocol) Destroy() {
	for _, ivk := range proto.invokers {
		ivk.Destroy()
	}
	proto.invokers = []protocol.Invoker{}

	proto.bounds.Range(func(key, value interface{}) bool {
		exporter := value.(protocol.Exporter)
		exporter.Unexport()
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
	//here add * for return a new url
	url := invoker.GetUrl()
	//if the protocol == registry ,set protocol the registry value in url.params
	if url.Protocol == constant.REGISTRY_PROTOCOL {
		protocol := url.GetParam(constant.REGISTRY_KEY, "")
		url.Protocol = protocol
	}
	return &url
}

func getProviderUrl(invoker protocol.Invoker) *common.URL {
	url := invoker.GetUrl()
	//be careful params maps in url is map type
	return url.SubURL.Clone()
}
func setProviderUrl(regURL *common.URL, providerURL *common.URL) {
	regURL.SubURL = providerURL
}

func GetProtocol() protocol.Protocol {
	if regProtocol != nil {
		return regProtocol
	}
	return newRegistryProtocol()
}

type wrappedInvoker struct {
	invoker protocol.Invoker
	protocol.BaseInvoker
}

func newWrappedInvoker(invoker protocol.Invoker, url *common.URL) *wrappedInvoker {
	return &wrappedInvoker{
		invoker:     invoker,
		BaseInvoker: *protocol.NewBaseInvoker(*url),
	}
}

func (ivk *wrappedInvoker) Invoke(invocation protocol.Invocation) protocol.Result {
	// get right url
	ivk.invoker.(*proxy_factory.ProxyInvoker).BaseInvoker = *protocol.NewBaseInvoker(ivk.GetUrl())
	return ivk.invoker.Invoke(invocation)
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

func (listener *serviceConfigurationListener) Process(event *config_center.ConfigChangeEvent) {
	listener.BaseConfigurationListener.Process(event)
	listener.overrideListener.doOverrideIfNecessary()
}
