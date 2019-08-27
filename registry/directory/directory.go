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
	"sync"
	"time"
)

import (
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/cluster/directory"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/protocol/protocolwrapper"
	"github.com/apache/dubbo-go/registry"
	"github.com/apache/dubbo-go/remoting"
)

const (
	RegistryConnDelay = 3
)

type Options struct {
	serviceTTL time.Duration
}

type Option func(*Options)

type registryDirectory struct {
	directory.BaseDirectory
	cacheInvokers    []protocol.Invoker
	listenerLock     sync.Mutex
	serviceType      string
	registry         registry.Registry
	cacheInvokersMap *sync.Map //use sync.map
	Options
}

func NewRegistryDirectory(url *common.URL, registry registry.Registry, opts ...Option) (*registryDirectory, error) {
	options := Options{
		//default 300s
		serviceTTL: time.Duration(300e9),
	}
	for _, opt := range opts {
		opt(&options)
	}
	if url.SubURL == nil {
		return nil, perrors.Errorf("url is invalid, suburl can not be nil")
	}
	return &registryDirectory{
		BaseDirectory:    directory.NewBaseDirectory(url),
		cacheInvokers:    []protocol.Invoker{},
		cacheInvokersMap: &sync.Map{},
		serviceType:      url.SubURL.Service(),
		registry:         registry,
		Options:          options,
	}, nil
}

//subscribe from registry
func (dir *registryDirectory) Subscribe(url common.URL) {
	for {
		if !dir.registry.IsAvailable() {
			logger.Warnf("event listener game over.")
			return
		}

		listener, err := dir.registry.Subscribe(url)
		if err != nil {
			if !dir.registry.IsAvailable() {
				logger.Warnf("event listener game over.")
				return
			}
			logger.Warnf("getListener() = err:%v", perrors.WithStack(err))
			time.Sleep(time.Duration(RegistryConnDelay) * time.Second)
			continue
		}

		for {
			if serviceEvent, err := listener.Next(); err != nil {
				logger.Warnf("Selector.watch() = error{%v}", perrors.WithStack(err))
				listener.Close()
				time.Sleep(time.Duration(RegistryConnDelay) * time.Second)
				return
			} else {
				logger.Infof("update begin, service event: %v", serviceEvent.String())
				go dir.update(serviceEvent)
			}

		}

	}
}

//subscribe service from registry , and update the cacheServices
func (dir *registryDirectory) update(res *registry.ServiceEvent) {
	if res == nil {
		return
	}

	logger.Debugf("registry update, result{%s}", res)

	logger.Debugf("update service name: %s!", res.Service)

	dir.refreshInvokers(res)
}

func (dir *registryDirectory) refreshInvokers(res *registry.ServiceEvent) {

	switch res.Action {
	case remoting.EventTypeAdd:
		//dir.cacheService.EventTypeAdd(res.Path, dir.serviceTTL)
		dir.cacheInvoker(res.Service)
	case remoting.EventTypeDel:
		//dir.cacheService.EventTypeDel(res.Path, dir.serviceTTL)
		dir.uncacheInvoker(res.Service)
		logger.Infof("selector delete service url{%s}", res.Service)
	default:
		return
	}

	newInvokers := dir.toGroupInvokers()

	dir.listenerLock.Lock()
	defer dir.listenerLock.Unlock()
	dir.cacheInvokers = newInvokers
}

func (dir *registryDirectory) toGroupInvokers() []protocol.Invoker {

	newInvokersList := []protocol.Invoker{}
	groupInvokersMap := make(map[string][]protocol.Invoker)
	groupInvokersList := []protocol.Invoker{}

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
	if len(groupInvokersMap) == 1 {
		//len is 1 it means no group setting ,so do not need cluster again
		for _, invokers := range groupInvokersMap {
			groupInvokersList = invokers
		}
	} else {
		for _, invokers := range groupInvokersMap {
			staticDir := directory.NewStaticDirectory(invokers)
			cluster := extension.GetCluster(dir.GetUrl().SubURL.GetParam(constant.CLUSTER_KEY, constant.DEFAULT_CLUSTER))
			groupInvokersList = append(groupInvokersList, cluster.Join(staticDir))
		}
	}

	return groupInvokersList
}

func (dir *registryDirectory) uncacheInvoker(url common.URL) {
	logger.Debugf("service will be deleted in cache invokers: invokers key is  %s!", url.Key())
	dir.cacheInvokersMap.Delete(url.Key())
}

func (dir *registryDirectory) cacheInvoker(url common.URL) {
	referenceUrl := dir.GetUrl().SubURL
	//check the url's protocol is equal to the protocol which is configured in reference config or referenceUrl is not care about protocol
	if url.Protocol == referenceUrl.Protocol || referenceUrl.Protocol == "" {
		url = common.MergeUrl(url, referenceUrl)

		if _, ok := dir.cacheInvokersMap.Load(url.Key()); !ok {
			logger.Debugf("service will be added in cache invokers: invokers key is  %s!", url.Key())
			newInvoker := extension.GetProtocol(protocolwrapper.FILTER).Refer(url)
			if newInvoker != nil {
				dir.cacheInvokersMap.Store(url.Key(), newInvoker)
			}
		}
	}
}

//select the protocol invokers from the directory
func (dir *registryDirectory) List(invocation protocol.Invocation) []protocol.Invoker {
	//TODO:router
	return dir.cacheInvokers
}

func (dir *registryDirectory) IsAvailable() bool {
	if !dir.BaseDirectory.IsAvailable() {
		return dir.BaseDirectory.IsAvailable()
	} else {
		for _, ivk := range dir.cacheInvokers {
			if ivk.IsAvailable() {
				return true
			}
		}
	}
	return false
}

func (dir *registryDirectory) Destroy() {
	//TODO:unregister & unsubscribe
	dir.BaseDirectory.Destroy(func() {
		for _, ivk := range dir.cacheInvokers {
			ivk.Destroy()
		}
		dir.cacheInvokers = []protocol.Invoker{}
	})
}
