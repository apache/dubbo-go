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
	"github.com/dubbogo/gost/log/logger"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/directory"
	"dubbo.apache.org/dubbo-go/v3/cluster/directory/base"
	"dubbo.apache.org/dubbo-go/v3/cluster/directory/static"
	"dubbo.apache.org/dubbo-go/v3/cluster/router/chain"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	_ "dubbo.apache.org/dubbo-go/v3/config_center/configurator"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/protocolwrapper"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

func init() {
	extension.SetDefaultRegistryDirectory(NewRegistryDirectory)
}

// RegistryDirectory implementation of Directory:
// Invoker list returned from this Directory's list method have been filtered by Routers
type RegistryDirectory struct {
	base.Directory
	cacheInvokers                  []protocol.Invoker
	invokersLock                   sync.RWMutex
	serviceType                    string
	registry                       registry.Registry
	cacheInvokersMap               *sync.Map // use sync.map
	consumerURL                    *common.URL
	cacheOriginUrl                 *common.URL
	configurators                  []config_center.Configurator
	consumerConfigurationListener  *consumerConfigurationListener
	referenceConfigurationListener *referenceConfigurationListener
	registerLock                   sync.Mutex // this lock if for register
}

// NewRegistryDirectory will create a new RegistryDirectory
func NewRegistryDirectory(url *common.URL, registry registry.Registry) (directory.Directory, error) {
	if url.SubURL == nil {
		return nil, perrors.Errorf("url is invalid, suburl can not be nil")
	}
	logger.Debugf("new RegistryDirectory for service :%s.", url.Key())
	dir := &RegistryDirectory{
		Directory:        base.NewDirectory(url),
		cacheInvokers:    []protocol.Invoker{},
		cacheInvokersMap: &sync.Map{},
		serviceType:      url.SubURL.Service(),
		registry:         registry,
	}

	dir.consumerURL = dir.getConsumerUrl(url.SubURL)

	if routerChain, err := chain.NewRouterChain(); err == nil {
		dir.Directory.SetRouterChain(routerChain)
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

	var oldInvoker []protocol.Invoker
	if event != nil {
		oldInvoker, _ = dir.cacheInvokerByEvent(event)
	}
	dir.setNewInvokers()
	for _, v := range oldInvoker {
		if v != nil {
			v.Destroy()
		}
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
		// MergeURL is executed once and put the result into Event. After this, the key will get from Event.Key().
		newUrl := dir.convertUrl(event)
		newUrl = common.MergeURL(newUrl, referenceUrl)
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
			logger.Debugf("[Registry Directory] registry update, result{%s}", event)
			if event != nil && event.Service != nil {
				logger.Infof("[Registry Directory] selector add service url{%s}", event.Service.String())
			}
			if event != nil && event.Service != nil && constant.RouterProtocol == event.Service.Protocol {
				dir.configRouters()
			}
			if oldInvoker, _ := dir.doCacheInvoker(event.Service, event); oldInvoker != nil {
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
	newUrl := common.MergeURL(event.Service, referenceUrl)
	event.Update(newUrl)
	return event.Key()
}

// setNewInvokers groups the invokers from the cache first, then set the result to both directory and router chain.
func (dir *RegistryDirectory) setNewInvokers() {
	newInvokers := dir.toGroupInvokers()
	dir.invokersLock.Lock()
	defer dir.invokersLock.Unlock()
	dir.cacheInvokers = newInvokers
	dir.RouterChain().SetInvokers(newInvokers)
}

// cacheInvokerByEvent caches invokers from the service event
func (dir *RegistryDirectory) cacheInvokerByEvent(event *registry.ServiceEvent) ([]protocol.Invoker, error) {
	// judge is override or others
	if event != nil {

		switch event.Action {
		case remoting.EventTypeAdd, remoting.EventTypeUpdate:
			u := dir.convertUrl(event)
			logger.Infof("[Registry Directory] selector add service url{%s}", event.Service)
			if u != nil && constant.RouterProtocol == u.Protocol {
				dir.configRouters()
			}
			return []protocol.Invoker{dir.cacheInvoker(u, event)}, nil
		case remoting.EventTypeDel:
			logger.Infof("[Registry Directory] selector delete service url{%s}", event.Service)
			return dir.uncacheInvoker(event), nil
		default:
			return nil, fmt.Errorf("illegal event type: %v", event.Action)
		}
	}
	return nil, nil
}

// configRouters configures dynamic routers into the router chain, but, the current impl is incorrect, see FIXME above.
func (dir *RegistryDirectory) configRouters() {
}

// convertUrl processes override:// and router://
func (dir *RegistryDirectory) convertUrl(res *registry.ServiceEvent) *common.URL {
	ret := res.Service
	if ret.Protocol == constant.OverrideProtocol || // 1.for override url in 2.6.x
		ret.GetParam(constant.CategoryKey, constant.DefaultCategory) == constant.ConfiguratorsCategory {
		dir.configurators = append(dir.configurators, extension.GetDefaultConfigurator(ret))
		ret = nil
	} else if ret.Protocol == constant.RouterProtocol || // 2.for router
		ret.GetParam(constant.CategoryKey, constant.DefaultCategory) == constant.RouterCategory {
		ret = nil
	}
	return ret
}

func (dir *RegistryDirectory) toGroupInvokers() []protocol.Invoker {
	groupInvokersMap := make(map[string][]protocol.Invoker)

	dir.cacheInvokersMap.Range(func(key, value interface{}) bool {
		invoker := value.(protocol.Invoker)
		group := invoker.GetURL().GetParam(constant.GroupKey, "")
		groupInvokersMap[group] = append(groupInvokersMap[group], invoker)
		return true
	})

	groupInvokersList := make([]protocol.Invoker, 0, len(groupInvokersMap))
	if len(groupInvokersMap) == 1 {
		// len is 1 it means no group setting ,so do not need cluster again
		for _, invokers := range groupInvokersMap {
			groupInvokersList = invokers
		}
	} else {
		for _, invokers := range groupInvokersMap {
			staticDir := static.NewDirectory(invokers)
			clusterKey := dir.GetURL().SubURL.GetParam(constant.ClusterKey, constant.DefaultCluster)
			cluster, err := extension.GetCluster(clusterKey)
			if err != nil {
				panic(err)
			}
			err = staticDir.BuildRouterChain(invokers)
			if err != nil {
				logger.Error(err)
				continue
			}
			groupInvokersList = append(groupInvokersList, cluster.Join(staticDir))
		}
	}

	return groupInvokersList
}

func (dir *RegistryDirectory) uncacheInvokerWithClusterID(clusterID string) []protocol.Invoker {
	logger.Debugf("All service will be deleted in cache invokers with clusterID %s!", clusterID)
	invokerKeys := make([]string, 0)
	dir.cacheInvokersMap.Range(func(key, cacheInvoker interface{}) bool {
		if cacheInvoker.(protocol.Invoker).GetURL().GetParam(constant.MeshClusterIDKey, "") == clusterID {
			invokerKeys = append(invokerKeys, key.(string))
		}
		return true
	})
	uncachedInvokers := make([]protocol.Invoker, 0)
	for _, v := range invokerKeys {
		uncachedInvokers = append(uncachedInvokers, dir.uncacheInvokerWithKey(v))
	}
	return uncachedInvokers
}

// uncacheInvoker will return abandoned Invoker, if no Invoker to be abandoned, return nil
func (dir *RegistryDirectory) uncacheInvoker(event *registry.ServiceEvent) []protocol.Invoker {
	if clusterID := event.Service.GetParam(constant.MeshClusterIDKey, ""); event.Service.Location == constant.MeshAnyAddrMatcher && clusterID != "" {
		dir.uncacheInvokerWithClusterID(clusterID)
	}
	return []protocol.Invoker{dir.uncacheInvokerWithKey(event.Key())}
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
func (dir *RegistryDirectory) cacheInvoker(url *common.URL, event *registry.ServiceEvent) protocol.Invoker {
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
		newUrl := common.MergeURL(url, referenceUrl)
		dir.overrideUrl(newUrl)
		event.Update(newUrl)
		if v, ok := dir.doCacheInvoker(newUrl, event); ok {
			return v
		}
	}
	return nil
}

func (dir *RegistryDirectory) doCacheInvoker(newUrl *common.URL, event *registry.ServiceEvent) (protocol.Invoker, bool) {
	key := event.Key()
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
		if common.GetCompareURLEqualFunc()(newUrl, cacheInvoker.(protocol.Invoker).GetURL()) {
			return nil, true
		}

		logger.Debugf("service will be updated in cache invokers: new invoker url is %s, old invoker url is %s", newUrl, cacheInvoker.(protocol.Invoker).GetURL())
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
	routerChain := dir.RouterChain()

	if routerChain == nil {
		dir.invokersLock.RLock()
		defer dir.invokersLock.RUnlock()
		return dir.cacheInvokers
	}
	return routerChain.Route(dir.consumerURL, invocation)
}

// IsAvailable  whether the directory is available
func (dir *RegistryDirectory) IsAvailable() bool {
	if !dir.Directory.IsAvailable() {
		return dir.Directory.IsAvailable()
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
	dir.Directory.Destroy(func() {
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
		url.EncodedServiceKey()+constant.ConfiguratorSuffix,
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
	application := config.GetRootConfig().Application
	listener.InitWith(
		application.Name+constant.ConfiguratorSuffix,
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
