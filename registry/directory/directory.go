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

package directory

import (
	"fmt"
	"net/url"
	"os"
	"sync"
)

import (
	perrors "github.com/pkg/errors"
	"go.uber.org/atomic"
)

import (
	"github.com/apache/dubbo-go/cluster"
	"github.com/apache/dubbo-go/cluster/directory"
	"github.com/apache/dubbo-go/cluster/router/chain"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/config_center"
	_ "github.com/apache/dubbo-go/config_center/configurator"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/protocol/protocolwrapper"
	"github.com/apache/dubbo-go/registry"
	"github.com/apache/dubbo-go/remoting"
)

func init() {
	extension.SetDefaultRegistryDirectory(NewRegistryDirectory)
}

// RegistryDirectory implementation of Directory:
// Invoker list returned from this Directory's list method have been filtered by Routers
type RegistryDirectory struct {
	directory.BaseDirectory
	cacheInvokers                  []protocol.Invoker
	listenerLock                   sync.Mutex
	serviceType                    string
	registry                       registry.Registry
	cacheInvokersMap               *sync.Map // use sync.map
	consumerURL                    *common.URL
	cacheOriginUrl                 *common.URL
	configurators                  []config_center.Configurator
	consumerConfigurationListener  *consumerConfigurationListener
	referenceConfigurationListener *referenceConfigurationListener
	serviceKey                     string
	forbidden                      atomic.Bool
}

// NewRegistryDirectory will create a new RegistryDirectory
func NewRegistryDirectory(url *common.URL, registry registry.Registry) (cluster.Directory, error) {
	if url.SubURL == nil {
		return nil, perrors.Errorf("url is invalid, suburl can not be nil")
	}
	dir := &RegistryDirectory{
		BaseDirectory:    directory.NewBaseDirectory(url),
		cacheInvokers:    []protocol.Invoker{},
		cacheInvokersMap: &sync.Map{},
		serviceType:      url.SubURL.Service(),
		registry:         registry,
	}

	dir.consumerURL = dir.getConsumerUrl(url.SubURL)

	if routerChain, err := chain.NewRouterChain(dir.consumerURL); err == nil {
		dir.BaseDirectory.SetRouterChain(routerChain)
	} else {
		logger.Warnf("fail to create router chain with url: %s, err is: %v", url.SubURL, err)
	}

	dir.consumerConfigurationListener = newConsumerConfigurationListener(dir)

	go dir.subscribe(url.SubURL)
	return dir, nil
}

// subscribe from registry
func (dir *RegistryDirectory) subscribe(url *common.URL) {
	dir.consumerConfigurationListener.addNotifyListener(dir)
	dir.referenceConfigurationListener = newReferenceConfigurationListener(dir, url)
	dir.registry.Subscribe(url, dir)
}

// Notify monitor changes from registry,and update the cacheServices
func (dir *RegistryDirectory) Notify(events ...*registry.ServiceEvent) {
	go dir.refreshInvokers(events...)
}

// refreshInvokers refreshes service's events. It supports two modes: incremental mode and batch mode. If a single
// service event is passed in, then it is incremental mode, and if an array of service events are passed in, it is
// batch mode, in this mode, we assume the registry center have the complete list of the service events, therefore
// in this case, we can safely assume any cached invoker not in the incoming list can be removed. It is necessary
// since in batch mode, the register center handles the different type of events by itself, then notify the directory
// a batch of 'Update' events, instead of omit the different type of event one by one.
func (dir *RegistryDirectory) refreshInvokers(events ...*registry.ServiceEvent) {
	var oldInvokers []protocol.Invoker

	// in batch mode, it is safe to remove since we have the complete list of events.
	if len(events) > 1 {
		dir.cacheInvokersMap.Range(func(k, v interface{}) bool {
			if !dir.eventMatched(k.(string), events) {
				if invoker := dir.uncacheInvokerWithKey(k.(string)); invoker != nil {
					oldInvokers = append(oldInvokers, invoker)
				}
			}
			return true
		})
	}

	for _, event := range events {
		logger.Debugf("registry update, result{%s}", event)
		if oldInvoker, _ := dir.cacheInvokerByEvent(event); oldInvoker != nil {
			oldInvokers = append(oldInvokers, oldInvoker)
		}
	}

	if len(events) > 0 {
		dir.setNewInvokers()
	}

	// After dir.cacheInvokers is updated,destroy the oldInvoker
	// Ensure that no request will enter the oldInvoker
	for _, invoker := range oldInvokers {
		invoker.Destroy()
	}
}

// eventMatched checks if a cached invoker appears in the incoming invoker list, if no, then it is safe to remove.
func (dir *RegistryDirectory) eventMatched(key string, events []*registry.ServiceEvent) bool {
	for _, event := range events {
		if dir.invokerCacheKey(event.Service) == key {
			return true
		}
	}
	return false
}

// invokerCacheKey generates the key in the cache for a given URL.
func (dir *RegistryDirectory) invokerCacheKey(url *common.URL) string {
	referenceUrl := dir.GetDirectoryUrl().SubURL
	newUrl := common.MergeUrl(url, referenceUrl)
	return newUrl.Key()
}

// setNewInvokers groups the invokers from the cache first, then set the result to both directory and router chain.
func (dir *RegistryDirectory) setNewInvokers() {
	newInvokers := dir.toGroupInvokers()
	dir.listenerLock.Lock()
	defer dir.listenerLock.Unlock()
	dir.cacheInvokers = newInvokers
	dir.RouterChain().SetInvokers(newInvokers)
}

// cacheInvokerByEvent caches invokers from the service event
func (dir *RegistryDirectory) cacheInvokerByEvent(event *registry.ServiceEvent) (protocol.Invoker, error) {
	// judge is override or others
	if event != nil {
		u := dir.convertUrl(event)
		switch event.Action {
		case remoting.EventTypeAdd, remoting.EventTypeUpdate:
			logger.Infof("selector add service url{%s}", event.Service)
			// FIXME: routers are built in every address notification?
			dir.configRouters()
			return dir.cacheInvoker(u), nil
		case remoting.EventTypeDel:
			logger.Infof("selector delete service url{%s}", event.Service)
			return dir.uncacheInvoker(u), nil
		default:
			return nil, fmt.Errorf("illegal event type: %v", event.Action)
		}
	}
	return nil, nil
}

// configRouters configures dynamic routers into the router chain, but, the current impl is incorrect, see FIXME above.
func (dir *RegistryDirectory) configRouters() {
	var urls []*common.URL
	for _, v := range config.GetRouterURLSet().Values() {
		urls = append(urls, v.(*common.URL))
	}

	if len(urls) > 0 {
		dir.SetRouters(urls)
	}
}

// convertUrl processes override:// and router://
func (dir *RegistryDirectory) convertUrl(res *registry.ServiceEvent) *common.URL {
	ret := res.Service
	if ret.Protocol == constant.OVERRIDE_PROTOCOL || // 1.for override url in 2.6.x
		ret.GetParam(constant.CATEGORY_KEY, constant.DEFAULT_CATEGORY) == constant.CONFIGURATORS_CATEGORY {
		dir.configurators = append(dir.configurators, extension.GetDefaultConfigurator(ret))
		ret = nil
	} else if ret.Protocol == constant.ROUTER_PROTOCOL || // 2.for router
		ret.GetParam(constant.CATEGORY_KEY, constant.DEFAULT_CATEGORY) == constant.ROUTER_CATEGORY {
		ret = nil
	}
	return ret
}

func (dir *RegistryDirectory) toGroupInvokers() []protocol.Invoker {
	var (
		err             error
		newInvokersList []protocol.Invoker
	)
	groupInvokersMap := make(map[string][]protocol.Invoker)

	dir.cacheInvokersMap.Range(func(key, value interface{}) bool {
		newInvokersList = append(newInvokersList, value.(protocol.Invoker))
		return true
	})

	for _, invoker := range newInvokersList {
		group := invoker.GetUrl().GetParam(constant.GROUP_KEY, "")

		if _, ok := groupInvokersMap[group]; ok {
			groupInvokersMap[group] = append(groupInvokersMap[group], invoker)
		} else {
			groupInvokersMap[group] = []protocol.Invoker{invoker}
		}
	}
	groupInvokersList := make([]protocol.Invoker, 0, len(groupInvokersMap))
	if len(groupInvokersMap) == 1 {
		// len is 1 it means no group setting ,so do not need cluster again
		for _, invokers := range groupInvokersMap {
			groupInvokersList = invokers
		}
	} else {
		for _, invokers := range groupInvokersMap {
			staticDir := directory.NewStaticDirectory(invokers)
			cst := extension.GetCluster(dir.GetUrl().SubURL.GetParam(constant.CLUSTER_KEY, constant.DEFAULT_CLUSTER))
			err = staticDir.BuildRouterChain(invokers)
			if err != nil {
				logger.Error(err)
				continue
			}
			groupInvokersList = append(groupInvokersList, cst.Join(staticDir))
		}
	}

	return groupInvokersList
}

// uncacheInvoker will return abandoned Invoker, if no Invoker to be abandoned, return nil
func (dir *RegistryDirectory) uncacheInvoker(url *common.URL) protocol.Invoker {
	return dir.uncacheInvokerWithKey(url.Key())
}

func (dir *RegistryDirectory) uncacheInvokerWithKey(key string) protocol.Invoker {
	logger.Debugf("service will be deleted in cache invokers: invokers key is  %s!", key)
	if cacheInvoker, ok := dir.cacheInvokersMap.Load(key); ok {
		dir.cacheInvokersMap.Delete(key)
		return cacheInvoker.(protocol.Invoker)
	}
	return nil
}

// cacheInvoker will return abandoned Invoker,if no Invoker to be abandoned,return nil
func (dir *RegistryDirectory) cacheInvoker(url *common.URL) protocol.Invoker {
	dir.overrideUrl(dir.GetDirectoryUrl())
	referenceUrl := dir.GetDirectoryUrl().SubURL

	if url == nil && dir.cacheOriginUrl != nil {
		url = dir.cacheOriginUrl
	} else {
		dir.cacheOriginUrl = url
	}
	if url == nil {
		logger.Error("URL is nil ,pls check if service url is subscribe successfully!")
		return nil
	}
	// check the url's protocol is equal to the protocol which is configured in reference config or referenceUrl is not care about protocol
	if url.Protocol == referenceUrl.Protocol || referenceUrl.Protocol == "" {
		newUrl := common.MergeUrl(url, referenceUrl)
		dir.overrideUrl(newUrl)
		if cacheInvoker, ok := dir.cacheInvokersMap.Load(newUrl.Key()); !ok {
			logger.Debugf("service will be added in cache invokers: invokers url is  %s!", newUrl)
			newInvoker := extension.GetProtocol(protocolwrapper.FILTER).Refer(newUrl)
			if newInvoker != nil {
				dir.cacheInvokersMap.Store(newUrl.Key(), newInvoker)
			}
		} else {
			// if cached invoker has the same URL with the new URL, then no need to re-refer, and no need to destroy
			// the old invoker.
			if common.IsEquals(newUrl, cacheInvoker.(protocol.Invoker).GetUrl()) {
				return nil
			}

			logger.Debugf("service will be updated in cache invokers: new invoker url is %s, old invoker url is %s", newUrl, cacheInvoker.(protocol.Invoker).GetUrl())
			newInvoker := extension.GetProtocol(protocolwrapper.FILTER).Refer(newUrl)
			if newInvoker != nil {
				dir.cacheInvokersMap.Store(newUrl.Key(), newInvoker)
				return cacheInvoker.(protocol.Invoker)
			}
		}
	}
	return nil
}

// List selected protocol invokers from the directory
func (dir *RegistryDirectory) List(invocation protocol.Invocation) []protocol.Invoker {
	invokers := dir.cacheInvokers
	routerChain := dir.RouterChain()

	if routerChain == nil {
		return invokers
	}
	return routerChain.Route(dir.consumerURL, invocation)
}

// IsAvailable  whether the directory is available
func (dir *RegistryDirectory) IsAvailable() bool {
	if !dir.BaseDirectory.IsAvailable() {
		return dir.BaseDirectory.IsAvailable()
	}

	for _, ivk := range dir.cacheInvokers {
		if ivk.IsAvailable() {
			return true
		}
	}

	return false
}

// Destroy method
func (dir *RegistryDirectory) Destroy() {
	// TODO:unregister & unsubscribe
	dir.BaseDirectory.Destroy(func() {
		invokers := dir.cacheInvokers
		dir.cacheInvokers = []protocol.Invoker{}
		for _, ivk := range invokers {
			ivk.Destroy()
		}
	})
}

func (dir *RegistryDirectory) overrideUrl(targetUrl *common.URL) {
	doOverrideUrl(dir.configurators, targetUrl)
	doOverrideUrl(dir.consumerConfigurationListener.Configurators(), targetUrl)
	doOverrideUrl(dir.referenceConfigurationListener.Configurators(), targetUrl)
}

func (dir *RegistryDirectory) getConsumerUrl(c *common.URL) *common.URL {
	processID := fmt.Sprintf("%d", os.Getpid())
	localIP := common.GetLocalIp()

	params := url.Values{}
	c.RangeParams(func(key, value string) bool {
		params.Add(key, value)
		return true
	})

	params.Add("pid", processID)
	params.Add("ip", localIP)
	params.Add("protocol", c.Protocol)

	return common.NewURLWithOptions(common.WithProtocol("consumer"), common.WithIp(localIP), common.WithPath(c.Path),
		common.WithParams(params))
}

func doOverrideUrl(configurators []config_center.Configurator, targetUrl *common.URL) {
	for _, v := range configurators {
		v.Configure(targetUrl)
	}
}

type referenceConfigurationListener struct {
	registry.BaseConfigurationListener
	directory *RegistryDirectory
	url       *common.URL
}

func newReferenceConfigurationListener(dir *RegistryDirectory, url *common.URL) *referenceConfigurationListener {
	listener := &referenceConfigurationListener{directory: dir, url: url}
	listener.InitWith(
		url.EncodedServiceKey()+constant.CONFIGURATORS_SUFFIX,
		listener,
		extension.GetDefaultConfiguratorFunc(),
	)
	return listener
}

// Process handle events and update Invokers
func (l *referenceConfigurationListener) Process(event *config_center.ConfigChangeEvent) {
	l.BaseConfigurationListener.Process(event)
	// FIXME: this doesn't trigger dir.overrideUrl()
	l.directory.refreshInvokers()
}

type consumerConfigurationListener struct {
	registry.BaseConfigurationListener
	listeners []registry.NotifyListener
	directory *RegistryDirectory
}

func newConsumerConfigurationListener(dir *RegistryDirectory) *consumerConfigurationListener {
	listener := &consumerConfigurationListener{directory: dir}
	listener.InitWith(
		config.GetConsumerConfig().ApplicationConfig.Name+constant.CONFIGURATORS_SUFFIX,
		listener,
		extension.GetDefaultConfiguratorFunc(),
	)
	return listener
}

func (l *consumerConfigurationListener) addNotifyListener(listener registry.NotifyListener) {
	l.listeners = append(l.listeners, listener)
}

// Process handles events from Configuration Center and update Invokers
func (l *consumerConfigurationListener) Process(event *config_center.ConfigChangeEvent) {
	l.BaseConfigurationListener.Process(event)
	// FIXME: this doesn't trigger dir.overrideUrl()
	l.directory.refreshInvokers()
}
