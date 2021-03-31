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
	//serviceKey                     string
	//forbidden                      atomic.Bool
	registerLock sync.Mutex // this lock if for register
}

// NewRegistryDirectory will create a new RegistryDirectory
func NewRegistryDirectory(url *common.URL, registry registry.Registry) (cluster.Directory, error) {
	if url.SubURL == nil {
		return nil, perrors.Errorf("url is invalid, suburl can not be nil")
	}
	logger.Debugf("new RegistryDirectory for service :%s.", url.Key())
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
	logger.Debugf("subscribe service :%s for RegistryDirectory.", url.Key())
	dir.consumerConfigurationListener.addNotifyListener(dir)
	dir.referenceConfigurationListener = newReferenceConfigurationListener(dir, url)
	if err := dir.registry.Subscribe(url, dir); err != nil {
		logger.Error("registry.Subscribe(url:%v, dir:%v) = error:%v", url, dir, err)
	}
}

// Notify monitor changes from registry,and update the cacheServices
func (dir *RegistryDirectory) Notify(event *registry.ServiceEvent) {
	if event == nil {
		return
	}
	dir.refreshInvokers(event)
}

// NotifyAll notify the events that are complete Service Event List.
// After notify the address, the callback func will be invoked.
func (dir *RegistryDirectory) NotifyAll(events []*registry.ServiceEvent, callback func()) {
	go dir.refreshAllInvokers(events, callback)
}

// refreshInvokers refreshes service's events.
func (dir *RegistryDirectory) refreshInvokers(event *registry.ServiceEvent) {
	if event != nil {
		logger.Debugf("refresh invokers with %+v", event)
	} else {
		logger.Debug("refresh invokers with nil")
	}

	var oldInvoker protocol.Invoker
	if event != nil {
		oldInvoker, _ = dir.cacheInvokerByEvent(event)
	}
	dir.setNewInvokers()
	if oldInvoker != nil {
		oldInvoker.Destroy()
	}
}

// refreshAllInvokers the argument is the complete list of the service events,  we can safely assume any cached invoker
// not in the incoming list can be removed.  The Action of serviceEvent should be EventTypeUpdate.
func (dir *RegistryDirectory) refreshAllInvokers(events []*registry.ServiceEvent, callback func()) {
	var (
		oldInvokers []protocol.Invoker
		addEvents   []*registry.ServiceEvent
	)
	dir.overrideUrl(dir.GetDirectoryUrl())
	referenceUrl := dir.GetDirectoryUrl().SubURL

	// loop the events to check the Action should be EventTypeUpdate.
	for _, event := range events {
		if event.Action != remoting.EventTypeUpdate {
			panic("Your implements of register center is wrong, " +
				"please check the Action of ServiceEvent should be EventTypeUpdate")
		}
		// Originally it will Merge URL many times, now we just execute once.
		// MergeUrl is executed once and put the result into Event. After this, the key will get from Event.Key().
		newUrl := dir.convertUrl(event)
		newUrl = common.MergeUrl(newUrl, referenceUrl)
		dir.overrideUrl(newUrl)
		event.Update(newUrl)
	}
	// After notify all addresses, do some callback.
	defer callback()
	func() {
		// this lock is work at batch update of InvokeCache
		dir.registerLock.Lock()
		defer dir.registerLock.Unlock()
		// get need clear invokers from original invoker list
		dir.cacheInvokersMap.Range(func(k, v interface{}) bool {
			if !dir.eventMatched(k.(string), events) {
				// delete unused invoker from cache
				if invoker := dir.uncacheInvokerWithKey(k.(string)); invoker != nil {
					oldInvokers = append(oldInvokers, invoker)
				}
			}
			return true
		})
		// get need add invokers from events
		for _, event := range events {
			// Get the key from Event.Key()
			if _, ok := dir.cacheInvokersMap.Load(event.Key()); !ok {
				addEvents = append(addEvents, event)
			}
		}
		// loop the updateEvents
		for _, event := range addEvents {
			logger.Debugf("registry update, result{%s}", event)
			if event.Service != nil {
				logger.Infof("selector add service url{%s}", event.Service.String())
			}
			if event != nil && event.Service != nil && constant.ROUTER_PROTOCOL == event.Service.Protocol {
				dir.configRouters()
			}
			if oldInvoker, _ := dir.doCacheInvoker(event.Service); oldInvoker != nil {
				oldInvokers = append(oldInvokers, oldInvoker)
			}
		}
	}()
	dir.setNewInvokers()
	// destroy unused invokers
	for _, invoker := range oldInvokers {
		go invoker.Destroy()
	}
}

// eventMatched checks if a cached invoker appears in the incoming invoker list, if no, then it is safe to remove.
func (dir *RegistryDirectory) eventMatched(key string, events []*registry.ServiceEvent) bool {
	for _, event := range events {
		if dir.invokerCacheKey(event) == key {
			return true
		}
	}
	return false
}

// invokerCacheKey generates the key in the cache for a given ServiceEvent.
func (dir *RegistryDirectory) invokerCacheKey(event *registry.ServiceEvent) string {
	// If the url is merged, then return Event.Key() directly.
	if event.Updated() {
		return event.Key()
	}
	referenceUrl := dir.GetDirectoryUrl().SubURL
	newUrl := common.MergeUrl(event.Service, referenceUrl)
	event.Update(newUrl)
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
			if u != nil && constant.ROUTER_PROTOCOL == u.Protocol {
				dir.configRouters()
			}
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

		groupInvokersMap[group] = append(groupInvokersMap[group], invoker)
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
	protocol.RemoveUrlKeyUnhealthyStatus(key)
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
		if v, ok := dir.doCacheInvoker(newUrl); ok {
			return v
		}
	}
	return nil
}

func (dir *RegistryDirectory) doCacheInvoker(newUrl *common.URL) (protocol.Invoker, bool) {
	key := newUrl.Key()
	if cacheInvoker, ok := dir.cacheInvokersMap.Load(key); !ok {
		logger.Debugf("service will be added in cache invokers: invokers url is  %s!", newUrl)
		newInvoker := extension.GetProtocol(protocolwrapper.FILTER).Refer(newUrl)
		if newInvoker != nil {
			dir.cacheInvokersMap.Store(key, newInvoker)
		} else {
			logger.Warnf("service will be added in cache invokers fail, result is null, invokers url is %+v", newUrl.String())
		}
	} else {
		// if cached invoker has the same URL with the new URL, then no need to re-refer, and no need to destroy
		// the old invoker.
		if common.GetCompareURLEqualFunc()(newUrl, cacheInvoker.(protocol.Invoker).GetUrl()) {
			return nil, true
		}

		logger.Debugf("service will be updated in cache invokers: new invoker url is %s, old invoker url is %s", newUrl, cacheInvoker.(protocol.Invoker).GetUrl())
		newInvoker := extension.GetProtocol(protocolwrapper.FILTER).Refer(newUrl)
		if newInvoker != nil {
			dir.cacheInvokersMap.Store(key, newInvoker)
			return cacheInvoker.(protocol.Invoker), true
		} else {
			logger.Warnf("service will be updated in cache invokers fail, result is null, invokers url is %+v", newUrl.String())
		}
	}
	return nil, false
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
	l.directory.refreshInvokers(nil)
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
	l.directory.refreshInvokers(nil)
}
